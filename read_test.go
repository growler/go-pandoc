package pandoc

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestAppendQuote(t *testing.T) {
	var tests = []struct {
		str, want string
	}{
		{"", `""`},
		{"a", `"a"`},
		{"\"", `"\""`},
	}
	for i := range tests {
		r := appendQuote(nil, tests[i].str)
		v := []byte(tests[i].want)
		if !bytes.Equal(r, v) {
			t.Errorf("expected [%s], got [%s]", v, r)
		}
	}
}

func TestCompareSemver(t *testing.T) {
	var tests = []struct {
		a, b []int64
		want int
	}{
		{[]int64{1, 23, 1}, []int64{1, 23, 1}, 0},
		{[]int64{1, 23, 1}, []int64{1, 23, 2}, -1},
		{[]int64{1, 23}, []int64{1, 23, 2}, -1},
		{[]int64{1, 23, 1}, []int64{1, 23}, 1},
		{[]int64{1}, []int64{1, 23, 1}, -1},
	}
	for _, tt := range tests {
		got := cmpSemver(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareSemver(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPipe(t *testing.T) {
	f, err := os.Open("testdata/test.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := Write(&b, doc); err != nil {
		t.Fatal(err)
	}
	b.WriteByte('\n')
	if !bytes.Equal(data, b.Bytes()) {
		o, err := os.Create("failed-test-output.json")
		if err != nil {
			t.Fatal(err)
		}
		defer o.Close()
		_, err = o.Write(b.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("data mismatch %d %d", len(data), len(b.Bytes()))
		for i := 0; i < len(data) && i < len(b.Bytes()); i++ {
			if data[i] != b.Bytes()[i] {
				t.Logf("mismatch at %d: \"%s\" != \"%s\"", i, string(data[i:i+25]), string(b.Bytes()[i:i+25]))
				break
			}
		}
	}
}

func BenchmarkParse(b *testing.B) {
	b.StopTimer()
	f, err := os.Open("testdata/test.json")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		b.Fatal(err)
	}
	r := bytes.NewReader(data)	
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r.Reset(data)
		if _, err := Read(r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQuery(b *testing.B) {
	b.StopTimer()
	f, err := os.Open("testdata/test.json")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		b.Fatal(err)
	}
	r := bytes.NewReader(data)	
	doc, err := Read(r)
	if err != nil {
		b.Fatal(err)
	}
	doc.Meta = nil
	i := 0
	Query(doc, func (Element) WalkResult {
		i++
		return WalkContinue
	})
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		Query(doc, func(elt *Str) WalkResult {
			return WalkContinue
		})
	}
}

func TestRead(t *testing.T) {
	r := strings.NewReader(t1)
	doc, err := Read(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", doc)
}

// func (intReader) read(d *json.Decoder) (int64, error) {
// 	tok, err := d.Token()
// 	if err != nil {
// 		return 0, err
// 	}
// 	if v, ok := tok.(json.Number); ok {
// 		return v.Int64()
// 	} else {
// 		return 0, errorf("expected number, got %v", tok)
// 	}
// }

// func TestD(t *testing.T) {
// 	const v = `[1,2,3,4,5,6]`
// 	j := json.NewDecoder(strings.NewReader(v))
// 	j.UseNumber()
// 	l, err := listr(readInt)(j)
// 	if err != nil {
// 		t.Fatal(err)
// 	} else {
// 		t.Logf("l=%v", l)
// 	}
// }

func readInt1(dec *json.Decoder) (int64, error) {
	// off := dec.InputOffset()
	tok, err := dec.Token()
	if err != nil {
		return 0, err
	}
	switch tok := tok.(type) {
	case json.Number:
		return tok.Int64()
	}
	return 0, errorf("expected int, got %v", tok)
	// return 0, errorf("expected int at %d, got %v", off, tok)
}

func testData() []byte {
	var b bytes.Buffer
	b.WriteString("[0")
	for i := 1; i < 1000; i++ {
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(i))
	}
	b.WriteByte(']')
	return b.Bytes()
}

func BenchmarkInline(b *testing.B) {
	b.StopTimer()
	v := []byte(`[{"t":"Space"},{"t":"Space"},{"t":"Space"},{"t":"Space"},{"t":"Space"},{"t":"Space"},{"t":"Space"},{"t":"Space"},{"t":"Space"},{"t":"Space"}]`)
	var r = bytes.NewReader(nil)
	var j scanner
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Reset(v)
		j.init(r)
		if _, err := testlistr(readInline)(&j); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkList(b *testing.B) {
	b.StopTimer()
	v := testData()
	var r = bytes.NewReader(nil)
	var j scanner
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Reset(v)
		j.init(r)
		if _, err := listr(readInt)(&j); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListStdStream(b *testing.B) {
	b.StopTimer()
	v := testData()
	b.StartTimer()
	b.ReportAllocs()
	var r []int64
	for i := 0; i < b.N; i++ {
		j := json.NewDecoder(bytes.NewReader(v))
		j.UseNumber()
		for {
			if tok, err := j.Token(); err != nil {
				break
			} else if n, ok := tok.(json.Number); ok {
				if n, err := n.Int64(); err != nil {
					b.Fatal(err)
				} else {
					r = append(r, n)
				}
			}
		}
	}
}

func BenchmarkListStd(b *testing.B) {
	b.StopTimer()
	v := testData()
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var r []int
		json.Unmarshal(v, &r)
	}
}

const t1 = `{"pandoc-api-version":[1,23,1],"meta":{},"blocks":[{"t":"Header","c":[1,["mainpage",["title"],[]],[{"t":"Str","c":"A"},{"t":"Space"},{"t":"Str","c":"document"}]]},{"t":"Para","c":[{"t":"Str","c":"Paragraph"}]},{"t":"Header","c":[2,["sec1",["h1"],[]],[{"t":"Str","c":"A"},{"t":"Space"},{"t":"Str","c":"section"}]]},{"t":"Para","c":[{"t":"Str","c":"Another"},{"t":"Space"},{"t":"Str","c":"paragraph"}]},{"t":"Header","c":[3,["sec1-1",["h2"],[]],[{"t":"Str","c":"A"},{"t":"Space"},{"t":"Str","c":"subsection"}]]},{"t":"Para","c":[{"t":"Str","c":"Yet"},{"t":"Space"},{"t":"Str","c":"another"},{"t":"Space"},{"t":"Str","c":"paragraph"}]},{"t":"Header","c":[4,["sec1-1-1",["h3"],[]],[{"t":"Str","c":"A"},{"t":"Space"},{"t":"Str","c":"subsubsection"}]]},{"t":"Para","c":[{"t":"Str","c":"And"},{"t":"Space"},{"t":"Str","c":"another"},{"t":"Space"},{"t":"Str","c":"paragraph"}]}]}`

func BenchmarkRead(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(t1)
		_, _ = Read(r)
	}
}

func BenchmarkQuery1(b *testing.B) {
	b.StopTimer()
	r := strings.NewReader(t1)
	doc, err := Read(r)
	if err != nil {
		b.Fatal(err)
	}
	var i int
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		Query(doc, func(elt *Header) WalkResult {
			i++
			return WalkContinue
		})
	}
	_ = i
	// b.Logf("i=%d", i)
}
