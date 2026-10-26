// Package shellpath is the one place abcd edits a person's shell settings: the
// line that puts the folder abcd installs into on the PATH
// (itd-2610240919030303, adr-2610241022330872). It never prints and never
// asks; `abcd ahoy install` asks the question and the setup flow lists the
// line when the answer is no. What it does on a yes is held to one bar:
//
//   - it edits only the zsh or bash startup file the person's login shell
//     reads in a new terminal, and only one it can read whole: a regular file,
//     not a link, owned by the person, writable by no one else, and within
//     MaxFileBytes;
//   - it never adds the folder twice: a startup file that already adds it (or
//     a PATH that already reaches it) is left alone;
//   - it keeps a copy of the file in abcd's backups folder first (no copy, no
//     write), replaces the file atomically, reads it back, and puts it back
//     from the copy when it does not say what was written;
//   - it adds exactly one line, ending in Marker, and records it in abcd's
//     home, so Remove takes out that line and nothing else.
//
// Which file each shell reads (Plan documents the choice):
//
//   - zsh: ${ZDOTDIR:-$HOME}/.zshrc, which every interactive zsh reads, the
//     login shell a macOS terminal opens and the non-login one most Linux
//     terminals open alike; .zprofile is read by login shells only.
//   - bash on macOS: the login file bash reads, the first of .bash_profile,
//     .bash_login and .profile that exists, else a new .bash_profile; a
//     macOS terminal opens a login shell, which never reads .bashrc, and
//     creating .bash_profile beside an existing .profile would stop bash
//     reading .profile.
//   - bash on Linux: ~/.bashrc, which the non-login shell a terminal opens
//     reads.
package shellpath

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Marker ends the one line abcd adds, so the line can be found and removed
// again and a person reading the file can tell who put it there.
const Marker = "# added by abcd ahoy install"

// MaxFileBytes is the largest startup file abcd edits. A file it cannot read
// whole is a file it cannot vouch for leaving otherwise unchanged.
const MaxFileBytes = 256 << 10

// recordLeaf is the record of the line abcd added, in abcd's home.
const recordLeaf = "shell-path.json"

// backupKeep is how many copies of each startup file the backups folder
// keeps, the bound the harness-settings copies are held to.
const backupKeep = 10

// Env is the part of the environment a plan reads. CurrentEnv fills it from
// the process; a test builds its own.
type Env struct {
	Home    string // the person's home directory, absolute
	Shell   string // $SHELL, the login shell
	ZDotDir string // $ZDOTDIR, where zsh looks for its startup files
	Path    string // $PATH
	GOOS    string // runtime.GOOS
}

// CurrentEnv reads Env from the process environment.
func CurrentEnv() Env {
	home, _ := os.UserHomeDir()
	return Env{
		Home:    home,
		Shell:   os.Getenv("SHELL"),
		ZDotDir: os.Getenv("ZDOTDIR"),
		Path:    os.Getenv("PATH"),
		GOOS:    runtime.GOOS,
	}
}

// State is what a plan found.
type State string

const (
	// OnPath: the PATH already reaches the folder; there is nothing to do.
	OnPath State = "on_path"
	// AlreadyAdded: a startup file already adds the folder (abcd's own line
	// or the person's); nothing is edited, and a new terminal picks it up.
	AlreadyAdded State = "already_added"
	// Offer: abcd may add its line to File, on the person's yes.
	Offer State = "offer"
	// Checklist: abcd will not edit the file (Reason says why); the line is
	// the person's to add.
	Checklist State = "checklist"
)

// Plan is what Plan found: the file and line to show the person before
// asking, or why there is nothing to ask.
type Plan struct {
	State  State  `json:"state"`
	Shell  string `json:"shell,omitempty"` // "zsh" or "bash"; "" for another shell
	File   string `json:"file,omitempty"`  // the file to edit, or the one that already adds the folder
	Line   string `json:"line"`            // the marked line abcd adds
	Export string `json:"export"`          // the same line without the marker, for the checklist
	Reason string `json:"reason,omitempty"`
	Create bool   `json:"create,omitempty"` // File does not exist yet; Apply creates it

	dir     string
	current []byte // File's bytes when planned; nil when absent
}

// Applied is what Apply did.
type Applied struct {
	File   string `json:"file"`
	Line   string `json:"line"`
	Backup string `json:"backup,omitempty"` // the copy of the file as it was; "" when Apply created it
	// RecordNote is set when the line was added but its record could not be
	// kept: Remove still finds the line by its marker.
	RecordNote string `json:"record_note,omitempty"`
	// PruneNote is set when older copies could not be pruned.
	PruneNote string `json:"prune_note,omitempty"`
}

// Removed is what Remove did.
type Removed struct {
	Removed bool   `json:"removed"`
	File    string `json:"file,omitempty"`
	Backup  string `json:"backup,omitempty"`
	Note    string `json:"note"`
}

// ExportLine is the line that puts dir on the PATH, in the form a person
// pastes: $HOME stands for the home directory so the line reads the same on
// every account. The folder comes first, ahead of the PATH it had, so the
// abcd abcd installs is the one that answers rather than an older copy in a
// system folder (the shadowing ahoy reports as symlink.shadowed).
func ExportLine(home, dir string) string {
	return `export PATH="` + shellDir(home, dir) + `:$PATH"`
}

// markedLine is ExportLine with Marker, the one line abcd adds.
func markedLine(home, dir string) string {
	return ExportLine(home, dir) + " " + Marker
}

// checklistLine is the line the checklist names when the person declines, or
// abcd will not edit the file: the export line, for the person to add.
func checklistLine(home, dir string) string { return ExportLine(home, dir) }

// shellDir renders dir for a double-quoted shell string.
func shellDir(home, dir string) string {
	if home != "" {
		if rel, err := filepath.Rel(home, dir); err == nil && rel != "." && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
			return "$HOME/" + filepath.ToSlash(rel)
		}
	}
	return dir
}

// unsafeInQuotes are the bytes a double-quoted shell string expands or ends
// on; a folder holding one is never written into a startup file.
const unsafeInQuotes = "\"$`\\\n"

// PlanFor decides, reading only, whether abcd may add dir to the PATH in the
// startup file the person's login shell reads, and which file and line that
// is. It never writes.
func PlanFor(env Env, dir string) Plan {
	p := Plan{dir: dir, Line: markedLine(env.Home, dir), Export: ExportLine(env.Home, dir)}
	if onPath(env.Path, dir) {
		p.State = OnPath
		return p
	}
	if env.Home == "" || !filepath.IsAbs(env.Home) {
		return checklist(p, "no home directory is resolved")
	}
	if strings.ContainsAny(dir, unsafeInQuotes) {
		return checklist(p, "the folder's name holds a character a shell would expand")
	}
	p.Shell = filepath.Base(env.Shell)
	switch p.Shell {
	case "zsh", "bash":
	default:
		shown := p.Shell
		if shown == "" || shown == "." {
			shown = "not named"
		}
		p.Shell = ""
		return checklist(p, "abcd edits zsh and bash settings only, and your login shell is "+shown)
	}
	target, read, why := files(env, p.Shell)
	if why != "" {
		return checklist(p, why)
	}
	p.File = target
	// Already added: abcd's own line, or a line of the person's that sets the
	// PATH to include the folder, in any file this shell reads.
	for _, f := range read {
		if f == "" {
			continue
		}
		if raw, ok := readFollowing(f); ok && addsDir(raw, env.Home, dir) {
			p.State = AlreadyAdded
			p.File = f
			return p
		}
	}
	cur, why := editable(target)
	if why != "" {
		return checklist(p, why)
	}
	p.current = cur
	p.Create = cur == nil
	p.State = Offer
	return p
}

func checklist(p Plan, why string) Plan {
	p.State = Checklist
	p.Reason = why
	return p
}

// files names the file abcd edits for shell, and every file that shell reads
// at startup, which the already-added check reads.
func files(env Env, shell string) (target string, read []string, why string) {
	h := env.Home
	if shell == "zsh" {
		zdot := h
		if env.ZDotDir != "" {
			z := filepath.Clean(env.ZDotDir)
			if !filepath.IsAbs(z) || !within(h, z) {
				return "", nil, "ZDOTDIR names a folder outside your home folder, which abcd does not edit"
			}
			zdot = z
		}
		read = []string{filepath.Join(h, ".zshenv")}
		for _, leaf := range []string{".zshenv", ".zprofile", ".zshrc", ".zlogin"} {
			if f := filepath.Join(zdot, leaf); f != read[0] {
				read = append(read, f)
			}
		}
		return filepath.Join(zdot, ".zshrc"), read, ""
	}
	login := []string{filepath.Join(h, ".bash_profile"), filepath.Join(h, ".bash_login"), filepath.Join(h, ".profile")}
	read = append(append([]string{}, login...), filepath.Join(h, ".bashrc"))
	if env.GOOS != "darwin" {
		return filepath.Join(h, ".bashrc"), read, ""
	}
	for _, f := range login {
		if _, err := os.Lstat(f); err == nil {
			return f, read, ""
		}
	}
	return login[0], read, ""
}

// within reports whether p is home or a path below it.
func within(home, p string) bool {
	rel, err := filepath.Rel(home, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// onPath reports whether dir is one of the entries of pathList, comparing
// each through its symlinks.
func onPath(pathList, dir string) bool {
	want := resolve(dir)
	for _, d := range filepath.SplitList(pathList) {
		if d != "" && resolve(d) == want {
			return true
		}
	}
	return false
}

func resolve(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// readFollowing reads a startup file for the already-added check, following a
// link (reading changes nothing) but never a non-regular file and never past
// MaxFileBytes.
func readFollowing(path string) ([]byte, bool) {
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxFileBytes {
		return nil, false
	}
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil || len(raw) > MaxFileBytes {
		return nil, false
	}
	return raw, true
}

// addsDir reports whether raw already puts dir on the PATH: abcd's own marked
// line for dir (markedLine, exactly), or an uncommented line that names the
// PATH and the folder in any of the spellings a person writes it in. The
// check is textual and conservative: it can mistake a line for one that adds
// the folder, which costs the offer, never a second line.
func addsDir(raw []byte, home, dir string) bool {
	forms := []string{dir}
	if home != "" && within(home, dir) && dir != home {
		rel, _ := filepath.Rel(home, dir)
		rel = filepath.ToSlash(rel)
		forms = append(forms, "$HOME/"+rel, "${HOME}/"+rel, "~/"+rel)
	}
	marked := markedLine(home, dir)
	for _, ln := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		// abcd's own line for this folder, compared whole as Remove compares
		// it: the marker on a line that adds another folder adds nothing.
		if t == marked {
			return true
		}
		if !strings.Contains(strings.ToLower(t), "path") {
			continue
		}
		for _, f := range forms {
			if containsWord(t, f) {
				return true
			}
		}
	}
	return false
}

// containsWord reports whether s holds f followed by the end of a PATH entry.
func containsWord(s, f string) bool {
	for i := 0; ; {
		j := strings.Index(s[i:], f)
		if j < 0 {
			return false
		}
		end := i + j + len(f)
		if end == len(s) || strings.ContainsRune(":\"' )\t;}/", rune(s[end])) {
			if end == len(s) || s[end] != '/' || end+1 == len(s) || strings.ContainsRune(":\"' )\t;}", rune(s[end+1])) {
				return true
			}
		}
		i = i + j + 1
	}
}

// editable reads the file abcd would edit and says why it will not, or ""
// when it may: absent (it is created), or a regular file, not a link, owned
// by the person, writable by no one else, within MaxFileBytes. A file that
// is a link is kept elsewhere (shared dotfiles), and an edit through the
// link would change that other copy.
func editable(path string) (current []byte, why string) {
	shown := fsutil.RedactHome(path)
	fi, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		parent, perr := os.Lstat(filepath.Dir(path))
		if perr != nil || !parent.IsDir() {
			return nil, "the folder " + fsutil.RedactHome(filepath.Dir(path)) + " that holds it is not there"
		}
		return nil, ""
	case err != nil:
		return nil, shown + " could not be read (" + err.Error() + ")"
	case fi.Mode()&os.ModeSymlink != 0:
		return nil, shown + " is a link to a file kept elsewhere, which abcd does not edit"
	case !fi.Mode().IsRegular():
		return nil, shown + " is not a regular file"
	case fi.Size() > MaxFileBytes:
		return nil, shown + " is larger than abcd reads whole"
	}
	if err := fsutil.CallersAlone(path, fi); err != nil {
		return nil, shown + " is owned by another account or writable by others"
	}
	raw, err := fsutil.ReadGuarded(path, MaxFileBytes)
	if err != nil {
		return nil, shown + " could not be read whole"
	}
	if raw == nil {
		raw = []byte{}
	}
	return raw, ""
}

// record is what abcd keeps of the line it added, so Remove takes out exactly
// that line.
type record struct {
	SchemaVersion int    `json:"schema_version"`
	File          string `json:"file"`
	Line          string `json:"line"`
	// Separated: the file did not end in a newline, so abcd added one before
	// its line, and Remove takes that newline out with it.
	Separated bool `json:"separated,omitempty"`
	// Created: the file did not exist, and Remove deletes it when nothing but
	// abcd's line was ever in it.
	Created bool `json:"created,omitempty"`
}

// Errors Apply and Remove return; the text completes "…: <text>".
var (
	ErrNotOffered = errors.New("the plan offers no edit")
	ErrChanged    = errors.New("the file changed since the question was asked, so nothing was written")
)

// Apply adds the plan's line to its file, on the person's yes. The file is
// judged and read again first, and refused when it is no longer the file the
// person was shown.
func Apply(env Env, p Plan) (Applied, error) {
	if p.State != Offer || p.File == "" {
		return Applied{}, ErrNotOffered
	}
	shown := fsutil.RedactHome(p.File)
	cur, why := editable(p.File)
	if why != "" {
		return Applied{}, errors.New(why + ", so nothing was written")
	}
	if (cur == nil) != p.Create || (cur != nil && !bytes.Equal(cur, p.current)) {
		return Applied{}, ErrChanged
	}
	rec := record{SchemaVersion: 1, File: p.File, Line: p.Line, Created: p.Create}
	next := append([]byte{}, cur...)
	if len(next) > 0 && next[len(next)-1] != '\n' {
		next = append(next, '\n')
		rec.Separated = true
	}
	next = append(next, []byte(p.Line+"\n")...)

	out := Applied{File: p.File, Line: p.Line}
	if p.Create {
		if err := createFile(p.File, next); err != nil {
			return Applied{}, errors.New(shown + " could not be created (" + fsutil.RedactHome(err.Error()) + "); nothing was written")
		}
	} else {
		backup, pruned, err := keepCopy(env.Home, p.File, cur)
		if err != nil {
			return Applied{}, errors.New("a copy of " + shown + " could not be kept in " + abcdhome.Display("backups") +
				"/ first (" + fsutil.RedactHome(err.Error()) + "), so nothing was written")
		}
		out.Backup = backup
		if pruned != nil {
			out.PruneNote = "older copies of " + shown + " in " + abcdhome.Display("backups") + "/ could not be pruned (" + pruned.Error() + ")"
		}
		judged, err := os.Lstat(p.File)
		if err != nil || !judged.Mode().IsRegular() {
			return Applied{}, ErrChanged
		}
		beforeReplace()
		// The rename replaces the leaf without following it, so a link put at
		// the path since it was judged would be lost rather than written
		// through; the leaf is judged again here and refused when it moved.
		if now, err := os.Lstat(p.File); err != nil || !now.Mode().IsRegular() || !os.SameFile(judged, now) {
			return Applied{}, ErrChanged
		}
		if err := replace(p.File, cur, next, backup); err != nil {
			return Applied{}, err
		}
	}
	if err := writeRecord(env.Home, rec); err != nil {
		out.RecordNote = "the record of the line could not be kept in " + abcdhome.Display(recordLeaf) +
			" (" + fsutil.RedactHome(err.Error()) + "); `abcd ahoy uninstall` still finds the line by its marker"
	}
	return out, nil
}

// createFile creates path holding data, never over a file that appeared
// since it was judged absent.
func createFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	if err := fsutil.Flush(f); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return err
	}
	return readBack(path, data)
}

// writeFile replaces a startup file atomically, keeping its mode. A variable
// so a test can make the write land something other than what was asked,
// which is what the read-back exists to catch.
var writeFile = fsutil.WriteFileAtomicPreserveMode

// beforeReplace runs between the last judgement of the leaf and the replace.
// A variable so a test can swap the leaf in that moment.
var beforeReplace = func() {}

// replace writes next over path, whose bytes now are cur and whose copy is at
// backup, reads it back, and puts cur back when it does not read as written.
func replace(path string, cur, next []byte, backup string) error {
	shown := fsutil.RedactHome(path)
	copyAt := fsutil.RedactHome(backup)
	if err := writeFile(path, next); err != nil {
		return errors.New(shown + " could not be written (" + fsutil.RedactHome(err.Error()) + "); it is as it was, and a copy is at " + copyAt)
	}
	if err := readBack(path, next); err == nil {
		return nil
	}
	if rerr := fsutil.WriteFileAtomicPreserveMode(path, cur); rerr != nil {
		return &RestoreError{File: shown, Backup: copyAt, Cause: fsutil.RedactHome(rerr.Error())}
	}
	return errors.New(shown + " did not read back as written, so it was put back from the copy at " + copyAt)
}

// RestoreError is the one failure that leaves a startup file other than as it
// was: it did not read back as written, and putting it back from the copy
// failed too, so the person has a step to take by hand. Its fields are in the
// tilde form.
type RestoreError struct {
	File   string // the startup file
	Backup string // the copy of it as it was
	Cause  string // why putting it back failed
}

func (e *RestoreError) Error() string {
	return e.File + " did not read back as written, and putting it back failed too (" + e.Cause +
		"): copy " + e.Backup + " over it by hand"
}

// readBack confirms path holds exactly want.
func readBack(path string, want []byte) error {
	got, err := fsutil.ReadGuarded(path, MaxFileBytes+int64(len(want)))
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return errors.New("the file does not hold what was written")
	}
	return nil
}

// Remove takes out the line abcd added and nothing else, for `abcd ahoy
// uninstall`. The file comes from the record abcd kept; with none, every
// startup file the person's shell reads is searched for abcd's exact line for
// dir. A line the person has since edited is no longer abcd's and is left.
func Remove(env Env, dir string) (Removed, error) {
	rec, rerr := readRecord(env.Home)
	var candidates []record
	switch {
	case rerr != nil:
		return Removed{Note: "the record in " + abcdhome.Display(recordLeaf) + " could not be read (" + rerr.Error() + "); no shell settings file was changed"}, nil
	case rec != nil:
		candidates = []record{*rec}
	default:
		if dir == "" || env.Home == "" {
			return Removed{Note: "abcd added no line to a shell settings file"}, nil
		}
		line := markedLine(env.Home, dir)
		for _, shell := range []string{"zsh", "bash"} {
			_, read, _ := files(env, shell)
			for _, f := range read {
				if raw, ok := readFollowing(f); ok && lineIndex(raw, line) >= 0 {
					candidates = append(candidates, record{File: f, Line: line})
				}
			}
		}
		if len(candidates) == 0 {
			return Removed{Note: "abcd added no line to a shell settings file"}, nil
		}
	}
	c := candidates[0]
	shown := fsutil.RedactHome(c.File)
	cur, why := editable(c.File)
	switch {
	case why != "":
		return Removed{File: c.File, Note: why + ", so abcd's line was left in it; remove `" + c.Line + "` by hand"}, nil
	case cur == nil:
		_ = removeRecord(env.Home)
		return Removed{File: c.File, Note: shown + " is not there; nothing to remove"}, nil
	}
	i := lineIndex(cur, c.Line)
	if i < 0 {
		_ = removeRecord(env.Home)
		return Removed{File: c.File, Note: shown + " no longer holds abcd's line; nothing in it was changed"}, nil
	}
	end := i + len(c.Line)
	if end < len(cur) && cur[end] == '\n' {
		end++
	}
	next := append(append([]byte{}, cur[:i]...), cur[end:]...)
	if c.Separated && end == len(cur) && len(next) > 0 && next[len(next)-1] == '\n' {
		next = next[:len(next)-1]
	}
	if c.Created && len(next) == 0 {
		if err := os.Remove(c.File); err != nil {
			return Removed{}, errors.New(shown + " could not be removed (" + fsutil.RedactHome(err.Error()) + ")")
		}
		_ = removeRecord(env.Home)
		return Removed{Removed: true, File: c.File, Note: "removed " + shown + ", which abcd created for its line alone"}, nil
	}
	backup, _, err := keepCopy(env.Home, c.File, cur)
	if err != nil {
		return Removed{}, errors.New("a copy of " + shown + " could not be kept in " + abcdhome.Display("backups") +
			"/ first (" + fsutil.RedactHome(err.Error()) + "), so abcd's line was left in it")
	}
	if err := replace(c.File, cur, next, backup); err != nil {
		return Removed{}, err
	}
	_ = removeRecord(env.Home)
	return Removed{Removed: true, File: c.File, Backup: backup,
		Note: "removed abcd's PATH line from " + shown + "; the file as it was is kept at " + fsutil.RedactHome(backup)}, nil
}

// lineIndex is the offset of the last whole line of raw that is exactly line,
// or -1.
func lineIndex(raw []byte, line string) int {
	at := -1
	off := 0
	for _, ln := range bytes.SplitAfter(raw, []byte("\n")) {
		if string(bytes.TrimSuffix(ln, []byte("\n"))) == line {
			at = off
		}
		off += len(ln)
	}
	return at
}

// ---------------------------------------------------------------------------
// abcd's home: the record and the copies
// ---------------------------------------------------------------------------

func writeRecord(home string, rec record) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	dir, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer dir.Close()
	return fsutil.WriteFileAtomicInRoot(dir, recordLeaf, append(data, '\n'), abcdhome.FileMode)
}

// readRecord reads the record: nil with no error when there is none.
func readRecord(home string) (*record, error) {
	if home == "" {
		return nil, nil
	}
	dir, err := fsutil.OpenHomeScope(home, abcdhome.Rel())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer dir.Close()
	raw, err := fsutil.ReadGuardedInRoot(dir, recordLeaf, 64<<10)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, errors.New("it is not valid JSON")
	}
	if rec.File == "" || !filepath.IsAbs(rec.File) || !strings.HasSuffix(rec.Line, Marker) || strings.Contains(rec.Line, "\n") {
		return nil, errors.New("it does not name an absolute file and a line ending in abcd's marker")
	}
	return &rec, nil
}

func removeRecord(home string) error {
	dir, err := fsutil.OpenHomeScope(home, abcdhome.Rel())
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Remove(recordLeaf)
}

// keepCopy keeps raw, the file's bytes now, in abcd's backups folder as
// shell-<leaf>.<UTC stamp>, created exclusively and private to the account,
// and prunes that file's copies to the newest backupKeep.
func keepCopy(home, path string, raw []byte) (backup string, pruned, err error) {
	if home == "" {
		return "", nil, errors.New("no home directory is resolved")
	}
	dir, err := fsutil.EnsureHomeScope(home, abcdhome.Rel("backups"), abcdhome.DirMode)
	if err != nil {
		return "", nil, err
	}
	defer dir.Close()
	prefix := backupPrefix(path)
	stamp := prefix + time.Now().UTC().Format("20060102T150405Z")
	for n := 1; n <= 100; n++ {
		leaf := stamp
		if n > 1 {
			leaf += "-" + strconv.Itoa(n)
		}
		err := fsutil.CreateExclusiveIn(dir, leaf, raw, abcdhome.FileMode)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", nil, err
		}
		return abcdhome.Path(home, "backups", leaf), prune(dir, prefix), nil
	}
	return "", nil, errors.New("a hundred copies already carry this second's stamp")
}

// backupPrefix names a startup file's copies: shell-zshrc. for ~/.zshrc.
func backupPrefix(path string) string {
	return "shell-" + strings.TrimPrefix(filepath.Base(path), ".") + "."
}

// prune removes all but the newest backupKeep copies carrying prefix. Only a
// regular file named exactly like a copy counts; anything else is left.
func prune(dir *os.Root, prefix string) error {
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `(\d{8}T\d{6}Z)(?:-(\d+))?$`)
	ents, err := fs.ReadDir(dir.FS(), ".")
	if err != nil {
		return err
	}
	type kept struct {
		leaf, stamp string
		n           int
	}
	var copies []kept
	for _, e := range ents {
		m := re.FindStringSubmatch(e.Name())
		if m == nil || !e.Type().IsRegular() {
			continue
		}
		n := 1
		if m[2] != "" {
			if n, err = strconv.Atoi(m[2]); err != nil {
				continue
			}
		}
		copies = append(copies, kept{leaf: e.Name(), stamp: m[1], n: n})
	}
	if len(copies) <= backupKeep {
		return nil
	}
	sort.Slice(copies, func(i, j int) bool {
		if copies[i].stamp != copies[j].stamp {
			return copies[i].stamp < copies[j].stamp
		}
		return copies[i].n < copies[j].n
	})
	var first error
	for _, c := range copies[:len(copies)-backupKeep] {
		if err := dir.Remove(c.leaf); err != nil && first == nil {
			first = err
		}
	}
	return first
}
