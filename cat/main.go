// cat echos the contents of files to standard output.
package main

import (
	"io"
	"os"
)

func main() {
	var err error
	status := 0
	if len(os.Args) == 1 {
		if _, err = io.Copy(os.Stdout, os.Stdin); err != nil {
			status = 1
			os.Stderr.WriteString(err.Error()+"\n")
		}
	}
	for _, path := range os.Args[1:] {
		var file *os.File
		if file, err = os.Open(path); err != nil {
			status = 1
			os.Stderr.WriteString(err.Error()+"\n")
			continue
		}
		if _, err = io.Copy(os.Stdout, file); err != nil {
			status = 1
			os.Stderr.WriteString(err.Error()+"\n")
		}
		file.Close()
	}
	os.Exit(status)
}
