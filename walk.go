package pandoc

import (
	"errors"
	"io"
	"strings"
	"unicode"
)

// AST traversal result (used by Filter and QueryE)
type traversalResult uint8

const (
	skipChildren = 1 << iota
	replaceElement
	haltTraversal
)

func (e traversalResult) replace() bool {
	return e&replaceElement != 0
}

func (e traversalResult) halt() bool {
	return e&haltTraversal != 0
}

func (e traversalResult) skipChildren() bool {
	return e&skipChildren != 0 || e&haltTraversal != 0
}

// error interface
func (e traversalResult) Error() string {
	switch e {
	case 0:
		return "traverse children"
	case skipChildren:
		return "skip children"
	case replaceElement:
		return "replace element and keep traversing"
	case replaceElement | skipChildren:
		return "replace element and skip children"
	case haltTraversal:
		return "halt traversal"
	case haltTraversal | skipChildren:
		return "halt traversal"
	case haltTraversal | replaceElement:
		return "replace element and halt traversal"
	case haltTraversal | replaceElement | skipChildren:
		return "replace element and halt traversal"
	default:
		return "unknown error"
	}
}

func isResult(err error) (traversalResult, bool) {
	if err == nil {
		return 0, true
	}
	r, ok := err.(traversalResult)
	if ok && r <= (skipChildren|replaceElement|haltTraversal) {
		return r, true
	} else {
		return 0, false
	}
}

var (
	// Continue indicates that the traversal should continue, including
	// children of the current element.
	Continue = func() error { return traversalResult(0) }()

	// Skip indicates that the traversal should continue, but skip processing
	// children of the current element.
	Skip = func() error { return traversalResult(skipChildren) }()

	// Halt indicates that the traversal should stop.
	Halt = func() error { return traversalResult(haltTraversal) }()

	// ReplaceContinue indicates that the current element should be
	// replaced with the elements returned by the function, and the
	// function should be applied to the children of the returned elements.
	ReplaceContinue = func() error { return traversalResult(replaceElement) }()

	// ReplaceSkip indicates that the current element should be replaced with the
	// elements returned by the function. No children of the returned elements
	// will be processed, but the traversal will continue with the siblings of
	// the current element (if any).
	ReplaceSkip = func() error { return traversalResult(replaceElement | skipChildren) }()

	// ReplaceHalt indicates that the current element should be replaced with the
	// elements returned by the function, and the traversal should stop.
	// No children of the returned elements will be processed.
	// Might be useful to change or remove the only specific element in the AST.
	ReplaceHalt = func() error { return traversalResult(replaceElement | haltTraversal) }()

	// Returned by Filter if the function returns the wrong type.
	ErrUnexpectedType = errors.New("unexpected type")
)

// Filter applies the specified function 'fun' to each child element of the provided
// element 'elt'. The function 'fun' is not applied to 'elt' itself, even if 'elt's type
// matches the parameter type of 'fun'.
//
// The parameter type P should be the same as or implement the return type R. This
// relationship is not enforced by the type system. If this condition is not met,
// the filter operation will still execute, but the intended modifications may not be applied.
//
// The parameter type P must be also a slice of R. In this case the function will
// be applied to all the lists of Inlines, or Blocks, (but not both).
// In latter case the order of traversal is reverted: fun will be applied to
// all the children and the to the current element's list.
//
// The behavior of the filter depends on the error returned by 'fun':
//
//   - TraverseChildren: Continie traversing including children of the
//     current element.
//   - SkipChildren: Continue traversing AST, but skip processing children
//     of the current element.
//   - StopTraversal: Skips processing children of the current element
//     and terminates the traversal process immediately.
//   - Replace: Replaces the current element with the elements returned by 'fun'.
//   - any other error: Terminates the traversal process immediately and
//     returns the error.
//
// To remove an element, 'fun' should return an empty slice of elements along with
// Replace.
//
// The function returns an updated version of the 'elt' after applying the
// specified function 'fun' (it might be the same 'elt' if no changes were made).
//
// Example:
//
//	 import . "github.com/growler/go-pandoc/cons"
//
//		doc = pandoc.Filter(doc, func (str *pandoc.Str) ([]pandoc.Inline, error) {
//		    return Inlines(Quoted(SingleQuote, Str("foo"))), Replace
//		})
func Filter[P any, E Element, R Element](elt E, fun func(P) ([]R, error)) (E, error) {
	elt, err := walkChildren(elt, fun)
	_, ok := isResult(err)
	if !ok {
		return elt, err
	} else {
		return elt, nil
	}
}

// Takes a filter function and returns a transformer function that can be used
// with element.Apply.
//
// Example:
//
//	  doc.Apply(
//		    pandoc.Transformer[*pandoc.Pandoc](func (e *pandoc.Span) ([]pandoc.Inline, error) {
//			    ...
//		    }),
//		    pandoc.Transformer[*pandoc.Pandoc](func (e *pandoc.Header) ([]pandoc.Block, error) {
//			    ...
//		    }),
//	  )
func Transformer[E Element, P any, R Element](fun func(P) ([]R, error)) func(E) (E, error) {
	return func(elt E) (E, error) {
		return Filter(elt, fun)
	}
}

func apply[E Element](elt E, transformer ...func(E) (E, error)) (E, error) {
	var err error
	for _, t := range transformer {
		if elt, err = t(elt); err != nil {
			return elt, err
		}
	}
	return elt, nil
}

type queryResult struct{}

func (queryResult) element()              {}
func (queryResult) clone() Element        { return nil }
func (queryResult) write(io.Writer) error { return nil }

// Query works the same way as QueryE, but fun does not return errors and
// traverse all the AST.
func Query[P any, E Element](elt E, fun func(P)) {
	walkChildren(elt, func(e P) ([]queryResult, error) {
		fun(e)
		return nil, nil
	})
}

// QueryE applies the specified function 'fun' to each child element of the provided
// element 'elt'. The function 'fun' is not applied to 'elt' itself, regardless of whether
// 'elt's type matches the parameter type of 'fun'.
//
// This function is used for walking through the child elements of 'elt' and applying
// the function 'fun' to perform checks or actions, without altering the structure of 'elt'.
// It is particularly useful for operations like searching or validation where modification
// of the element is not required.
//
// The function 'fun' returns an error to control the traversal process:
//
//   - Continue: Continie traversing the tree. Also nil works as well.
//   - Skip: Skips processing children of the current element.
//   - SkipAll: Skips processing children of the current element
//     and terminates the traversal process immediately.
//   - Replace: Not allowed. The function should not return this error.
//     However, works as Continue.
//   - any other error: Terminates the traversal process immediately and
//     returns the error.
//
// Unlike Filter, Query does not modify the element 'elt' or its children.
// It strictly performs read-only operations as defined in 'fun'.
//
// The fun may update the elements in-place, however this must be used
// with caution.
//
// Example:
//
//	var headers int
//	pandoc.Query(doc, func (str *pandoc.Header) { headers++ })
//
//	pandoc.QueryE(doc, func (str *pandoc.Header) error {
//
//	})
func QueryE[P any, E Element](elt E, fun func(P) error) error {
	_, err := walkChildren(elt, func(e P) ([]queryResult, error) {
		return nil, fun(e)
	})
	_, ok := isResult(err)
	if !ok {
		return err
	} else {
		return nil
	}
}

// Index returns index of the first element of type E in the list of elements
// implementing interface L (either Block or Inline), and the element itself.
// Returns -1, nil if []L does not contain any element of type E
//
// Example:
//
//	  Filter(doc, func(lst []Inline) ([]Inline, error) {
//		     ...
//	      if span, idx := Index[*Span](lst); idx >= 0 {
func Index[E Element, L Element](lst []L) (int, E) {
	for i := range lst {
		if e, ok := any(lst[i]).(E); ok {
			return i, e
		}
	}
	var cero E
	return -1, cero
}

// Index2 returns index of the first sequence of elements of types E1 and E2 in
// the list of elements implementing interface L (either Block or Inline),
// and the elements themselves.
// Returns -1, nil, nil if []L does not contain any element of type E
//
// Example:
//
//	  Filter(doc, func(lst []Block) ([]Block, error) {
//		     ...
//	      if para, block, idx := Index2[*Plain, *CodeBlock](lst); idx >= 0 {
func Index2[E1 Element, E2 Element, L Element](lst []L) (int, E1, E2) {
	for i := 0; i < len(lst)-1; i++ {
		if e1, ok := any(lst[i]).(E1); ok {
			if e2, ok := any(lst[i+1]).(E2); ok {
				return i, e1, e2
			}
		}
	}
	var (
		cero1 E1
		cero2 E2
	)
	return -1, cero1, cero2
}

// Index3 returns index of the first sequence of elements of types E1, E2 and E3
// in  the list of elements implementing interface L (either Block or Inline),
// and the elements themselves.
// Returns -1, nil, nil, nil if []L does not contain any element of type E
//
// Example:
//
//	  Filter(doc, func(lst []Inline) ([]Inline, error) {
//		     ...
//	      if _, str, _, idx := Index3[*Space, *Str, *Space](lst); idx >= 0 {
func Index3[E1 Element, E2 Element, E3 Element, L Element](lst []L) (int, E1, E2, E3) {
	for i := 0; i < len(lst)-2; i++ {
		if e1, ok := any(lst[i]).(E1); ok {
			if e2, ok := any(lst[i+1]).(E2); ok {
				if e3, ok := any(lst[i+2]).(E3); ok {
					return i, e1, e2, e3
				}
			}
		}
	}
	var (
		cero1 E1
		cero2 E2
		cero3 E3
	)
	return -1, cero1, cero2, cero3
}

// Converts string to identifier.
func StringToIdent(s string) string {
	var sb strings.Builder
	var prev rune
	for _, r := range s {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
			prev = 0
		} else if unicode.IsLetter(r) {
			sb.WriteRune(unicode.ToLower(r))
			prev = 0
		} else if (r == '-' || r == '_') && prev != r {
			sb.WriteByte(byte(r))
			prev = r
		} else if (unicode.IsSpace(r) || unicode.IsPunct(r)) && prev != '-' {
			sb.WriteByte('-')
			prev = '-'
		}
	}
	return sb.String()
}

// Converts list of inlines to identifier.
func InlinesToIdent(inlines []Inline) string {
	var sb strings.Builder
	walkList(inlines, func(elt Inline) ([]Inline, error) {
		switch e := any(elt).(type) {
		case *Str:
			sb.WriteString(StringToIdent(e.Text))
		case *Code:
			sb.WriteString(StringToIdent(e.Text))
		case *Space:
			if sb.Len() > 0 && sb.String()[sb.Len()-1] != '-' {
				sb.WriteByte('-')
			}
		case *SoftBreak:
			if sb.Len() > 0 && sb.String()[sb.Len()-1] != '-' {
				sb.WriteByte('-')
			}
		case *LineBreak:
			if sb.Len() > 0 && sb.String()[sb.Len()-1] != '-' {
				sb.WriteByte('-')
			}
		case *Note:
			return nil, Skip
		}
		return nil, nil
	})
	return sb.String()
}

func matchList[T Element](t []T, l []T) bool {
	if len(t) != len(l) {
		return false
	}
	for i := range t {
		if _, ok := Match(t[i], l[i]); !ok {
			return false
		}
	}
	return true
}

// Matches element E of type E agains template m of type T.
// Returns T and true if e matches. Does not modify m.
//
// Example:
//
//	  var tmpl = &Para{[]Inline{&Code{}, &Link{}}}
//	  Query(doc, func(e Block) {
//		     if para, ok := Match(tmpl, e); ok {
//		         ... // para is Para that consists of a single Code followed by Link
func Match[T Element, E Element](m T, e E) (T, bool) {
	var zero T
	if r, ok := any(e).(T); !ok {
		return zero, false
	} else {
		switch e := any(e).(type) {
		case inlinesContainer:
			if !matchList(any(m).(inlinesContainer).inlines(), e.inlines()) {
				return zero, false
			}
		case blocksContainer:
			if matchList(any(m).(blocksContainer).blocks(), e.blocks()) {
				return zero, false
			}
		}
		return r, true
	}
}

// Walk support following filter input/output combinations (input columns, output rows):
//
//  |     R     | Inline | Block | []Inline | []Block | *E (E <: R) |
//  |-----------|--------|-------|----------|---------|-------------|
//  | []Inline  |   X    |       |    X     |         |      X      |
//  | []Block   |        |   X   |          |    X    |      X      |
//  | []*E      |        |       |          |         |      X      |
//
//
// So from the walk's point of view, following functions are possible
//
//    func (elt Inline) ([]Inline, WalkResult)
//    func (elt Block) ([]Block, WalkResult)
//
//    func (elt []Inline) ([]Inline, WalkResult)
//    func (elt []Block) ([]Block, WalkResult)
//
//    func (elt *E) ([]R, WalkResult) // *E <: R, R \in {Inline, Block}

func walkLists[P any, E1 Element, E2 Element, R Element](l1 []E1, l2 []E2, fun func(P) ([]R, error)) ([]E1, []E2, error) {
	nl1, err := walkList(l1, fun)
	rl1, ok := isResult(err)
	if !ok {
		return l1, l2, err
	}
	if rl1.halt() {
		if rl1.replace() {
			return nl1, l2, ReplaceHalt
		} else {
			return l1, l2, Halt
		}
	}
	nl2, err := walkList(l2, fun)
	rl2, ok := isResult(err)
	if !ok {
		return l1, l2, err
	}
	if rl2.halt() {
		if rl1.replace() && !rl2.replace() {
			return nl1, l2, ReplaceHalt
		} else if !rl1.replace() && rl2.replace() {
			return l1, nl2, ReplaceHalt
		} else if rl1.replace() && rl2.replace() {
			return nl1, nl2, ReplaceHalt
		} else {
			return l1, l2, Halt
		}
	} else {
		if rl1.replace() && !rl2.replace() {
			return nl1, l2, ReplaceContinue
		} else if !rl1.replace() && rl2.replace() {
			return l1, nl2, ReplaceContinue
		} else if rl1.replace() && rl2.replace() {
			return nl1, nl2, ReplaceContinue
		} else {
			return l1, l2, Continue
		}
	}
}

// walkChildren traverses all the children on the provided element e
// and applies function fun. walkChildren returns the element itself
// if no changes were made, or a new element. walkChildren may return
// one of the following errors:
// - ReplaceAndStop
// - StopTraversal
// - TraverseChildren
func walkChildren[P any, E Element, R Element](e E, fun func(P) ([]R, error)) (E, error) {
	switch e := any(e).(type) {
	case *Pandoc:
		meta, blocks, err := walkLists(e.Meta, e.Blocks, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Pandoc{Meta: meta, Blocks: blocks}
		}
		return any(e).(E), err
	// Inlines
	case *Emph:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Emph{Inlines: lst}
		}
		return any(e).(E), err
	case *Strong:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Strong{Inlines: lst}
		}
		return any(e).(E), err
	case *Strikeout:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Strikeout{Inlines: lst}
		}
		return any(e).(E), err
	case *Superscript:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Superscript{Inlines: lst}
		}
		return any(e).(E), err
	case *Subscript:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Subscript{Inlines: lst}
		}
		return any(e).(E), err
	case *SmallCaps:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &SmallCaps{Inlines: lst}
		}
		return any(e).(E), err
	case *Quoted:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Quoted{QuoteType: e.QuoteType, Inlines: lst}
		}
		return any(e).(E), err
	case *Citation:
		pref, suff, err := walkLists(e.Prefix, e.Suffix, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			newElt := *e
			newElt.Prefix = pref
			newElt.Suffix = suff
			e = &newElt
		}
		return any(e).(E), err
	case *Cite:
		cts, lst, err := walkLists(e.Citations, e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Cite{Citations: cts, Inlines: lst}
		}
		return any(e).(E), err
	case *Link:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Link{Attr: e.Attr, Target: e.Target, Inlines: lst}
		}
		return any(e).(E), err
	case *Image:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Image{Attr: e.Attr, Target: e.Target, Inlines: lst}
		}
		return any(e).(E), err
	case *Note:
		lst, err := walkList(e.Blocks, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Note{Blocks: lst}
		}
		return any(e).(E), err
	case *Span:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Span{Attr: e.Attr, Inlines: lst}
		}
		return any(e).(E), err

	// following inlines have no children
	//
	// case *Str:
	// case *Code:
	// case *Space:
	// case *SoftBreak:
	// case *LineBreak:
	// case *Math:
	// case *RawInline:

	// Blocks
	case *Plain:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Plain{Inlines: lst}
		}
		return any(e).(E), err
	case *Para:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Para{Inlines: lst}
		}
		return any(e).(E), err
	case *LineBlock:
		lst, err := walkListOfLists(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &LineBlock{Inlines: lst}
		}
		return any(e).(E), err
	// case *CodeBlock: // no children
	// case *RawBlock: // no children
	case *BlockQuote:
		lst, err := walkList(e.Blocks, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &BlockQuote{Blocks: lst}
		}
		return any(e).(E), err
	case *OrderedList:
		lst, err := walkListOfLists(e.Items, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &OrderedList{Attr: e.Attr, Items: lst}
		}
		return any(e).(E), err
	case *BulletList:
		lst, err := walkListOfLists(e.Items, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &BulletList{Items: lst}
		}
		return any(e).(E), err
	case *DefinitionList:
		var (
			updated bool
			inlines []Inline
			blocks  [][]Block
			err     error
			items   = e.Items
			orig    = e
		)
		for i := range items {
			inlines, err = walkList(items[i].Term, fun)
			rslt, ok := isResult(err)
			if !ok {
				return any(orig).(E), err
			}
			if rslt.replace() {
				if !updated {
					updated = true
					items = append([]Definition(nil), items...)
				}
				items[i].Term = inlines
			}
			if rslt.halt() {
				if updated {
					e = &DefinitionList{Items: items}
					return any(e).(E), ReplaceHalt
				} else {
					return any(orig).(E), Halt
				}
			}
			blocks, err = walkListOfLists(items[i].Definition, fun)
			rslt, ok = isResult(err)
			if !ok {
				return any(orig).(E), err
			}
			if rslt.replace() {
				if !updated {
					updated = true
					items = append([]Definition(nil), items...)
				}
				items[i].Definition = blocks
			}
			if rslt.halt() {
				if updated {
					e = &DefinitionList{Items: items}
					return any(e).(E), ReplaceHalt
				} else {
					return any(orig).(E), Halt
				}
			}
		}
		if updated {
			e = &DefinitionList{Items: items}
			return any(e).(E), ReplaceContinue
		} else {
			return any(orig).(E), Continue
		}
	case *Header:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Header{
				Level:   e.Level,
				Attr:    e.Attr,
				Inlines: lst,
			}
		}
		return any(e).(E), err
	// case *HorizontalRule: // no children
	case *Table:
		table, err := walkTable(e, fun)
		return any(table).(E), err
	case *TableHeadFoot:
		lst, err := walkList(e.Rows, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &TableHeadFoot{Attr: e.Attr, Rows: lst}
		}
		return any(e).(E), err
	case *TableBody:
		hdr, body, err := walkLists(e.Head, e.Body, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &TableBody{Attr: e.Attr, RowHeadColumns: e.RowHeadColumns, Head: hdr, Body: body}
		}
		return any(e).(E), err
	case *TableRow:
		lst, err := walkList(e.Cells, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &TableRow{Attr: e.Attr, Cells: lst}
		}
		return any(e).(E), err
	case *TableCell:
		lst, err := walkList(e.Blocks, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &TableCell{
				Attr:    e.Attr,
				Align:   e.Align,
				RowSpan: e.RowSpan,
				ColSpan: e.ColSpan,
				Blocks:  lst,
			}
		}
		return any(e).(E), err
	case *Figure:
		caption, err := walkCaption(e.Caption, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		newF := e
		if rslt.replace() {
			newF = &Figure{
				Attr:    e.Attr,
				Caption: caption,
				Blocks:  e.Blocks,
			}
		}
		if rslt.halt() {
			return any(newF).(E), err
		}
		lst, err := walkList(e.Blocks, fun)
		rslt, ok = isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			newF = &Figure{
				Attr:    e.Attr,
				Caption: caption,
				Blocks:  lst,
			}
		}
		if newF != e {
			if rslt.halt() {
				return any(newF).(E), ReplaceHalt
			} else {
				return any(newF).(E), ReplaceContinue
			}
		}
		return any(e).(E), err
	case *Div:
		lst, err := walkList(e.Blocks, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &Div{Attr: e.Attr, Blocks: lst}
		}
		return any(e).(E), err

	// Meta
	case *MetaMap:
		lst, err := walkList(e.Entries, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &MetaMap{lst}
		}
		return any(e).(E), err
	case MetaMapEntry:
		val, err := walkChildren(e.Value, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			return any(MetaMapEntry{Key: e.Key, Value: val}).(E), err
		} else {
			return any(e).(E), err
		}
	case *MetaList:
		lst, err := walkList(e.Entries, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &MetaList{lst}
		}
		return any(e).(E), err
	case *MetaBlocks:
		lst, err := walkList(e.Blocks, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &MetaBlocks{lst}
		}
		return any(e).(E), err
	case *MetaInlines:
		lst, err := walkList(e.Inlines, fun)
		rslt, ok := isResult(err)
		if !ok {
			return any(e).(E), err
		}
		if rslt.replace() {
			e = &MetaInlines{lst}
		}
		return any(e).(E), err
	default:
		return any(e).(E), nil
	}
}

func walkTableHeadFoot[P any, R Element](hf *TableHeadFoot, fun func(P) ([]R, error)) (*TableHeadFoot, error) {
	if param, ok := any(hf).(P); ok {
		replace, err := fun(param)
		rslt, ok := isResult(err)
		if !ok {
			return hf, err
		}
		src := hf
		var updated bool
		if rslt.replace() {
			if len(replace) != 1 {
				return hf, ErrUnexpectedType
			} else if newhf, ok := any(replace[0]).(*TableHeadFoot); !ok {
				return hf, ErrUnexpectedType
			} else {
				hf = newhf
				updated = true
			}
		}
		if rslt.skipChildren() {
			return hf, err
		}
		hf, err := walkChildren(hf, fun)
		rslt, ok = isResult(err)
		if !ok {
			return src, err
		}

		//  replace update result
		//     0      0    err
		//     0      1    if hlt ReplaceAndStop else Replace
		//     1      0    err
		//     1      1    err

		if !rslt.replace() && updated {
			if rslt.halt() {
				return hf, ReplaceHalt
			} else {
				return hf, ReplaceContinue
			}
		} else {
			return hf, err
		}
	} else {
		return walkChildren(hf, fun)
	}
}

func walkTable[P any, R Element](table *Table, fun func(P) ([]R, error)) (*Table, error) {
	var (
		updated bool
		err     error
		caption = table.Caption
		head    = &table.Head
		foot    = &table.Foot
		bodies  = table.Bodies
	)
	caption, err = walkCaption(table.Caption, fun)
	rslt, ok := isResult(err)
	if !ok {
		return table, err
	}
	updated = updated || rslt.replace()
	if rslt.halt() {
		goto fin
	}
	head, err = walkTableHeadFoot(&table.Head, fun)
	rslt, ok = isResult(err)
	if !ok {
		return table, err
	}
	updated = updated || rslt.replace()
	if rslt.halt() {
		goto fin
	}
	bodies, err = walkList(table.Bodies, fun)
	rslt, ok = isResult(err)
	if !ok {
		return table, err
	}
	updated = updated || rslt.replace()
	if rslt.halt() {
		goto fin
	}
	foot, err = walkTableHeadFoot(&table.Foot, fun)
	rslt, ok = isResult(err)
	if !ok {
		return table, err
	}
	updated = updated || rslt.replace()
fin:
	if updated {
		table = &Table{
			Attr:    table.Attr,
			Caption: caption,
			Aligns:  table.Aligns,
			Head:    *head,
			Bodies:  bodies,
			Foot:    *foot,
		}
		if rslt.halt() {
			return table, ReplaceHalt
		} else {
			return table, ReplaceContinue
		}
	} else {
		return table, err
	}
}

func walkCaption[P any, R Element](caption Caption, fun func(P) ([]R, error)) (Caption, error) {
	var cap = caption
	short, long, err := walkLists(caption.Short, caption.Long, fun)
	rslt, ok := isResult(err)
	if !ok {
		return cap, err
	}
	if rslt.replace() {
		cap.Short = short
		cap.Long = long
	}
	return cap, err
}

func walkListOfLists[P any, S Element, R Element](source [][]S, fun func(P) ([]R, error)) ([][]S, error) {
	var (
		newList []S
		err     error
		updated bool
		src     = source
	)
	for i := 0; i < len(source); {
		newList, err = walkList(source[i], fun)
		rslt, ok := isResult(err)
		if !ok {
			return src, err
		}
		if rslt.replace() {
			if !updated {
				updated = true
				source = append([][]S(nil), source...)
			}
			if len(newList) == 0 {
				source = append(source[:i], source[i+1:]...)
			} else {
				source[i] = newList
				i++
			}
		} else {
			i++
		}
		if rslt.halt() {
			if updated {
				return source, ReplaceHalt
			} else {
				return source, Halt
			}
		}
	}
	if updated {
		return source, ReplaceContinue
	} else {
		return source, Continue
	}
}

// walkList
func walkList[P any, S Element, R Element](source []S, fun func(P) ([]R, error)) ([]S, error) {
	var (
		replace   []R
		err       error
		sameInOut bool
		updated   = false
	)
	_, sameInOut = any(replace).([]S)
	var src = source
	if _, ok := any(source).(P); ok {
		list := any(source).(P)
		replace, err = fun(list)
		rslt, ok := isResult(err)
		if !ok {
			return src, err
		}
		if updated = rslt.replace(); updated {
			if sameInOut {
				source = any(replace).([]S)
			} else {
				source = make([]S, len(replace))
				for i := range replace {
					if s, ok := any(replace[i]).(S); ok {
						source[i] = s
					} else {
						return src, ErrUnexpectedType
					}
				}
			}
		}
		if rslt.skipChildren() {
			return source, err
		}
		for i := range source {
			var item S
			item, err = walkChildren(source[i], fun)
			rslt, ok := isResult(err)
			if !ok {
				return src, err
			}
			if rslt.replace() {
				if !updated {
					updated = true
					source = append([]S(nil), source...)
				}
				source[i] = item
			}
			if rslt.halt() {
				if updated {
					return source, ReplaceHalt
				} else {
					return source, Halt
				}
			}
		}
		if updated {
			return source, ReplaceContinue
		} else {
			return source, Continue
		}
	}
	for i := 0; i < len(source); {
		if val, ok := any(source[i]).(P); !ok {
			item, err := walkChildren(source[i], fun)
			rslt, ok := isResult(err)
			if !ok {
				return src, err
			}
			if rslt.replace() {
				if !updated {
					updated = true
					source = append([]S(nil), source...)
				}
				source[i] = item
			}
			if rslt.halt() {
				if updated {
					return source, ReplaceHalt
				} else {
					return source, Halt
				}
			}
			i++
		} else {
			replace, err = fun(val)
			rslt, ok := isResult(err)
			if !ok {
				return src, err
			}
			if !rslt.replace() {
				if !rslt.skipChildren() {
					item, err := walkChildren(source[i], fun)
					rslt, ok := isResult(err)
					if !ok {
						return src, err
					}
					if rslt.replace() {
						if !updated {
							updated = true
							source = append([]S(nil), source...)
						}
						source[i] = item
					}
					if rslt.halt() {
						if updated {
							return source, ReplaceHalt
						} else {
							return source, Halt
						}
					}
				}
				i++
			} else {
				if !updated {
					updated = true
					source = append([]S(nil), source...)
				}
				if len(replace) == 0 {
					source = append(source[:i], source[i+1:]...)
				} else {
					if len(replace) == 1 {
						if s, ok := any(replace[0]).(S); !ok {
							return src, ErrUnexpectedType
						} else {
							if rslt.skipChildren() {
								source[i] = s
							} else {
								item, err := walkChildren(s, fun)
								rslt, ok := isResult(err)
								if !ok {
									return src, err
								}
								if rslt.replace() {
									source[i] = item
								} else {
									source[i] = s
								}
								if rslt.halt() {
									return source, ReplaceHalt
								}
							}
						}
					} else if sameInOut {
						source = append(source[:i], append(any(replace).([]S), source[i+1:]...)...)
						if !rslt.skipChildren() {
							for j := range replace {
								item, err := walkChildren(source[i+j], fun)
								rslt, ok := isResult(err)
								if !ok {
									return src, err
								}
								if rslt.replace() {
									source[i+j] = item
								}
								if rslt.halt() {
									return source, ReplaceHalt
								}
							}
						}
					} else {
						source = append(source[:i], append(make([]S, len(replace)), source[i+1:]...)...)
						for j := range replace {
							if s, ok := any(replace[j]).(S); !ok {
								return src, ErrUnexpectedType
							} else {
								item, err := walkChildren(s, fun)
								rslt, ok := isResult(err)
								if !ok {
									return src, err
								}
								if rslt.replace() {
									source[i+j] = item
								} else {
									source[i+j] = s
								}
								if rslt.halt() {
									return source, ReplaceHalt
								}
							}
						}
					}
					i += len(replace)
				}
			}
			if rslt.halt() {
				if updated {
					return source, ReplaceHalt
				} else {
					return source, Halt
				}
			}
		}
	}
	if updated {
		return source, ReplaceContinue
	} else {
		return source, Continue
	}
}
