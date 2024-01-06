package pandoc

import (
	"strings"
	"testing"
)

func testTable() *Table {
	return &Table{
		Head: TableHeadFoot{
			Rows: []*TableRow{{
				Cells: []*TableCell{{Blocks: []Block{&Plain{[]Inline{&Str{"TableHead"}}}}}},
			}},
		},
		Foot: TableHeadFoot{
			Rows: []*TableRow{{
				Cells: []*TableCell{{Blocks: []Block{&Plain{[]Inline{&Str{"TableFoot"}}}}}},
			}},
		},
		Bodies: []*TableBody{
			{
				Head: []*TableRow{{
					Cells: []*TableCell{{Blocks: []Block{&Plain{[]Inline{&Str{"BodyHead"}}}}}},
				}},
				Body: []*TableRow{{
					Cells: []*TableCell{{Blocks: []Block{&Plain{[]Inline{&Str{"BodyBody"}}}}}},
				}},
			},
		},
	}
}

func BenchmarkWalkTable(b *testing.B) {
	b.StopTimer()
	doc := testTable()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		Query(doc, func(e Element) {})
	}
}

func TestWalkTable(t *testing.T) {
	var items []string
	Query(testTable(), func(e *Str) { items = append(items, e.Text) })
	const expected = "TableHead,BodyHead,BodyBody,TableFoot"
	if result := strings.Join(items, ","); result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
