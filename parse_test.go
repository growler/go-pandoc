package pandoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	var (
		v = []byte(`"`)
		p scanner
	)
	for i := 0; i < 32; i++ {
		v = append(v[:i+1], ' ')
		v = append(v, `💩"`...)
		p.init(bytes.NewReader(v))
		if tok := p.next(); tok == tokErr {
			t.Errorf("error: %s", p.err)
		} else if tok != tokStr {
			t.Errorf("expected string, got %s", tok)
		} else if p.string() != string(v[1:len(v)-1]) {
			t.Errorf("expected %s, got %s", v[1:len(v)-1], p.string())
		}
	}
}

func TestParser(t *testing.T) {
	// const v = `["0","1","2","3","4","5","6","7","8","9","10","11","12","13","14","15","16",null,false,true,0,-0,3,1E-15,0.12,1.23]`
	const v = `[0,1,false,2,false,3,false,4,false,5,false,6,false,7,false,8,false,9,false]`
	var p scanner
	p.init(strings.NewReader(v))
	for {
		tok := p.next()
		if tok == tokEOF {
			break
		}
		if tok == tokErr {
			t.Errorf("error: %s", p.err)
			break
		}
		if tok == tokStr {
			if p.sb.Len() > 0 {
				t.Logf("tok = %s sb(%s)", tok, p.sb.String())
			} else {
				t.Logf("tok = %s str(%s)", tok, p.buf[p.str:p.pos-1])
			}
		} else if tok == tokNumber {
			if p.numberIsInt() {
				t.Logf("tok = %s(%d)", tok, p.int())
			} else {
				t.Logf("tok = %s(%g)", tok, p.float())
			}
		} else {
			t.Logf("tok = %s", tok)
		}
	}
	if p.err != nil && !errors.Is(p.err, io.EOF) {
		t.Errorf("error: %s", p.err)
	}
}

func testD() []byte {
	var b bytes.Buffer
	b.WriteString("[null")
	for i := 1; i < 1000; i++ {
		b.WriteByte(',')
		b.WriteString("true")
		b.WriteByte(',')
		b.WriteString("null")
		b.WriteByte(',')
		b.WriteString("false")
	}
	b.WriteByte(']')
	return b.Bytes()
}

func testD1() []byte {
	var b bytes.Buffer
	b.WriteString("[0")
	for i := 1; i < 1000; i++ {
		b.WriteByte(',')
		b.WriteString(strconv.FormatInt(int64(i), 10))
	}
	b.WriteByte(']')
	return b.Bytes()
}

func BenchmarkParser(b *testing.B) {
	b.StopTimer()
	v := testD()
	var p scanner
	rdr := bytes.NewReader(nil)
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		rdr.Reset(v)
		p.init(rdr)
		for {
			if tok := p.next(); tok == tokEOF {
				break
			} else if tok == tokErr {
				b.Fatal(p.err)
			}
		}
	}
}

func BenchmarkParserStd(b *testing.B) {
	b.StopTimer()
	v := testD()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var r interface{}
		if err := json.Unmarshal(v, &r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserStdStream(b *testing.B) {
	b.StopTimer()
	v := testD()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		j := json.NewDecoder(bytes.NewReader(v))
		for {
			if _, err := j.Token(); err != nil {
				if err == io.EOF {
					break
				} else {
					b.Fatal(err)
				}
			}
		}
	}
}

func FuzzScanner(f *testing.F) {
	f.Add([]byte(`{"a":0,"b":0,"c":0,"d":0,"e":0,"f":[0,1E1],"g":null,"h":true}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		var p scanner
		p.init(bytes.NewReader(b))
		for {
			tok := p.next()
			if tok == tokEOF || tok == tokErr {
				break
			}
		}
	})
}
