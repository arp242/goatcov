package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/tools/cover"
	"zgo.at/utils/sliceutil"
)

type (
	overview struct {
		Coverage float64
		Files    []file
	}

	file struct {
		Name     string
		Coverage float64
		Funcs    []fun
	}

	fun struct {
		Name     string
		Coverage float64
	}
)

func (o overview) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "Total: %3.0f%%\n\n", o.Coverage)

	for i, f := range o.Files {
		if i > 0 {
			fmt.Fprintf(b, "\n")
		}

		tab := tabwriter.NewWriter(b, 1, 8, 1, '\t', 0)

		fmt.Fprintf(tab, "%s\t%3.0f%%\n", f.Name, f.Coverage)
		for _, fn := range f.Funcs {
			fmt.Fprintf(tab, "    %s\t%3.0f%%\n", fn.Name, fn.Coverage)
		}
		tab.Flush()
	}

	return b.String()
}

const usage = `
goatcov creates and compares coverage reports for Go programs.

Flags:

  -profile  Coverage profile, as created by "go test -coverprofile".

  -diff     Diff against a previously generated report.

  -src      Source directory; defaults to current directory.

  -exclude  Paths to exclude, matched as strings.HasPrefix() to the full package
            path with the filename.

  -html     Output as HTML.

  -link     Link to files in the output.

            github:zgoat/goatcov
`

func main() {
	var (
		coverfile string
		prevfile  string
		srcdir    string
		exclude   []string
		html      bool
		link      string
	)
	flag.StringVar(&coverfile, "profile", "coverage", "")
	flag.StringVar(&prevfile, "diff", "", "")
	flag.StringVar(&srcdir, "src", ".", "")
	flag.BoolVar(&html, "html", false, "")
	flag.StringVar(&link, "link", "", "")
	e := flag.String("exclude", "", "")
	flag.Usage = func() { fmt.Fprintf(os.Stderr, usage) }
	flag.Parse()
	if len(flag.Args()) > 0 {
		flag.Usage()
		os.Exit(1)
	}

	if e != nil && *e != "" {
		exclude = strings.Split(*e, ",")
	}

	var err error
	if prevfile != "" {
		err = diff(html, coverfile, prevfile, srcdir, exclude)
	} else {
		err = printreport(html, coverfile, srcdir, exclude)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		flag.Usage()
		os.Exit(1)
	}
}

func printreport(html bool, coverfile, srcdir string, exclude []string) error {
	r, err := report(coverfile, srcdir, exclude)
	if err != nil {
		return err
	}

	if html {
		tpl := template.New("tpl").Funcs(template.FuncMap{
			"float": func(f float64) string { return fmt.Sprintf("%3.0f", f) },
		})
		tpl, err := tpl.Parse(htmlTemplate)
		if err != nil {
			return err
		}

		tpl.Execute(os.Stdout, map[string]interface{}{
			"Overview": r,
		})
	} else {
		fmt.Print(r.String())
	}

	return nil
}

func diff(html bool, coverfile, prevfile, srcdir string, exclude []string) error {
	r1, err := report(prevfile, srcdir, exclude)
	if err != nil {
		return err
	}
	r2, err := report(coverfile, srcdir, exclude)
	if err != nil {
		return err
	}

	b := new(strings.Builder)
	if r1.Coverage != r2.Coverage {
		fmt.Fprintf(b, "Total %.0f%% → %.0f%% (%+3.2f%%)\n",
			r1.Coverage, r2.Coverage, r2.Coverage-r1.Coverage)
	}

	files := map[string]file{}
	for _, f := range r1.Files {
		files[f.Name] = f
	}

	for _, f2 := range r2.Files {
		f1, ok := files[f2.Name]
		if !ok {
			f1 = file{Name: f2.Name, Coverage: 0}
		}

		funcs := map[string]fun{}
		for _, fn := range f1.Funcs {
			funcs[fn.Name] = fn
		}

		tab := tabwriter.NewWriter(b, 1, 8, 1, '\t', 0)

		shownFile := false
		for _, fn2 := range f2.Funcs {
			fn1, ok := funcs[fn2.Name]
			if !ok {
				fn1 = fun{Name: fn2.Name, Coverage: 0}
			}

			if fn1.Coverage != fn2.Coverage {
				if !shownFile {
					shownFile = true
					fmt.Fprintf(tab, "\n%s\t%3.0f%% → %3.0f%% (%+3.2f%%)\n",
						f2.Name, f1.Coverage, f2.Coverage, f2.Coverage-f1.Coverage)
				}
				fmt.Fprintf(tab, "    %s\t%3.0f%% → %3.0f%% (%+3.2f%%)\n",
					fn2.Name, fn1.Coverage, fn2.Coverage, fn2.Coverage-fn1.Coverage)
			}
		}
		if shownFile {
			tab.Flush()
		}
	}

	fmt.Println(b.String())
	return nil
}

func report(coverfile, srcdir string, exclude []string) (overview, error) {
	prof, err := cover.ParseProfiles(coverfile)
	if err != nil {
		return overview{}, err
	}
	dirs, err := findPkgs(srcdir, prof)
	if err != nil {
		return overview{}, err
	}

	var (
		ret            overview
		total, covered int64
	)
	for _, p := range prof {
		if sliceutil.InStringSlice(exclude, p.FileName) {
			continue
		}

		f, err := findFile(dirs, p.FileName)
		if err != nil {
			return overview{}, err
		}
		funcs, err := findFuncs(f)
		if err != nil {
			return overview{}, err
		}

		o := file{
			Name:     p.FileName,
			Coverage: percentCovered(p),
		}
		for _, fn := range funcs {
			c, t := fn.coverage(p)
			o.Funcs = append(o.Funcs, fun{
				Name:     fn.name,
				Coverage: percent(c, t),
			})
			total += t
			covered += c
		}
		ret.Files = append(ret.Files, o)
	}

	ret.Coverage = percent(covered, total)
	return ret, nil
}
