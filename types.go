// Package pandoc implements the [Pandoc] AST as defined in [Pandoc Types].
//
// [Pandoc]: https://pandoc.org/
// [Pandoc Types]: https://hackage.haskell.org/package/pandoc-types
package pandoc

import (
	"strconv"
	"strings"
)

// Implemented Pandoc protocol version.
const Version = "1.23.1"

var _Version = func() []int {
	c := strings.Split(Version, ".")
	v := make([]int, len(c))
	for i, s := range c {
		n, _ := strconv.ParseInt(s, 10, 64)
		v[i] = int(n)
	}
	return v
}()

// A convenience function to check if an element is of a particular type.
//
// Example:
//
//	if pandoc.Is[*pandoc.Str](elt) {
//	    ...
//
//	if pandoc.Is[pandoc.Inline](elt) {
//	    ...
func Is[P any, S Element](elt S) bool {
	_, ok := any(elt).(*P)
	return ok
}

// Returs a shallow copy of an element. Intended for use in Filter.
func Clone[P Element](elt P) P {
	return elt.clone().(P)
}

// Pandoc AST element interface
type Element interface {
	writable
	element()
	clone() Element
}

type inlinesContainer interface {
	inlines() []Inline
}

type blocksContainer interface {
	blocks() []Block
}

// Pandoc AST element that can be referred to.
type Linkable interface {
	Element
	Ident() string
	SetIdent(string)
}

// Pandoc AST object tag
type Tag string

func (t Tag) Tag() Tag       { return t }
func (t Tag) String() string { return string(t) }

// Pandoc AST object with tag
type Tagged interface {
	Tag() Tag
}

// Pandoc AST inline element
type Inline interface {
	Element
	Tagged
	inline()
}

// Pandoc AST inline's whitespaces (Space, SoftBreak, LineBreak)
type WhiteSpace interface {
	Inline
	space()
}

// Pandoc AST block element
type Block interface {
	Element
	Tagged
	block()
}

// Pandoc document metadata value
type MetaValue interface {
	Element
	Tagged
	meta()
}

// Pandoc document
type Pandoc struct {
	Meta   Meta
	Blocks []Block
}

func (p *Pandoc) element() {}
func (p *Pandoc) clone() Element {
	c := *p
	return &c
}
func (p *Pandoc) blocks() []Block { return p.Blocks }
func (p *Pandoc) Apply(transformers ...func(*Pandoc) (*Pandoc, error)) (*Pandoc, error) {
	return apply(p, transformers...)
}

// Pandoc's MetaMap entry.
type MetaMapEntry struct {
	Key   string
	Value MetaValue
}

func (m MetaMapEntry) element()       {}
func (m MetaMapEntry) clone() Element { return m }

// Pandoc's Meta
type Meta []MetaMapEntry

// Returns a value of the given key or nil if the key is not present.
func (m *Meta) Get(key string) MetaValue {
	for _, e := range *m {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

// Sets a value for the given key. If the value is nil, the key is removed.
func (m *Meta) Set(key string, value MetaValue) {
	for i, e := range *m {
		if e.Key == key {
			if value == nil {
				*m = append((*m)[:i], (*m)[i+1:]...)
			} else {
				(*m)[i].Value = value
			}
			return
		}
	}
	if value != nil {
		*m = append(*m, MetaMapEntry{key, value})
	}
}

// Sets a boolean value for the given key.
func (m *Meta) SetBool(key string, value bool) {
	m.Set(key, MetaBool(value))
}

// Sets a list of blocks for the given key.
func (m *Meta) SetBlocks(key string, value ...Block) {
	m.Set(key, &MetaBlocks{value})
}

// Sets a list of inlines for the given key.
func (m *Meta) SetInlines(key string, value ...Inline) {
	m.Set(key, &MetaInlines{value})
}

// Sets a string value for the given key.
func (m *Meta) SetString(key string, value string) {
	m.Set(key, MetaString(value))
}

// Pandoc document metadata map
type MetaMap struct {
	Entries Meta
}

const MetaMapTag = Tag("MetaMap")

func (m *MetaMap) Tag() Tag { return MetaMapTag }
func (m *MetaMap) clone() Element {
	c := *m
	return &c
}
func (m *MetaMap) element() {}
func (m *MetaMap) meta()    {}

// Returns a value of the given key or nil if the key is not present.
func (m *MetaMap) Get(key string) MetaValue {
	return m.Entries.Get(key)
}

// Sets a value for the given key. If the value is nil, the key is removed.
func (m *MetaMap) Set(key string, value MetaValue) {
	m.Entries.Set(key, value)
}

// Pandoc document metadata list
type MetaList struct {
	Entries []MetaValue
}

const MetaListTag = Tag("MetaList")

func (m *MetaList) Tag() Tag { return MetaListTag }
func (m *MetaList) clone() Element {
	c := *m
	return &c
}
func (m *MetaList) element() {}
func (m *MetaList) meta()    {}

// Pandoc document metadata inlines block
type MetaInlines struct {
	Inlines []Inline
}

const MetaInlinesTag = Tag("MetaInlines")

func (m *MetaInlines) Tag() Tag          { return MetaInlinesTag }
func (m *MetaInlines) inlines() []Inline { return m.Inlines }
func (m *MetaInlines) clone() Element {
	c := *m
	return &c
}
func (m *MetaInlines) element() {}
func (m *MetaInlines) meta()    {}
func (m *MetaInlines) Text() string {
	var sb strings.Builder
	walkList(m.Inlines, func(i Inline) ([]Inline, error) {
		switch i := i.(type) {
		case *Str:
			sb.WriteString(i.Text)
		case *Space:
			sb.WriteByte(' ')
		case *SoftBreak:
			sb.WriteByte('\n')
		case *LineBreak:
			sb.WriteByte('\n')
		case *Note:
			return nil, Skip
		}
		return nil, Continue
	})
	return sb.String()
}

// Pandoc document metadata blocks block
type MetaBlocks struct {
	Blocks []Block
}

const MetaBlocksTag = Tag("MetaBlocks")

func (m *MetaBlocks) Tag() Tag        { return MetaBlocksTag }
func (m *MetaBlocks) blocks() []Block { return m.Blocks }
func (m *MetaBlocks) clone() Element {
	c := *m
	return &c
}
func (m *MetaBlocks) element() {}
func (m *MetaBlocks) meta()    {}

// Pandoc document metadata boolean
type MetaBool bool

const MetaBoolTag = Tag("MetaBool")

func (b MetaBool) Tag() Tag       { return MetaBoolTag }
func (b MetaBool) clone() Element { return b }
func (b MetaBool) element()       {}
func (b MetaBool) meta()          {}

// Pandoc document metadata string
type MetaString string

const MetaStringTag = Tag("MetaString")

func (MetaString) Tag() Tag         { return MetaStringTag }
func (s MetaString) clone() Element { return s }
func (s MetaString) String() string { return string(s) }
func (MetaString) element()         {}
func (MetaString) meta()            {}

// Pandoc elements attribute' key-value pair.
type KV struct {
	Key   string
	Value string
}

// Pandoc elements attribute.
type Attr struct {
	Id      string   // Element ID
	Classes []string // Element classes
	KVs     []KV     // Element attributes' key-value pairs
}

// Returns the element's ID.
func (a *Attr) Ident() string {
	return a.Id
}

// Sets the element's ID in-place. This method is intended to quickly
// modify an element's ID in Query or QueryE without cloning it.
func (a *Attr) SetIdent(id string) {
	a.Id = id
}

// Returns a copy of attributes with the given ID.
func (a Attr) WithIdent(id string) Attr {
	a.Id = id
	return a
}

// Returns true if attribute has the given class.
func (a *Attr) HasClass(c string) bool {
	for _, cl := range a.Classes {
		if cl == c {
			return true
		}
	}
	return false
}

// Returns true if attribute has one of the given classes.
func (a *Attr) HasOneOfClasses(c ...string) bool {
	for _, cl := range a.Classes {
		for _, c := range c {
			if cl == c {
				return true
			}
		}
	}
	return false
}

// Returns a value of the given key or false if the key is not present.
func (a *Attr) Get(key string) (string, bool) {
	for _, kv := range a.KVs {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

// Returns a copy of attributes with the given class.
func (a Attr) WithClass(c string) Attr {
	if !a.HasClass(c) {
		a.Classes = append(a.Classes, c)
	}
	return a
}

// Returns a copy of attributes without the given class.
func (a Attr) WithoutClass(c string) Attr {
	for i, cl := range a.Classes {
		if cl == c {
			a.Classes = append(a.Classes[:i], a.Classes[i+1:]...)
			return a
		}
	}
	return a
}

// Returns a copy of attributes with the given key-value pair.
func (a Attr) WithKV(key, value string) Attr {
	for i, kv := range a.KVs {
		if kv.Key == key {
			a.KVs[i].Value = value
			return a
		}
	}
	a.KVs = append(a.KVs, KV{key, value})
	return a
}

// Returns a copy of attributes without the given key.
func (a Attr) WithoutKey(key string) Attr {
	for i, kv := range a.KVs {
		if kv.Key == key {
			a.KVs = append(a.KVs[:i], a.KVs[i+1:]...)
			return a
		}
	}
	return a
}

// Returns a copy of attributes with the given key-value pairs.
func (a Attr) WithKVs(pairs ...string) Attr {
	kvs := append(make([]KV, 0, len(a.KVs)+len(pairs)/2), a.KVs...)
next:
	for i := 0; i+1 < len(pairs); i += 2 {
		for j := range kvs {
			if kvs[j].Key == pairs[i] {
				kvs[j].Value = pairs[i+1]
				continue next
			}
		}
		kvs = append(kvs, KV{pairs[i], pairs[i+1]})
	}
	a.KVs = kvs
	return a
}

// Returns a copy of attributes without the given keys.
func (a Attr) WithoutKeys(keys ...string) Attr {
	kvs := append(make([]KV, 0, len(a.KVs)), a.KVs...)
	for i := range keys {
		for j := range kvs {
			if a.KVs[j].Key == keys[i] {
				copy(kvs[j:], kvs[j+1:])
				kvs = kvs[:len(kvs)-1]
				break
			}
		}
	}
	a.KVs = kvs
	return a
}

// Text (string)
type Str struct {
	Text string
}

const StrTag = Tag("Str")

func (s *Str) Tag() Tag { return StrTag }
func (s *Str) clone() Element {
	c := *s
	return &c
}
func (s *Str) inline()  {}
func (s *Str) element() {}

// Emphasized text (list of inlines)
type Emph struct {
	Inlines []Inline
}

const EmphTag = Tag("Emph")

func (e *Emph) Tag() Tag          { return EmphTag }
func (e *Emph) inlines() []Inline { return e.Inlines }
func (e *Emph) clone() Element {
	c := *e
	return &c
}
func (e *Emph) inline()  {}
func (e *Emph) element() {}
func (e *Emph) Apply(transformers ...func(*Emph) (*Emph, error)) (*Emph, error) {
	return apply(e, transformers...)
}

// Underlined text (list of inlines)
type Underline struct {
	Inlines []Inline
}

const UnderlineTag = Tag("Underline")

func (u *Underline) Tag() Tag          { return UnderlineTag }
func (u *Underline) inlines() []Inline { return u.Inlines }
func (u *Underline) inline()           {}
func (u *Underline) clone() Element {
	c := *u
	return &c
}
func (u *Underline) element() {}
func (u *Underline) Apply(transformers ...func(*Underline) (*Underline, error)) (*Underline, error) {
	return apply(u, transformers...)
}

// Strongly emphasized text (list of inlines)
type Strong struct {
	Inlines []Inline
}

const StrongTag = Tag("Strong")

func (s *Strong) Tag() Tag          { return StrongTag }
func (s *Strong) inlines() []Inline { return s.Inlines }
func (s *Strong) inline()           {}
func (s *Strong) clone() Element {
	c := *s
	return &c
}
func (s *Strong) element() {}
func (s *Strong) Apply(transformers ...func(*Strong) (*Strong, error)) (*Strong, error) {
	return apply(s, transformers...)
}

// Strikeout text (list of inlines)
type Strikeout struct {
	Inlines []Inline
}

const StrikeoutTag = Tag("Strikeout")

func (s *Strikeout) Tag() Tag          { return StrikeoutTag }
func (s *Strikeout) inlines() []Inline { return s.Inlines }
func (s *Strikeout) inline()           {}
func (s *Strikeout) clone() Element {
	c := *s
	return &c
}
func (s *Strikeout) element() {}
func (s *Strikeout) Apply(transformers ...func(*Strikeout) (*Strikeout, error)) (*Strikeout, error) {
	return apply(s, transformers...)
}

// Superscripted text (list of inlines)
type Superscript struct {
	Inlines []Inline
}

const SuperscriptTag = Tag("Superscript")

func (s *Superscript) Tag() Tag          { return SuperscriptTag }
func (s *Superscript) inlines() []Inline { return s.Inlines }
func (s *Superscript) clone() Element {
	c := *s
	return &c
}
func (s *Superscript) inline()  {}
func (s *Superscript) element() {}
func (s *Superscript) Apply(transformers ...func(*Superscript) (*Superscript, error)) (*Superscript, error) {
	return apply(s, transformers...)
}

// Subscripted text (list of inlines)
type Subscript struct {
	Inlines []Inline
}

const SubscriptTag = Tag("Subscript")

func (s *Subscript) Tag() Tag          { return SubscriptTag }
func (s *Subscript) inlines() []Inline { return s.Inlines }
func (s *Subscript) inline()           {}
func (s *Subscript) clone() Element {
	c := *s
	return &c
}
func (s *Subscript) element() {}
func (s *Subscript) Apply(transformers ...func(*Subscript) (*Subscript, error)) (*Subscript, error) {
	return apply(s, transformers...)
}

// Small capitals (list of inlines)
type SmallCaps struct {
	Inlines []Inline
}

const SmallCapsTag = Tag("SmallCaps")

func (s *SmallCaps) Tag() Tag          { return SmallCapsTag }
func (s *SmallCaps) inlines() []Inline { return s.Inlines }
func (s *SmallCaps) inline()           {}
func (s *SmallCaps) clone() Element {
	c := *s
	return &c
}
func (s *SmallCaps) element() {}
func (s *SmallCaps) Apply(transformers ...func(*SmallCaps) (*SmallCaps, error)) (*SmallCaps, error) {
	return apply(s, transformers...)
}

type QuoteType Tag

const (
	SingleQuote QuoteType = "SingleQuote"
	DoubleQuote QuoteType = "DoubleQuote"
)

// Quoted text (list of inlines)
type Quoted struct {
	QuoteType QuoteType
	Inlines   []Inline
}

const QuotedTag = Tag("Quoted")

func (q *Quoted) Tag() Tag          { return QuotedTag }
func (q *Quoted) inlines() []Inline { return q.Inlines }
func (q *Quoted) inline()           {}
func (q *Quoted) clone() Element {
	c := *q
	return &c
}
func (q *Quoted) element() {}
func (q *Quoted) Apply(transformers ...func(*Quoted) (*Quoted, error)) (*Quoted, error) {
	return apply(q, transformers...)
}

type CitationMode Tag

const (
	NormalCitation CitationMode = "NormalCitation"
	SuppressAuthor CitationMode = "SuppressAuthor"
	AuthorInText   CitationMode = "AuthorInText"
)

type Citation struct {
	Id      string
	Prefix  []Inline
	Suffix  []Inline
	Mode    CitationMode
	NoteNum int
	Hash    int
}

func (c *Citation) element() {}
func (c *Citation) clone() Element {
	c1 := *c
	return &c1
}
func (c *Citation) Apply(transformers ...func(*Citation) (*Citation, error)) (*Citation, error) {
	return apply(c, transformers...)
}

// Citation (list of inlines)
type Cite struct {
	Citations []*Citation
	Inlines   []Inline
}

const CiteTag = Tag("Cite")

func (c *Cite) Tag() Tag { return CiteTag }

func (c *Cite) inline() {}
func (c *Cite) clone() Element {
	c2 := *c
	return &c2
}
func (c *Cite) element() {}
func (c *Cite) Apply(transformers ...func(*Cite) (*Cite, error)) (*Cite, error) {
	return apply(c, transformers...)
}

// Inline code (literal)
type Code struct {
	Attr
	Text string
}

const CodeTag = Tag("Code")

func (c *Code) Tag() Tag { return CodeTag }
func (c *Code) clone() Element {
	c1 := *c
	return &c1
}
func (c *Code) inline()  {}
func (c *Code) element() {}

var SP = &Space{}

// Inter-word space
type Space struct{}

const SpaceTag = Tag("Space")

func (*Space) Tag() Tag       { return SpaceTag }
func (*Space) space()         {}
func (*Space) clone() Element { return SP }
func (*Space) inline()        {}
func (*Space) element()       {}

var SB = &SoftBreak{}

// Soft line break
type SoftBreak struct{}

const SoftBreakTag = Tag("SoftBreak")

func (*SoftBreak) Tag() Tag       { return SoftBreakTag }
func (*SoftBreak) space()         {}
func (*SoftBreak) clone() Element { return SB }
func (*SoftBreak) inline()        {}
func (*SoftBreak) element()       {}

var LB = &LineBreak{}

// Hard line break
type LineBreak struct{}

const LineBreakTag = Tag("LineBreak")

func (*LineBreak) Tag() Tag       { return LineBreakTag }
func (*LineBreak) space()         {}
func (*LineBreak) clone() Element { return LB }
func (*LineBreak) inline()        {}
func (*LineBreak) element()       {}

type MathType Tag

const (
	DisplayMath MathType = "DisplayMath"
	InlineMath  MathType = "InlineMath"
)

// TeX math (literal)
type Math struct {
	MathType MathType
	Text     string
}

const MathTag = Tag("Math")

func (m *Math) Tag() Tag { return MathTag }
func (m *Math) clone() Element {
	c := *m
	return &c
}
func (m *Math) inline()  {}
func (m *Math) element() {}

// Raw inline
type RawInline struct {
	Format string
	Text   string
}

const RawInlineTag = Tag("RawInline")

func (r *RawInline) Tag() Tag { return RawInlineTag }
func (r *RawInline) clone() Element {
	c := *r
	return &c
}
func (r *RawInline) element() {}
func (r *RawInline) inline()  {}

type Target struct {
	Url   string
	Title string
}

// Hyperlink: alt text (list of inlines), target
type Link struct {
	Attr
	Inlines []Inline
	Target  Target
}

const LinkTag = Tag("Link")

func (l *Link) Tag() Tag          { return LinkTag }
func (l *Link) inlines() []Inline { return l.Inlines }
func (l *Link) clone() Element {
	c := *l
	return &c
}
func (l *Link) inline()  {}
func (l *Link) element() {}
func (l *Link) Apply(transformers ...func(*Link) (*Link, error)) (*Link, error) {
	return apply(l, transformers...)
}

// Image: alt text (list of inlines), target
type Image struct {
	Attr
	Inlines []Inline
	Target  Target
}

const ImageTag = Tag("Image")

func (i *Image) Tag() Tag          { return ImageTag }
func (i *Image) inlines() []Inline { return i.Inlines }
func (i *Image) clone() Element {
	c := *i
	return &c
}
func (i *Image) element() {}
func (i *Image) inline()  {}
func (i *Image) Apply(transformers ...func(*Image) (*Image, error)) (*Image, error) {
	return apply(i, transformers...)
}

// Footnote: list of blocks
type Note struct {
	Blocks []Block
}

const NoteTag = Tag("Note")

func (n *Note) Tag() Tag        { return NoteTag }
func (n *Note) blocks() []Block { return n.Blocks }
func (n *Note) clone() Element {
	c := *n
	return &c
}
func (n *Note) element() {}
func (n *Note) inline()  {}
func (n *Note) Apply(transformers ...func(*Note) (*Note, error)) (*Note, error) {
	return apply(n, transformers...)
}

// Generic inline container with attributes
type Span struct {
	Attr
	Inlines []Inline
}

const SpanTag = Tag("Span")

func (s *Span) Tag() Tag          { return SpanTag }
func (s *Span) inlines() []Inline { return s.Inlines }
func (s *Span) clone() Element {
	c := *s
	return &c
}
func (s *Span) inline()  {}
func (s *Span) element() {}
func (s *Span) Apply(transformers ...func(*Span) (*Span, error)) (*Span, error) {
	return apply(s, transformers...)
}

// Plain text, not a paragraph
type Plain struct {
	Inlines []Inline
}

const PlainTag = Tag("Plain")

func (p *Plain) Tag() Tag          { return PlainTag }
func (p *Plain) inlines() []Inline { return p.Inlines }
func (p *Plain) clone() Element {
	c := *p
	return &c
}
func (p *Plain) block()   {}
func (p *Plain) element() {}
func (p *Plain) Apply(transformers ...func(*Plain) (*Plain, error)) (*Plain, error) {
	return apply(p, transformers...)
}

// Paragraph (list of inlines)
type Para struct {
	Inlines []Inline
}

const ParaTag = Tag("Para")

func (p *Para) Tag() Tag          { return ParaTag }
func (p *Para) inlines() []Inline { return p.Inlines }
func (p *Para) clone() Element {
	c := *p
	return &c
}
func (p *Para) block()   {}
func (p *Para) element() {}
func (p *Para) Apply(transformers ...func(*Para) (*Para, error)) (*Para, error) {
	return apply(p, transformers...)
}

// Multiple non-breaking lines
type LineBlock struct {
	Inlines [][]Inline
}

const LineBlockTag = Tag("LineBlock")

func (b *LineBlock) Tag() Tag { return LineBlockTag }
func (b *LineBlock) clone() Element {
	c := *b
	return &c
}
func (b *LineBlock) block()   {}
func (b *LineBlock) element() {}
func (b *LineBlock) Apply(transformers ...func(*LineBlock) (*LineBlock, error)) (*LineBlock, error) {
	return apply(b, transformers...)
}

// Code block (literal)
type CodeBlock struct {
	Attr
	Text string
}

const CodeBlockTag = Tag("CodeBlock")

func (b *CodeBlock) Tag() Tag { return CodeBlockTag }
func (b *CodeBlock) clone() Element {
	c := *b
	return &c
}
func (b *CodeBlock) block()   {}
func (b *CodeBlock) element() {}

// Raw block
type RawBlock struct {
	Format string
	Text   string
}

const RawBlockTag = Tag("RawBlock")

func (b *RawBlock) Tag() Tag { return RawBlockTag }
func (b *RawBlock) clone() Element {
	c := *b
	return &c
}
func (b *RawBlock) block()   {}
func (b *RawBlock) element() {}

// Block quote (list of blocks)
type BlockQuote struct {
	Blocks []Block
}

const BlockQuoteTag = Tag("BlockQuote")

func (b *BlockQuote) Tag() Tag        { return BlockQuoteTag }
func (b *BlockQuote) blocks() []Block { return b.Blocks }
func (b *BlockQuote) clone() Element {
	c := *b
	return &c
}
func (b *BlockQuote) block()   {}
func (b *BlockQuote) element() {}
func (b *BlockQuote) Apply(transformers ...func(*BlockQuote) (*BlockQuote, error)) (*BlockQuote, error) {
	return apply(b, transformers...)
}

type ListNumberStyle Tag

const (
	DefaultStyle ListNumberStyle = "DefaultStyle"
	Example      ListNumberStyle = "Example"
	Decimal      ListNumberStyle = "Decimal"
	LowerRoman   ListNumberStyle = "LowerRoman"
	UpperRoman   ListNumberStyle = "UpperRoman"
	LowerAlpha   ListNumberStyle = "LowerAlpha"
	UpperAlpha   ListNumberStyle = "UpperAlpha"
)

type ListNumberDelim Tag

const (
	DefaultDelim ListNumberDelim = "DefaultDelim"
	Period       ListNumberDelim = "Period"
	OneParen     ListNumberDelim = "OneParen"
	TwoParens    ListNumberDelim = "TwoParens"
)

type ListAttrs struct {
	Start     int
	Style     ListNumberStyle
	Delimiter ListNumberDelim
}

// Ordered list (attributes and a list of items, each a list of blocks)
type OrderedList struct {
	Attr  ListAttrs
	Items [][]Block
}

const OrderedListTag = Tag("OrderedList")

func (l *OrderedList) Tag() Tag { return OrderedListTag }
func (l *OrderedList) clone() Element {
	c := *l
	return &c
}
func (l *OrderedList) block()   {}
func (l *OrderedList) element() {}
func (l *OrderedList) Apply(transformers ...func(*OrderedList) (*OrderedList, error)) (*OrderedList, error) {
	return apply(l, transformers...)
}

// Bullet list (list of items, each a list of blocks)
type BulletList struct {
	Items [][]Block
}

const BulletListTag = Tag("BulletList")

func (l *BulletList) Tag() Tag { return BulletListTag }
func (l *BulletList) clone() Element {
	c := *l
	return &c
}
func (l *BulletList) block()   {}
func (l *BulletList) element() {}
func (l *BulletList) Apply(transformers ...func(*BulletList) (*BulletList, error)) (*BulletList, error) {
	return apply(l, transformers...)
}

type Definition struct {
	Term       []Inline
	Definition [][]Block
}

// Definition list (list of items, each a pair of inlines and a list of blocks)
type DefinitionList struct {
	Items []Definition
}

const DefinitionListTag = Tag("DefinitionList")

func (d *DefinitionList) Tag() Tag { return DefinitionListTag }
func (d *DefinitionList) clone() Element {
	c := *d
	return &c
}
func (d *DefinitionList) block()   {}
func (d *DefinitionList) element() {}
func (d *DefinitionList) Apply(transformers ...func(*DefinitionList) (*DefinitionList, error)) (*DefinitionList, error) {
	return apply(d, transformers...)
}

var HR = &HorizontalRule{}

// Horizontal rule
type HorizontalRule struct{}

const HorizontalRuleTag = Tag("HorizontalRule")

func (*HorizontalRule) Tag() Tag       { return HorizontalRuleTag }
func (*HorizontalRule) clone() Element { return HR }
func (*HorizontalRule) block()         {}
func (*HorizontalRule) element()       {}

// Header - level (integer) and text (inlines)
type Header struct {
	Attr
	Level   int
	Inlines []Inline
}

const HeaderTag = Tag("Header")

func (h *Header) Tag() Tag          { return HeaderTag }
func (h *Header) inlines() []Inline { return h.Inlines }
func (h *Header) clone() Element {
	c := *h
	return &c
}
func (h *Header) block()   {}
func (h *Header) element() {}
func (h *Header) Apply(transformers ...func(*Header) (*Header, error)) (*Header, error) {
	return apply(h, transformers...)
}

func (h *Header) Title() string {
	var sb strings.Builder
	walkList(h.Inlines, func(i Inline) ([]Inline, error) {
		switch i := i.(type) {
		case *Str:
			sb.WriteString(i.Text)
		case *Space:
			sb.WriteByte(' ')
		case *SoftBreak:
			sb.WriteByte('\n')
		case *LineBreak:
			sb.WriteByte('\n')
		case *Note:
			return nil, Skip
		}
		return nil, Continue
	})
	return sb.String()
}

type Caption struct {
	Short []Inline
	Long  []Block
}

type Alignment Tag

const (
	AlignLeft    Alignment = "AlignLeft"
	AlignRight   Alignment = "AlignRight"
	AlignCenter  Alignment = "AlignCenter"
	AlignDefault Alignment = "AlignDefault"
)

type ColWidth struct {
	Width   float64
	Default bool
}

const (
	_ColWidth        = "ColWidth"
	_ColWidthDefault = "ColWidthDefault"
)

func DefaultColWidth() ColWidth { return ColWidth{Default: true} }
func (c ColWidth) Tag() string {
	if c.Default {
		return _ColWidthDefault
	} else {
		return _ColWidth
	}
}

type ColSpec struct {
	Align Alignment
	Width ColWidth
}

type TableHeadFoot struct {
	Attr
	Rows []*TableRow
}

func (t *TableHeadFoot) element() {}
func (t *TableHeadFoot) clone() Element {
	c := *t
	return &c
}
func (t *TableHeadFoot) Apply(transformers ...func(*TableHeadFoot) (*TableHeadFoot, error)) (*TableHeadFoot, error) {
	return apply(t, transformers...)
}

type TableRow struct {
	Attr
	Cells []*TableCell
}

func (t *TableRow) element() {}
func (t *TableRow) clone() Element {
	c := *t
	return &c
}
func (t *TableRow) Apply(transformers ...func(*TableRow) (*TableRow, error)) (*TableRow, error) {
	return apply(t, transformers...)
}

type TableCell struct {
	Attr
	Align   Alignment
	RowSpan int
	ColSpan int
	Blocks  []Block
}

func (t *TableCell) element() {}
func (t *TableCell) clone() Element {
	c := *t
	return &c
}
func (t *TableCell) blocks() []Block { return t.Blocks }
func (t *TableCell) Apply(transformers ...func(*TableCell) (*TableCell, error)) (*TableCell, error) {
	return apply(t, transformers...)
}

type TableBody struct {
	Attr
	RowHeadColumns int
	Head           []*TableRow
	Body           []*TableRow
}

func (t *TableBody) element() {}
func (t *TableBody) clone() Element {
	c := *t
	return &c
}
func (t *TableBody) Apply(transformers ...func(*TableBody) (*TableBody, error)) (*TableBody, error) {
	return apply(t, transformers...)
}

// Table, with attributes, caption, optional short caption, column alignments
// and widths (required), table head, table bodies, and table foot
type Table struct {
	Attr
	Caption Caption
	Aligns  []ColSpec
	Head    TableHeadFoot
	Bodies  []*TableBody
	Foot    TableHeadFoot
}

const TableTag = Tag("Table")

func (t *Table) Tag() Tag { return TableTag }
func (t *Table) clone() Element {
	c := *t
	return &c
}
func (t *Table) block()   {}
func (t *Table) element() {}
func (t *Table) Apply(transformers ...func(*Table) (*Table, error)) (*Table, error) {
	return apply(t, transformers...)
}

// Figure, with attributes, caption, and content (list of blocks)
type Figure struct {
	Attr
	Caption Caption
	Blocks  []Block
}

const FigureTag = Tag("Figure")

func (f *Figure) Tag() Tag { return FigureTag }
func (f *Figure) clone() Element {
	c := *f
	return &c
}
func (f *Figure) block()   {}
func (f *Figure) element() {}
func (f *Figure) Apply(transformers ...func(*Figure) (*Figure, error)) (*Figure, error) {
	return apply(f, transformers...)
}

// Generic block container with attributes
type Div struct {
	Attr
	Blocks []Block
}

const DivTag = Tag("Div")

func (d *Div) Tag() Tag        { return DivTag }
func (d *Div) blocks() []Block { return d.Blocks }
func (d *Div) clone() Element {
	c := *d
	return &c
}
func (d *Div) block()   {}
func (d *Div) element() {}
func (d *Div) Apply(transformers ...func(*Div) (*Div, error)) (*Div, error) {
	return apply(d, transformers...)
}
