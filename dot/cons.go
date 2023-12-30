package dot

import "github.com/growler/go-pandoc"

var (
	Continue = pandoc.Continue
	Replace  = pandoc.Replace
	Skip     = pandoc.Skip
	SkipAll  = pandoc.SkipAll
)

func Blocks(b ...pandoc.Block) []pandoc.Block {
	return b
}

func Inlines(i ...pandoc.Inline) []pandoc.Inline {
	return i
}

// Text (string)
func Str(s string) pandoc.Inline { 
	return &pandoc.Str{Text: s} 
}

// Emphasized text (list of inlines)
func Emph(i ...pandoc.Inline) *pandoc.Emph { 
	return &pandoc.Emph{Inlines: i} 
}

// Underlined text (list of inlines)
func Underline(i ...pandoc.Inline) *pandoc.Underline { 
	return &pandoc.Underline{Inlines: i} 
}

// Strongly emphasized text (list of inlines)
func Strong(i ...pandoc.Inline) *pandoc.Strong {
	return &pandoc.Strong{Inlines: i}
}

// Strikeout text (list of inlines)
func Strikeout(i ...pandoc.Inline) *pandoc.Strikeout {
	return &pandoc.Strikeout{Inlines: i}
}

// Superscripted text (list of inlines)
func Superscript(i ...pandoc.Inline) *pandoc.Superscript {
	return &pandoc.Superscript{Inlines: i}
}

// Subscripted text (list of inlines)
func Subscript(i ...pandoc.Inline) *pandoc.Subscript {
	return &pandoc.Subscript{Inlines: i}
}

// Small capitals (list of inlines)
func SmallCaps(i ...pandoc.Inline) *pandoc.SmallCaps {
	return &pandoc.SmallCaps{Inlines: i}
}

const (
	DoubleQuote = pandoc.DoubleQuote
	SingleQuote = pandoc.SingleQuote
)

// Quoted text (list of inlines). The first argument is the quote type.
func Quoted(t pandoc.QuoteType, i ...pandoc.Inline) *pandoc.Quoted {
	return &pandoc.Quoted{QuoteType: t, Inlines: i}
}

const (
	NormalCitation = pandoc.NormalCitation
	SuppressAuthor = pandoc.SuppressAuthor
	AuthorInText   = pandoc.AuthorInText
)

func Citation(id string, mode pandoc.CitationMode, noteNum int, prefix, suffix []pandoc.Inline) pandoc.Citation {
	return pandoc.Citation{
		Id:      id,
		Prefix:  prefix,
		Suffix:  suffix,
		Mode:    mode,
		NoteNum: noteNum,
	}
}

// Citation (list of inlines as citation prefix).
func Cite(c ...pandoc.Citation) *pandoc.Cite {
	return &pandoc.Cite{Citations: c}
}

// Inline code (literal). The first argument is the span attributes.
func Code(attr pandoc.Attr, text string) *pandoc.Code {
	return &pandoc.Code{Attr: attr, Text: text}
}

// Inter-word space
func Space() pandoc.Inline { return pandoc.SP }

// Soft line break
func SoftBreak() pandoc.Inline { return pandoc.SB }

// Hard line break
func LineBreak() pandoc.Inline { return pandoc.LB }

const (
	DisplayMath = pandoc.DisplayMath
	InlineMath  = pandoc.InlineMath
)

// TeX math (literal). The first argument is the math type.
func Math(t pandoc.MathType, text string) *pandoc.Math { 
	return &pandoc.Math{MathType: t, Text: text}
}

// Raw inline (literal). The first argument is the format
// the literal must be export in.
func RawInline(format string, text string) *pandoc.RawInline {
	return &pandoc.RawInline{Format: format, Text: text}
}

// Link (list of inlines as link text).
func Link(attr pandoc.Attr, url string, title string, i ...pandoc.Inline) *pandoc.Link {
	return &pandoc.Link{Attr: attr, Target: pandoc.Target{Url: url, Title: title}, Inlines: i}
}

// Image (list of inlines as alternate text).
func Image(attr pandoc.Attr, url string, title string, i ...pandoc.Inline) *pandoc.Image {
	return &pandoc.Image{Attr: attr, Target: pandoc.Target{Url: url, Title: title}, Inlines: i}
}

// Footnote or endnote (list of blocks)
func Note(i ...pandoc.Block) pandoc.Inline { 
	return &pandoc.Note{Blocks: i} 
}

// Generic inline container with attributes.
func Span(attr pandoc.Attr, i ...pandoc.Inline) *pandoc.Span {
	return &pandoc.Span{Attr: attr, Inlines: i}
}

// Horizontal rule.
func HorizontalRule() pandoc.Block {
	return pandoc.HR
}

var NoAttr = pandoc.Attr{}

func KVs(kvs ...string) []pandoc.KV {
	var res = make([]pandoc.KV, 0, len(kvs)/2)
	for i := 0; i < len(kvs)-1; i += 2 {
		res = append(res, pandoc.KV{Key: kvs[i], Value: kvs[i+1]})
	}
	return res
}

func Attr(id string, classes ...string) pandoc.Attr {
	return pandoc.Attr{Id: id, Classes: classes}
}

func AttrKVs(id string, kvs []pandoc.KV, classes ...string) pandoc.Attr {
	return pandoc.Attr{Id: id, Classes: classes, KVs: kvs}
}

func Plain(i ...pandoc.Inline) *pandoc.Plain {
	return &pandoc.Plain{Inlines: i}
}

func Para(i ...pandoc.Inline) *pandoc.Para {
	return &pandoc.Para{Inlines: i}
}

func BulletList(i ...[]pandoc.Block) *pandoc.BulletList {
	return &pandoc.BulletList{Blocks: i}
}

func CodeBlock(attr pandoc.Attr, text string) *pandoc.CodeBlock {
	return &pandoc.CodeBlock{Attr: attr, Text: text}
}


func Div(attr pandoc.Attr, i ...pandoc.Block) *pandoc.Div {
	return &pandoc.Div{Attr: attr, Blocks: i}
}

func Header(level int, attr pandoc.Attr, i ...pandoc.Inline) *pandoc.Header {
	return &pandoc.Header{Level: level, Attr: attr, Inlines: i}
}

func RawBlock(format string, text string) *pandoc.RawBlock {
	return &pandoc.RawBlock{Format: format, Text: text}
}

func Filter[P any, E pandoc.Element, R pandoc.Element](elt E, fun func(P) ([]R, error)) (E, error) {
	return pandoc.Filter[P, E, R](elt, fun)
}

func QueryE[P any, E pandoc.Element](elt E, fun func(P) error) error {
	return pandoc.QueryE[P, E](elt, fun)
}

func Query[P any, E pandoc.Element](elt E, fun func(P)) {
	pandoc.Query[P, E](elt, fun)
}

func StringToIdent(s string) string {
	return pandoc.StringToIdent(s)
}

func InlinesToIdent(inlines []pandoc.Inline) string {
	return pandoc.InlinesToIdent(inlines)
}

func Match[T pandoc.Element, E pandoc.Element](m T, e E) (T, bool) {
	return pandoc.Match[T, E](m, e)
}

func Index[E pandoc.Element, L pandoc.Element](lst []L) (int, E) {
	return pandoc.Index[E, L](lst)
}

func Index2[E1 pandoc.Element, E2 pandoc.Element, L pandoc.Element](lst []L) (int, E1, E2) {
	return pandoc.Index2[E1, E2, L](lst)
}

func Index3[E1 pandoc.Element, E2 pandoc.Element, E3 pandoc.Element, L pandoc.Element](lst []L) (int, E1, E2, E3) {
	return pandoc.Index3[E1, E2, E3, L](lst)
}
