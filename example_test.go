package tblfmt_test

import (
	"fmt"
	"log"
	"os"

	"github.com/xo/tblfmt"
)

// result is a simple type providing a tblfmt.ResultSet.
type result struct {
	pos  int
	cols []string
	vals [][]interface{}
}

// Columns  satisfies the tblfmt.ResultSet interface.
func (res *result) Columns() ([]string, error) {
	return res.cols, nil
}

// Next satisfies the tblfmt.ResultSet interface.
func (res *result) Next() bool {
	return res.pos < len(res.vals)
}

// Scan satisfies the tblfmt.ResultSet interface.
func (res *result) Scan(vals ...interface{}) error {
	for i := range vals {
		x, ok := vals[i].(*interface{})
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

// getDatabaseResults returns a tblfmt.ResultSet, which is an interface that is
// compatible with Go's standard
func getDatabaseResults() tblfmt.ResultSet {
	return &result{
		cols: []string{"author_id", "name", "z"},
		vals: [][]interface{}{
			{14, "a\tb\tc\td", nil},
			{15, "aoeu\ntest\n", nil},
			{2, "袈\t袈\t\t袈", nil},
		},
	}
}

func ExampleEncodeAll() {
	res := getDatabaseResults()
	if err := tblfmt.EncodeAll(os.Stdout, res, map[string]string{
		"format":   "csv",
		"quote":    "true",
		"fieldsep": "|",
		"null":     "<nil>",
	}); err != nil {
		log.Fatal(err)
	}
	// Output:
	//author_id,name,z
	//14,"a	b	c	d",
	//15,"aoeu
	//test
	//",
	//2,"袈	袈		袈",
}

func ExampleEncodeTemplateAll_html() {
	res := getDatabaseResults()
	if err := tblfmt.EncodeTemplateAll(os.Stdout, res, tblfmt.WithTemplate("html")); err != nil {
		log.Fatal(err)
	}
	// Output:
	//<table>
	//  <caption></caption>
	//  <thead>
	//    <tr>
	//      <th align="left">author_id</th>
	//      <th align="left">name</th>
	//      <th align="left">z</th>
	//    </tr>
	//  </thead>
	//  <tbody>
	//    <tr>
	//      <td align="right">14</td>
	//      <td align="left">a	b	c	d</td>
	//      <td align="left"></td>
	//    </tr>
	//    <tr>
	//      <td align="right">15</td>
	//      <td align="left">aoeu
	//test
	//</td>
	//      <td align="left"></td>
	//    </tr>
	//    <tr>
	//      <td align="right">2</td>
	//      <td align="left">袈	袈		袈</td>
	//      <td align="left"></td>
	//    </tr>
	//  </tbody>
	//</table>
}

func ExampleNewTableEncoder_encodeAll() {
	res := getDatabaseResults()
	enc, err := tblfmt.NewTableEncoder(
		res,
		tblfmt.WithBorder(2),
		tblfmt.WithLineStyle(tblfmt.UnicodeDoubleLineStyle()),
		tblfmt.WithWidths([]int{20, 20}),
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
	// ║ author_id            ║ name                      ║ z ║
	// ╠══════════════════════╬═══════════════════════════╬═══╣
	// ║                   14 ║ a	b	c	d  ║   ║
	// ║                   15 ║ aoeu                     ↵║   ║
	// ║                      ║ test                     ↵║   ║
	// ║                      ║                           ║   ║
	// ║                    2 ║ 袈	袈		袈 ║   ║
	// ╚══════════════════════╩═══════════════════════════╩═══╝
	//(3 rows)
}
