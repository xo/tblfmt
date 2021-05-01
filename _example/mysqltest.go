// +build ignore

package main

import (
	"flag"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/xo/dburl"
	"github.com/xo/tblfmt"
	_ "github.com/ziutek/mymysql/godrv"
)

func main() {
	useColumnTypes := flag.Bool("u", false, "use column types")
	flag.Parse()
	if err := run(*useColumnTypes); err != nil {
		log.Fatal(err)
	}
}

func run(useColumnTypes bool) error {
	db, err := dburl.Open("mysql://root:P4ssw0rd@localhost/testdb?parseTime=true")
	// db, err := dburl.Open("mymysql://root:P4ssw0rd@localhost/testdb")
	if err != nil {
		return err
	}
	res, err := db.Query("select * from a_bit_of_everything")
	if err != nil {
		return err
	}
	defer res.Close()
	return tblfmt.EncodeTableAll(os.Stdout, res, tblfmt.WithUseColumnTypes(useColumnTypes), tblfmt.WithEmpty("stuff"))
}
