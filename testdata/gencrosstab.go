//go:build ignore

package main

// Examples taken from: https://wiki.postgresql.org/wiki/Crosstabview

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

func main() {
	dsn := flag.String("dsn", "postgres://postgres:P4ssw0rd@localhost", "dsn")
	out := flag.String("out", "crosstab.txt", "out")
	flag.Parse()
	if err := run(context.Background(), *dsn, *out); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dsn, out string) error {
	// setup data
	if _, err := psqlExec(ctx, dsn, dropsql, createsql); err != nil {
		return err
	}
	settings := []string{
		`\pset format unaligned`,
		`\pset footer off`,
	}
	buf := new(bytes.Buffer)
	for _, q := range crosstabQueries {
		s := strings.Split(q, ` \crosstabview`)
		if len(s) != 2 {
			return fmt.Errorf(`expected query to have \crosstabview: %v`, s)
		}
		b0, err := psqlExec(ctx, dsn, append(settings, s[0])...)
		if err != nil {
			return err
		}
		b1, err := psqlExec(ctx, dsn, append(settings, q)...)
		if err != nil {
			return err
		}
		if _, err = fmt.Fprintf(buf, "%s\n--\n%s\n--\n%s\n\n", q, string(b0), string(b1)); err != nil {
			return err
		}
	}
	if _, err := psqlExec(ctx, dsn, dropsql); err != nil {
		return err
	}
	b := bytes.TrimRightFunc(buf.Bytes(), unicode.IsSpace)
	if err := os.WriteFile(out, append(b, '\n'), 0644); err != nil {
		return err
	}
	return nil
}

func psqlExec(ctx context.Context, dsn string, sqlstrs ...string) ([]byte, error) {
	stdin, stdout, stderr := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)
	for _, sqlstr := range sqlstrs {
		sqlstr = strings.TrimSpace(sqlstr)
		if !strings.Contains(sqlstr, `\`) {
			sqlstr += ";"
		}
		fmt.Fprintf(stdin, "%s\n", sqlstr)
	}
	cmd := exec.CommandContext(ctx, `psql`, dsn, "-q")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	if err := cmd.Run(); err != nil {
		log.Printf(">>> stderr:\n%s\n---", strings.TrimRightFunc(stderr.String(), unicode.IsSpace))
		return nil, err
	}
	return bytes.TrimRightFunc(stdout.Bytes(), unicode.IsSpace), nil
}

const (
	dropsql   = `drop view if exists v_data; drop view if exists my_table;`
	createsql = `create view v_data as
select * from (values
   ('v1','h2','foo', '2015-04-01'::date),
   ('v2','h1','bar', '2015-01-02'),
   ('v1','h0','baz', '2015-07-12'),
   ('v0','h4','qux', '2015-07-15')
 ) as l(v,h,c,d);
create view my_table as
select * from (values
  (1, 'one'),
  (2, 'two'),
  (3, 'three'),
  (4, 'four')
) as my_table(first, second);`
)

var crosstabQueries = []string{
	`select v,h,c from v_data \crosstabview`,                                                                                  // example 0
	`select v,h,c from v_data order by 1 \crosstabview v h c`,                                                                 // example 1
	`select v,h,c from v_data order by 1 desc \crosstabview v h c`,                                                            // example 2
	`select v,h,c from v_data order by 2 \crosstabview v h c`,                                                                 // example 3
	`select v,h,c,row_number() over(order by h) as hsort from v_data order by 1 \crosstabview v h c`,                          // no example
	`select v,h,c,row_number() over(order by h) as hsort from v_data order by 1 \crosstabview v h c hsort`,                    // example 4
	`select v,h,c,row_number() over(order by h desc) as hsort from v_data order by 1 \crosstabview v h c hsort`,               // example 5
	`select v,to_char(d,'Mon') as m, c from v_data order by 1 \crosstabview v m c`,                                            // example 6
	`select v,to_char(d,'Mon') as m, c from v_data order by d \crosstabview v m c`,                                            // example 7
	`select v,to_char(d,'Mon') as m, c, extract(month from d) as mnum from v_data order by v \crosstabview v m c mnum`,        // example 8
	`select v,to_char(d,'Mon') as m, c, -1*extract(month from d) as revnum from v_data order by v \crosstabview v m c revnum`, // example 9
	// psql manual example 1
	`SELECT first, second, first > 2 AS gt2 FROM my_table \crosstabview first second`,
	// psql manual example 2
	`SELECT t1.first as A, t2.first+100 AS B, t1.first*(t2.first+100) as AxB, row_number()` +
		` over(order by t2.first) AS ord FROM my_table t1 CROSS JOIN my_table t2 ORDER BY 1 DESC \crosstabview A B AxB ord `,
}
