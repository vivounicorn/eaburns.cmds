package main

import (
	"bufio"
	"container/heap"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
)

const (
	chunkSize = 500000
	mergeSize = 10
)

func main() {
	errs := make(chan error)
	go mergeSort(os.Args[1:], errs)

	status := 0
	for err := range errs {
		fmt.Fprintln(os.Stderr, err)
		status = 1
	}
	os.Exit(status)
}

func mergeSort(paths []string, errs chan<- error) {
	lines := readAllLines(os.Args[1:], errs)
	var tmps []string
	for c := range chunks(lines, chunkSize) {
		if len(tmps) == 0 && len(c) < chunkSize {
			out := bufio.NewWriter(os.Stdout)
			defer out.Flush()
			for _, l := range c {
				fmt.Fprintln(out, l.str)
			}
			goto out
		}
		if tmp, err := writeChunk(c); err != nil {
			errs <- err
			goto out
		} else {
			tmps = append(tmps, tmp)
		}
	}

	for len(tmps) > 0 {
		if len(tmps) < mergeSize {
			if err := merge(os.Stdout, tmps); err != nil {
				errs <- err
			}
			break
		}

		f, err := ioutil.TempFile(os.TempDir(), "sort")
		if err != nil {
			errs <- err
			break
		}
		err = merge(f, tmps[:mergeSize])
		tmps = append(tmps[mergeSize:], f.Name())
		if err != nil {
			errs <- err
			break
		}
		f.Close()
	}

out:
	for _, t := range tmps {
		os.Remove(t)
	}
	close(errs)
}

func writeChunk(c chunk) (string, error) {
	f, err := ioutil.TempFile(os.TempDir(), "sort")
	if err != nil {
		return "", err
	}
	defer f.Close()

	out := bufio.NewWriter(f)
	defer out.Flush()

	for _, l := range c {
		if _, err := fmt.Fprintln(out, l.str); err != nil {
			os.Remove(f.Name())
			return "", err
		}
	}
	return f.Name(), nil
}

type chunkFile struct {
	file *os.File
	in   *bufio.Reader
	cur  line
}

func newChunkFile(p string) (*chunkFile, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	in := bufio.NewReader(f)
	// prefix cannot be true since we wrote these
	// lines and we didn't write one that is too long.
	bytes, _, err := in.ReadLine()
	if err != nil {
		os.Remove(f.Name())
		f.Close()
		return nil, err
	}
	return &chunkFile{file: f, in: in, cur: makeLine(string(bytes))}, nil
}

func (c *chunkFile) nextLine() error {
	bytes, _, err := c.in.ReadLine()
	if err != nil {
		os.Remove(c.file.Name())
		c.file.Close()
		return err
	}
	c.cur = makeLine(string(bytes))
	return nil
}

type chunkHeap []*chunkFile

func (h chunkHeap) Len() int {
	return len(h)
}

func (h chunkHeap) Less(i, j int) bool {
	return h[i].cur.less(h[j].cur)
}

func (h chunkHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *chunkHeap) Push(x interface{}) {
	*h = append(*h, x.(*chunkFile))
}

func (h *chunkHeap) Pop() interface{} {
	x := (*h)[len(*h)-1]
	*h = (*h)[:len(*h)-1]
	return x
}

func merge(w io.Writer, paths []string) error {
	var q chunkHeap
	for _, p := range paths {
		if c, err := newChunkFile(p); err != nil {
			return err
		} else {
			heap.Push(&q, c)
		}
	}

	out := bufio.NewWriter(w)
	defer out.Flush()
	for len(q) > 0 {
		c := heap.Pop(&q).(*chunkFile)
		if _, err := fmt.Fprintln(out, c.cur.str); err != nil {
			return err
		}
		if err := c.nextLine(); err == nil {
			heap.Push(&q, c)
		} else if err != io.EOF {
			return err
		}
	}
	return nil
}

type line struct {
	num int
	str string
}

func makeLine(s string) line {
	return line{str: s}
}

func (a line) less(b line) bool {
	return a.str < b.str
}

type chunk []line

func (c chunk) Len() int {
	return len(c)
}

func (c chunk) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c chunk) Less(i, j int) bool {
	return c[i].less(c[j])
}

func chunks(lines <-chan string, sz int) <-chan chunk {
	ch := make(chan chunk)
	go func(ch chan<- chunk) {
		c := make(chunk, 0, sz)
		for l := range lines {
			c = append(c, makeLine(l))
			if len(c) == sz {
				sort.Sort(c)
				ch <- c
				c = make(chunk, 0, sz)
			}
		}
		if len(c) > 0 {
			sort.Sort(c)
			ch <- c
		}
		close(ch)
	}(ch)
	return ch
}

func readAllLines(paths []string, errs chan<- error) <-chan string {
	ch := make(chan string)
	go func(lines chan<- string) {
		if len(paths) == 0 {
			paths = []string{"-"}
		}
		for _, p := range paths {
			readLines(p, ch, errs)
		}
		close(ch)
	}(ch)
	return ch
}

func readLines(path string, lines chan<- string, errs chan<- error) {
	var r io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			errs <- err
			return
		}
		defer f.Close()
		r = f
	}

	in := bufio.NewReader(r)
	for {
		bytes, prefix, err := in.ReadLine()
		if prefix {
			errs <- fmt.Errorf("%s: truncating long line", path)
			for prefix && err != nil {
				_, prefix, err = in.ReadLine()
			}
		}
		if err != nil {
			if err != io.EOF {
				errs <- err
			}
			break
		}
		lines <- string(bytes)
	}
}
