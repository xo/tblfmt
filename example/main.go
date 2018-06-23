// example/main.go
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

	err = tblfmt.NewTableEncoder(result).Encode(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
