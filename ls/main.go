// ls lists directory entries.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"
)

var (
	listDirectory = flag.Bool("d", false, "List directories instead of their contents")
	longFormat    = flag.Bool("l", false, "Print each item with a longer format")
	baseName      = flag.Bool("p", false, "Only print the base name of each entry")
	classify      = flag.Bool("F", false, "Print / after directories")
)

type errors []error

func (errs errors) Error() (s string) {
	for _, e := range errs {
		s += e.Error() + "\n"
	}
	return
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [<path> ...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	status := 0
	var items listItems
	for _, path := range paths {
		is, err := getItems(path)
		if err != nil {
			status = 1
			os.Stderr.WriteString(err.Error()+"\n")
		}
		items = append(items, is...)
	}

	sort.Sort(items)
	for _, item := range items {
		var err error
		if *longFormat {
			err = item.printLong()
		} else {
			err = item.print()
		}
		if err != nil {
			status = 1
			os.Stderr.WriteString(err.Error()+"\n")
		}
	}

	os.Exit(status)
}

// getItems returns all of the items to be listed.
func getItems(path string) ([]listItem, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsDir() || *listDirectory {
		return []listItem{{path, info}}, nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	ents, err := dir.Readdirnames(-1)
	if err != nil && err != io.EOF {
		return nil, err
	}

	var items []listItem
	var errs errors
	for _, ent := range ents {
		p := filepath.Join(path, ent)
		if info, err := os.Stat(p); err != nil {
			errs = append(errs, err)
		} else {
			items = append(items, listItem{p, info})
		}
	}
	if errs != nil {
		err = errs
	}
	return items, err
}

// listItem is an item that must be listed.
type listItem struct {
	path string
	info os.FileInfo
}

// listItems is a slice of listItems, implementing
// sort.Interface.
type listItems []listItem

func (l listItems) Len() int {
	return len(l)
}

func (l listItems) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l listItems) Less(i, j int) bool {
	return l[i].path < l[j].path
}

// pathName returns the path name of this item.
func (i listItem) pathName() string {
	p := i.path
	if *baseName {
		p = filepath.Base(i.path)
	}
	if *classify && i.info.Mode().IsDir() {
		p += "/"
	}
	return p
}

// print prints the item.
func (i listItem) print() error {
	_, err := os.Stdout.WriteString(i.pathName()+"\n")
	return err
}

// printLong prints the item in the long format.
func (i listItem) printLong() error {
	uid, gid := -1, -1
	if sys, ok := i.info.Sys().(*syscall.Stat_t); ok {
		uid = int(sys.Uid)
		gid = int(sys.Gid)
	}

	userStr := strconv.Itoa(uid)
	var errs errors
	if u, err := user.LookupId(userStr); err != nil {
		errs = append(errs, err)
	} else {
		userStr = u.Username
	}

	size := i.info.Size()
	time := i.info.ModTime().Format("Jan 2 15:04")
	mode := i.info.Mode().String()
	name := i.pathName()
	if _, err := fmt.Println(mode, userStr, gid, size, time, name); err != nil {
		errs = append(errs, err)
	}

	if errs == nil {
		return nil
	}
	return errs
}
