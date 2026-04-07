package tblfmt_test

import (
	"fmt"
	"log"
	"os"

	"github.com/xo/tblfmt"
)

func Example() {
	res := getDatabaseResults()
	if err := tblfmt.EncodeAll(os.Stdout, res, map[string]string{
		"format": "aligned",
		"border": "2",
	}); err != nil {
		log.Fatal(err)
	}
}

func ExampleEncodeAll() {
	res := getDatabaseResults()
	if err := tblfmt.EncodeAll(os.Stdout, res, map[string]string{
		"format":   "csv",
		"fieldsep": "|",
		"null":     "<nil>",
	}); err != nil {
		log.Fatal(err)
	}
	// Output:
	// author_id,name,z
	// 14,"a	b	c	d",<nil>
	// 15,"aoeu
	// test
	// ",<nil>
	// 2,"袈	袈		袈",<nil>
}

func ExampleNewTableEncoder() {
	res := getDatabaseResults()
	enc, err := tblfmt.NewTableEncoder(
		res,
		tblfmt.WithBorder(2),
		tblfmt.WithLineStyle(tblfmt.UnicodeDoubleLineStyle()),
		tblfmt.WithWidths(20, 20),
		tblfmt.WithSummary(tblfmt.DefaultTableSummary()),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := enc.EncodeAll(os.Stdout); err != nil {
		log.Fatal(err)
	}
	// Output:
	// ╔══════════════════════╦═══════════════════════════╦═══╗
	// ║      author_id       ║           name            ║ z ║
	// ╠══════════════════════╬═══════════════════════════╬═══╣
	// ║                   14 ║ a	b	c	d  ║   ║
	// ║                   15 ║ aoeu                     ↵║   ║
	// ║                      ║ test                     ↵║   ║
	// ║                      ║                           ║   ║
	// ║                    2 ║ 袈	袈		袈 ║   ║
	// ╚══════════════════════╩═══════════════════════════╩═══╝
	// (3 rows)
}

func ExampleEncodeTemplateAll() {
	res := getDatabaseResults()
	if err := tblfmt.EncodeTemplateAll(os.Stdout, res, tblfmt.WithTemplate("html")); err != nil {
		log.Fatal(err)
	}
	// Output:
	// <table>
	//   <caption></caption>
	//   <thead>
	//     <tr>
	//       <th align="left">author_id</th>
	//       <th align="left">name</th>
	//       <th align="left">z</th>
	//     </tr>
	//   </thead>
	//   <tbody>
	//     <tr>
	//       <td align="right">14</td>
	//       <td align="left">a	b	c	d</td>
	//       <td align="left"></td>
	//     </tr>
	//     <tr>
	//       <td align="right">15</td>
	//       <td align="left">aoeu
	// test
	// </td>
	//       <td align="left"></td>
	//     </tr>
	//     <tr>
	//       <td align="right">2</td>
	//       <td align="left">袈	袈		袈</td>
	//       <td align="left"></td>
	//     </tr>
	//   </tbody>
	// </table>
}

//func Example_fromMap() {
//	res := getDatabaseResults()
//	builder, opts := tblfmt.FromMap(map[string]string{
//		"format": "table",
//	})
//	enc, err := builder(res, opts...)
//	if err != nil {
//		log.Fatal(err)
//	}
//	var buf bytes.Buffer
//	if err := enc.EncodeAll(&buf); err != nil {
//		log.Fatal(err)
//	}
//	// extra padding to deal with Go's example output
//	b := append([]byte{'#'}, append(bytes.TrimSpace(buf.Bytes()), '#')...)
//	b = bytes.ReplaceAll(b, []byte{'\n'}, []byte{'#', '\n', '#'})
//	if _, err := os.Stdout.Write(b); err != nil {
//		log.Fatal(err)
//	}
//	// Output:
//	// AUTHOR_ID NAME                     Z
//	// 14        a     b       c       d
//	// 15        aoeu
//	//           test
//	//
//	// 2         袈    袈              袈
//	// (3 rows)
//}

// getDatabaseResults returns a tblfmt.ResultSet, which is an interface that is
// compatible with Go's standard.
func getDatabaseResults() tblfmt.ResultSet {
	return &result{
		cols: []string{"author_id", "name", "z"},
		vals: [][]any{
			{14, "a\tb\tc\td", nil},
			{15, "aoeu\ntest\n", nil},
			{2, "袈\t袈\t\t袈", nil},
		},
	}
}

// result is a simple type providing a tblfmt.ResultSet.
type result struct {
	pos  int
	cols []string
	vals [][]any
}

// Columns satisfies the tblfmt.ResultSet interface.
func (res *result) Columns() ([]string, error) {
	return res.cols, nil
}

// Next satisfies the tblfmt.ResultSet interface.
func (res *result) Next() bool {
	return res.pos < len(res.vals)
}

// Scan satisfies the tblfmt.ResultSet interface.
func (res *result) Scan(vals ...any) error {
	for i := range vals {
		x, ok := vals[i].(*any)
		if !ok {
			return fmt.Errorf("scan for col %d expected *interface{}, got: %T", i, vals[i])
		}
		*x = res.vals[res.pos][i]
	}
	res.pos++
	return nil
}

// Err satisfies the tblfmt.ResultSet interface.
func (res *result) Err() error {
	return nil
}

// Close satisfies the tblfmt.ResultSet interface.
func (res *result) Close() error {
	return nil
}

// NextResultSet satisfies the tblfmt.ResultSet interface.
func (res *result) NextResultSet() bool {
	return false
}
