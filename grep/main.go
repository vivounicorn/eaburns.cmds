package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
)

var (
	vFlag = flag.Bool("v", false, "reverse: print lines not matching the pattern")
	nFlag = flag.Bool("n", false, "print line numbers")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <regexp> [<path> ...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	re, err := regexp.Compile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(flag.Args()) == 1 {
		grep(re, "", os.Stdin)
	}
	status := 0
	for _, path := range flag.Args()[1:] {
		file, err := os.Open(path)
		if err != nil {
			status = 1
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if err := grep(re, path, file); err != nil {
			status = 1
			fmt.Fprintln(os.Stderr, err)
		}
		file.Close()
	}
	os.Exit(status)
}

func grep(re *regexp.Regexp, path string, r io.Reader) error {
	in := bufio.NewReader(r)
	lineNo := 0
	for {
		switch line, prefix, err := in.ReadLine(); {
		case prefix:
			return errors.New("Line is too long")
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		default:
			lineNo++
			match := re.Match(line)
			if (match && !*vFlag) || (!match && *vFlag) {
				if *nFlag && path != "" {
					fmt.Print(path, ":")
				}
				if *nFlag {
					fmt.Print(lineNo, ":")
				}
				fmt.Println(string(line))
			}
		}
	}
	panic("Unreachable")
}
