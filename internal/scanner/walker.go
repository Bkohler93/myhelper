package scanner

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Walk traverses the directory tree rooted at root and returns relative paths
// of all Go source files that should be indexed. It excludes:
//   - directories named .git, vendor, testdata, .myhelper
//   - any .go file whose first 256 bytes contain "// Code generated"
func Walk(root string) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "testdata" || name == ".myhelper" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		// Check first 256 bytes for generated file marker.
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		buf := make([]byte, 256)
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if strings.Contains(string(buf[:n]), "// Code generated") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	})

	return paths, err
}
