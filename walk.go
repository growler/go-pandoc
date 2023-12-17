package pandoc

import "io"

// WalkResult is the result of a walk operation.
type WalkResult int

// WalkContinue indicates that the walk operation should continue.
const WalkContinue = 0

// WalkReplace indicates that the current element should be replaced with the
// elements returned by the function.
const WalkReplace = 1

// WalkSkip indicates that the current element should be skipped and
// no children should be processed.
const WalkSkip = 2

// WalkStop indicates that the walk operation should stop immediately.
const WalkStop = 3

// Filter applies the specified function 'fun' to each child element of the provided
// element 'elt'. The function 'fun' is not applied to 'elt' itself, even if 'elt's type
// matches the parameter type of 'fun'.
//
// The parameter type P should be the same as or implement the return type R. This
// relationship is not enforced by the type system. If this condition is not met,
// the filter operation will still execute, but the intended modifications may not be applied.
//
// The behavior of the filter depends on the WalkResult returned by 'fun':
//
//   - WalkStop: Terminates the traversal process immediately.
//   - WalkSkip: Skips processing of the current element.
//   - WalkReplace: Replaces the current element with the elements returned by 'fun'.
//   - WalkContinue: Continues without replacing the current element.
//
// To remove an element, 'fun' should return an empty slice of elements along with WalkReplace.
//
// The function returns an updated version of the 'elt' after applying the specified function 'fun'.
//
// Example:
//
//	doc = pandoc.Filter(doc, func (str *pandoc.Str) ([]pandoc.Inline, pandoc.WaklResult) {
//	    return []pandoc.Inline{&pandoc.Quoted{
//	            QuoteType: pandoc.SingleQuote,
//	            Inlines: []pandoc.Inline{&pandoc.Str{Str: "foo"}},
//	        }}, pandoc.WalkReplace
//	})
func Filter[P any, E Element, R Element](elt E, fun func(P) ([]R, WalkResult)) E {
	elt, _, _ = walkChildren(elt, fun)
	return elt
}

type queryResult struct{}

func (queryResult) element()              {}
func (queryResult) write(io.Writer) error { return nil }

// Query applies the specified function 'fun' to each child element of the provided
// element 'elt'. The function 'fun' is not applied to 'elt' itself, regardless of whether
// 'elt's type matches the parameter type of 'fun'.
//
// This function is used for walking through the child elements of 'elt' and applying
// the function 'fun' to perform checks or actions, without altering the structure of 'elt'.
// It is particularly useful for operations like searching or validation where modification
// of the element is not required.
//
// The function 'fun' returns a WalkResult to control the traversal process:
//
//   - WalkStop: Terminates the traversal process immediately.
//   - WalkSkip: Skips processing of the current element.
//   - WalkContinue: Continues to the next element without any special action.
//
// Unlike Filter, Query does not modify the element 'elt' or its children. It strictly
// performs read-only operations as defined in 'fun'.
//
// Example:
//
//	var headers int
//	pandoc.Query(doc, func (str *pandoc.Header) pandoc.WalkResult {
//	    headers++
//	    return pandoc.WalkSkip
//	})
//	fmt.Printf("doc has %d headers\n", headers)
func Query[P any, E Element](elt E, fun func(P) WalkResult) {
	walkChildren(elt, func(e P) ([]queryResult, WalkResult) {
		return nil, fun(e)
	})
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

func walkChildren[P any, E Element, R Element](e E, fun func(P) ([]R, WalkResult)) (E, bool, WalkResult) {
	switch e := any(e).(type) {
	case *Doc:
		meta, metaUpdated, _ := walkList(e.Meta, fun)
		blocks, blocksUpdated, _ := walkList(e.Blocks, fun)
		if metaUpdated || blocksUpdated {
			e = &Doc{Version: e.Version, File: e.File, Meta: meta, Blocks: blocks}
		}
		return any(e).(E), metaUpdated || blocksUpdated, WalkContinue
	// Inlines
	case *Emph:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Emph{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Strong:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Strong{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Strikeout:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Strikeout{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Superscript:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Superscript{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Subscript:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Subscript{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *SmallCaps:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &SmallCaps{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Quoted:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Quoted{QuoteType: e.QuoteType, Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Cite:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Cite{Citations: e.Citations, Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Link:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Link{Attr: e.Attr, Target: e.Target, Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Image:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Image{Attr: e.Attr, Target: e.Target, Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Note:
		lst, updated, result := walkList(e.Blocks, fun)
		if updated {
			e = &Note{Blocks: lst}
		}
		return any(e).(E), updated, result
	case *Span:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Span{Attr: e.Attr, Inlines: lst}
		}
		return any(e).(E), updated, result

	// following have no children
	//
	case *Str:
	case *Code:
	case *Space:
	case *SoftBreak:
	case *LineBreak:
	case *Math:
	case *RawInline:

	// Blocks
	case *Plain:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Plain{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Para:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Para{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *LineBlock:
		lst, updated, result := walkListOfLists(e.Inlines, fun)
		if updated {
			e = &LineBlock{Inlines: lst}
		}
		return any(e).(E), updated, result
	case *Header:
		lst, updated, result := walkList(e.Inlines, fun)
		if updated {
			e = &Header{
				Level:   e.Level,
				Attr:    e.Attr,
				Inlines: lst,
			}
		}
		return any(e).(E), updated, result
	case *Figure:
		caption, updated, result := walkCaption(e.Caption, fun)
		if updated {
			e = &Figure{
				Attr:    e.Attr,
				Caption: caption,
				Blocks:  e.Blocks,
			}
		}
		if result == WalkStop {
			return any(e).(E), updated, WalkStop
		}
		lst, blocksUpdated, result := walkList(e.Blocks, fun)
		if blocksUpdated {
			if !updated {
				updated = true
				e = &Figure{
					Attr:    e.Attr,
					Caption: e.Caption,
					Blocks:  lst,
				}
			} else {
				e.Blocks = lst
			}
		}
		return any(e).(E), updated, result
	case *BlockQuote:
		lst, updated, result := walkList(e.Blocks, fun)
		if updated {
			e = &BlockQuote{Blocks: lst}
		}
		return any(e).(E), updated, result
	case *DefinitionList:
		var (
			updated bool
			inlines []Inline
			blocks  [][]Block
			result  WalkResult
			items   = e.Items
		)
		for i := range items {
			inlines, updated, result = walkList(items[i].Term, fun)
			if updated {
				if !updated {
					updated = true
					items = append([]Definition(nil), items...)
				}
				items[i].Term = inlines
			}
			if result == WalkStop {
				if updated {
					e = &DefinitionList{Items: items}
				}
				return any(e).(E), updated, WalkStop
			}
			blocks, updated, result = walkListOfLists(items[i].Definition, fun)
			if updated {
				if !updated {
					updated = true
					items = append([]Definition(nil), items...)
				}
				items[i].Definition = blocks
			}
			if result == WalkStop {
				if updated {
					e = &DefinitionList{Items: items}
				}
				return any(e).(E), updated, WalkStop
			}
		}
		if updated {
			e = &DefinitionList{Items: items}
		}
		return any(e).(E), updated, WalkContinue
	// following have no children
	case *CodeBlock:
	case *RawBlock:

	// Meta
	case *MetaMap:
		lst, updated, result := walkList(*e, fun)
		if updated {
			*e = lst
		}
		return any(e).(E), updated, result
	case *MetaList:
		lst, updated, result := walkList(*e, fun)
		if updated {
			*e = lst
		}
		return any(e).(E), updated, result
	case *MetaBlocks:
		lst, updated, result := walkList(*e, fun)
		if updated {
			*e = lst
		}
		return any(e).(E), updated, result
	case *MetaInlines:
		lst, updated, result := walkList(*e, fun)
		if updated {
			*e = lst
		}
		return any(e).(E), updated, result
	}
	return e, false, WalkContinue
}

func walkCaption[P any, R Element](caption Caption, fun func(P) ([]R, WalkResult)) (Caption, bool, WalkResult) {
	item, shortUpdated, result := walkList(caption.Short, fun)
	if shortUpdated {
		caption.Short = item
	}
	if result == WalkStop {
		return caption, shortUpdated, WalkStop
	}
	lst, longUpdated, result := walkList(caption.Long, fun)
	if longUpdated {
		caption.Long = lst
	}
	return caption, shortUpdated || longUpdated, result
}

func walkListOfLists[P any, S Element, R Element](source [][]S, fun func(P) ([]R, WalkResult)) ([][]S, bool, WalkResult) {
	var (
		newList []S
		result  WalkResult
		updated bool
	)
	for i := 0; i < len(source); {
		newList, updated, result = walkList(source[i], fun)
		if updated {
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
		if result == WalkStop {
			return source, false, WalkStop
		}
	}
	return source, false, WalkContinue
}

func walkList[P any, S Element, R Element](source []S, fun func(P) ([]R, WalkResult)) ([]S, bool, WalkResult) {
	var (
		replace                   []R
		result                    WalkResult
		updated                   = false
		update                    bool
		sameInOut, coercibleInOut bool
	)
	if _, ok := any(source).(P); ok { // special case, func handles lists and works down-top
		for i := range source {
			var item S
			item, update, result = walkChildren(source[i], fun)
			if update {
				if !updated {
					updated = true
					source = append([]S(nil), source...)
				}
				source[i] = item
			}
			if result == WalkStop {
				return source, updated, WalkStop
			}
		}
		list := any(source).(P)
		replace, result = fun(list)
		switch result {
		case WalkReplace:
			return any(replace).([]S), true, WalkContinue
		case WalkStop:
			return source, updated, WalkStop
		}
		return source, updated, WalkContinue
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
			replace, result = fun(v)
			switch result {
			case WalkStop:
				return source, updated, WalkStop
			case WalkSkip:
				i++
				continue
			case WalkReplace:
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
			case WalkContinue:
			}
		}
		var item S
		item, update, result = walkChildren(source[i], fun)
		if update {
			if !updated {
				updated = true
				source = append([]S(nil), source...)
			}
			source[i] = item
		}
		if result == WalkStop {
			return source, updated, WalkStop
		}
		i++
	}
	return source, updated, WalkContinue
}
