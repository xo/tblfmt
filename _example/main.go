// _example/main.go
package main

import (
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/xo/dburl"
	"github.com/xo/tblfmt"
)

func main() {
	db, err := dburl.Open("postgres://booktest:booktest@localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	result, err := db.Query("select * from authors")
	if err != nil {
		log.Fatal(err)
	}
	defer result.Close()
	enc, err := tblfmt.NewTableEncoder(result,
		// force minimum column widths
		tblfmt.WithWidths(20, 20),
	)
	if err = enc.EncodeAll(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
