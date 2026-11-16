package examples

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Find walks the tree under -p (default ".") and prints every entry of type -t
// (file, dir or symlink) whose name contains -n. Only -t is required: an empty
// -n matches every name.
func Find() {
	// A private FlagSet instead of flag.CommandLine - see Cat.
	flags := flag.NewFlagSet("find", flag.ExitOnError)
	path := flags.String("p", ".", "Start path")
	name := flags.String("n", "", "Substring of the name to match (empty matches everything)")
	fileType := flags.String("t", "", "Type to match (file, dir, symlink)")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	switch *fileType {
	case "file", "dir", "symlink":
	case "":
		fmt.Fprintln(os.Stderr, "find: -t is required")
		flags.Usage()
		os.Exit(2)
	default:
		log.Fatalf("unknown type %q, allowed: file, dir, symlink", *fileType)
	}

	// WalkDir rather than Walk: since Go 1.16 it is the recommended form, because
	// it hands the callback an fs.DirEntry instead of an fs.FileInfo and therefore
	// avoids a stat(2) call for every single entry.
	if err := filepath.WalkDir(*path, onElement(*fileType, *name)); err != nil {
		log.Fatalf("error walking %q: %v", *path, err)
	}
}

func onElement(fileType, name string) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			// One unreadable directory must not abort the whole walk - report it
			// and carry on, which is what grep.go does too. Returning err here
			// would stop everything.
			log.Printf("skipping %q: %v", path, err)
			return nil
		}
		// strings.Contains with an empty substring is always true, so an empty
		// -n matches everything.
		if !strings.Contains(entry.Name(), name) {
			return nil
		}
		// Note: the first case used to read `case fileType:`, which compared the
		// variable with itself - it matched ALWAYS, so the "dir" and "symlink"
		// branches were dead code and -t dir behaved like -t file.
		switch fileType {
		case "file":
			if entry.Type().IsRegular() {
				fmt.Println(path)
			}
		case "dir":
			if entry.IsDir() {
				fmt.Println(path)
			}
		case "symlink":
			if entry.Type()&os.ModeSymlink != 0 {
				fmt.Println(path)
			}
		}
		return nil
	}
}
