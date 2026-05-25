package proxy

import (
	"fmt"
	"os"
	"path/filepath"
)

// templateSearchPaths is the list of directories searched for a template
// matching subdir/filename, in priority order: relative to cwd first, then
// system-wide locations a packaged install would use.
var templateSearchPaths = []string{
	"templates",
	"/usr/local/etc/tube",
	"/etc/tube",
}

// readTemplate finds and reads a template file under one of the search paths.
// Returns a useful error listing all paths tried when none matches.
func readTemplate(subdir, filename string) (string, error) {
	var tried []string
	for _, root := range templateSearchPaths {
		path := filepath.Join(root, subdir, filename)
		tried = append(tried, path)
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), nil
		}
		if !os.IsNotExist(err) {
			// Real error (permission, etc) — surface it directly.
			return "", fmt.Errorf("failed to read template %s: %w", path, err)
		}
	}
	return "", fmt.Errorf("template %s/%s not found in any of: %v", subdir, filename, tried)
}
