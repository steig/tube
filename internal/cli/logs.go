package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
)

// logCandidates returns the search paths for a given log type, in priority order.
// First path that exists wins.
func logCandidates(cfg *config.Config, logType string) []string {
	logsDir := cfg.Directories.Logs
	switch logType {
	case "access":
		return []string{
			filepath.Join(logsDir, "access.log"),
			filepath.Join(logsDir, "nginx-access.log"),
			"/var/log/nginx/access.log",
		}
	case "error":
		return []string{
			filepath.Join(logsDir, "error.log"),
			filepath.Join(logsDir, "nginx-error.log"),
			"/var/log/nginx/error.log",
		}
	case "tube":
		return []string{filepath.Join(logsDir, "tube.log")}
	case "tunnel":
		return []string{filepath.Join(logsDir, "tunnel.log")}
	case "health":
		return []string{filepath.Join(logsDir, "health.log")}
	default:
		return nil
	}
}

// NewLogsCmd creates the logs command
func NewLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [type]",
		Short: "View tube logs",
		Long: `View tube logs.

Types: tube (default), access, error, tunnel, health

Examples:
  tube logs           # Show main tube logs
  tube logs access    # Show nginx access logs
  tube logs error     # Show error logs
  tube logs -f        # Follow logs in real-time
  tube logs -n 100    # Show last 100 lines`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			logType := "tube"
			if len(args) > 0 {
				logType = args[0]
			}

			candidates := logCandidates(cfg, logType)
			if candidates == nil {
				return fmt.Errorf("unknown log type %q (try: tube, access, error, tunnel, health)", logType)
			}

			path := firstExisting(candidates)
			if path == "" {
				return fmt.Errorf("no %s log found (searched: %v)", logType, candidates)
			}

			follow, _ := cmd.Flags().GetBool("follow")
			lines, _ := cmd.Flags().GetInt("lines")

			if err := tailFile(cmd.OutOrStdout(), path, lines, follow); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolP("follow", "f", false, "follow logs in real-time")
	cmd.Flags().IntP("lines", "n", 50, "number of lines to show")

	return cmd
}

func firstExisting(paths []string) string {
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// tailFile prints the last `lines` lines of path. If follow is set, it then
// streams new content as the file grows. Detects truncation (file shrinks)
// and re-seeks to the start.
func tailFile(w io.Writer, path string, lines int, follow bool) error {
	if err := printLastLines(w, path, lines); err != nil {
		return err
	}
	if !follow {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek %s: %w", path, err)
	}

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if _, werr := io.WriteString(w, line); werr != nil {
				return werr
			}
			continue
		}
		if err != nil && err != io.EOF {
			return fmt.Errorf("read error: %w", err)
		}

		// EOF — check for truncation, then sleep.
		if fi, statErr := os.Stat(path); statErr == nil {
			pos, _ := f.Seek(0, io.SeekCurrent)
			if fi.Size() < pos {
				// File was truncated; seek back to start.
				if _, err := f.Seek(0, io.SeekStart); err != nil {
					return fmt.Errorf("failed to re-seek truncated file: %w", err)
				}
				reader.Reset(f)
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// printLastLines reads the last N lines of a file and writes them to w.
// Uses a backward block read so it does not buffer the whole file.
func printLastLines(w io.Writer, path string, n int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}
	if fi.Size() == 0 {
		return nil
	}

	const block = 4096
	var (
		size      = fi.Size()
		buf       []byte
		lineCount int
		offset    = size
	)

	for offset > 0 && lineCount <= n {
		read := min(int64(block), offset)
		offset -= read
		chunk := make([]byte, read)
		if _, err := f.ReadAt(chunk, offset); err != nil {
			return fmt.Errorf("read error: %w", err)
		}
		// Count newlines in this chunk.
		for _, b := range chunk {
			if b == '\n' {
				lineCount++
			}
		}
		buf = append(chunk, buf...)
	}

	// Trim leading lines so we emit at most n.
	if lineCount > n {
		extra := lineCount - n
		for i, b := range buf {
			if b == '\n' {
				extra--
				if extra == 0 {
					buf = buf[i+1:]
					break
				}
			}
		}
	}

	_, err = w.Write(buf)
	return err
}
