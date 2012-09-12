// wc counts words, characters, and lines.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"unicode"
)

var (
	printLines = flag.Bool("l", false, "Print the number of lines")
	printWords = flag.Bool("w", false, "Print the number of words")
	printRunes = flag.Bool("r", false, "Print the number of runes")
	printChars = flag.Bool("c", false, "Print the number of characters")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [<path> ...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if !*printLines && !*printWords && !*printRunes && !*printChars {
		*printLines = true
		*printWords = true
		*printChars = true
	}

	status := 0
	var totalLines, totalWords, totalRunes, totalChars int
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
	for _, path := range flag.Args() {
		file, err := os.Open(path)
		if err != nil {
			status = 1
			os.Stderr.WriteString(err.Error()+"\n")
			continue
		}

		nl, nw, nr, nc, err := count(bufio.NewReader(file))
		file.Close()

		if err != nil {
			status = 1
			os.Stderr.WriteString(err.Error()+"\n")
			continue
		} else {
			printCounts(w, nl, nw, nr, nc)
			fmt.Fprintln(w, path)
			totalLines += nl
			totalWords += nw
			totalRunes += nr
			totalChars += nc
		}
	}

	if len(flag.Args()) == 0 {
		if nl, nw, nr, nc, err := count(bufio.NewReader(os.Stdin)); err != nil {
			status = 1
			os.Stderr.WriteString(err.Error()+"\n")
		} else {
			printCounts(w, nl, nw, nr, nc)
			fmt.Fprintln(w, "")
		}
	}

	if len(flag.Args()) > 1 {
		if *printLines {
			fmt.Fprint(w, totalLines, "\t")
		}
		if *printWords {
			fmt.Fprint(w, totalWords, "\t")
		}
		if *printRunes {
			fmt.Fprint(w, totalRunes, "\t")
		}
		if *printChars {
			fmt.Fprint(w, totalChars, "\t")
		}
		fmt.Fprintln(w, "total")
	}
	w.Flush()
	os.Exit(status)
}

func printCounts(w io.Writer, nl, nw, nr, nc int) {
	if *printLines {
		fmt.Fprint(w, nl, "\t")
	}
	if *printWords {
		fmt.Fprint(w, nw, "\t")
	}
	if *printRunes {
		fmt.Fprint(w, nr, "\t")
	}
	if *printChars {
		fmt.Fprint(w, nc, "\t")
	}
}

func count(in *bufio.Reader) (nl, nw, nr, nc int, err error) {
	inword := false
	for {
		var r rune
		var sz int
		r, sz, err = in.ReadRune()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}

		nr++
		nc += sz
		if r == '\n' {
			nl++
		}
		if unicode.IsSpace(r) && inword {
			inword = false
			nw++
		} else if !unicode.IsSpace(r) {
			inword = true
		}
	}
	return
}
