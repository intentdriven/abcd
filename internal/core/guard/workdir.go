package guard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// A host whose shell tool takes a per-call working directory runs the command
// there instead of in the session directory, so a guard that reads only the
// command string decides a command whose meaning was moved out of the string
// (iss-2609212142557657, the older lab finding iss-2609012040019014). The ruled
// contract (option A, 2026-09-23) is that the guard reads that directory where a
// host supplies one and resolves the command against it.
//
// What "resolves" means was settled by a probe, not by the host's documentation
// (prefer-the-experiment-to-the-inference). opencode 1.18.31 is the one host
// with the field; its bash tool, driven directly with
// `opencode debug agent build --tool bash --params …`, showed:
//
//   - a relative workdir is resolved against the session directory, and an
//     empty one IS the session directory;
//   - a workdir that does not exist fails the call with NotFound, and one that
//     names a regular file fails it with ENOTDIR — in both cases nothing runs,
//     and nothing falls back to the session directory.
//
// The second observation is the one that shapes the registry. The
// rm-rf-after-cd-chain entry exists because a FAILED shell cd leaves the delete
// running wherever the shell already was; a host-managed working directory that
// cannot be entered fails the whole call instead, so that hazard is absent and
// the workdir is never read as a `cd` (folding it into the string as
// `cd <workdir> && ` would block every workdir'd recursive delete for a hazard
// the host does not have). What the workdir DOES change is which repository the
// command runs in, and therefore which repo registry names its hazards: the
// front door checks the command against the session's registry and, when the
// workdir is an existing directory resolving to a different repository, against
// that repository's registry too, with Strictest choosing the answer — the
// workdir's registry can add a hazard and can never take one away.
//
// A host adapter for a host that DOES fall back to the session directory must
// not use the field: it folds the directory into the command as a `cd`, which
// the guard already reads as the failed-cd hazard it then is.

// ErrMalformedWorkdir is a host-supplied working directory that no directory
// could be named by — a NUL byte, a control character, invalid UTF-8, a value
// over MaxWorkdirBytes, or a relative value with no absolute session directory to
// resolve it against. The value is written by the model, so it is untrusted
// input; the front door refuses the call rather than guess where it would run.
var ErrMalformedWorkdir = errors.New("guard: malformed working directory")

// MaxWorkdirBytes caps a host-supplied working directory: PATH_MAX on Linux, and
// above macOS's 1024, so no path a host could chdir into is refused.
const MaxWorkdirBytes = 4096

// Workdir is a host-supplied per-call working directory after validation.
type Workdir struct {
	// Path is absolute and lexically clean. Empty means the host supplied none
	// and the command runs in the session directory.
	Path string
	// Exists reports whether Path named a directory at check time. A workdir that
	// does not is one the probed host refuses to run in, so there is no
	// directory whose registry could apply.
	Exists bool
}

// ResolveWorkdir validates raw, a host-supplied per-call working directory, and
// resolves it against sessionDir the way the probed host does. An empty raw is no
// workdir at all. A refusal wraps ErrMalformedWorkdir and names what it found,
// never echoing a control byte back.
func ResolveWorkdir(sessionDir, raw string) (Workdir, error) {
	if raw == "" {
		return Workdir{}, nil
	}
	if len(raw) > MaxWorkdirBytes {
		return Workdir{}, fmt.Errorf("%w: it is %d bytes, over the %d-byte cap", ErrMalformedWorkdir, len(raw), MaxWorkdirBytes)
	}
	if !utf8.ValidString(raw) {
		return Workdir{}, fmt.Errorf("%w: it is not valid UTF-8", ErrMalformedWorkdir)
	}
	if i := strings.IndexByte(raw, 0); i >= 0 {
		return Workdir{}, fmt.Errorf("%w: it holds a NUL byte at offset %d, which no path can contain", ErrMalformedWorkdir, i)
	}
	for i, r := range raw {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return Workdir{}, fmt.Errorf("%w: it holds a control character (U+%04X) at offset %d", ErrMalformedWorkdir, r, i)
		}
	}
	path := raw
	if !filepath.IsAbs(path) {
		if !filepath.IsAbs(sessionDir) {
			return Workdir{}, fmt.Errorf("%w: it is relative and there is no absolute session directory to resolve it against", ErrMalformedWorkdir)
		}
		path = filepath.Join(sessionDir, path)
	}
	path = filepath.Clean(path)
	fi, err := os.Stat(path)
	return Workdir{Path: path, Exists: err == nil && fi.IsDir()}, nil
}

// severity orders the verdicts for Strictest.
func severity(v Verdict) int {
	switch v {
	case VerdictBlock:
		return 2
	case VerdictWarn:
		return 1
	default:
		return 0
	}
}

// Strictest returns the most severe of the decisions — block over warn over
// allow — keeping the earliest on a tie, so a caller that lists the session's
// decision first keeps its lesson whenever the other registry adds nothing
// stricter. No decisions at all is an allow.
func Strictest(ds ...Decision) Decision {
	best := Decision{Verdict: VerdictAllow}
	for i, d := range ds {
		if i == 0 || severity(d.Verdict) > severity(best.Verdict) {
			best = d
		}
	}
	return best
}
