package pandoc

import (
	"fmt"
	"io"
)

// ----------- inlines -------------

func readInline(s *scanner) (ret Inline, err error) {
	if err := s.expect(tokLBrace); err != nil {
		return nil, err
	}
	if err := s.expectString("t"); err != nil {
		return nil, err
	}
	if err := s.expect(tokColon); err != nil {
		return nil, err
	}
	if err := s.expect(tokStr); err != nil {
		return nil, err
	}
	if !s.stringInBuffer() {
		return nil, errorf("expected tag, got %s", s.string())
	}
	switch Tag(s.buf[s.str : s.pos-1]) {
	case SpaceTag:
		return readEmptyObj(SP)(s)
	case SoftBreakTag:
		return readEmptyObj(SB)(s)
	case LineBreakTag:
		return readEmptyObj(LB)(s)
	case StrTag:
		return readObj(readStr)(s)
	case EmphTag:
		return readObj(readEmph)(s)
	case UnderlineTag:
		return readObj(readUnderline)(s)
	case StrongTag:
		return readObj(readStrong)(s)
	case StrikeoutTag:
		return readObj(readStrikeout)(s)
	case SuperscriptTag:
		return readObj(readSuperscript)(s)
	case SubscriptTag:
		return readObj(readSubscript)(s)
	case SmallCapsTag:
		return readObj(readSmallCaps)(s)
	case QuotedTag:
		return readObj(readQuoted)(s)
	case CodeTag:
		return readObj(readCode)(s)
	case SpanTag:
		return readObj(readSpan)(s)
	case RawInlineTag:
		return readObj(readRawInline)(s)
	case MathTag:
		return readObj(readMath)(s)
	case CiteTag:
		return readObj(readCite)(s)
	case LinkTag:
		return readObj(readLink)(s)
	case ImageTag:
		return readObj(readImage)(s)
	case NoteTag:
		return readObj(readNote)(s)
	default:
		return nil, errorf("unknown inline type %q", s.string())
	}
}

// Str
func readStr(s *scanner) (*Str, error) {
	if str, err := readString(s); err != nil {
		return nil, err
	} else {
		return &Str{str}, nil
	}
}

// Emph
func readEmph(s *scanner) (*Emph, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &Emph{list}, nil
	}
}

// Underline
func readUnderline(s *scanner) (*Underline, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &Underline{list}, nil
	}
}

// Strong
func readStrong(s *scanner) (*Strong, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &Strong{list}, nil
	}
}

// Strikeout
func readStrikeout(s *scanner) (*Strikeout, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &Strikeout{list}, nil
	}
}

// Superscript
func readSuperscript(s *scanner) (*Superscript, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &Superscript{list}, nil
	}
}

// Subscript
func readSubscript(s *scanner) (*Subscript, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &Subscript{list}, nil
	}
}

// SmallCaps
func readSmallCaps(s *scanner) (*SmallCaps, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &SmallCaps{list}, nil
	}
}

var readQuoteType = readTags(SingleQuote, DoubleQuote)

// Quoted
func readQuoted(s *scanner) (*Quoted, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	typ, tup, err := readItem(readQuoteType)(s, tup)
	if err != nil {
		return nil, err
	}
	inlines, _, err := readItem(listr(readInline))(s, tup)
	if err != nil {
		return nil, err
	}
	return &Quoted{typ, inlines}, nil
}

// RawInline
func readRawInline(s *scanner) (*RawInline, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	format, tup, err := readItem(readString)(s, tup)
	if err != nil {
		return nil, err
	}
	str, _, err := readItem(readString)(s, tup)
	if err != nil {
		return nil, err
	}
	return &RawInline{format, str}, nil
}

var readMathType = readTags(DisplayMath, InlineMath)

// Math
func readMath(s *scanner) (*Math, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	typ, tup, err := readItem(readMathType)(s, tup)
	if err != nil {
		return nil, err
	}
	str, _, err := readItem(readString)(s, tup)
	if err != nil {
		return nil, err
	}
	return &Math{typ, str}, nil
}

// Code
func readCode(s *scanner) (*Code, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	code, _, err := readItem(readString)(s, tup)
	if err != nil {
		return nil, err
	}
	return &Code{attr, code}, nil
}

// Span
func readSpan(s *scanner) (*Span, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	inlines, _, err := readItem(listr(readInline))(s, tup)
	if err != nil {
		return nil, err
	}
	return &Span{attr, inlines}, nil
}

var readCitationMode = readTags(AuthorInText, SuppressAuthor, NormalCitation)

func readCitation(s *scanner) (citation Citation, err error) {
	if err := s.expect(tokLBrace); err != nil {
		return Citation{}, err
	}
	for i := 6; i > 0; i-- {
		if err := s.expect(tokStr); err != nil {
			return Citation{}, err
		}
		if !s.stringInBuffer() {
			return Citation{}, errorf("expected string, got %s", s.string())
		}
		switch string(s.buf[s.str : s.pos-1]) {
		case "citationId":
			citation.Id, err = readField(s, i, readString)
		case "citationPrefix":
			citation.Prefix, err = readField(s, i, listr(readInline))
		case "citationSuffix":
			citation.Suffix, err = readField(s, i, listr(readInline))
		case "citationMode":
			citation.Mode, err = readField(s, i, readCitationMode)
		case "citationNoteNum":
			citation.NoteNum, err = readField(s, i, readInt)
		case "citationHash":
			citation.Hash, err = readField(s, i, readInt)	
		default:
			return Citation{}, errorf("unknown citation field %q", s.string())
		}
		if err != nil {
			return Citation{}, err
		}
	}
	return citation, nil
}

// Cite
func readCite(s *scanner) (*Cite, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	citations, tup, err := readItem(listr(readCitation))(s, tup)
	if err != nil {
		return nil, err
	}
	inlines, _, err := readItem(listr(readInline))(s, tup)
	if err != nil {
		return nil, err
	}
	return &Cite{citations, inlines}, nil
}

// Note
func readNote(s *scanner) (*Note, error) {
	blocks, err := listr(readBlock)(s)
	if err != nil {
		return nil, err
	}
	return &Note{blocks}, nil
}

func readImage(s *scanner) (Inline, error) {
	tup, err := tupler(s, 3)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	inlines, tup, err := readItem(listr(readInline))(s, tup)
	if err != nil {
		return nil, err
	}
	target, _, err := readItem(readTarget)(s, tup)
	if err != nil {
		return nil, err
	}
	return &Image{attr, inlines, target}, nil
}

func readLink(s *scanner) (Inline, error) {
	tup, err := tupler(s, 3)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	inlines, tup, err := readItem(listr(readInline))(s, tup)
	if err != nil {
		return nil, err
	}
	target, _, err := readItem(readTarget)(s, tup)
	if err != nil {
		return nil, err
	}
	return &Link{attr, inlines, target}, nil
}

// ----------- blocks -------------

func readBlock(s *scanner) (ret Block, err error) {
	if err := s.expect(tokLBrace); err != nil {
		return nil, err
	}
	if err := s.expectString("t"); err != nil {
		return nil, err
	}
	if err := s.expect(tokColon); err != nil {
		return nil, err
	}
	if err := s.expect(tokStr); err != nil {
		return nil, err
	}
	if !s.stringInBuffer() {
		return nil, errorf("expected tag, got %s", s.string())
	}
	switch Tag(s.buf[s.str : s.pos-1]) {
	case HorizontalRuleTag:
		return readEmptyObj(HR)(s)
	case PlainTag:
		return readObj(readPlain)(s)
	case ParaTag:
		return readObj(readPara)(s)
	case HeaderTag:
		return readObj(readHeader)(s)
	case CodeBlockTag:
		return readObj(readCodeBlock)(s)
	case DivTag:
		return readObj(readDiv)(s)
	case LineBlockTag:
		return readObj(readLineBlock)(s)
	case RawBlockTag:
		return readObj(readRawBlock)(s)
	case FigureTag:
		return readObj(readFigure)(s)
	case TableTag:
		return readObj(readTable)(s)
	case DefinitionListTag:
		return readObj(readDefinitionList)(s)
	case BulletListTag:
		return readObj(readBulletList)(s)
	case OrderedListTag:
		return readObj(readOrderedList)(s)
	case BlockQuoteTag:
		return readObj(readBlockQuote)(s)
	default:
		return nil, errorf("unknown block type %q", s.string())
	}
}

// Plain
func readPlain(s *scanner) (*Plain, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &Plain{list}, nil
	}
}

// Para
func readPara(s *scanner) (*Para, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &Para{list}, nil
	}
}

// CodeBlock
func readCodeBlock(s *scanner) (*CodeBlock, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	code, _, err := readItem(readString)(s, tup)
	if err != nil {
		return nil, err
	}
	return &CodeBlock{attr, code}, nil
}

// Div
func readDiv(s *scanner) (*Div, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	blocks, _, err := readItem(listr(readBlock))(s, tup)
	if err != nil {
		return nil, err
	}
	return &Div{attr, blocks}, nil
}

// RawBlock
func readRawBlock(s *scanner) (*RawBlock, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	format, tup, err := readItem(readString)(s, tup)
	if err != nil {
		return nil, err
	}
	text, _, err := readItem(readString)(s, tup)
	if err != nil {
		return nil, err
	}
	return &RawBlock{format, text}, nil
}

// Header
func readHeader(s *scanner) (*Header, error) {
	tup, err := tupler(s, 3)
	if err != nil {
		return nil, err
	}
	lvl, tup, err := readItem(readInt)(s, tup)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	inlines, _, err := readItem(listr(readInline))(s, tup)
	if err != nil {
		return nil, err
	}
	return &Header{attr, lvl, inlines}, nil
}

// LineBlock
func readLineBlock(s *scanner) (*LineBlock, error) {
	if list, err := dlistr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &LineBlock{list}, nil
	}
}

// BulletList
func readBulletList(s *scanner) (*BulletList, error) {
	list, err := dlistr(readBlock)(s)
	if err != nil {
		return nil, err
	}
	return &BulletList{list}, nil
}

// Figure
func readFigure(s *scanner) (*Figure, error) {
	tup, err := tupler(s, 3)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	caption, tup, err := readItem(readCaption)(s, tup)
	if err != nil {
		return nil, err
	}
	content, _, err := readItem(listr(readBlock))(s, tup)
	if err != nil {
		return nil, err
	}
	return &Figure{attr, caption, content}, nil
}

// OrderedList
func readOrderedList(s *scanner) (*OrderedList, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readListAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	list, _, err := readItem(dlistr(readBlock))(s, tup)
	if err != nil {
		return nil, err
	}
	return &OrderedList{attr, list}, nil

}

// Table
func readTable(s *scanner) (*Table, error) {
	tup, err := tupler(s, 6)
	if err != nil {
		return nil, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return nil, err
	}
	caption, tup, err := readItem(readCaption)(s, tup)
	if err != nil {
		return nil, err
	}
	colSpec, tup, err := readItem(listr(readColSpec))(s, tup)
	if err != nil {
		return nil, err
	}
	head, tup, err := readItem(readTableHeadFoot)(s, tup)
	if err != nil {
		return nil, err
	}
	bodies, tup, err := readItem(listr(readTableBody))(s, tup)
	if err != nil {
		return nil, err
	}
	foot, _, err := readItem(readTableHeadFoot)(s, tup)
	if err != nil {
		return nil, err
	}
	return &Table{attr, caption, colSpec, head, bodies, foot}, nil
}

// DefinitionList
func readDefinitionList(s *scanner) (*DefinitionList, error) {
	list, err := listr(readDefinition)(s)
	if err != nil {
		return nil, err
	}
	return &DefinitionList{list}, nil
}

// BlockQuote
func readBlockQuote(s *scanner) (*BlockQuote, error) {
	list, err := listr(readBlock)(s)
	if err != nil {
		return nil, err
	}
	return &BlockQuote{list}, nil
}

// ----------- other types -------------

var readListNumberStyle = readTags(DefaultStyle, Example, Decimal, LowerRoman, UpperRoman, LowerAlpha, UpperAlpha)
var readListNumberDelim = readTags(DefaultDelim, Period, OneParen, TwoParens)

func readListAttr(s *scanner) (ListAttrs, error) {
	tup, err := tupler(s, 3)
	if err != nil {
		return ListAttrs{}, err
	}
	start, tup, err := readItem(readInt)(s, tup)
	if err != nil {
		return ListAttrs{}, err
	}
	style, tup, err := readItem(readListNumberStyle)(s, tup)
	if err != nil {
		return ListAttrs{}, err
	}
	delim, _, err := readItem(readListNumberDelim)(s, tup)
	if err != nil {
		return ListAttrs{}, err
	}
	return ListAttrs{start, style, delim}, nil
}

func readDefinition(s *scanner) (Definition, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return Definition{}, err
	}
	terms, tup, err := readItem(listr(readInline))(s, tup)
	if err != nil {
		return Definition{}, err
	}
	defs, _, err := readItem(dlistr(readBlock))(s, tup)
	if err != nil {
		return Definition{}, err
	}
	return Definition{terms, defs}, nil
}

func readTableBody(s *scanner) (TableBody, error) {
	tup, err := tupler(s, 4)
	if err != nil {
		return TableBody{}, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return TableBody{}, err
	}
	rhc, tup, err := readItem(readInt)(s, tup)
	if err != nil {
		return TableBody{}, err
	}
	head, tup, err := readItem(listr(readTableRow))(s, tup)
	if err != nil {
		return TableBody{}, err
	}
	body, _, err := readItem(listr(readTableRow))(s, tup)
	if err != nil {
		return TableBody{}, err
	}
	return TableBody{attr, rhc, head, body}, nil
}

func readTableCell(s *scanner) (TableCell, error) {
	tup, err := tupler(s, 5)
	if err != nil {
		return TableCell{}, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return TableCell{}, err
	}
	align, tup, err := readItem(readAlignment)(s, tup)
	if err != nil {
		return TableCell{}, err
	}
	rowspan, tup, err := readItem(readInt)(s, tup)
	if err != nil {
		return TableCell{}, err
	}
	colspan, tup, err := readItem(readInt)(s, tup)
	if err != nil {
		return TableCell{}, err
	}
	blocks, _, err := readItem(listr(readBlock))(s, tup)
	if err != nil {
		return TableCell{}, err
	}
	return TableCell{attr, align, rowspan, colspan, blocks}, nil
}

func readTableRow(s *scanner) (TableRow, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return TableRow{}, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return TableRow{}, err
	}
	cells, _, err := readItem(listr(readTableCell))(s, tup)
	if err != nil {
		return TableRow{}, err
	}
	return TableRow{attr, cells}, nil
}

func readTableHeadFoot(s *scanner) (TableHeadFoot, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return TableHeadFoot{}, err
	}
	attr, tup, err := readItem(readAttr)(s, tup)
	if err != nil {
		return TableHeadFoot{}, err
	}
	rows, _, err := readItem(listr(readTableRow))(s, tup)
	if err != nil {
		return TableHeadFoot{}, err
	}
	return TableHeadFoot{attr, rows}, nil
}

var readAlignment = readTags(AlignLeft, AlignRight, AlignCenter, AlignDefault)

func readColWidth(s *scanner) (ColWidth, error) {
	if err := s.expect(tokLBrace); err != nil {
		return ColWidth{}, err
	}
	if err := s.expectString("t"); err != nil {
		return ColWidth{}, err
	}
	if err := s.expect(tokColon); err != nil {
		return ColWidth{}, err
	}
	if err := s.expect(tokStr); err != nil {
		return ColWidth{}, err
	}
	if !s.stringInBuffer() {
		return ColWidth{}, errorf("expected tag, got %s", s.string())
	}
	switch Tag(s.buf[s.str : s.pos-1]) {
	case _ColWidthDefault:
		if err := s.expect(tokRBrace); err != nil {
			return ColWidth{}, err
		}
		return ColWidth{0, true}, nil
	case _ColWidth:
		if flt, err := readObj(readFloat)(s); err != nil {
			return ColWidth{}, err
		} else {
			return ColWidth{flt, false}, nil
		}
	default:
		return ColWidth{}, errorf("unknown col width type %q", s.string())
	}
}

func readColSpec(s *scanner) (ColSpec, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return ColSpec{}, err
	}
	align, tup, err := readItem(readAlignment)(s, tup)
	if err != nil {
		return ColSpec{}, err
	}
	width, _, err := readItem(readColWidth)(s, tup)
	if err != nil {
		return ColSpec{}, err
	}
	return ColSpec{align, width}, nil
}

func readCaption(s *scanner) (Caption, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return Caption{}, err
	}
	var hasShort bool
	var short []Inline
	if tok := s.peek(); tok == tokNull {
		_, tup, err = readItem(readNull)(s, tup)
		if err != nil {
			return Caption{}, err
		}
	} else {
		hasShort = true
		short, tup, err = readItem(listr(readInline))(s, tup)
		if err != nil {
			return Caption{}, err
		}
	}
	long, _, err := readItem(listr(readBlock))(s, tup)
	if err != nil {
		return Caption{}, err
	}
	return Caption{hasShort, short, long}, nil
}

func readAttrKV(s *scanner) (KV, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return KV{}, err
	}
	key, tup, err := readItem(readString)(s, tup)
	if err != nil {
		return KV{}, err
	}
	val, _, err := readItem(readString)(s, tup)
	if err != nil {
		return KV{}, err
	}
	return KV{key, val}, nil
}

func readAttr(s *scanner) (Attr, error) {
	tup, err := tupler(s, 3)
	if err != nil {
		return Attr{}, err
	}
	id, tup, err := readItem(readString)(s, tup)
	if err != nil {
		return Attr{}, err
	}
	classes, tup, err := readItem(listr(readString))(s, tup)
	if err != nil {
		return Attr{}, err
	}
	kvs, _, err := readItem(listr(readAttrKV))(s, tup)
	if err != nil {
		return Attr{}, err
	}
	return Attr{id, classes, kvs}, nil
}

func readTarget(s *scanner) (Target, error) {
	tup, err := tupler(s, 2)
	if err != nil {
		return Target{}, err
	}
	url, tup, err := readItem(readString)(s, tup)
	if err != nil {
		return Target{}, err
	}
	title, _, err := readItem(readString)(s, tup)
	if err != nil {
		return Target{}, err
	}
	return Target{url, title}, nil
}

// ----------- meta -------------

func readMetaValue(s *scanner) (MetaValue, error) {
	if err := s.expect(tokLBrace); err != nil {
		return nil, err
	}
	if err := s.expectString("t"); err != nil {
		return nil, err
	}
	if err := s.expect(tokColon); err != nil {
		return nil, err
	}
	if err := s.expect(tokStr); err != nil {
		return nil, err
	}
	if !s.stringInBuffer() {
		return nil, errorf("expected tag, got %s", s.string())
	}
	switch Tag(s.buf[s.str : s.pos-1]) {
	case MetaMapTag:
		return readObj(readMetaMap)(s)
	case MetaListTag:
		return readObj(readMetaList)(s)
	case MetaBoolTag:
		return readObj(readMetaBool)(s)
	case MetaStringTag:
		return readObj(readMetaString)(s)
	case MetaInlinesTag:
		return readObj(readMetaInlines)(s)
	case MetaBlocksTag:
		return readObj(readMetaBlocks)(s)
	default:
		return nil, errorf("unknown meta value type %q", s.string())
	}
}

func readMetaMap(s *scanner) (*MetaMap, error) {
	if m, err := readMeta(s); err != nil {
		return nil, err
	} else {
		return &MetaMap{m}, nil
	}
}

func readMeta(s *scanner) ([]MetaMapEntry, error) {
	if err := s.expect(tokLBrace); err != nil {
		return nil, err
	}
	var m []MetaMapEntry
	for {
		if tok := s.peek(); tok == tokRBrace {
			s.next()
			break
		}
		if err := s.expect(tokStr); err != nil {
			return nil, err
		}
		key := s.string()
		if err := s.expect(tokColon); err != nil {
			return nil, err
		}
		if val, err := readMetaValue(s); err != nil {
			return nil, err
		} else {
			m = append(m, MetaMapEntry{key, val})
		}
		if tok := s.next(); tok == tokRBrace {
			break
		} else if tok != tokComma {
			return nil, errorf("expected comma or right bracket, got %s", tok)
		}
	}
	return m, nil
}

func readMetaBool(s *scanner) (MetaBool, error) {
	if b, err := readBool(s); err != nil {
		return false, err
	} else {
		return MetaBool(b), nil
	}
}

func readMetaList(s *scanner) (*MetaList, error) {
	if list, err := listr(readMetaValue)(s); err != nil {
		return nil, err
	} else {
		return &MetaList{list}, nil
	}
}

func readMetaString(s *scanner) (MetaString, error) {
	if str, err := readString(s); err != nil {
		return "", err
	} else {
		return MetaString(str), nil
	}
}

func readMetaInlines(s *scanner) (*MetaInlines, error) {
	if list, err := listr(readInline)(s); err != nil {
		return nil, err
	} else {
		return &MetaInlines{list}, nil
	}
}

func readMetaBlocks(s *scanner) (*MetaBlocks, error) {
	if list, err := listr(readBlock)(s); err != nil {
		return nil, err
	} else {
		return &MetaBlocks{list}, nil
	}
}

// ----------- helpers -------------

// reads content of a tagged object
func readObj[T any](r func(*scanner) (T, error)) func(*scanner) (T, error) {
	return func(s *scanner) (ret T, err error) {
		if err = s.expect(tokComma); err != nil {
			return
		}
		if err = s.expectString("c"); err != nil {
			return
		}
		if err = s.expect(tokColon); err != nil {
			return
		}
		if ret, err = r(s); err != nil {
			return
		}
		if err = s.expect(tokRBrace); err != nil {
			return
		}
		return
	}
}

// reads empty tagged object
func readEmptyObj[T any](v T) func(*scanner) (T, error) {
	return func(s *scanner) (ret T, err error) {
		if err = s.expect(tokRBrace); err != nil {
			return
		}
		return v, nil
	}
}

// reads one of the tags
func readTags[T ~string](tags ...T) func(*scanner) (T, error) {
	var m = make(map[string]T, len(tags))
	for _, elt := range tags {
		m[string(elt)] = elt
	}
	return func(s *scanner) (ret T, err error) {
		if err = s.expect(tokLBrace); err != nil {
			return
		}
		if err = s.expectString("t"); err != nil {
			return
		}
		if err = s.expect(tokColon); err != nil {
			return
		}
		if err = s.expect(tokStr); err != nil {
			return
		}
		if !s.stringInBuffer() {
			err = errorf("expected one of %v, got %s", tags, s.string())
			return
		}
		if elt, ok := m[string(s.buf[s.str:s.pos-1])]; !ok {
			err = errorf("expected one of %v, got %s", tags, s.string())
			return
		} else if err = s.expect(tokRBrace); err != nil {
			return
		} else {
			ret = elt
			return
		}
	}
}

// tuple reader
type tuple int

// creates a new tuple reader
func tupler(s *scanner, cnt int) (tuple, error) {
	if err := s.expect(tokLBrack); err != nil {
		return 0, err
	} else {
		return tuple(cnt), nil
	}
}

// reads tuple item
func readItem[T any](r func(*scanner) (T, error)) func(*scanner, tuple) (T, tuple, error) {
	return func(s *scanner, t tuple) (ret T, rt tuple, err error) {
		ret, err = r(s)
		if err != nil {
			return
		}
		rt = t - 1
		if rt == 0 {
			err = s.expect(tokRBrack)
			return
		} else {
			err = s.expect(tokComma)
			return
		}
	}
}

// reads a field of an object
func readField[T any](s *scanner, n int, r func(*scanner) (T, error)) (ret T, err error) {
	if err = s.expect(tokColon); err != nil {
		return
	}
	ret, err = r(s)
	if err != nil {
		return
	}
	if n == 1 {
		err = s.expect(tokRBrace)
	} else {
		err = s.expect(tokComma)
	}
	return
}

// ----------- list readers -------------

// a test reader that only verifies that the input is a list
func testlistr[T any, R func(*scanner) (T, error)](r R) func(*scanner) ([]T, error) {
	return func(s *scanner) ([]T, error) {
		if err := s.expect(tokLBrack); err != nil {
			return nil, err
		}
		for {
			_, err := r(s)
			if err != nil {
				return nil, err
			}
			off := s.current()
			if tok := s.next(); tok == tokRBrack {
				break
			} else if tok != tokComma {
				return nil, errorf("expected comma or right bracket, got %s at %d", tok, off)
			}
		}
		return nil, nil
	}
}

// a list of lists reader
func dlistr[T any, R func(*scanner) (T, error)](r R) func(*scanner) ([][]T, error) {
	return listr(listr(r))
}

// a list reader
func listr[T any, R func(*scanner) (T, error)](r R) func(*scanner) ([]T, error) {
	return func(s *scanner) ([]T, error) {
		ret := make([]T, 0, 1)
		if err := s.expect(tokLBrack); err != nil {
			return nil, err
		}
		for {
			if s.peek() == tokRBrack {
				s.next()
				break
			}
			item, err := r(s)
			if err != nil {
				return nil, err
			}
			ret = append(ret, item)
			off := s.current()
			if tok := s.next(); tok == tokRBrack {
				break
			} else if tok != tokComma {
				return nil, errorf("expected comma or right bracket, got %s at %d", tok, off)
			}
		}
		return ret, nil
	}
}

// int reader
func readInt(s *scanner) (int, error) {
	if err := s.expect(tokNumber); err != nil {
		return 0, err
	}
	return s.int(), nil
}

// int reader
func readFloat(s *scanner) (float64, error) {
	if err := s.expect(tokNumber); err != nil {
		return 0, err
	}
	return s.float(), nil
}

// null reader
func readNull(s *scanner) (any, error) {
	off := s.current()
	if tok := s.next(); tok == tokNull {
		return nil, nil
	} else {
		return nil, errorf("expected null, got %s at %d", tok, off)
	}
}

// bool reader
func readBool(s *scanner) (bool, error) {
	off := s.current()
	if tok := s.next(); tok == tokTrue {
		return true, nil
	} else if tok == tokFalse {
		return false, nil
	} else {
		return false, errorf("expected boolean, got %s at %d", tok, off)
	}
}

// string reader
func readString(s *scanner) (string, error) {
	if err := s.expect(tokStr); err != nil {
		return "", err
	}
	return s.string(), nil
}

func errorf(f string, a ...any) error {
	panic(fmt.Errorf(f, a...).Error())
}

// compares two semver versions
func cmpSemver(mine, their []int) int {
	var i int
	for i = 0; i < len(mine); i++ {
		if i >= len(their) {
			return 1
		}
		if mine[i] > their[i] {
			return 1
		}
		if mine[i] < their[i] {
			return -1
		}
	}
	if i < len(their) {
		return -1
	} else {
		return 0
	}
}

// ReadFrom parses a Pandoc AST JSON from the reader.
func ReadFrom(r io.Reader) (*Pandoc, error) {
	var s = scanner{}
	s.init(r)
	if err := s.expect(tokLBrace); err != nil {
		return nil, err
	}
	var (
		doc                            = &Pandoc{}
		err                            error
	)
	for i := 3; i > 0; i-- {
		if err := s.expect(tokStr); err != nil {
			return nil, err
		}
		if !s.stringInBuffer() {
			return nil, errorf("expected string, got %s", s.string())
		}
		switch string(s.buf[s.str : s.pos-1]) {
		case "pandoc-api-version":			
			if version, err := readField(&s, i, listr(readInt)); err != nil {
				return nil, err
			} else if cmpSemver(version, _Version) < 0 {
				return nil, errorf("unsupported pandoc version %v", version)
			}
		case "meta":
			if doc.Meta, err = readField(&s, i, readMeta); err != nil {
				return nil, err
			}
		case "blocks":
			if doc.Blocks, err = readField(&s, i, listr(readBlock)); err != nil {
				return nil, err
			}
		default:
			return nil, errorf("unknown pandoc field %q", s.string())
		}
	}
	return doc, nil
}
