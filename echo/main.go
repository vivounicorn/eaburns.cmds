// echo echos command-line arguments to standard output.
package main

import (
	"flag"
	"fmt"
)

var nflag = flag.Bool("n", false, "Elide the final newline")

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		fmt.Print(args[0])
		for _, s := range args[1:] {
			fmt.Print(" ", s)
		}
	}
	if !*nflag {
		fmt.Println("")
	}
}