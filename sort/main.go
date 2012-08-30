package main

import (
	"io"
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"container/heap"
)

const (
	// bufSize is the read and write buffer size.
	bufSize = 8192

	// chunkSize is the number of lines in allowed
	// before going to disk.
	chunkSize = 500000

	// mergeSize is the number of files to merge
	// at once.
	mergeSize = 10
)

func main() {
	status := 0
	stored := make([]string, 0)
	for c := range chunks(allLines(os.Args[1:]), chunkSize) {
		if c.err != nil {
			fmt.Fprintln(os.Stderr, c.err)
			status = 1
			continue
		}

		// Check if we only have a single chunk, if so then
		// just print it directly without going to a file.
		if len(c.chunk) < chunkSize && len(stored) == 0 {
			if err := writeChunk(os.Stdout, c.chunk); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			break
		}

		if s, err := storeChunk(c.chunk); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		} else {
			stored = append(stored, s)
		}
	}

	cleanUp, err := mergeSort(stored, mergeSize)
	for _, tmp := range cleanUp {
		os.Remove(tmp)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		status = 1
	}

	os.Exit(status)
}

// chunkErr is either a chunk of lines or an error.
type chunkErr struct {
	chunk chunk
	err error
}

type chunk []string

func (c chunk) Len() int {
	return len(c)
}

func (c chunk) Less(i, j int) bool {
	return less(c[i], c[j])
}

func (c chunk) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// chunks returns a channel upon which chunks of the file
// are sent.  All chunks have n lines except for the final
// chunk which may have fewer, each chunk is sorted.
func chunks(ls <-chan strErr, n int) <-chan chunkErr {
	ch := make(chan chunkErr)
	go func() {
		c := make(chunk, 0, n)
		for l := range ls {
			if l.err != nil {
				ch <- chunkErr{err: l.err}
				continue
			}
			c = append(c, l.str)
			if len(c) == n {
				sort.Sort(c)
				ch <- chunkErr{chunk: c}
				c = make(chunk, 0, n)
			}
		}
		if len(c) > 0 {
			sort.Sort(c)
			ch <- chunkErr{chunk: c}
		}
		close(ch)
	}()
	return ch
}

// storeChunk stores the chunk to a temporary file,
// returning the temporary file name.
func storeChunk(c chunk) (string, error) {
	f, err := ioutil.TempFile(os.TempDir(), "sort")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := writeChunk(f, c); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func writeChunk(w io.Writer, c chunk) error {
	out := bufio.NewWriterSize(w, bufSize)
	defer out.Flush()

	for _, l := range c {
		if _, err := fmt.Fprintln(out, l); err != nil {
			return err
		}
	}
	return nil
}

type strErr struct {
	str string
	err error
}

// allLines returns a channes on which lines from a
// slice of files are sent.
func allLines(paths []string) <-chan strErr {
	ch := make(chan strErr)
	if len(paths) == 0 {
		paths = []string{"-"}
	}
	go func() {
		for _, p := range paths {
			if p == "-" {	
				in := bufio.NewReaderSize(os.Stdin, bufSize)
				for l := range lines(in) {
					ch <- l
				}
				continue
			}
			f, err := os.Open(p)
			if err != nil {
				ch <- strErr{err: err}
				continue
			}
			for l := range lines(bufio.NewReaderSize(f, bufSize)) {
				ch <- l
			}
			f.Close()
		}
		close(ch)
	}()
	return ch
}

// lines returns a channel on which lines from the a
// reader are sent.
func lines(r *bufio.Reader) <-chan strErr {
	ch := make(chan strErr)
	go func() {
		for {	
			bytes, prefix, err := r.ReadLine()
			if prefix {
				ch <- strErr{err: fmt.Errorf("truncating long line")}
				for prefix && err != nil {
					_, prefix, err = r.ReadLine()
				}
			}
			if err != nil {
				if err != io.EOF {
					ch <- strErr{err: err}
				}
				break
			}
			ch <- strErr{str: string(bytes)}
		}
		close(ch)
	}()
	return ch
}

// mergeSort merges sorted files n at a time.
// The returned slice contains temporary file
// names that may need to be removed (some
// may already be removed).
func mergeSort(paths []string, n int) ([]string, error) {
	for len(paths) > 0 {
		if len(paths) <= n {
			return nil, merge(os.Stdout, paths)
		}

		f, err := ioutil.TempFile(os.TempDir(), "sort")
		if err != nil {
			return paths, err
		}

		err = merge(f, paths[:n])
		f.Close()
		if err != nil {
			paths = append(paths, f.Name())
			return paths, err
		}
		paths = append(paths[n:], f.Name())
	}
	return paths, nil
}

type heapEntry struct {
	lines <-chan strErr
	line string
	file *os.File
}

func newHeapEntry(path string) (*heapEntry, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return &heapEntry{}, false, err
	}
	ent := &heapEntry{
		lines: lines(bufio.NewReaderSize(f, bufSize)),
		file: f,
	}
	ok, err := ent.next()
	return ent, ok, err
}

// next gets the next line for this file.  Returns true
// if there was another line.
func (h *heapEntry) next() (bool, error) {
	l, ok := <- h.lines
	if !ok || l.err != nil {
		os.Remove(h.file.Name())
		h.file.Close()
		return false, l.err
	}
	h.line = l.str
	return true, nil
}

type mergeHeap []*heapEntry

func (m mergeHeap) Len() int {
	return len(m)
}

func (m mergeHeap) Less(i, j int) bool {
	return less(m[i].line, m[j].line)
}

func (m mergeHeap) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m *mergeHeap) Pop() interface{} {
	x := (*m)[len(*m)-1]
	*m = (*m)[:len(*m)-1]
	return x
}

func (m *mergeHeap) Push(x interface{}) {
	*m = append(*m, x.(*heapEntry))
}

// merge merges the given paths to a writer.
// The merged paths are removed.
func merge(w io.Writer, paths []string) error {
	out := bufio.NewWriterSize(w, bufSize)
	defer out.Flush()

	var q mergeHeap
	for _, p := range paths {
		if ent, ok, err := newHeapEntry(p); err != nil {
			return err
		} else if ok {
			heap.Push(&q, ent)
		}
	}

	for len(q) > 0 {
		ent := heap.Pop(&q).(*heapEntry)
		line := ent.line
		if ok, err := ent.next(); err != nil {
			return err
		} else if ok {
			heap.Push(&q, ent)
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}

	return nil
}


func less(a, b string) bool {
	return a < b
}