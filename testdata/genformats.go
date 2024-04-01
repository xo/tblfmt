//go:build ignore

package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

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
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	m := make(map[string]*bytes.Buffer)
	for _, cfg := range configs() {
		for _, data := range []struct {
			name string
			rs   tblfmt.ResultSet
		}{
			{"big", internal.Big(seed)},
			{"multi", internal.Multi()},
			{"tiny", internal.Tiny()},
			{"wide", internal.Wide()},
		} {
			buf := m[data.name]
			if buf == nil {
				buf = new(bytes.Buffer)
				m[data.name] = buf
			}
			optdesc := ""
			if len(cfg.desc) != 0 {
				optdesc = "\n" + strings.Join(cfg.desc, "\n")
			}
			_, err := fmt.Fprintf(
				buf,
				"%s\nformat: %s%s\n%s\n",
				internal.Divider,
				cfg.format,
				optdesc,
				internal.Divider,
			)
			if err != nil {
				return err
			}
			enc, err := cfg.f(data.rs, cfg.opts...)
			if err != nil {
				return err
			}
			if err := enc.EncodeAll(buf); err != nil {
				return err
			}
		}
	}
	for name, buf := range m {
		out := new(bytes.Buffer)
		w := gzip.NewWriter(out)
		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		if err := os.WriteFile(name+".gz", out.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

type config struct {
	f      tblfmt.Builder
	opts   []tblfmt.Option
	format string
	desc   []string
}

func configs() []config {
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
	var v []config
	for _, o := range opts {
		v = append(v, config{
			f:      tblfmt.NewTableEncoder,
			opts:   o.opts[:],
			format: "aligned",
			desc:   o.desc,
		})
	}
	for _, o := range opts {
		v = append(v, config{
			f:      tblfmt.NewExpandedEncoder,
			opts:   o.opts[:],
			format: "expanded",
			desc:   o.desc,
		})
	}
	return append(v,
		config{f: tblfmt.NewJSONEncoder, format: "json"},
		config{f: tblfmt.NewUnalignedEncoder, format: "unaligned"},
		config{f: tblfmt.NewCSVEncoder, format: "csv"},
		config{f: tblfmt.NewHTMLEncoder, format: "html"},
		config{f: tblfmt.NewAsciiDocEncoder, format: "asciidoc"},
	)
}
