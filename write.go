package pandoc

import (
	"io"
	"math"
	"strconv"
	"strings"
)

type writable interface {
	write(io.Writer) error
}

// interface check

var _ []writable = []writable{
	MetaMap{},
	MetaList{},

	&Attr{},
	&Str{},
	&Emph{},
	&Underline{},
	&Strong{},
	&Strikeout{},
	&Superscript{},
	&Subscript{},
	&SmallCaps{},
	&Quoted{},
	&Cite{},
	&Code{},
	&Math{},
	&RawInline{},
	&Link{},
	&Image{},
	&Note{},
	&Span{},

	&Plain{},
	&Para{},
	&LineBlock{},
	&CodeBlock{},
	&RawBlock{},
	&BlockQuote{},
	&OrderedList{},
	&BulletList{},
	&DefinitionList{},
	&Header{},
	&Table{},
	&Figure{},
	&Div{},
}

func (s *Str) write(w io.Writer) error {
	return withTag(s, str(s.Str)).write(w)
}

func (s *Emph) write(w io.Writer) error {
	return withTag(s, list(s.Inlines)).write(w)
}

func (s *Underline) write(w io.Writer) error {
	return withTag(s, list(s.Inlines)).write(w)
}

func (s *Strong) write(w io.Writer) error {
	return withTag(s, list(s.Inlines)).write(w)
}

func (s *Strikeout) write(w io.Writer) error {
	return withTag(s, list(s.Inlines)).write(w)
}

func (s *Superscript) write(w io.Writer) error {
	return withTag(s, list(s.Inlines)).write(w)
}

func (s *Subscript) write(w io.Writer) error {
	return withTag(s, list(s.Inlines)).write(w)
}

func (s *SmallCaps) write(w io.Writer) error {
	return withTag(s, list(s.Inlines)).write(w)
}

func (q *Quoted) write(w io.Writer) error {
	return withTag(q, tuple2(taggedStr(q.QuoteType), list(q.Inlines))).write(w)
}

func writeField[T writable](w io.Writer, name string, d byte, v T) error {
	if err := writeKey(w, name); err != nil {
		return err
	}
	if err := v.write(w); err != nil {
		return err
	}
	return writeDelim(w, d)
}

type citationList []Citation

func (c citationList) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	for i := range c {
		if i > 0 {
			if err := writeDelim(w, ','); err != nil {
				return err
			}
		}
		if err := writeDelim(w, '{'); err != nil {
			return err
		}		
		if err := writeField(w, "citationId", ',', str(c[i].Id)); err != nil {
			return err
		}
		if err := writeField(w, "citationPrefix", ',', list(c[i].Prefix)); err != nil {
			return err
		}
		if err := writeField(w, "citationSuffix", ',', list(c[i].Suffix)); err != nil {
			return err
		}
		if err := writeField(w, "citationMode", ',', taggedStr(c[i].Mode)); err != nil {
			return err
		}
		if err := writeField(w, "citationNoteNum", ',', num(c[i].NoteNum)); err != nil {
			return err
		}
		if err := writeField(w, "citationHash", '}', num(c[i].Hash)); err != nil {
			return err
		}
	}
	return writeDelim(w, ']')
}

func (c *Cite) write(w io.Writer) error {
	return withTag(c, tuple2(citationList(c.Citations), list(c.Inlines))).write(w)
}
func (c *Code) write(w io.Writer) error {
	return withTag(c, tuple2(&c.Attr, str(c.Text))).write(w)
}

func (c *Space) write(w io.Writer) error {
	return taggedStr(c.Tag()).write(w)
}

func (b *SoftBreak) write(w io.Writer) error {
	return taggedStr(b.Tag()).write(w)
}

func (b *LineBreak) write(w io.Writer) error {
	return taggedStr(b.Tag()).write(w)
}

func (p Caption) write(w io.Writer) error {
	return tuple2(captionShort{p.HasShort, p.Short}, list(p.Long)).write(w)
}

func (m *Math) write(w io.Writer) error {
	return withTag(m, tuple2(taggedStr(m.MathType), str(m.Text))).write(w)
}

func (r *RawInline) write(w io.Writer) error {
	return withTag(r, tuple2(str(r.Format), str(r.Text))).write(w)
}

func (a AttrKV) write(w io.Writer) error {
	return tuple2(str(a.Key), str(a.Value)).write(w)
}

func (a *Attr) write(w io.Writer) error {
	return tuple3(str(a.Id), strList(a.Classes), list(a.Attrs)).write(w)
}

func (t *Target) write(w io.Writer) error {
	return tuple2(str(t.Url), str(t.Title)).write(w)
}

func (l *Link) write(w io.Writer) error {
	return withTag(l, tuple3(&l.Attr, list(l.Inlines), &l.Target)).write(w)
}

func (i *Image) write(w io.Writer) error {
	return withTag(i, tuple3(&i.Attr, list(i.Inlines), &i.Target)).write(w)
}

func (n *Note) write(w io.Writer) error {
	return withTag(n, list(n.Blocks)).write(w)
}

func (s *Span) write(w io.Writer) error {
	return withTag(s, tuple2(&s.Attr, list(s.Inlines))).write(w)
}

func (p *Plain) write(w io.Writer) error {
	return withTag(p, list(p.Inlines)).write(w)
}

func (p *Para) write(w io.Writer) error {
	return withTag(p, list(p.Inlines)).write(w)
}

func (p *LineBlock) write(w io.Writer) error {
	return withTag(p, dlist(p.Inlines)).write(w)
}

func (p *CodeBlock) write(w io.Writer) error {
	return withTag(p, tuple2(&p.Attr, str(p.Text))).write(w)
}

func (p *RawBlock) write(w io.Writer) error {
	return withTag(p, tuple2(str(p.Format), str(p.Text))).write(w)
}

func (p *BlockQuote) write(w io.Writer) error {
	return withTag(p, list(p.Blocks)).write(w)
}

func (a *ListAttrs) write(w io.Writer) error {
	return tuple3(num(a.Start), taggedStr(a.Style), taggedStr(a.Delimiter)).write(w)
}

func (p *OrderedList) write(w io.Writer) error {
	return withTag(p, tuple2(&p.Attr, dlist(p.Items))).write(w)
}

func (p *BulletList) write(w io.Writer) error {
	return withTag(p, dlist(p.Blocks)).write(w)
}

func (d Definition) write(w io.Writer) error {
	return tuple2(list(d.Term), dlist(d.Definition)).write(w)
}

func (p *DefinitionList) write(w io.Writer) error {
	return withTag(p, list(p.Items)).write(w)
}

func (l *HorizontalRule) write(w io.Writer) error {
	return taggedStr(l.Tag()).write(w)
}

func (p *Header) write(w io.Writer) error {
	return withTag(p, tuple3(num(p.Level), &p.Attr, list(p.Inlines))).write(w)
}

func (c ColWidth) write(w io.Writer) error {
	if c.Default {
		return taggedStr(_ColWidthDefault).write(w)
	} else {
		if _, err := w.Write(appendFloat([]byte("{\"t\":\""+_ColWidth+"\",\"c\":"), c.Width)); err != nil {
			return err
		}
		return writeDelim(w, '}')
	}
}

func (c ColSpec) write(w io.Writer) error {
	return tuple2(taggedStr(c.Align), c.Width).write(w)
}

func (r TableCell) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	if err := r.Attr.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := taggedStr(r.Align).write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := num(r.RowSpan).write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := num(r.ColSpan).write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := list(r.Blocks).write(w); err != nil {
		return err
	}
	return writeDelim(w, ']')
}

func (r TableRow) write(w io.Writer) error {
	return tuple2(&r.Attr, list(r.Cells)).write(w)
}

func (hf *TableHeadFoot) write(w io.Writer) error {
	return tuple2(&hf.Attr, list(hf.Rows)).write(w)
}

func (b TableBody) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	if err := b.Attr.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := num(b.RowHeadColumns).write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := list(b.Head).write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := list(b.Body).write(w); err != nil {
		return err
	}
	return writeDelim(w, ']')
}

func (p *Table) write(w io.Writer) error {
	if _, err := w.Write([]byte("{\"t\":\"Table\",\"c\":[")); err != nil {
		return err
	}
	if err := p.Attr.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := p.Caption.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := list(p.Aligns).write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := p.Head.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := list(p.Bodies).write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := p.Foot.write(w); err != nil {
		return err
	}
	_, err := w.Write([]byte("]}"))
	return err
}

func (p *Figure) write(w io.Writer) error {
	return withTag(p, tuple3(&p.Attr, &p.Caption, list(p.Blocks))).write(w)
}

func (p *Div) write(w io.Writer) error {
	return withTag(p, tuple2(&p.Attr, list(p.Blocks))).write(w)
}

// -------------------

func mv(v MetaValue) metaValue {
	return metaValue{v}
}

type metaValue struct {
	v MetaValue
}

func (m metaValue) write(w io.Writer) error {
	if _, err := w.Write([]byte("{\"t\":\"")); err != nil {
		return err
	}
	if _, err := w.Write([]byte(m.v.Tag())); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\",\"c\":")); err != nil {
		return err
	}
	if err := m.v.write(w); err != nil {
		return err
	}
	if _, err := w.Write([]byte("}")); err != nil {
		return err
	}
	return nil
}

func (m MetaInlines) write(w io.Writer) error {
	return list(m).write(w)
}

func (m MetaBlocks) write(w io.Writer) error {
	return list(m).write(w)
}

func (m MetaString) write(w io.Writer) error {
	return str(m).write(w)
}

func (m MetaBool) write(w io.Writer) error {
	if m {
		if _, err := w.Write([]byte("true")); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte("false")); err != nil {
			return err
		}
	}
	return nil
}

// MetaMapEntry
func (m MetaMapEntry) write(w io.Writer) error {
	if err := writeKey(w, m.Key); err != nil {
		return err
	}
	if err := mv(m.Value).write(w); err != nil {
		return err
	}
	return nil
}

// MetaMap
func (m MetaMap) write(w io.Writer) error {
	if err := writeDelim(w, '{'); err != nil {
		return err
	}
	for i := range m {
		if i > 0 {
			if err := writeDelim(w, ','); err != nil {
				return err
			}
		}
		if err := m[i].write(w); err != nil {
			return err
		}
	}
	return writeDelim(w, '}')
}

// MetaList
func (m MetaList) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	for i := range m {
		if i > 0 {
			if err := writeDelim(w, ','); err != nil {
				return err
			}
		}
		if err := mv(m[i]).write(w); err != nil {
			return err
		}
	}
	return writeDelim(w, ']')
}

// -------------------

type captionShort struct {
	notNull bool
	inlines []Inline
}

func (w captionShort) write(wrt io.Writer) error {
	if w.notNull {
		return list(w.inlines).write(wrt)
	} else {
		return writeNull(wrt)
	}
}

func taggedStr[T ~string](t T) tstr { return tstr(t) }

type tstr string

func (s tstr) write(w io.Writer) error {
	if _, err := w.Write([]byte(appendQuote([]byte("{\"t\":"), string(s)))); err != nil {
		return err
	}
	return writeDelim(w, '}')
}

func num(n int64) wnum { return wnum(n) }

type wnum int64

func (n wnum) write(w io.Writer) error {
	if _, err := w.Write(strconv.AppendInt(nil, int64(n), 10)); err != nil {
		return err
	}
	return nil
}

type wlstr[T ~string] []T

func strList[T ~string](l []T) wlstr[T] { return wlstr[T](l) }
func (s wlstr[T]) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	for i := range s {
		if i > 0 {
			if err := writeDelim(w, ','); err != nil {
				return err
			}
		}
		if _, err := w.Write(appendQuote(nil, string(s[i]))); err != nil {
			return err
		}
	}
	return writeDelim(w, ']')
}

func str[T ~string](s T) wstr { return wstr(s) }

type wstr string

func (s wstr) write(w io.Writer) error {
	_, err := w.Write(appendQuote(nil, string(s)))
	return err
}

// tuples
type t2[T1, T2 writable] struct {
	e1 T1
	e2 T2
}

func tuple2[T1, T2 writable](e1 T1, e2 T2) t2[T1, T2] {
	return t2[T1, T2]{e1, e2}
}
func (t t2[T1, T2]) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	if err := t.e1.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := t.e2.write(w); err != nil {
		return err
	}
	return writeDelim(w, ']')
}

type t3[T1, T2, T3 writable] struct {
	e1 T1
	e2 T2
	e3 T3
}

func tuple3[T1, T2, T3 writable](e1 T1, e2 T2, e3 T3) t3[T1, T2, T3] {
	return t3[T1, T2, T3]{e1, e2, e3}
}
func (t t3[T1, T2, T3]) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	if err := t.e1.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := t.e2.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := t.e3.write(w); err != nil {
		return err
	}
	return writeDelim(w, ']')
}

func withTag[T Tagged, C writable](e T, c C) t[C] {
	return t[C]{t: e.Tag(), c: c}
}

type t[C writable] struct {
	t Tag
	c C
}

func (e t[T]) write(w io.Writer) error {
	if _, err := w.Write([]byte("{\"t\":\"")); err != nil {
		return err
	}
	if _, err := w.Write([]byte(e.t)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\",\"c\":")); err != nil {
		return err
	}
	if err := e.c.write(w); err != nil {
		return err
	}
	if _, err := w.Write([]byte("}")); err != nil {
		return err
	}
	return nil
}

func list[T writable](lst []T) l[T] {
	return l[T](lst)
}

type l[T writable] []T

func (lst l[T]) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	for i := range lst {
		if i > 0 {
			if err := writeDelim(w, ','); err != nil {
				return err
			}
		}
		if err := lst[i].write(w); err != nil {
			return err
		}
	}
	return writeDelim(w, ']')
}

type dl[T writable] [][]T

func dlist[T writable](l [][]T) dl[T] {
	return dl[T](l)
}
func (l dl[T]) write(w io.Writer) error {
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	for i := range l {
		if i > 0 {
			if err := writeDelim(w, ','); err != nil {
				return err
			}
		}
		if err := list(l[i]).write(w); err != nil {
			return err
		}
	}
	return writeDelim(w, ']')
}

func writeDelim(w io.Writer, b byte) error {
	if _, err := w.Write([]byte{b}); err != nil {
		return err
	}
	return nil
}

func writeKey(wrt io.Writer, name string) error {
	if _, err := wrt.Write(strconv.AppendQuote(nil, name)); err != nil {
		return err
	}
	if _, err := wrt.Write([]byte{':'}); err != nil {
		return err
	}
	return nil
}

func writeNull(wrt io.Writer) error {
	if _, err := wrt.Write([]byte("null")); err != nil {
		return err
	}
	return nil
}

// pandoc uses different exponent cutoffs than strconv.AppendFloat,
// and it also does not pad the exponent to two digits.
func appendFloat(b []byte, f float64) []byte {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return append(b, "null"...)
	}
	abs := math.Abs(f)
	fmt := byte('f')
	if abs != 0 {
		if abs < 1e-1 || abs >= 1e21 {
			fmt = 'e'
		}
	}
	b = strconv.AppendFloat(b, f, fmt, -1, 64)
	if fmt == 'e' {
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}
	return b
}

func appendQuote(b []byte, s string) []byte {
	const escapable = "\"\\\b\f\n\r\t"
	var r = 2
	for i := 0; i < len(s); {
		if j := strings.IndexAny(s[i:], escapable); j >= 0 {
			i += j+1
			r += j+2
		} else {
			r += len(s)-i
			break
		}
	}
	p := len(b)	
	b = append(b, make([]byte, r)...)
	b[p] = '"'
	p++
	for i := 0; i < len(s); {
		if j := strings.IndexAny(s[i:], escapable); j >= 0 {
			copy(b[p:], s[i:i+j])
			p += j
			b[p] = '\\'
			p++
			switch s[i+j] {
			case '"':
				b[p] = '"'
			case '\\':
				b[p] = '\\'
			case '\b':
				b[p] = 'b'
			case '\f':
				b[p] = 'f'
			case '\n':
				b[p] = 'n'
			case '\r':
				b[p] = 'r'
			case '\t':
				b[p] = 't'
			}
			p++
			i += j+1
		} else {
			copy(b[p:], s[i:])
			p += len(s)-i
			break
		}
	}
	b[p] = '"'
	return b
}

func (p *Doc) write(w io.Writer) error {
	if err := writeDelim(w, '{'); err != nil {
		return err
	}
	if err := writeKey(w, "pandoc-api-version"); err != nil {
		return err
	}
	if err := writeDelim(w, '['); err != nil {
		return err
	}
	for i, n := range p.Version {
		if i > 0 {
			if _, err := w.Write([]byte{','}); err != nil {
				return err
			}
		}
		if _, err := w.Write(strconv.AppendInt(nil, n, 10)); err != nil {
			return err
		}
	}
	if err := writeDelim(w, ']'); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := writeKey(w, "meta"); err != nil {
		return err
	}
	if err := p.Meta.write(w); err != nil {
		return err
	}
	if err := writeDelim(w, ','); err != nil {
		return err
	}
	if err := writeKey(w, "blocks"); err != nil {
		return err
	}
	if err := list(p.Blocks).write(w); err != nil {
		return err
	}
	if err := writeDelim(w, '}'); err != nil {
		return err
	}
	return nil
}

// Write writes the JSON encoding of elt to w.
//
// Example:
//
//	var doc *pandoc.Doc
//	...
//	if err := pandoc.Write(os.Stdout, doc); err != nil {
//		log.Fatal(err)
//	}
func Write[E Element](w io.Writer, elt E) error {
	return elt.write(w)
}
