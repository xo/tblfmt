// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/xo/tblfmt"
	"github.com/xo/tblfmt/internal"
)

func main() {
	seed := flag.Int64("seed", 1549508725559526476, "seed")
	flag.Parse()
	if err := run(*seed); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(seed int64) error {
	bufMap := make(map[string]*bytes.Buffer)
	for _, config := range buildConfigs() {
		for _, data := range []struct {
			resultSet tblfmt.ResultSet
			desc      string
		}{
			{internal.NewRsetBig(seed), "big"},
			{internal.NewRsetMulti(), "multi"},
			{internal.NewRsetTiny(), "tiny"},
			{internal.NewRsetWide(), "wide"},
		} {
			buf := bufMap[data.desc]
			if buf == nil {
				buf = new(bytes.Buffer)
				bufMap[data.desc] = buf
			}
			optdesc := ""
			if len(config.desc) != 0 {
				optdesc = "\noptions:\n  " + strings.Join(config.desc, "\n  ")
			}
			_, err := fmt.Fprintf(
				buf,
				"%s\nencoder: %s%s\n%s\n",
				divider,
				config.encoder,
				optdesc,
				divider,
			)
			if err != nil {
				return err
			}
			enc, err := config.f(data.resultSet, config.opts...)
			if err != nil {
				return err
			}
			if err := enc.EncodeAll(buf); err != nil {
				return err
			}
		}
	}
	for k, buf := range bufMap {
		if err := ioutil.WriteFile(k+".txt", buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

type config struct {
	f       tblfmt.Builder
	opts    []tblfmt.Option
	encoder string
	desc    []string
}

func buildConfigs() []config {
	type opt struct {
		opts []tblfmt.Option
		desc []string
	}
	opts := []opt{{}}
	for _, s := range []struct {
		s    tblfmt.LineStyle
		desc string
	}{
		{tblfmt.ASCIILineStyle(), "ascii"},
		{tblfmt.OldASCIILineStyle(), "old-ascii"},
		{tblfmt.UnicodeLineStyle(), "unicode"},
		{tblfmt.UnicodeDoubleLineStyle(), "unicode-double"},
	} {
		for i := 0; i < 3; i++ {
			opts = append(opts, opt{
				opts: []tblfmt.Option{
					tblfmt.WithLineStyle(s.s),
					tblfmt.WithBorder(i),
				},
				desc: []string{"linestyle: " + s.desc, fmt.Sprintf("border: %d", i)},
			})
		}
	}
	var configs []config
	for _, o := range opts {
		configs = append(configs, config{
			f:       tblfmt.NewTableEncoder,
			opts:    o.opts[:],
			encoder: "table",
			desc:    o.desc,
		})
	}
	for _, o := range opts {
		configs = append(configs, config{
			f:       tblfmt.NewExpandedEncoder,
			opts:    o.opts[:],
			encoder: "expanded",
			desc:    o.desc,
		})
	}
	return append(
		configs,
		config{f: tblfmt.NewJSONEncoder, encoder: "json"},
		config{f: tblfmt.NewUnalignedEncoder, encoder: "unaligned"},
		config{f: tblfmt.NewCSVEncoder, encoder: "csv"},
		config{f: tblfmt.NewHTMLEncoder, encoder: "html"},
		config{f: tblfmt.NewAsciiDocEncoder, encoder: "asciidoc"},
	)
}

const divider = "==============================================="
