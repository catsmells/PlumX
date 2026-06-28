package main
import (
	"os"
	"path/filepath"
	"strings"
)
var skipSearchDirs = map[string]bool{
	"/proc": true,
	"/sys":  true,
	"/dev":  true,
	"/run":  true,
}
func searchFiles(root, query string, showHidden bool, cancelled func() bool, onMatch func(fileEntry)) {
	q := strings.ToLower(query)
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if cancelled() {
			return filepath.SkipAll
		}
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path != root && skipSearchDirs[path] {
			return filepath.SkipDir
		}
		if path != root && !showHidden && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), q) {
			onMatch(fileEntryFor(path, d))
		}
		return nil
	})
}
