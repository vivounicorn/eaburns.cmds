// touch sets the modification time of a file or files.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	create = flag.Bool("c", true, "Create files that do not exist")
	mtime  = flag.String("t", "", "The modification time to set (YYYY-MM-DD:HH:MM:SS)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [<path> ...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	t := time.Now()
	if *mtime != "" {
		var err error
		t, err = time.Parse("2006-01-02:15:04:05", *mtime)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	status := 0
	for _, path := range flag.Args() {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			if !*create {
				continue
			}
			if f, err := os.Create(path); err != nil {
				status = 1
				fmt.Fprintln(os.Stderr, err)
				continue
			} else {
				f.Close()
			}
		} else if err != nil {
			status = 1
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if err = os.Chtimes(path, t, t); err != nil {
			status = 1
			fmt.Fprintln(os.Stderr, err)
		}
	}
	os.Exit(status)
}
