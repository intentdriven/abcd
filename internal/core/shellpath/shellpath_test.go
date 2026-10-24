package shellpath

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// env is a person with a temporary home, the given login shell, and a PATH
// that does not reach ~/.local/bin.
func env(t *testing.T, shell, goos string) (Env, string) {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return Env{Home: home, Shell: shell, Path: "/usr/bin:/bin", GOOS: goos}, dir
}

func write(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

const wantLine = `export PATH="$HOME/.local/bin:$PATH" # added by abcd ahoy install`

// TestOfferAddsOneLineAndRemoveTakesOnlyIt is the whole round trip the intent
// promises: the plan names the file and the line, a yes keeps a copy and adds
// that one line, a second plan finds it and offers nothing, and removal puts
// the file back byte for byte, a file with no final newline included.
func TestOfferAddsOneLineAndRemoveTakesOnlyIt(t *testing.T) {
	e, dir := env(t, "/bin/zsh", "darwin")
	rc := filepath.Join(e.Home, ".zshrc")
	orig := "alias ll='ls -l'\nexport EDITOR=vi" // no final newline
	write(t, rc, orig, 0o644)

	p := PlanFor(e, dir)
	if p.State != Offer || p.File != rc || p.Line != wantLine || p.Create {
		t.Fatalf("plan = %+v, want an offer to add %q to %s", p, wantLine, rc)
	}
	if p.Export != `export PATH="$HOME/.local/bin:$PATH"` || ChecklistLine(e.Home, dir) != p.Export {
		t.Errorf("checklist line = %q", p.Export)
	}
	got, err := Apply(e, p)
	if err != nil {
		t.Fatal(err)
	}
	if want := orig + "\n" + wantLine + "\n"; read(t, rc) != want {
		t.Errorf("after apply the file is %q, want %q", read(t, rc), want)
	}
	if got.Backup == "" || read(t, got.Backup) != orig {
		t.Errorf("the copy kept is %q, want the file as it was", got.Backup)
	}
	if fi, err := os.Stat(rc); err != nil || fi.Mode().Perm() != 0o644 {
		t.Errorf("the file's mode changed: %v %v", fi.Mode(), err)
	}
	if _, err := os.Stat(abcdhome.Path(e.Home, recordLeaf)); err != nil {
		t.Errorf("no record of the line: %v", err)
	}

	if again := PlanFor(e, dir); again.State != AlreadyAdded || again.File != rc {
		t.Fatalf("second plan = %+v, want already added (never a second line)", again)
	}

	// The person adds a line after abcd's; removal takes abcd's alone.
	write(t, rc, read(t, rc)+"alias g=git\n", 0o644)
	r, err := Remove(e, dir)
	if err != nil || !r.Removed {
		t.Fatalf("remove = %+v, %v", r, err)
	}
	if want := orig + "\nalias g=git\n"; read(t, rc) != want {
		t.Errorf("after removal the file is %q, want %q", read(t, rc), want)
	}
	if _, err := os.Stat(abcdhome.Path(e.Home, recordLeaf)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the record outlived the line: %v", err)
	}
}

// TestRemoveRestoresAFileWithNoFinalNewline holds the byte-for-byte half: the
// newline abcd added before its line goes with it.
func TestRemoveRestoresAFileWithNoFinalNewline(t *testing.T) {
	e, dir := env(t, "/bin/zsh", "linux")
	rc := filepath.Join(e.Home, ".zshrc")
	orig := "setopt autocd"
	write(t, rc, orig, 0o600)
	if _, err := Apply(e, PlanFor(e, dir)); err != nil {
		t.Fatal(err)
	}
	if _, err := Remove(e, dir); err != nil {
		t.Fatal(err)
	}
	if read(t, rc) != orig {
		t.Errorf("after removal the file is %q, want %q", read(t, rc), orig)
	}
	if fi, _ := os.Stat(rc); fi.Mode().Perm() != 0o600 {
		t.Errorf("the mode is %v, want 0600 kept", fi.Mode().Perm())
	}
}

// TestCreatedFileIsRemovedWithTheLine: a file abcd created for its line alone
// goes when the line does.
func TestCreatedFileIsRemovedWithTheLine(t *testing.T) {
	e, dir := env(t, "/usr/bin/zsh", "linux")
	p := PlanFor(e, dir)
	if p.State != Offer || !p.Create {
		t.Fatalf("plan = %+v, want an offer to create .zshrc", p)
	}
	if _, err := Apply(e, p); err != nil {
		t.Fatal(err)
	}
	if read(t, p.File) != wantLine+"\n" {
		t.Errorf("created file holds %q", read(t, p.File))
	}
	if _, err := Remove(e, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(p.File); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the file abcd created is still there: %v", err)
	}
}

// TestLinkedFileGoesOnTheChecklist: a startup file that is a link (shared
// dotfiles) is never edited.
func TestLinkedFileGoesOnTheChecklist(t *testing.T) {
	e, dir := env(t, "/bin/zsh", "darwin")
	kept := filepath.Join(t.TempDir(), "dotfiles-zshrc")
	write(t, kept, "export EDITOR=vi\n", 0o644)
	if err := os.Symlink(kept, filepath.Join(e.Home, ".zshrc")); err != nil {
		t.Fatal(err)
	}
	p := PlanFor(e, dir)
	if p.State != Checklist || !strings.Contains(p.Reason, "link") {
		t.Fatalf("plan = %+v, want the checklist for a linked file", p)
	}
	if _, err := Apply(e, p); err == nil {
		t.Error("apply edited a plan that offers nothing")
	}
	if read(t, kept) != "export EDITOR=vi\n" {
		t.Error("the linked file changed")
	}
}

// TestAlreadyAddedByThePerson: a line of the person's own that puts the folder
// on the PATH, in any file the shell reads, means nothing is offered.
func TestAlreadyAddedByThePerson(t *testing.T) {
	for _, line := range []string{
		`export PATH="$HOME/.local/bin:$PATH"`,
		`export PATH=~/.local/bin:$PATH`,
		`path=(${HOME}/.local/bin $path)`,
	} {
		e, dir := env(t, "/bin/zsh", "darwin")
		write(t, filepath.Join(e.Home, ".zprofile"), line+"\n", 0o644)
		if p := PlanFor(e, dir); p.State != AlreadyAdded {
			t.Errorf("%q: plan = %+v, want already added", line, p)
		}
		if _, err := os.Lstat(filepath.Join(e.Home, ".zshrc")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%q: .zshrc was touched", line)
		}
	}
	// A commented line, or one naming a deeper folder, adds nothing.
	for _, line := range []string{`# export PATH="$HOME/.local/bin:$PATH"`, `export PATH="$HOME/.local/bin/sub:$PATH"`} {
		e, dir := env(t, "/bin/zsh", "darwin")
		write(t, filepath.Join(e.Home, ".zshrc"), line+"\n", 0o644)
		if p := PlanFor(e, dir); p.State != Offer {
			t.Errorf("%q: plan = %+v, want an offer", line, p)
		}
	}
}

// TestOnPathOffersNothing.
func TestOnPathOffersNothing(t *testing.T) {
	e, dir := env(t, "/bin/zsh", "darwin")
	e.Path = "/usr/bin:" + dir
	if p := PlanFor(e, dir); p.State != OnPath {
		t.Errorf("plan = %+v, want on path", p)
	}
}

// TestBashFilePerSystem holds the file choice for bash.
func TestBashFilePerSystem(t *testing.T) {
	e, dir := env(t, "/bin/bash", "linux")
	if p := PlanFor(e, dir); p.File != filepath.Join(e.Home, ".bashrc") {
		t.Errorf("linux bash file = %s, want ~/.bashrc", p.File)
	}
	e, dir = env(t, "/bin/bash", "darwin")
	if p := PlanFor(e, dir); p.File != filepath.Join(e.Home, ".bash_profile") || !p.Create {
		t.Errorf("macOS bash with no login file: %+v, want a new ~/.bash_profile", p)
	}
	write(t, filepath.Join(e.Home, ".profile"), "umask 022\n", 0o644)
	if p := PlanFor(e, dir); p.File != filepath.Join(e.Home, ".profile") || p.Create {
		t.Errorf("macOS bash with ~/.profile: %+v, want ~/.profile, never a .bash_profile that hides it", p)
	}
}

// TestUnsupportedAndUnsafe: another shell, a ZDOTDIR outside home, and a file
// another account could write all go on the checklist.
func TestUnsupportedAndUnsafe(t *testing.T) {
	e, dir := env(t, "/opt/homebrew/bin/fish", "darwin")
	if p := PlanFor(e, dir); p.State != Checklist || p.Shell != "" || !strings.Contains(p.Reason, "fish") {
		t.Errorf("fish: %+v", p)
	}
	e, dir = env(t, "/bin/zsh", "darwin")
	e.ZDotDir = t.TempDir()
	if p := PlanFor(e, dir); p.State != Checklist || !strings.Contains(p.Reason, "ZDOTDIR") {
		t.Errorf("ZDOTDIR outside home: %+v", p)
	}
	e, dir = env(t, "/bin/zsh", "darwin")
	write(t, filepath.Join(e.Home, ".zshrc"), "x=1\n", 0o666)
	if p := PlanFor(e, dir); p.State != Checklist || !strings.Contains(p.Reason, "writable by others") {
		t.Errorf("other-writable file: %+v", p)
	}
	e, dir = env(t, "/bin/zsh", "darwin")
	write(t, filepath.Join(e.Home, ".zshrc"), strings.Repeat("#\n", MaxFileBytes), 0o644)
	if p := PlanFor(e, dir); p.State != Checklist {
		t.Errorf("oversize file: %+v", p)
	}
}

// TestZDotDirInsideHome is honoured.
func TestZDotDirInsideHome(t *testing.T) {
	e, dir := env(t, "/bin/zsh", "darwin")
	e.ZDotDir = filepath.Join(e.Home, ".config", "zsh")
	if err := os.MkdirAll(e.ZDotDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if p := PlanFor(e, dir); p.State != Offer || p.File != filepath.Join(e.ZDotDir, ".zshrc") {
		t.Errorf("plan = %+v, want $ZDOTDIR/.zshrc", p)
	}
}

// TestApplyRefusesAFileChangedSinceTheQuestion.
func TestApplyRefusesAFileChangedSinceTheQuestion(t *testing.T) {
	e, dir := env(t, "/bin/zsh", "darwin")
	rc := filepath.Join(e.Home, ".zshrc")
	write(t, rc, "a=1\n", 0o644)
	p := PlanFor(e, dir)
	write(t, rc, "a=2\n", 0o644)
	if _, err := Apply(e, p); !errors.Is(err, ErrChanged) {
		t.Errorf("apply over a changed file: %v, want ErrChanged", err)
	}
	if read(t, rc) != "a=2\n" {
		t.Error("the changed file was written")
	}
}

// TestReadBackMismatchPutsTheFileBack: a write that lands something other than
// what was asked is undone from the copy.
func TestReadBackMismatchPutsTheFileBack(t *testing.T) {
	e, dir := env(t, "/bin/zsh", "darwin")
	rc := filepath.Join(e.Home, ".zshrc")
	write(t, rc, "a=1\n", 0o644)
	orig := writeFile
	writeFile = func(path string, _ []byte) error { return os.WriteFile(path, []byte("garbled"), 0o644) }
	t.Cleanup(func() { writeFile = orig })
	_, err := Apply(e, PlanFor(e, dir))
	if err == nil || !strings.Contains(err.Error(), "put back") {
		t.Fatalf("apply = %v, want the file put back", err)
	}
	if read(t, rc) != "a=1\n" {
		t.Errorf("the file is %q, want it put back", read(t, rc))
	}
}

// TestRemoveWithoutRecordFindsTheMarkedLine, and leaves a line the person
// edited alone.
func TestRemoveWithoutRecordFindsTheMarkedLine(t *testing.T) {
	e, dir := env(t, "/bin/zsh", "darwin")
	rc := filepath.Join(e.Home, ".zshrc")
	write(t, rc, "a=1\n"+wantLine+"\nb=2\n", 0o644)
	if r, err := Remove(e, dir); err != nil || !r.Removed {
		t.Fatalf("remove = %+v, %v", r, err)
	}
	if read(t, rc) != "a=1\nb=2\n" {
		t.Errorf("file = %q", read(t, rc))
	}

	edited := strings.Replace(wantLine, "$PATH", "$PATH:/x", 1)
	write(t, rc, "a=1\n"+edited+"\n", 0o644)
	if r, err := Remove(e, dir); err != nil || r.Removed {
		t.Errorf("remove of an edited line = %+v, %v; want it left", r, err)
	}
	if read(t, rc) != "a=1\n"+edited+"\n" {
		t.Error("an edited line was removed")
	}
}

// TestCopiesArePrunedToTen.
func TestCopiesArePrunedToTen(t *testing.T) {
	e, _ := env(t, "/bin/zsh", "darwin")
	rc := filepath.Join(e.Home, ".zshrc")
	for i := 0; i < backupKeep+3; i++ {
		if _, pruned, err := keepCopy(e.Home, rc, []byte("x")); err != nil || pruned != nil {
			t.Fatal(err, pruned)
		}
	}
	other := abcdhome.Path(e.Home, "backups", "settings.json.20200101T000000Z")
	write(t, other, "{}", 0o600)
	if _, _, err := keepCopy(e.Home, rc, []byte("x")); err != nil {
		t.Fatal(err)
	}
	ents, _ := os.ReadDir(abcdhome.Path(e.Home, "backups"))
	n := 0
	for _, d := range ents {
		if strings.HasPrefix(d.Name(), "shell-zshrc.") {
			n++
		}
	}
	if n != backupKeep {
		t.Errorf("%d copies kept, want %d", n, backupKeep)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("another file's copy was pruned: %v", err)
	}
}
