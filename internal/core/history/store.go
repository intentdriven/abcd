package history

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// rootSHARe is the immutable repo key: a lowercase hex commit SHA, 40 chars for
// git's SHA-1 object format or 64 for SHA-256. Accepting only 40 made every
// history verb (Capture/List/Read) fail for a SHA-256 repo, whose root SHA the
// ahoy layer derives at 64 chars — mirrors the voyage-ledger key fix.
var rootSHARe = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// rootSHAErrMsg is the diagnostic for a value that fails rootSHARe; it names both
// accepted widths so it cannot drift from the regex (which accepts SHA-256's 64
// as well as SHA-1's 40).
const rootSHAErrMsg = "history: rootSHA must be a 40- or 64-character lowercase hex commit SHA"

// sessionIDRe restricts a vendor session id to filesystem-safe characters so it
// can be embedded verbatim in a record filename with no path-traversal or
// separator surprises.
var sessionIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// validKinds are the accepted source_kind values.
var validKinds = map[string]struct{}{
	"native":           {},
	"specstory-import": {},
}

// maxTranscriptBytes caps a single guarded record read from the store. It
// matches the transcript-capture cap on the write side; a record grown past it
// out of band is refused rather than read wholly into memory.
const maxTranscriptBytes = 64 << 20 // 64 MiB

// repoLock takes a per-<rootSHA> advisory lock on records/.lock, disjoint
// from ahoy's index lock. The lock file is opened O_NOFOLLOW mode 0o600 so a
// pre-planted lock-file symlink is refused. The returned release closes the fd
// (which drops the flock). Ports the two-domain lock model from
// history_store.py.
func repoLock(tdir string) (func(), error) {
	lockPath := filepath.Join(tdir, ".lock")
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, &StorePathError{Path: lockPath, Msg: "lock file open refused (symlinked or unwritable): " + err.Error()}
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("history: acquire lock %s: %w", lockPath, err)
	}
	return func() { f.Close() }, nil
}

// recordFilename is <compact-utc>-<session-id>.md — sorts chronologically and,
// with nanosecond precision, does not collide within a session.
func recordFilename(capturedAt time.Time, sessionID string) string {
	return capturedAt.UTC().Format("20060102T150405.000000000Z") + "-" + sessionID + ".md"
}

// frontmatter fields (flat, one scalar per line) — a small fixed schema parsed
// by a line reader, so no YAML dependency is pulled in.
const (
	fmSchema      = "schema"
	fmSessionID   = "session_id"
	fmRootCommit  = "root_commit"
	fmCapturedAt  = "captured_at"
	fmSourceKind  = "source_kind"
	fmSourceSHA   = "source_sha256"
	fmRedSecrets  = "redacted_secrets"
	fmRedHomePath = "redacted_home_paths"
)

// marshalRecord renders a record file: YAML frontmatter then the redacted body.
func marshalRecord(r Record, body string) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "%s: %d\n", fmSchema, recordSchemaVersion)
	fmt.Fprintf(&b, "%s: %s\n", fmSessionID, r.SessionID)
	fmt.Fprintf(&b, "%s: %s\n", fmRootCommit, r.RootCommit)
	fmt.Fprintf(&b, "%s: %s\n", fmCapturedAt, r.CapturedAt.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&b, "%s: %s\n", fmSourceKind, r.SourceKind)
	fmt.Fprintf(&b, "%s: %s\n", fmSourceSHA, r.SourceSHA256)
	fmt.Fprintf(&b, "%s: %d\n", fmRedSecrets, r.Secrets)
	fmt.Fprintf(&b, "%s: %d\n", fmRedHomePath, r.HomePaths)
	b.WriteString("---\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// parseRecord splits a record file into its metadata and redacted body. The
// Path field is set by the caller. Returns an error when the frontmatter fence
// is missing or a required field is malformed.
func parseRecord(data []byte) (Record, string, error) {
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return Record{}, "", fmt.Errorf("history: record missing frontmatter fence")
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return Record{}, "", fmt.Errorf("history: record frontmatter not terminated")
	}
	head := rest[:end]
	body := rest[end+len("\n---\n"):]

	fields := map[string]string{}
	for _, line := range strings.Split(head, "\n") {
		if line == "" {
			continue
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			continue
		}
		fields[line[:i]] = line[i+2:]
	}

	var r Record
	r.SessionID = fields[fmSessionID]
	r.RootCommit = fields[fmRootCommit]
	r.SourceKind = fields[fmSourceKind]
	r.SourceSHA256 = fields[fmSourceSHA]
	if r.SessionID == "" || r.RootCommit == "" || r.SourceSHA256 == "" {
		return Record{}, "", fmt.Errorf("history: record frontmatter missing a required field")
	}
	if ts := fields[fmCapturedAt]; ts != "" {
		t, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return Record{}, "", fmt.Errorf("history: record captured_at unparseable: %w", err)
		}
		r.CapturedAt = t.UTC()
	}
	r.Secrets, _ = strconv.Atoi(fields[fmRedSecrets])
	r.HomePaths, _ = strconv.Atoi(fields[fmRedHomePath])
	return r, body, nil
}

// listRecords reads every *.md record under tdir, newest first. A record file
// that fails to parse is skipped (an individual corrupt transcript is not
// fatal to the rest), mirroring ScanBundle's per-file tolerance.
func listRecords(tdir string) ([]Record, error) {
	entries, err := os.ReadDir(tdir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p := filepath.Join(tdir, e.Name())
		// Guarded read: the transcripts dir is a cross-repo store under HOME,
		// so a *.md symlink or FIFO planted by anything running as the caller
		// must not be followed or block the listing (iss-383). A refused leaf
		// is skipped like an unparseable one.
		data, err := fsutil.ReadGuarded(p, maxTranscriptBytes)
		if err != nil {
			continue
		}
		r, _, err := parseRecord(data)
		if err != nil {
			continue
		}
		r.Path = p
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].CapturedAt.Equal(out[j].CapturedAt) {
			return out[i].CapturedAt.After(out[j].CapturedAt)
		}
		return out[i].Path > out[j].Path
	})
	return out, nil
}

// StorePathError is a preflight fault: an owned store path is absent, a symlink,
// or otherwise unsafe. It is returned (never panicked) so the caller can surface
// a clean diagnostic and refuse the operation.
type StorePathError struct {
	Path string
	Msg  string
}

func (e *StorePathError) Error() string { return "history: " + e.Msg + " (" + e.Path + ")" }
