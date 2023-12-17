package pandoc

import (
	"fmt"
	"io"
	"math"
	"math/bits"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Simple streaming JSON parser suitable for parsing pandoc JSON AST.
// It does not support unicode escapes in strings, but pandoc never
// produces them.
//
// On the other hand, it's much faster than encoding/json and
// also does not allocate that much memory.

type token int

const (
	tokErr token = iota - 1
	tokEOF
	tokLBrack
	tokRBrack
	tokLBrace
	tokRBrace
	tokComma
	tokColon
	tokStr
	tokNumber
	tokTrue
	tokFalse
	tokNull
)

func (t token) String() string {
	switch t {
	case tokLBrack:
		return "["
	case tokRBrack:
		return "]"
	case tokLBrace:
		return "{"
	case tokRBrace:
		return "}"
	case tokComma:
		return ","
	case tokColon:
		return ":"
	case tokStr:
		return "string"
	case tokNumber:
		return "number"
	case tokTrue:
		return "true"
	case tokFalse:
		return "false"
	case tokNull:
		return "null"
	case tokEOF:
		return "EOF"
	default:
		return fmt.Sprintf("token(%d)", int(t))
	}
}

type scanner struct {
	r      io.Reader       // reader
	buf    []byte          // current buffer
	sb     strings.Builder // string buffer (for large and escaped strings)
	err    error           // an error
	off    int             // offset of the current buffer in the reader
	pos    int             // next unread byte in the buffer
	str    int             // start of the current string/atom/number. -1 if there is no any.
	num    int64           // parsed number
	intnum bool            // true if the number is an integer
}

func (p *scanner) stringInBuffer() bool {
	return p.str >= 0
}

func (p *scanner) string() string {
	if p.str >= 0 {
		return string(p.buf[p.str : p.pos-1])
	} else {
		return p.sb.String()
	}
}

func (p *scanner) expectString(s string) error {
	if t := p.peek(); t != tokStr {
		if p.err != nil {
			return fmt.Errorf("expected %s, got %s at %d (%w)", s, t.String(), p.off+p.pos, p.err)
		} else {
			return fmt.Errorf("expected %s, got %s at %d", s, t.String(), p.off+p.pos)
		}
	} else {
		o := p.off + p.pos
		p.next()
		if p.sb.Len() != 0 && p.sb.String() != s {
			return fmt.Errorf("expected string %s, got %s at %d", s, p.sb.String(), o)
		} else if string(p.buf[p.str:p.pos-1]) != s {
			return fmt.Errorf("expected string %s, got %s at %d", s, string(p.buf[p.str:p.pos-1]), o)
		}
	}
	return nil
}

func (p *scanner) expect(tok token) error {
	if t := p.peek(); t != tok {
		if t == tokErr {
			p.err = fmt.Errorf("expected %s, got %s at %d (%w) \"%s\"", tok.String(), t.String(), p.off+p.pos, p.err, p.buf[p.pos:])
		} else {
			p.err = fmt.Errorf("expected %s, got %s at %d \"%s\"", tok.String(), t.String(), p.off+p.pos, p.buf[p.pos:])
			panic(p.err.Error())
		}
		return p.err
	}
	p.next()
	return nil
}

func (p *scanner) current() int {
	return p.off + p.pos
}

func (p *scanner) numberIsInt() bool {
	return p.intnum
}

func (p *scanner) int() int64 {
	if !p.intnum {
		return int64(math.Float64frombits(uint64(p.num)))
	} else {
		return p.num
	}
}

func (p *scanner) float() float64 {
	if p.intnum {
		return float64(p.num)
	} else {
		return math.Float64frombits(uint64(p.num))
	}
}

// spills the small string buffer into the string buffer.
func (p *scanner) spillstr() {
	p.sb.Write(p.buf[p.str:p.pos])
	p.str = -1
}

// ensures that there is at least size bytes in the buffer after the current position.
// if there is not enough space, it shifts the buffer to the left and fills it with new data.
// if str is > 0, shifts the buffer to the left so that str becomes 0.
// if str is already 0, it stores the current [str:pos] slice in the string buffer.
func (p *scanner) ensure(size int) bool {
	var bs = len(p.buf)
	if bs-p.pos >= size {
		return true
	} else if p.str > 0 {
		copy(p.buf, p.buf[p.str:])
		bs -= p.str
		p.pos -= p.str
		p.off += p.str
		p.str = 0
	} else if p.str == 0 {
		p.sb.Write(p.buf[:p.pos])
		p.off += p.pos
		bs -= p.pos
		p.pos = 0
	} else if p.str < 0 {
		p.off += p.pos
		bs -= p.pos
		p.pos = 0
	}
	n, err := p.r.Read(p.buf[bs:cap(p.buf)])
	p.buf = p.buf[:bs+n]
	if err != nil {
		p.err = err
	}
	if len(p.buf)-p.pos >= size {
		return true
	}
	return false
}

func (p *scanner) init(r io.Reader) {
	var buf []byte
	if cap(p.buf) == 0 {
		buf = make([]byte, 0, 128)
	} else {
		buf = p.buf[:0]
	}
	*p = scanner{r: r, buf: buf}
}

func (p *scanner) skipws() {
	for p.ensure(1) {
		switch p.buf[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *scanner) peek() token {
	p.skipws()
	if !p.ensure(1) {
		return tokEOF
	}
	switch p.buf[p.pos] {
	case ',':
		return tokComma
	case ':':
		return tokColon
	case '[':
		return tokLBrack
	case ']':
		return tokRBrack
	case '{':
		return tokLBrace
	case '}':
		return tokRBrace
	case '"':
		return tokStr
	case 'n':
		if p.ensure(4) && string(p.buf[p.pos:p.pos+4]) == "null" {
			return tokNull
		}
	case 't':
		if p.ensure(4) && string(p.buf[p.pos:p.pos+4]) == "true" {
			return tokTrue
		}
	case 'f':
		if p.ensure(5) && string(p.buf[p.pos:p.pos+5]) == "false" {
			return tokFalse
		}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return tokNumber
	}
	return tokErr
}

func (p *scanner) next() token {
	p.str = -1
scan:
	if !p.ensure(1) {
		return tokEOF
	}
	switch c := p.buf[p.pos]; c {
	case '[':
		p.pos++
		return tokLBrack
	case ']':
		p.pos++
		return tokRBrack
	case '{':
		p.pos++
		return tokLBrace
	case '}':
		p.pos++
		return tokRBrace
	case ',':
		p.pos++
		return tokComma
	case ':':
		p.pos++
		return tokColon
	case 'n':
		p.str = p.pos
		if p.ensure(4) && string(p.buf[p.pos:p.pos+4]) == "null" {
			p.pos += 4
			return tokNull
		} else {
			p.err = fmt.Errorf("unexpected character %c at %d", c, p.off+p.pos)
			return tokErr
		}
	case 't':
		p.str = p.pos
		if p.ensure(4) && string(p.buf[p.pos:p.pos+4]) == "true" {
			p.pos += 4
			return tokTrue
		} else {
			p.err = fmt.Errorf("unexpected character %c at %d", c, p.off+p.pos)
			return tokErr
		}
	case 'f':
		p.str = p.pos
		if p.ensure(5) && string(p.buf[p.pos:p.pos+5]) == "false" {
			p.pos += 5
			return tokFalse
		} else {
			p.err = fmt.Errorf("unexpected character %c at %d", c, p.off+p.pos)
			return tokErr
		}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNum(byte(c))
	case '"':
		p.pos++
		return p.parseStr()
	case ' ', '\t', '\n', '\r':
		p.pos++
		goto scan
	default:
		p.err = fmt.Errorf("unexpected character %c at %d", c, p.off+p.pos)
		return tokErr
	}
}

func (p *scanner) parseNum(c byte) token {
	var (
		sign int
		num  uint64
		flt  bool
		off  = p.off
	)
	p.sb.Reset()
	p.str = p.pos
	if c == '-' {
		p.pos++
		if !p.ensure(1) {
			p.err = fmt.Errorf("unexpected EOF at %d", off)
			return tokErr
		}
		if c = p.buf[p.pos]; c < '0' || c > '9' {
			p.err = fmt.Errorf("invalid number literal at %d", off)
			return tokErr
		}
		sign = -1
	}
	if c == '0' {
		p.pos++
		if p.ensure(1) {
			c = p.buf[p.pos]
		} else {
			c = 0
		}
		switch {
		case c >= '0' && c <= '9':
			p.err = fmt.Errorf("invalid number literal at %d", off)
			return tokErr
		case c == '.':
			flt = true
			p.pos++
			goto frac
		case c == 'e' || c == 'E':
			flt = true
			p.pos++
			goto exp
		default:
			if sign < 0 {
				p.num = int64(math.Float64bits(math.Copysign(0, -1)))
				p.intnum = false
				return tokNumber
			} else {
				p.num = 0
				p.intnum = true
				return tokNumber
			}
		}
	} else {
		p.pos++
		num = uint64(c - '0')
	}
	for {
		if !p.ensure(1) {
			goto done
		}
		c = p.buf[p.pos]
		if c >= '0' && c <= '9' {
			if !flt {
				if h, l := bits.Mul64(num, 10); h != 0 {
					flt = true
				} else if l, h = bits.Add64(l, uint64(c-'0'), 0); h != 0 {
					flt = true
				} else {
					num = l
				}
			}
			p.pos++
		} else if c == '.' {
			p.pos++
			flt = true
			goto frac
		} else if c == 'e' || c == 'E' {
			p.pos++
			flt = true
			goto exp
		} else {
			goto done
		}
	}
frac:
	if !p.ensure(1) || p.buf[p.pos] < '0' || p.buf[p.pos] > '9' {
		p.err = fmt.Errorf("invalid number literal at %d", off)
		return tokErr
	}
	c = p.buf[p.pos]
	for {
		if c >= '0' && c <= '9' {
			p.pos++
		} else if c == 'e' || c == 'E' {
			p.pos++
			goto exp
		} else {
			goto done
		}
		if !p.ensure(1) {
			goto done
		}
		c = p.buf[p.pos]
	}
exp:
	if !p.ensure(1) {
		p.err = fmt.Errorf("invalid number literal at %d", off)
		return tokErr
	}
	c = p.buf[p.pos]
	if c == '+' || c == '-' {
		p.pos++
		if !p.ensure(1) {
			p.err = fmt.Errorf("invalid number literal at %d", off)
			return tokErr
		}
		c = p.buf[p.pos]
	}
	if c < '0' || c > '9' {
		p.err = fmt.Errorf("invalid number literal at %d", off)
		return tokErr
	}
	for {
		if c >= '0' && c <= '9' {
			p.pos++
		} else {
			goto done
		}
		if !p.ensure(1) {
			goto done
		}
		c = p.buf[p.pos]
	}
done:
	if !flt {
		if sign < 0 {
			if num > math.MaxInt64+1 {
				p.num = int64(math.Float64bits(math.Copysign(float64(num), -1)))
				p.intnum = false
				return tokNumber
			} else {
				p.num = -int64(num)
				p.intnum = true
				return tokNumber
			}
		} else {
			if num > math.MaxInt64 {
				p.num = int64(math.Float64bits(float64(num)))
				p.intnum = false
				return tokNumber
			} else {
				p.num = int64(num)
				p.intnum = true
				return tokNumber
			}
		}
	} else {
		var (
			float float64
			err   error
		)
		if p.sb.Len() != 0 {
			p.spillstr()
			float, err = strconv.ParseFloat(p.sb.String(), 64)
		} else {
			float, err = strconv.ParseFloat(string(p.buf[p.str:p.pos]), 64)
		}
		if err != nil {
			p.err = fmt.Errorf("invalid number literal at %d (%w)", off, err)
			return tokErr
		} else {
			p.num = int64(math.Float64bits(float))
			p.intnum = false
			return tokNumber
		}
	}
}

func (p *scanner) parseStr() token {
	p.sb.Reset()
scan:
	p.str = p.pos
	for p.ensure(1) {
		if c := p.buf[p.pos]; c == '"' {
			if p.sb.Len() != 0 {
				p.spillstr()
			}
			p.pos++
			return tokStr
		} else if c == '\\' {
			p.spillstr()
			p.pos++
			goto escape
		} else if c >= utf8.RuneSelf {
			b := bits.LeadingZeros8(^c)
			if b > 4 {
				p.err = fmt.Errorf("invalid UTF-8 encoding at %d", p.off+p.pos)
				return tokErr
			} else {

				if len(p.buf)-p.pos < b {
					p.spillstr()
					p.str = p.pos
				}
				if !p.ensure(b) {
					p.err = fmt.Errorf("unexpected EOF at %d", p.off+p.pos)
					return tokErr
				}
			}
			if _, n := utf8.DecodeRune(p.buf[p.pos:]); n != b {
				p.err = fmt.Errorf("invalid UTF-8 encoding at %d", p.off+p.pos)
				return tokErr
			} else {
				p.pos += n
			}
		} else {
			p.pos++
		}
	}
	p.err = fmt.Errorf("unexpected EOF at %d", p.off+p.pos)
	return tokErr
escape:
	if !p.ensure(1) {
		p.err = fmt.Errorf("unexpected EOF at %d", p.off+p.pos)
		return tokErr
	}
	switch p.buf[p.pos] {
	case '"':
		p.sb.WriteByte('"')
	case '/':
		p.sb.WriteByte('/')
	case '\\':
		p.sb.WriteByte('\\')
	case 'b':
		p.sb.WriteByte('\b')
	case 'f':
		p.sb.WriteByte('\f')
	case 'n':
		p.sb.WriteByte('\n')
	case 'r':
		p.sb.WriteByte('\r')
	case 't':
		p.sb.WriteByte('\t')
	case 'u':
		// pandoc never produces unicode escapes
		fallthrough
	default:
		p.err = fmt.Errorf("invalid escape sequence at %d", p.off+p.pos)
		return tokErr
	}
	p.pos++
	goto scan
}
