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
const Version = "1.21.1"

var _Version = func() []int64 {
	c := strings.Split(Version, ".")
	v := make([]int64, len(c))
	for i, s := range c {
		v[i], _ = strconv.ParseInt(s, 10, 64)
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

// Pandoc AST element interface
type Element interface {
	writable
	element()
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

// Doc document
type Doc struct {
	Version []int64
	File    string
	Meta    MetaMap
	Blocks  []Block
}

func (*Doc) element() {}

// Pandoc's MetaMap entry.
type MetaMapEntry struct {
	Key   string
	Value MetaValue
}

func (MetaMapEntry) element() {}

// Pandoc document metadata map
type MetaMap []MetaMapEntry

const MetaMapTag = Tag("MetaMap")

func (MetaMap) Tag() Tag { return MetaMapTag }
func (MetaMap) element() {}
func (MetaMap) meta()    {}

func (m MetaMap) Get(key string) MetaValue {
	for _, e := range m {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

func (m *MetaMap) Set(key string, value MetaValue) {
	for i, e := range *m {
		if e.Key == key {
			(*m)[i].Value = value
			return
		}
	}
	*m = append(*m, MetaMapEntry{key, value})
}

// Pandoc document metadata list
type MetaList []MetaValue

const MetaListTag = Tag("MetaList")

func (MetaList) Tag() Tag { return MetaListTag }
func (MetaList) element() {}
func (MetaList) meta()    {}

// Pandoc document metadata inlines block
type MetaInlines []Inline

const MetaInlinesTag = Tag("MetaInlines")

func (MetaInlines) Tag() Tag { return MetaInlinesTag }
func (MetaInlines) element() {}
func (MetaInlines) meta()    {}

// Pandoc document metadata blocks block
type MetaBlocks []Block

const MetaBlocksTag = Tag("MetaBlocks")

func (MetaBlocks) Tag() Tag { return MetaBlocksTag }
func (MetaBlocks) element() {}
func (MetaBlocks) meta()    {}

// Pandoc document metadata boolean
type MetaBool bool

const MetaBoolTag = Tag("MetaBool")

func (MetaBool) Tag() Tag { return MetaBoolTag }
func (MetaBool) element() {}
func (MetaBool) meta()    {}

// Pandoc document metadata string
type MetaString string

const MetaStringTag = Tag("MetaString")

func (MetaString) Tag() Tag { return MetaStringTag }
func (MetaString) element() {}
func (MetaString) meta()    {}

type AttrKV struct {
	Key   string
	Value string
}

type Attr struct {
	Id      string
	Classes []string
	Attrs   []AttrKV
}

// Text (string)
type Str struct {
	Str string
}

const StrTag = Tag("Str")

func (Str) Tag() Tag  { return StrTag }
func (*Str) inline()  {}
func (*Str) element() {}

// Emphasized text (list of inlines)
type Emph struct {
	Inlines []Inline
}

const EmphTag = Tag("Emph")

func (Emph) Tag() Tag  { return EmphTag }
func (*Emph) inline()  {}
func (*Emph) element() {}

// Underlined text (list of inlines)
type Underline struct {
	Inlines []Inline
}

const UnderlineTag = Tag("Underline")

func (Underline) Tag() Tag  { return UnderlineTag }
func (*Underline) inline()  {}
func (*Underline) element() {}

// Strongly emphasized text (list of inlines)
type Strong struct {
	Inlines []Inline
}

const StrongTag = Tag("Strong")

func (Strong) Tag() Tag  { return StrongTag }
func (*Strong) inline()  {}
func (*Strong) element() {}

// Strikeout text (list of inlines)
type Strikeout struct {
	Inlines []Inline
}

const StrikeoutTag = Tag("Strikeout")

func (Strikeout) Tag() Tag  { return StrikeoutTag }
func (*Strikeout) inline()  {}
func (*Strikeout) element() {}

// Superscripted text (list of inlines)
type Superscript struct {
	Inlines []Inline
}

const SuperscriptTag = Tag("Superscript")

func (Superscript) Tag() Tag  { return SuperscriptTag }
func (*Superscript) inline()  {}
func (*Superscript) element() {}

// Subscripted text (list of inlines)
type Subscript struct {
	Inlines []Inline
}

const SubscriptTag = Tag("Subscript")

func (Subscript) Tag() Tag  { return SubscriptTag }
func (*Subscript) inline()  {}
func (*Subscript) element() {}

// Small capitals (list of inlines)
type SmallCaps struct {
	Inlines []Inline
}

const SmallCapsTag = Tag("SmallCaps")

func (SmallCaps) Tag() Tag  { return SmallCapsTag }
func (*SmallCaps) inline()  {}
func (*SmallCaps) element() {}

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

func (Quoted) Tag() Tag  { return QuotedTag }
func (*Quoted) inline()  {}
func (*Quoted) element() {}

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
	NoteNum int64
	Hash    int64
}

// Citation (list of inlines)
type Cite struct {
	Citations []Citation
	Inlines   []Inline
}

const CiteTag = Tag("Cite")

func (Cite) Tag() Tag  { return CiteTag }
func (*Cite) inline()  {}
func (*Cite) element() {}

// Inline code (literal)
type Code struct {
	Attr Attr
	Text string
}

const CodeTag = Tag("Code")

func (Code) Tag() Tag  { return CodeTag }
func (*Code) inline()  {}
func (*Code) element() {}

var SP = &Space{}

// Inter-word space
type Space struct {
}

const SpaceTag = Tag("Space")

func (Space) Tag() Tag  { return SpaceTag }
func (*Space) inline()  {}
func (*Space) element() {}

var SB = &SoftBreak{}

// Soft line break
type SoftBreak struct {
}

const SoftBreakTag = Tag("SoftBreak")

func (SoftBreak) Tag() Tag  { return SoftBreakTag }
func (*SoftBreak) inline()  {}
func (*SoftBreak) element() {}

var LB = &LineBreak{}

// Hard line break
type LineBreak struct {
}

const LineBreakTag = Tag("LineBreak")

func (LineBreak) Tag() Tag  { return LineBreakTag }
func (*LineBreak) inline()  {}
func (*LineBreak) element() {}

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

func (Math) Tag() Tag  { return MathTag }
func (*Math) inline()  {}
func (*Math) element() {}

// Raw inline
type RawInline struct {
	Format string
	Text   string
}

const RawInlineTag = Tag("RawInline")

func (RawInline) Tag() Tag  { return RawInlineTag }
func (*RawInline) element() {}
func (*RawInline) inline()  {}

type Target struct {
	Url   string
	Title string
}

// Hyperlink: alt text (list of inlines), target
type Link struct {
	Attr    Attr
	Inlines []Inline
	Target  Target
}

const LinkTag = Tag("Link")

func (Link) Tag() Tag  { return LinkTag }
func (*Link) inline()  {}
func (*Link) element() {}

// Image: alt text (list of inlines), target
type Image struct {
	Attr    Attr
	Inlines []Inline
	Target  Target
}

const ImageTag = Tag("Image")

func (Image) Tag() Tag  { return ImageTag }
func (*Image) element() {}
func (*Image) inline()  {}

// Footnote: list of blocks
type Note struct {
	Blocks []Block
}

const NoteTag = Tag("Note")

func (Note) Tag() Tag  { return NoteTag }
func (*Note) element() {}
func (*Note) inline()  {}

// Generic inline container with attributes
type Span struct {
	Attr    Attr
	Inlines []Inline
}

const SpanTag = Tag("Span")

func (Span) Tag() Tag  { return SpanTag }
func (*Span) inline()  {}
func (*Span) element() {}

// Plain text, not a paragraph
type Plain struct {
	Inlines []Inline
}

const PlainTag = Tag("Plain")

func (Plain) Tag() Tag  { return PlainTag }
func (*Plain) block()   {}
func (*Plain) element() {}

// Paragraph (list of inlines)
type Para struct {
	Inlines []Inline
}

const ParaTag = Tag("Para")

func (Para) Tag() Tag  { return ParaTag }
func (*Para) block()   {}
func (*Para) element() {}

// Multiple non-breaking lines
type LineBlock struct {
	Inlines [][]Inline
}

const LineBlockTag = Tag("LineBlock")

func (LineBlock) Tag() Tag  { return LineBlockTag }
func (*LineBlock) block()   {}
func (*LineBlock) element() {}

// Code block (literal)
type CodeBlock struct {
	Attr Attr
	Text string
}

const CodeBlockTag = Tag("CodeBlock")

func (CodeBlock) Tag() Tag  { return CodeBlockTag }
func (*CodeBlock) block()   {}
func (*CodeBlock) element() {}

// Raw block
type RawBlock struct {
	Format string
	Text   string
}

const RawBlockTag = Tag("RawBlock")

func (RawBlock) Tag() Tag  { return RawBlockTag }
func (*RawBlock) block()   {}
func (*RawBlock) element() {}

// Block quote (list of blocks)
type BlockQuote struct {
	Blocks []Block
}

const BlockQuoteTag = Tag("BlockQuote")

func (BlockQuote) Tag() Tag  { return BlockQuoteTag }
func (*BlockQuote) block()   {}
func (*BlockQuote) element() {}

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
	Start     int64
	Style     ListNumberStyle
	Delimiter ListNumberDelim
}

// Ordered list (attributes and a list of items, each a list of blocks)
type OrderedList struct {
	Attr  ListAttrs
	Items [][]Block
}

const OrderedListTag = Tag("OrderedList")

func (OrderedList) Tag() Tag  { return OrderedListTag }
func (*OrderedList) block()   {}
func (*OrderedList) element() {}

// Bullet list (list of items, each a list of blocks)
type BulletList struct {
	Blocks [][]Block
}

const BulletListTag = Tag("BulletList")

func (BulletList) Tag() Tag  { return BulletListTag }
func (*BulletList) block()   {}
func (*BulletList) element() {}

type Definition struct {
	Term       []Inline
	Definition [][]Block
}

// Definition list (list of items, each a pair of inlines and a list of blocks)
type DefinitionList struct {
	Items []Definition
}

const DefinitionListTag = Tag("DefinitionList")

func (DefinitionList) Tag() Tag  { return DefinitionListTag }
func (*DefinitionList) block()   {}
func (*DefinitionList) element() {}

var HR = &HorizontalRule{}

// Horizontal rule
type HorizontalRule struct {
}

const HorizontalRuleTag = Tag("HorizontalRule")

func (HorizontalRule) Tag() Tag  { return HorizontalRuleTag }
func (*HorizontalRule) block()   {}
func (*HorizontalRule) element() {}

// Header - level (integer) and text (inlines)
type Header struct {
	Level   int64
	Attr    Attr
	Inlines []Inline
}

const HeaderTag = Tag("Header")

func (Header) Tag() Tag  { return HeaderTag }
func (*Header) block()   {}
func (*Header) element() {}

type Caption struct {
	HasShort bool
	Short    []Inline
	Long     []Block
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
	Attr Attr
	Rows []TableRow
}

type TableRow struct {
	Attr  Attr
	Cells []TableCell
}

type TableCell struct {
	Attr    Attr
	Align   Alignment
	RowSpan int64
	ColSpan int64
	Blocks  []Block
}

type TableBody struct {
	Attr           Attr
	RowHeadColumns int64
	Head           []TableRow
	Body           []TableRow
}

// Table, with attributes, caption, optional short caption, column alignments
// and widths (required), table head, table bodies, and table foot
type Table struct {
	Attr    Attr
	Caption Caption
	Aligns  []ColSpec
	Head    TableHeadFoot
	Bodies  []TableBody
	Foot    TableHeadFoot
}

const TableTag = Tag("Table")

func (Table) Tag() Tag  { return TableTag }
func (*Table) block()   {}
func (*Table) element() {}

// Figure, with attributes, caption, and content (list of blocks)
type Figure struct {
	Attr    Attr
	Caption Caption
	Blocks  []Block
}

const FigureTag = Tag("Figure")

func (Figure) Tag() Tag  { return FigureTag }
func (*Figure) block()   {}
func (*Figure) element() {}

// Generic block container with attributes
type Div struct {
	Attr   Attr
	Blocks []Block
}

const DivTag = Tag("Div")

func (Div) Tag() Tag  { return DivTag }
func (*Div) block()   {}
func (*Div) element() {}
