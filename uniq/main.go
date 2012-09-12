// uniq prints lines read from standard input
// to standard output, filtering out adjacent lines
// that are the same.
package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
)

var stdin = bufio.NewReader(os.Stdin)
var stdout = bufio.NewWriter(os.Stdout)

func main() {
	defer stdout.Flush()
	var prevLine []byte

	for {
		line, err := stdin.ReadBytes('\n')
		line = bytes.TrimRight(line, "\r\n")
		if err == io.EOF {
			return
		} else if err != nil {
			die(err)
		}

		if !bytes.Equal(line, prevLine) {
			_, err = stdout.Write(append(line, '\n'))
			if err != nil {
				die(err)
			}
			prevLine = line
		}
	}
}

func die(err error) {
	os.Stderr.WriteString(err.Error()+"\n")
	os.Exit(1)
}