// sorts prints lines of files in sorted order to standard output.
package main

import (
	"bufio"
	"container/heap"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"sync/atomic"
)

const (
	// Number of lines to read before sorting and
	// dumping to disk.
	chunkSize = 500000

	// Number of files to merge at a single time.
	mergeSize = 10

	// bufSize is the output file buffer size.
	bufSize = 8 * 1024
)

var (
	// nerrors counts the non-fatal errors
	nerrors int32

	// tempFiles is a list of all temporary files
	// created for the sorting so that they can
	// be cleaned up on exit.
	tempFiles = []string{}
)

func main() {
	w := bufio.NewWriterSize(os.Stdout, bufSize)
	defer w.Flush()
	err := mergeSort(w, os.Args[1:])
	remove(tempFiles...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err != nil || nerrors > 0 {
		os.Exit(1)
	}
}

// mergeSort sorts the lines of the files and prints
// them in sorted order to the writer.
func mergeSort(out io.Writer, paths []string) error {
	files, err := sortChunks(out, allLines(os.Args[1:]))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	for len(files) > 1 {
		if len(files) > mergeSize {
			f, err := ioutil.TempFile(os.TempDir(), "sort-merge")
			if err != nil {
				return err
			}
			tempFiles = append(tempFiles, f.Name())
			files = append(files[mergeSize:], f.Name())

			w := bufio.NewWriterSize(f, bufSize)
			err = mergeFiles(w, files[:mergeSize])
			w.Flush()
			f.Close()
			if err != nil {
				return err
			}
		} else {
			if err := mergeFiles(out, files); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// mergeFiles merges the given files, writing
// the result into another temporary file.  The
// resulting file name is returned and the input
// files are removed.
func mergeFiles(out io.Writer, paths []string) error {
	var q fileHeap
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		lines := readerLines(f)
		if l, ok := <-lines; ok {
			heap.Push(&q, fileHeapEntry{l, lines, f})
		} else {
			f.Close()
			remove(f.Name())
		}
	}
	for len(q) > 0 {
		l := heap.Pop(&q).(fileHeapEntry)
		line := l.line
		if nxt, ok := <-l.lines; ok {
			l.line = nxt
			heap.Push(&q, l)
		} else {
			remove(l.file.Name())
			l.file.Close()
		}
		if err := writeLine(out, line); err != nil {
			return err
		}
	}
	return nil
}

// sortChunks breaks the input into chunks of
// a fixed number of lines, sorts each chunk and
// writes it to a temporary file.  The return value
// is a slice of the temporary file names.
//
// If there is only one chunk then it is sorted and
// printed without going to disk.  In this case, the
// returned slice is empty.
func sortChunks(out io.Writer, ls <-chan string) ([]string, error) {
	tmpFiles := []string{}

	chunk := make(lines, 0, chunkSize)
	for line := range ls {
		if len(chunk) == chunkSize {
			sort.Sort(chunk)
			tfile, err := writeChunk(chunk)
			if err != nil {
				return nil, err
			}
			tmpFiles = append(tmpFiles, tfile)
			chunk = make(lines, 0, chunkSize)
		}
		chunk = append(chunk, line)
	}
	if len(chunk) > 0 {
		sort.Sort(chunk)
		if len(tmpFiles) == 0 {
			for _, l := range chunk {
				if err := writeLine(out, l); err != nil {
					atomic.AddInt32(&nerrors, 1)
					fmt.Fprintln(os.Stderr, err)
					break
				}
			}
			return nil, nil
		}
		tfile, err := writeChunk(chunk)
		if err != nil {
			return nil, err
		}
		tmpFiles = append(tmpFiles, tfile)
	}
	return tmpFiles, nil
}

// writeChunk writes a chunk to a temporary file,
// returning the temporary file name.
func writeChunk(chunk lines) (string, error) {
	f, err := ioutil.TempFile(os.TempDir(), "sort-sort")
	if err != nil {
		return "", err
	}
	tempFiles = append(tempFiles, f.Name())
	defer f.Close()

	w := bufio.NewWriterSize(f, bufSize)
	defer w.Flush()

	for _, l := range chunk {
		if err := writeLine(w, l); err != nil {
			return "", err
		}
	}

	return f.Name(), nil
}

// allLines returns a channel upon which all
// of the lines of all of the files will be sent.
func allLines(paths []string) <-chan string {
	ch := make(chan string)
	go func() {
		if len(paths) == 0 {
			paths = []string{"-"}
		}
		for _, p := range paths {
			var in io.Reader = os.Stdin
			if p != "-" {
				var err error
				in, err = os.Open(p)
				if err != nil {
					atomic.AddInt32(&nerrors, 1)
					fmt.Fprintln(os.Stderr, err)
					continue
				}
			}
			for l := range readerLines(in) {
				ch <- l
			}
			if p != "-" {
				in.(*os.File).Close()
			}
		}
		close(ch)
	}()
	return ch
}

// readerLines returns a channel, each line from the
// reader is sent on the channel.
func readerLines(r io.Reader) <-chan string {
	ch := make(chan string)
	go func() {
		in := bufio.NewReader(r)
		var err error
		var l []byte
		var prefix bool
		for {
			l, prefix, err = in.ReadLine()
			if err != nil {
				break
			}
			ch <- string(l)
			if !prefix {
				continue
			}

			fmt.Fprintln(os.Stderr, "line is too long: truncating")
			_, prefix, err = in.ReadLine()
			for prefix && err == nil {
				_, prefix, err = in.ReadLine()
			}
			if err != nil {
				break
			}
		}
		if err != io.EOF {
			atomic.AddInt32(&nerrors, 1)
			fmt.Fprintln(os.Stderr, err)
		}
		close(ch)
	}()
	return ch
}

// remove removes a slice of files
func remove(paths ...string) {
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			atomic.AddInt32(&nerrors, 1)
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

type lines []string

func (l lines) Len() int {
	return len(l)
}

func (l lines) Less(i, j int) bool {
	return less(l[i], l[j])
}

func (l lines) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

type fileHeapEntry struct {
	line  string
	lines <-chan string
	file  *os.File
}

type fileHeap []fileHeapEntry

func (h fileHeap) Len() int {
	return len(h)
}

func (h fileHeap) Less(i, j int) bool {
	return less(h[i].line, h[j].line)
}

func (h fileHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *fileHeap) Push(x interface{}) {
	*h = append(*h, x.(fileHeapEntry))
}

func (h *fileHeap) Pop() interface{} {
	x := (*h)[len(*h)-1]
	*h = (*h)[:len(*h)-1]
	return x
}

// less compares two lines.
func less(a, b string) bool {
	return a < b
}

// writeLine writes the line to a writer
func writeLine(w io.Writer, s string) error {
	_, err := io.WriteString(w, s+"\n")
	return err
}
