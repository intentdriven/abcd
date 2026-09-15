package gitleaks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// execRunner is the production Runner: it writes the text to a private temp
// directory and runs `gitleaks dir` over it with the default ruleset, reading
// back the JSON report. This mirrors the invocation iss-96's measurement used
// (gitleaks 8.24.3, `gitleaks dir`, default rules). It is the ONE place that
// spawns a process; every other path through this package is pure.
type execRunner struct{}

// Run scans text and returns gitleaks' raw JSON report bytes. gitleaks exits
// non-zero when it finds a leak, so --exit-code 0 makes a found leak a success
// and reserves a non-zero exit for a genuine tool failure. The report is read
// from a file rather than stdout so banner/log noise cannot corrupt the JSON.
func (execRunner) Run(ctx context.Context, binPath, text string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "abcd-gitleaks-")
	if err != nil {
		return nil, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "transcript.txt")
	if err := os.WriteFile(src, []byte(text), 0o600); err != nil {
		return nil, fmt.Errorf("write scan input: %w", err)
	}
	report := filepath.Join(dir, "report.json")

	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	// #nosec G204 -- binPath originates in a COMMITTED config (or a PATH
	// lookup) and is therefore untrusted input; it reaches this call only after
	// admitBinary has held it to the allow-shape (absolute, resolved outside
	// the repository, regular, executable), which is what makes the variable
	// argument safe to exec. The remaining arguments are fixed flags and paths
	// under a temp dir this process just created; the transcript content is
	// never on the command line.
	cmd := exec.CommandContext(ctx, binPath,
		"dir", dir,
		"--report-format", "json",
		"--report-path", report,
		"--no-banner",
		"--exit-code", "0",
	)
	// Run from the private temp dir, never from the caller's working directory:
	// the capture usually runs inside the checkout being scanned, and a binary
	// that consults its cwd (a config it looks for at ./, a tool shim) would be
	// reading repository content again by the back door the path rule closed.
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("gitleaks run: %w (%s)", err, string(out))
	}

	data, err := fsutil.ReadGuarded(report, maxReportBytes)
	if err != nil {
		if os.IsNotExist(err) {
			// gitleaks writes no report file when it finds nothing; treat as empty.
			return nil, nil
		}
		return nil, fmt.Errorf("read report: %w", err)
	}
	return data, nil
}

// maxReportBytes caps the report we read back (trust boundary on a file gitleaks
// wrote into a temp dir we own — a bound, not an expected size).
const maxReportBytes = 8 * 1024 * 1024
