package merger

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

var filePattern = regexp.MustCompile(`^(\d+)_.*\.sql$`)

type FileEntry struct {
	Path   string
	Prefix int
	Name   string
}

func sortedSQLFiles(dir string) ([]FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("input directory not found: %s", dir)
	}

	var files []FileEntry
	var duplicates []int

	prefixCount := map[int]int{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := filePattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		prefix, _ := strconv.Atoi(m[1])
		prefixCount[prefix]++
		files = append(files, FileEntry{
			Path:   filepath.Join(dir, e.Name()),
			Prefix: prefix,
			Name:   e.Name(),
		})
	}

	for prefix, count := range prefixCount {
		if count > 1 {
			duplicates = append(duplicates, prefix)
		}
	}
	if len(duplicates) > 0 {
		sort.Ints(duplicates)
		for _, p := range duplicates {
			fmt.Fprintf(os.Stderr, "warning: duplicate prefix %d found, sorting by filename\n", p)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Prefix != files[j].Prefix {
			return files[i].Prefix < files[j].Prefix
		}
		return files[i].Name < files[j].Name
	})

	if len(files) == 0 {
		return nil, fmt.Errorf("no SQL files found in %s", dir)
	}

	return files, nil
}
