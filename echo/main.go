// echo echos command-line arguments to standard output.
package main

import (
	"flag"
	"os"
)

var nflag = flag.Bool("n", false, "Elide the final newline")

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		os.Stdout.WriteString(args[0])
		for _, s := range args[1:] {
			os.Stdout.WriteString(" "+s)
		}
	}
	if !*nflag {
		os.Stdout.WriteString("\n")
	}
}