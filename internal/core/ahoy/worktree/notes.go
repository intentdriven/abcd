package worktree

// notes.go is the notes archive beside the worktree store
// (spc-2609301811532881): before prune removes a worktree, its local tier
// (.abcd/.work.local/, the handover notes, logs and scratch git ignores and
// `git worktree remove` deletes with the tree) is copied, redacted, to
// ~/.abcd.noindex/notes/<root-sha>/<YYYYMMDDTHHMMSSZ>-<name>/ beside a manifest.
//
// Every file and manifest field goes through history.Redact, the pass the
// transcript store runs, so there is one redaction path. The tier is read
// whole and redacted before anything is written: a file the pass cannot vouch
// for (not text, over the cap, a residual it cannot mask) fails the archive
// with nothing written, and a failed archive keeps the worktree.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// NotesRel is the notes archive, relative to the caller's home.
var NotesRel = abcdhome.Rel("notes")

// tierRel is a checkout's local tier, the part of a worktree the archive keeps.
const tierRel = ".abcd/.work.local"

// manifestName is the manifest's name at the top of an archive entry.
const manifestName = "manifest.json"

// The archive's bounds: a tier past them is not notes, and fails the archive
// rather than being copied in part.
const (
	maxNoteBytes  = 4 << 20
	maxNotesTotal = 64 << 20
	maxNoteFiles  = 4096
)

// note is one entry of a tier, redacted: a file's text, or a link's target.
type note struct {
	rel    string // slash-separated, relative to the tier
	data   []byte
	link   string
	isLink bool
}

// manifestFile is one entry of the manifest's files.
type manifestFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
	// Link is a symlink's target, home-redacted; the link is never followed.
	Link string `json:"link,omitempty"`
}

// manifest is an archive entry's manifest.json.
type manifest struct {
	SchemaVersion int            `json:"schema_version"`
	Worktree      string         `json:"worktree"`
	Branch        string         `json:"branch"`
	Source        string         `json:"source"`
	ArchivedAt    string         `json:"archived_at"`
	RootSHA       string         `json:"root_sha"`
	Files         []manifestFile `json:"files"`
}

// collectNotes reads the worktree at wt's local tier without following a
// symlink, and redacts each file through repoRoot's scanner. An absent tier,
// or one holding nothing, is no notes. Any file it cannot archive is an error
// naming the file's path in the tier.
func collectNotes(repoRoot, wt string) ([]note, error) {
	tier := wt
	for _, seg := range strings.Split(tierRel, "/") {
		tier = filepath.Join(tier, seg)
		fi, err := os.Lstat(tier)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return nil, nil
		case err != nil:
			return nil, err
		case fi.Mode()&fs.ModeSymlink != 0:
			return nil, fmt.Errorf("%s is a symlink, which the archive never follows", strings.TrimPrefix(tier, wt+string(filepath.Separator)))
		case !fi.IsDir():
			return nil, nil
		}
	}
	var notes []note
	total := 0
	err := filepath.WalkDir(tier, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == tier || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(tier, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if len(notes) == maxNoteFiles {
			return fmt.Errorf("the tier holds more than %d files", maxNoteFiles)
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			red, err := history.Redact(repoRoot, fsutil.RedactHome(target), rel)
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			notes = append(notes, note{rel: rel, link: red.Text, isLink: true})
		case !d.Type().IsRegular():
			return fmt.Errorf("%s: not a regular file", rel)
		default:
			data, err := fsutil.ReadGuarded(p, maxNoteBytes)
			switch {
			case errors.Is(err, fsutil.ErrTooBig):
				return fmt.Errorf("%s: over the archive's per-file cap of %d bytes", rel, maxNoteBytes)
			case err != nil:
				return fmt.Errorf("%s: %w", rel, err)
			case !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0:
				return fmt.Errorf("%s: not text, which the redaction pass cannot vouch for", rel)
			}
			red, err := history.Redact(repoRoot, string(data), rel)
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			if total += len(red.Text); total > maxNotesTotal {
				return fmt.Errorf("the tier holds more than the archive's cap of %d bytes", maxNotesTotal)
			}
			notes = append(notes, note{rel: rel, data: []byte(red.Text)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return notes, namesSurvive(repoRoot, notes)
}

// namesSurvive refuses a tier whose file names the redaction pass would
// change: a name is written as a path, and a masked one is not the file's.
func namesSurvive(repoRoot string, notes []note) error {
	if len(notes) == 0 {
		return nil
	}
	rels := make([]string, len(notes))
	for i, n := range notes {
		rels[i] = n.rel
	}
	red, err := history.Redact(repoRoot, strings.Join(rels, "\n"), "notes file names")
	if err != nil {
		return fmt.Errorf("the tier's file names: %w", err)
	}
	for i, got := range strings.Split(red.Text, "\n") {
		if i >= len(rels) || got != rels[i] {
			return errors.New("a file name holds what the redaction pass masks")
		}
	}
	return nil
}

// writeNotes writes notes into a new entry of the archive for the lane, and
// the manifest beside them. The entry is made exclusively, so two runs in one
// second never share one, and each file is written inside an os.Root on it.
// The entry's path is returned once it is made, a failure after that
// included.
func writeNotes(repoRoot string, lane Lane, e entry, notes []note, now time.Time) (string, error) {
	if !ValidName(e.name) {
		return "", errors.New("the worktree's name is not one plain path segment to name the entry with")
	}
	for _, n := range notes {
		if n.rel == manifestName {
			return "", fmt.Errorf("%s: the tier's own file takes the name the archive's manifest needs", n.rel)
		}
	}
	at := now.UTC()
	m := manifest{SchemaVersion: 1, ArchivedAt: at.Format(time.RFC3339), RootSHA: lane.RootSHA, Files: []manifestFile{}}
	fields := []string{e.name, shortBranch(e.wt), fsutil.RedactHome(e.path)}
	for _, f := range fields {
		if strings.ContainsAny(f, "\r\n") {
			return "", errors.New("the worktree's branch or path holds a line break")
		}
	}
	red, err := history.Redact(repoRoot, strings.Join(fields, "\n"), "notes manifest")
	if err != nil {
		return "", fmt.Errorf("the manifest: %w", err)
	}
	got := strings.Split(red.Text, "\n")
	if len(got) != len(fields) {
		return "", errors.New("the manifest's fields did not survive redaction intact")
	}
	m.Worktree, m.Branch, m.Source = got[0], got[1], got[2]

	rel := NotesRel + "/" + lane.RootSHA
	if _, err := levels(lane.Home, rel, "notes archive", true); err != nil {
		return "", err
	}
	dir := filepath.Join(lane.Home, filepath.FromSlash(rel), at.Format("20060102T150405Z")+"-"+e.name)
	if err := os.Mkdir(dir, dirPerm); err != nil {
		return "", fmt.Errorf("making the archive entry %s: %w", fsutil.RedactHome(dir), err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return dir, fmt.Errorf("opening the archive entry: %w", err)
	}
	defer root.Close()
	for _, n := range notes {
		if n.isLink {
			m.Files = append(m.Files, manifestFile{Path: n.rel, Link: n.link})
			continue
		}
		if d := path.Dir(n.rel); d != "." {
			if err := root.MkdirAll(d, dirPerm); err != nil {
				return dir, fmt.Errorf("%s: %w", n.rel, err)
			}
		}
		if err := writeNew(root, n.rel, n.data); err != nil {
			return dir, fmt.Errorf("%s: %w", n.rel, err)
		}
		sum := sha256.Sum256(n.data)
		m.Files = append(m.Files, manifestFile{Path: n.rel, Size: int64(len(n.data)), SHA256: hex.EncodeToString(sum[:])})
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return dir, err
	}
	if err := writeNew(root, manifestName, append(data, '\n')); err != nil {
		return dir, fmt.Errorf("%s: %w", manifestName, err)
	}
	return dir, nil
}

// writeNew writes data to a file rel inside root that must not exist yet,
// the caller's alone.
func writeNew(root *os.Root, rel string, data []byte) error {
	f, err := root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, abcdhome.FileMode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
