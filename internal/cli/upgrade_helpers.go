package cli

import (
	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/update"
)

// currentVsLatest synchronously fetches the latest release tag, updates
// the cache, and returns the comparison against the current build. Used
// by `tube upgrade --check`, where the user is explicitly waiting on an
// answer (in contrast to the ambient post-run notice which is async).
func currentVsLatest(cmd *cobra.Command) (latest, current string, newer bool, err error) {
	current = parseCurrentVersion(cmd.Root().Version)
	r, err := update.Fetch(current, update.DefaultCachePath())
	if err != nil {
		return "", current, false, err
	}
	return r.Latest, current, r.Newer, nil
}

// parseCurrentVersion pulls the bare "0.2.0" out of the formatted version
// string "0.2.0 (commit: ..., date: ...)" set by main.go.
func parseCurrentVersion(v string) string {
	for i, c := range v {
		if c == ' ' {
			return v[:i]
		}
	}
	return v
}
