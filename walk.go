package pandoc

import (
	"io"
	"strings"
	"unicode"
)

// AST traversal result (used by Filter and QueryE)
type TraversalError uint8

// error interface
func (e TraversalError) Error() string {
	switch e {
	case Continue:
		return "continue"
	case Replace:
		return "replace"
	case Skip:
		return "skip"
	case SkipAll:
		return "skip all"
	default:
		return "unknown error"
	}
}

func fatal(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(TraversalError)
	return !ok
}

// Continue indicates that the walk operation should continue.
// nil works as well.
var Continue error = TraversalError(0)

// Replace indicates that the current element should be replaced with the
// elements returned by the function.
var Replace error = TraversalError(1)

// Skip indicates that the current element should be skipped and
// no children should be processed.
var Skip error = TraversalError(2)

// SkipAll indicates that the walk operation should stop immediately.
var SkipAll error = TraversalError(3)

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
//   - Skip: Skips processing children of the current element.
//   - SkipAll: Skips processing children of the current element 
//     and terminates the traversal process immediately.
//   - Replace: Replaces the current element with the elements returned by 'fun'.
//   - Continue: Continie travesing. Also nil works as well.
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
	if elt, _, err := walkChildren(elt, fun); fatal(err) {
		return elt, err
	} else {
		return elt, nil
	}
}

type queryResult struct{}

func (queryResult) element() {}
func (queryResult) clone() Element { return nil}
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
//	 var headers int
//	 pandoc.Query(doc, func (str *pandoc.Header) { headers++ })
//
//	 pandoc.QueryE(doc, func (str *pandoc.Header) error {
//
//	 })
func QueryE[P any, E Element](elt E, fun func(P) error) error {
	if _, _, err := walkChildren(elt, func(e P) ([]queryResult, error) {
		return nil, fun(e)
	}); fatal(err) {
		return err
	}
	return nil
}

// Index returns index of the first element of type E in the list of elements
// implementing interface L (either Block or Inline), and the element itself.
// Returns -1, nil if []L does not contain any element of type E
// 
// Example:
//
//   Filter(doc, func(lst []Inline) ([]Inline, error) {
// 	     ...
//       if span, idx := Index[*Span](lst); idx >= 0 {
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
//   Filter(doc, func(lst []Block) ([]Block, error) {
// 	     ...
//       if para, block, idx := Index2[*Plain, *CodeBlock](lst); idx >= 0 {
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
//   Filter(doc, func(lst []Inline) ([]Inline, error) {
// 	     ...
//       if _, str, _, idx := Index3[*Space, *Str, *Space](lst); idx >= 0 {
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
//   var tmpl = &Para{[]Inline{&Code{}, &Link{}}}
//   Query(doc, func(e Block) {
//	     if para, ok := Match(tmpl, e); ok {
//	         ... // para is Para that consists of a single Code followed by Link
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

func walkChildren[P any, E Element, R Element](e E, fun func(P) ([]R, error)) (E, bool, error) {
	var ret error
	switch e := any(e).(type) {
	case *Pandoc:
		meta, metaUpdated, err := walkList(e.Meta, fun)
		if fatal(err) {
			return any(e).(E), false, err
		}
		blocks, blocksUpdated, err := walkList(e.Blocks, fun)
		if fatal(err) {
			return any(e).(E), false, err
		} else if metaUpdated || blocksUpdated {
			e = &Pandoc{Meta: meta, Blocks: blocks}
			return any(e).(E), true, nil
		} else {
			ret = err
		}
	// Inlines
	case *Emph:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Emph{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Strong:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Strong{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Strikeout:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Strikeout{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Superscript:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Superscript{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Subscript:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Subscript{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *SmallCaps:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &SmallCaps{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Quoted:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Quoted{QuoteType: e.QuoteType, Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Cite:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Cite{Citations: e.Citations, Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Link:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Link{Attr: e.Attr, Target: e.Target, Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Image:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Image{Attr: e.Attr, Target: e.Target, Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Note:
		if lst, updated, err := walkList(e.Blocks, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Note{Blocks: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Span:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Span{Attr: e.Attr, Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}

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
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Plain{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Para:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Para{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *LineBlock:
		if lst, updated, err := walkListOfLists(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &LineBlock{Inlines: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Header:
		if lst, updated, err := walkList(e.Inlines, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Header{
				Level:   e.Level,
				Attr:    e.Attr,
				Inlines: lst,
			}
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *Figure:
		caption, updated, err := walkCaption(e.Caption, fun)
		if fatal(err) {
			return any(e).(E), false, err
		}
		var newF *Figure
		if updated {
			newF = &Figure{
				Attr:    e.Attr,
				Caption: caption,
				Blocks:  e.Blocks,
			}
		}
		if err == SkipAll {
			return any(newF).(E), updated, SkipAll
		}
		lst, blocksUpdated, err := walkList(e.Blocks, fun)
		if fatal(err) {
			return any(e).(E), false, err
		}
		if blocksUpdated {
			if !updated {
				updated = true
				newF = &Figure{
					Attr:    e.Attr,
					Caption: e.Caption,
					Blocks:  lst,
				}
			} else {
				newF.Blocks = lst
			}
		}
		if updated {
			return any(newF).(E), true, Continue
		} else {
			ret = err
		}
	case *BlockQuote:
		if lst, updated, err := walkList(e.Blocks, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &BlockQuote{Blocks: lst}
			return any(e).(E), updated, err
		} else {
			ret = err
		}		
	case *DefinitionList:
		var (
			updated bool
			inlines []Inline
			blocks  [][]Block
			err     error
			items   = e.Items
			orig = e
		)
		for i := range items {
			inlines, updated, err = walkList(items[i].Term, fun)
			if fatal(err) {
				return any(orig).(E), false, err
			}
			if updated {
				if !updated {
					updated = true
					items = append([]Definition(nil), items...)
				}
				items[i].Term = inlines
			}
			if err == SkipAll {
				if updated {
					e = &DefinitionList{Items: items}
				}
				return any(e).(E), updated, SkipAll
			}
			blocks, updated, err = walkListOfLists(items[i].Definition, fun)
			if fatal(err) {
				return any(orig).(E), false, err
			}
			if updated {
				if !updated {
					updated = true
					items = append([]Definition(nil), items...)
				}
				items[i].Definition = blocks
			}
			if err == SkipAll {
				if updated {
					e = &DefinitionList{Items: items}
				}
				return any(e).(E), updated, SkipAll
			}
		}
		if updated {
			e = &DefinitionList{Items: items}
			return any(e).(E), true, Continue
		} else {
			ret = Continue
		}		
	case *Div:
		if lst, updated, err := walkList(e.Blocks, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e = &Div{Attr: e.Attr, Blocks: lst}
			return any(e).(E), true, err
		} else {
			ret = err
		}		
	// following have no children
	// case *CodeBlock:
	// case *RawBlock:

	// Meta
	case *MetaMap:
		if lst, updated, err := walkList(e.Entries, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e.Entries = lst
			return any(e).(E), true, err
		} else {
			ret = err
		}		
	case *MetaList:
		if lst, updated, err := walkList(e.Entries, fun); fatal(err) {
			return any(e).(E), false, err
		} else if updated {
			e.Entries = lst
			return any(e).(E), true, err
		} else {
			ret = err
		}
	case *MetaBlocks:
		lst, updated, result := walkList(e.Blocks, fun)
		if updated {
			e.Blocks = lst
		}
		return any(e).(E), updated, result
	case *MetaInlines:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e.Inlines = lst
		}
		return any(e).(E), updated, result
	}
	return e, false, ret
}

func walkCaption[P any, R Element](caption Caption, fun func(P) ([]R, error)) (Caption, bool, error) {
	var cap = caption
	item, shortUpdated, err := walkList(caption.Short, fun)
	if fatal(err) {
		return cap, false, err
	}
	if shortUpdated {
		caption.Short = item
	}
	if err == SkipAll {
		return caption, shortUpdated, SkipAll
	}
	lst, longUpdated, err := walkList(caption.Long, fun)
	if fatal(err) {
		return cap, false, err
	}
	if longUpdated {
		caption.Long = lst
	}
	return caption, shortUpdated || longUpdated, err
}

func walkListOfLists[P any, S Element, R Element](source [][]S, fun func(P) ([]R, error)) ([][]S, bool, error) {
	var (
		newList []S
		err     error
		updated bool
		listup bool
		src = source
	)
	for i := 0; i < len(source); {
		newList, listup, err = walkList(source[i], fun)
		if fatal(err) {
			return src, false, err
		}
		if listup {
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
		if err == SkipAll {
			return source, updated, SkipAll
		}
	}
	return source, updated, Continue
}

func walkList[P any, S Element, R Element](source []S, fun func(P) ([]R, error)) ([]S, bool, error) {
	var (
		replace        []R
		err            error
		update         bool
		sameInOut      bool
		coercibleInOut bool
		updated        = false
	)
	var src = source
	if _, ok := any(source).(P); ok { // special case, func handles lists and works down-top
		for i := range source {
			var item S
			item, update, err = walkChildren(source[i], fun)
			if fatal(err) {
				return src, false, err
			}
			if update {
				if !updated {
					updated = true
					source = append([]S(nil), source...)
				}
				source[i] = item
			}
			if err == SkipAll {
				return source, updated, SkipAll
			}
		}
		list := any(source).(P)
		replace, err = fun(list)
		if fatal(err) {
			return src, false, err
		}
		switch err {
		case Replace:
			return any(replace).([]S), true, Continue
		case SkipAll:
			return source, updated, SkipAll
		}
		return source, updated, Continue
	}
	_, sameInOut = any(replace).([]S)
	if !sameInOut {
		var item R
		_, coercibleInOut = any(item).(S)
		if !coercibleInOut {
			_, coercibleInOut = any(replace).([]Element)
		}
	}
	for i := 0; i < len(source); {
		if v, ok := any(source[i]).(P); ok {
			replace, err = fun(v)
			if fatal(err) {
				return src, false, err
			}
			switch err {
			case SkipAll:
				return source, updated, SkipAll
			case Skip:
				i++
				continue
			case Replace:
				if sameInOut || coercibleInOut {
					if !updated {
						updated = true
						source = append([]S(nil), source...)
					}
					if len(replace) == 0 {
						source = append(source[:i], source[i+1:]...)
						continue
					} else if len(replace) == 1 {
						source[i] = any(replace[0]).(S)
					} else if sameInOut {
						source = append(source[:i], append(any(replace).([]S), source[i+1:]...)...)
					} else {
						source = append(source[:i], append(make([]S, len(replace)), source[i+1:]...)...)
						for j := range replace {
							source[i+j] = any(replace[j]).(S)
						}
					}
					i += len(replace)
				} else {
					i++
				}
				continue
			// case Continue, nil:
			}
		}
		var item S
		item, update, err = walkChildren(source[i], fun)
		if fatal(err) {
			return src, false, err
		}
		if update {
			if !updated {
				updated = true
				source = append([]S(nil), source...)
			}
			source[i] = item
		}
		if err == SkipAll {
			return source, updated, SkipAll
		}
		i++
	}
	return source, updated, Continue
}
