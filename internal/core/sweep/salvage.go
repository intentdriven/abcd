package sweep

// salvage.go is patch salvage (spc-2610230730269295, Patch salvage): a dirty
// worktree nothing holds has its unstaged and staged changes, each as `git
// diff --binary`, and a tar of its untracked, non-ignored files saved to
// ~/.abcd.noindex/notes/<root-sha>/<UTC>-<name>-salvage/ beside a manifest.
// The worktree is kept: a salvage is a copy, never a reclaim.
//
// The salvage is collected whole, and redacted, while the run judges, before
// anything is written: so a dry run says exactly what apply would do, and a
// worktree whose changes cannot be saved (a file the redaction pass cannot
// vouch for, a cap passed) is kept with that reason and nothing written.
// Every file and manifest field goes through history.Redact, the one pass the
// notes archive and the transcript store run, and the archive's directory is
// made by the store's own maker (Lane.NotesDir). A second run whose salvage is
// identical, by sha256, to one already saved writes nothing and points at it.
//
// The embargo store and a worktree outside the store are never salvaged: the
// dry run keeps both before a salvage is considered, and collectSalvage
// refuses either again.

import (
	"archive/tar"
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
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/abcdhome"
	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The names inside a salvage entry, and the entry's own shape.
const (
	salvageUnstaged  = "unstaged.patch"
	salvageStaged    = "staged.patch"
	salvageUntracked = "untracked.tar"
	salvageManifest  = "manifest.json"
	// salvageKind marks a salvage's manifest apart from a prune's notes
	// manifest, which shares the archive and could share an entry's name
	// shape (a worktree called x-salvage archived by prune).
	salvageKind = "salvage"
	// salvageStamp is the entry's UTC time, the notes archive's own format.
	salvageStamp  = "20060102T150405Z"
	salvageSuffix = "-salvage"
)

// The salvage's bounds, the notes archive's: changes past them are not saved
// in part, and the worktree is kept with the reason.
const (
	maxSalvageFileBytes = 4 << 20
	maxSalvageBytes     = 64 << 20
	maxSalvageFiles     = 4096
	maxManifestBytes    = 1 << 20
)

// salvage is one worktree's changes, collected and redacted, ready to write.
type salvage struct {
	// name is the worktree's name, which names the entry.
	name     string
	files    []salvageFile
	manifest salvageManifestFile
}

// salvageFile is one file of a salvage entry, its data redacted.
type salvageFile struct {
	name string
	data []byte
}

// salvageManifestFile is a salvage entry's manifest.json.
type salvageManifestFile struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Worktree      string `json:"worktree"`
	Branch        string `json:"branch"`
	Head          string `json:"head"`
	// Source is the worktree's path, home-redacted.
	Source  string `json:"source"`
	SavedAt string `json:"saved_at"`
	RootSHA string `json:"root_sha"`
	// SHA256 is the salvage's identity: the sha256 of each file's name and
	// sha256, so a second run with the same changes finds this entry.
	SHA256 string             `json:"sha256"`
	Files  []salvageEntryFile `json:"files"`
	// Untracked is every path the tar holds, as git listed it.
	Untracked []string `json:"untracked"`
}

// salvageEntryFile is one entry of the manifest's files.
type salvageEntryFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// The reasons on a dirty worktree nothing holds. Each keeps the worktree; the
// salvage, when one is made, is a copy of its changes.
const (
	dirtyToSave     = "dirty, and nothing holds it: its changes are saved as a patch to the notes archive, and the worktree kept"
	dirtySavedAt    = "dirty, and nothing holds it: its changes were saved as a patch to "
	dirtySavedEarly = "dirty, and nothing holds it: kept; its changes are already saved, unchanged, at "
	dirtyUnsaved    = "dirty, and nothing holds it: kept, and its changes cannot be saved: "
)

// planSalvage judges the save-patch row for the store worktree src, of the
// repository at repoRoot whose lane is lane, collecting its salvage and
// writing nothing. ok is whether apply has a salvage to write: a salvage that
// cannot be collected, an archive that cannot be read, and changes saved,
// unchanged, by an earlier run each turn the row into a keep, naming why or
// where.
func (j judge) planSalvage(repoRoot string, lane store.Lane, src store.Row, row *Row) (s salvage, ok bool) {
	s, err := collectSalvage(repoRoot, j.home, lane, src)
	if err == nil {
		var earlier string
		if earlier, err = earlierSalvage(lane, s); err == nil && earlier != "" {
			row.Action, row.Reason = ActionKeep, clean(dirtySavedEarly+earlier)
			return salvage{}, false
		}
	}
	if err != nil {
		row.Action, row.Reason = ActionKeep, clean(dirtyUnsaved+err.Error())
		return salvage{}, false
	}
	return s, true
}

// save writes the salvage planned for row, the result's i-th, marking row
// with the outcome. A failed write keeps the worktree with its changes
// unsaved, and is returned as the run's fault.
func (p plan) save(i int, row *Row) error {
	s, ok := p.salvages[i]
	if !ok {
		row.Reason = "no salvage was collected for it, so the worktree is kept with its changes unsaved"
		return errors.New("sweep: no salvage was collected for " + row.Name)
	}
	entry, err := writeSalvage(p.lane, s, p.j.probe.now)
	if err != nil {
		why := err.Error()
		if entry != "" {
			why += "; the entry it left part-way, with no manifest, is " + entry
		}
		row.Reason = clean("the salvage failed, so the worktree is kept with its changes unsaved: " + why)
		return errors.New(clean("sweep: the salvage of " + row.Name + " failed: " + why))
	}
	row.Outcome = OutcomeSaved
	row.Reason = clean(dirtySavedAt + entry + "; the worktree is kept")
	return nil
}

// collectSalvage reads the changes of the store worktree r, for the
// repository at repoRoot whose lane is lane, and redacts them, writing
// nothing. An error says why the changes cannot be saved, already free of an
// unredacted name.
func collectSalvage(repoRoot, home string, lane store.Lane, r store.Row) (salvage, error) {
	abs := r.Abs
	if r.Where != store.WhereStore || storepath.IsStorePath(home, abs) {
		return salvage{}, errors.New("only a worktree in the store is salvaged, and never the embargo store")
	}
	if !store.ValidName(r.Name) {
		return salvage{}, errors.New("the worktree's name is not one plain path segment to name the salvage with")
	}
	head, err := gitutil.RunCapped(abs, 4096, "rev-parse", "--verify", "-q", "HEAD")
	if err != nil {
		return salvage{}, fmt.Errorf("reading its HEAD: %w", err)
	}
	diff := []string{"diff", "--binary", "--no-ext-diff", "--no-textconv", "--no-color"}
	unstaged, err := redactedDiff(repoRoot, abs, "unstaged changes", diff...)
	if err != nil {
		return salvage{}, err
	}
	staged, err := redactedDiff(repoRoot, abs, "staged changes", append(slices.Clone(diff), "--cached")...)
	if err != nil {
		return salvage{}, err
	}
	if len(unstaged)+len(staged) > maxSalvageBytes {
		return salvage{}, fmt.Errorf("its changes pass the salvage's cap of %d bytes", maxSalvageBytes)
	}
	untracked, tarball, err := untrackedTar(repoRoot, abs, maxSalvageBytes-len(unstaged)-len(staged))
	if err != nil {
		return salvage{}, err
	}
	if len(unstaged) == 0 && len(staged) == 0 && len(untracked) == 0 {
		return salvage{}, errors.New("git shows no uncommitted change to save")
	}

	fields := []string{r.Name, r.Branch, head, fsutil.RedactHome(abs)}
	for _, f := range fields {
		if strings.ContainsAny(f, "\r\n") {
			return salvage{}, errors.New("the worktree's name, branch or path holds a line break")
		}
	}
	red, err := history.Redact(repoRoot, strings.Join(fields, "\n"), "salvage manifest")
	if err != nil {
		return salvage{}, fmt.Errorf("the manifest: %w", err)
	}
	got := strings.Split(red.Text, "\n")
	if len(got) != len(fields) {
		return salvage{}, errors.New("the manifest's fields did not survive redaction intact")
	}
	s := salvage{
		name: r.Name,
		files: []salvageFile{
			{name: salvageUnstaged, data: unstaged},
			{name: salvageStaged, data: staged},
			{name: salvageUntracked, data: tarball},
		},
		manifest: salvageManifestFile{
			SchemaVersion: 1, Kind: salvageKind, RootSHA: lane.RootSHA,
			Worktree: got[0], Branch: got[1], Head: got[2], Source: got[3],
			Untracked: untracked,
		},
	}
	id := sha256.New()
	for _, f := range s.files {
		sum := sha256.Sum256(f.data)
		hexSum := hex.EncodeToString(sum[:])
		s.manifest.Files = append(s.manifest.Files, salvageEntryFile{Path: f.name, Size: int64(len(f.data)), SHA256: hexSum})
		fmt.Fprintf(id, "%s %s\n", f.name, hexSum)
	}
	s.manifest.SHA256 = hex.EncodeToString(id.Sum(nil))
	return s, nil
}

// redactedDiff is git's diff in the worktree at abs, redacted. A diff the
// redaction pass cannot vouch for (not text: a change to a file in another
// encoding) cannot be saved.
func redactedDiff(repoRoot, abs, what string, args ...string) ([]byte, error) {
	out, err := gitutil.RunCappedBytes(abs, maxSalvageBytes, args...)
	if err != nil {
		return nil, fmt.Errorf("reading its %s: %w", what, err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	if !isText(out) {
		return nil, fmt.Errorf("its %s are not text, which the redaction pass cannot vouch for", what)
	}
	red, err := history.Redact(repoRoot, string(out), what)
	if err != nil {
		return nil, fmt.Errorf("its %s: %w", what, err)
	}
	return []byte(red.Text), nil
}

// untrackedTar is the worktree at abs's untracked, non-ignored files as git
// lists them, and a tar of them, each redacted, within budget bytes. Every
// file is read inside an os.Root on the worktree, a link's target recorded and
// never followed, so nothing outside the worktree is read. The tar is the same
// bytes for the same files: sorted, and with no time, owner or name of the
// account in a header.
func untrackedTar(repoRoot, abs string, budget int) ([]string, []byte, error) {
	out, err := gitutil.RunCappedBytes(abs, maxSalvageBytes, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		return nil, nil, fmt.Errorf("listing its untracked files: %w", err)
	}
	var rels []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			rels = append(rels, p)
		}
	}
	if len(rels) == 0 {
		return nil, nil, nil
	}
	if len(rels) > maxSalvageFiles {
		return nil, nil, fmt.Errorf("it holds more than %d untracked files", maxSalvageFiles)
	}
	slices.Sort(rels)
	if err := salvageNamesSurvive(repoRoot, rels); err != nil {
		return nil, nil, err
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, nil, fmt.Errorf("opening the worktree: %v", causeOf(err))
	}
	defer root.Close()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, rel := range rels {
		if strings.HasSuffix(rel, "/") {
			return nil, nil, salvageNamed(repoRoot, rel, "a repository nested in the worktree, which the salvage cannot save")
		}
		fi, err := root.Lstat(rel)
		if err != nil {
			return nil, nil, salvageNamed(repoRoot, rel, causeOf(err))
		}
		hdr := &tar.Header{Name: rel, Mode: int64(fi.Mode().Perm()), ModTime: time.Unix(0, 0), Format: tar.FormatPAX}
		var data []byte
		switch {
		case fi.Mode()&fs.ModeSymlink != 0:
			target, err := root.Readlink(rel)
			if err != nil {
				return nil, nil, salvageNamed(repoRoot, rel, causeOf(err))
			}
			red, err := history.Redact(repoRoot, fsutil.RedactHome(target), rel)
			if err != nil {
				return nil, nil, salvageNamed(repoRoot, rel, err.Error())
			}
			hdr.Typeflag, hdr.Linkname = tar.TypeSymlink, red.Text
		case !fi.Mode().IsRegular():
			return nil, nil, salvageNamed(repoRoot, rel, "not a regular file")
		default:
			raw, err := fsutil.ReadGuardedInRoot(root, rel, maxSalvageFileBytes)
			switch {
			case errors.Is(err, fsutil.ErrTooBig):
				return nil, nil, salvageNamed(repoRoot, rel, fmt.Sprintf("over the salvage's per-file cap of %d bytes", maxSalvageFileBytes))
			case err != nil:
				return nil, nil, salvageNamed(repoRoot, rel, causeOf(err))
			case !isText(raw):
				return nil, nil, salvageNamed(repoRoot, rel, "not text, which the redaction pass cannot vouch for")
			}
			red, err := history.Redact(repoRoot, string(raw), rel)
			if err != nil {
				return nil, nil, salvageNamed(repoRoot, rel, err.Error())
			}
			data = []byte(red.Text)
			hdr.Typeflag, hdr.Size = tar.TypeReg, int64(len(data))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, nil, salvageNamed(repoRoot, rel, err.Error())
		}
		if _, err := tw.Write(data); err != nil {
			return nil, nil, salvageNamed(repoRoot, rel, err.Error())
		}
		if buf.Len() > budget {
			return nil, nil, fmt.Errorf("its changes pass the salvage's cap of %d bytes", maxSalvageBytes)
		}
	}
	if err := tw.Close(); err != nil {
		return nil, nil, err
	}
	return rels, buf.Bytes(), nil
}

// salvageNamesSurvive refuses untracked names the redaction pass would
// change: a name is written as a path in the tar, and a masked one is not the
// file's.
func salvageNamesSurvive(repoRoot string, rels []string) error {
	for _, r := range rels {
		if strings.ContainsAny(r, "\r\n") {
			return errors.New("an untracked file's name holds a line break")
		}
	}
	red, err := history.Redact(repoRoot, strings.Join(rels, "\n"), "salvage file names")
	if err != nil {
		return fmt.Errorf("the untracked files' names: %w", err)
	}
	if got := strings.Split(red.Text, "\n"); !slices.Equal(got, rels) {
		return errors.New("an untracked file's name holds what the redaction pass masks")
	}
	return nil
}

// salvageNamed is a reason about the untracked file rel. Its name passes the
// redaction pass before any reason carries it, and a name the pass refuses is
// not named at all.
func salvageNamed(repoRoot, rel, why string) error {
	shown := "an untracked file"
	if red, err := history.Redact(repoRoot, rel, "salvage file name"); err == nil {
		shown = red.Text
	}
	return errors.New(shown + ": " + why)
}

// causeOf is err without the path an *fs.PathError names, which is the
// worktree's own and unredacted.
func causeOf(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	var le *os.LinkError
	if errors.As(err, &le) {
		return le.Err.Error()
	}
	return err.Error()
}

// isText reports whether data is what the redaction pass can vouch for.
func isText(data []byte) bool {
	return utf8.Valid(data) && bytes.IndexByte(data, 0) < 0
}

// salvageEntryName is the entry a salvage of the worktree name made at the
// time at takes, and parseSalvageEntry its inverse: the worktree a directory
// in the archive is named for, ok false for any other name.
func salvageEntryName(name string, at time.Time) string {
	return at.UTC().Format(salvageStamp) + "-" + name + salvageSuffix
}

func parseSalvageEntry(entry string) (name string, ok bool) {
	n := len(salvageStamp)
	if len(entry) <= n+1+len(salvageSuffix) || entry[n] != '-' || !strings.HasSuffix(entry, salvageSuffix) {
		return "", false
	}
	if _, err := time.Parse(salvageStamp, entry[:n]); err != nil {
		return "", false
	}
	return strings.TrimSuffix(entry[n+1:], salvageSuffix), true
}

// earlierSalvage is the entry in the lane's archive holding a salvage
// identical to s, the latest when there are several; "" when none does or the
// archive is not made yet. An entry whose manifest cannot be read is no match:
// it is a salvage that failed part-way, or not a salvage at all.
func earlierSalvage(lane store.Lane, s salvage) (string, error) {
	dir, present, err := lane.NotesDir(false)
	if err != nil || !present {
		return "", err
	}
	es, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading the notes archive: %v", causeOf(err))
	}
	for i := len(es) - 1; i >= 0; i-- {
		e := es[i]
		if name, ok := parseSalvageEntry(e.Name()); !ok || name != s.name || !e.IsDir() {
			continue
		}
		entry := filepath.Join(dir, e.Name())
		if m, ok := readSalvageManifest(entry); ok && m.Kind == salvageKind &&
			m.SHA256 == s.manifest.SHA256 && m.Worktree == s.manifest.Worktree {
			return entry, nil
		}
	}
	return "", nil
}

// readSalvageManifest reads the manifest of the archive entry at entry,
// inside an os.Root on it, never through a link.
func readSalvageManifest(entry string) (salvageManifestFile, bool) {
	root, err := os.OpenRoot(entry)
	if err != nil {
		return salvageManifestFile{}, false
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, salvageManifest, maxManifestBytes)
	if err != nil {
		return salvageManifestFile{}, false
	}
	var m salvageManifestFile
	if json.Unmarshal(data, &m) != nil {
		return salvageManifestFile{}, false
	}
	return m, true
}

// writeSalvage writes s into a new entry of the lane's archive, stamped at, and
// answers the entry's path. The entry is made exclusively, each file inside an
// os.Root on it, and the manifest last, so an entry a failure leaves part-way
// has no manifest and is never taken for a salvage. The entry's path is
// returned once it is made, a failure after that included.
func writeSalvage(lane store.Lane, s salvage, at time.Time) (string, error) {
	dir, _, err := lane.NotesDir(true)
	if err != nil {
		return "", err
	}
	if !store.ValidName(s.name) {
		return "", errors.New("the worktree's name is not one plain path segment to name the salvage with")
	}
	entry := filepath.Join(dir, salvageEntryName(s.name, at))
	if err := os.Mkdir(entry, abcdhome.DirMode); err != nil {
		return "", fmt.Errorf("making the salvage entry %s: %v", fsutil.RedactHome(entry), causeOf(err))
	}
	root, err := os.OpenRoot(entry)
	if err != nil {
		return entry, fmt.Errorf("opening the salvage entry: %v", causeOf(err))
	}
	defer root.Close()
	for _, f := range s.files {
		if err := writeFresh(root, f.name, f.data); err != nil {
			return entry, fmt.Errorf("%s: %v", f.name, causeOf(err))
		}
	}
	m := s.manifest
	m.SavedAt = at.UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return entry, err
	}
	if err := writeFresh(root, salvageManifest, append(data, '\n')); err != nil {
		return entry, fmt.Errorf("%s: %v", salvageManifest, causeOf(err))
	}
	// The salvage may be the only copy of a dead session's work once a person
	// cleans the worktree up, so the entry and the directory it was made in
	// are flushed too. A filesystem that refuses a directory flush is
	// tolerated, as the notes archive tolerates it.
	for _, d := range []string{entry, dir} {
		if f, err := os.Open(d); err == nil {
			_ = fsutil.Flush(f)
			f.Close()
		}
	}
	return entry, nil
}

// writeFresh writes data to a file rel inside root that must not exist yet,
// the caller's alone, and flushes it before it is closed.
func writeFresh(root *os.Root, rel string, data []byte) error {
	if path.Base(rel) != rel {
		return errors.New("not one plain file name")
	}
	f, err := root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, abcdhome.FileMode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := fsutil.Flush(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
