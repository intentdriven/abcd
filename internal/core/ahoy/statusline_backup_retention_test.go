package ahoy

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// seedBackups lays down the backups folder with the named copies, each a
// regular file, plus whatever else the test puts beside them.
func seedBackups(t *testing.T, home string, names ...string) string {
	t.Helper()
	dir := abcdhome.Path(home, "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// copiesIn lists the names in the backups folder that have a copy's shape.
func copiesIn(t *testing.T, home string) []string {
	t.Helper()
	var out []string
	for _, n := range backupsIn(t, home) {
		if backupNameRe.MatchString(n) {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// TestStatusLineBackupsKeepTheNewestTen: every write into the harness file
// keeps a copy first, so without a bound the folder grows by one on every
// install, repair and uninstall. The newest ten copies are kept. Only a regular
// file named exactly like a copy is ever removed — never a symlink, whatever
// it is named and wherever it points, and never anything else in the folder —
// and a suffixed copy (two in one second) is ordered by its number, not by its
// spelling.
func TestStatusLineBackupsKeepTheNewestTen(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(""))
	repo := installedRepo(t)
	old := []string{
		"settings.json.20200101T000001Z",
		"settings.json.20200101T000001Z-2",
		"settings.json.20200101T000001Z-10",
	}
	for i := 2; i <= 9; i++ {
		old = append(old, "settings.json.20200101T00000"+string(rune('0'+i))+"Z")
	}
	dir := seedBackups(t, home, old...)
	others := []string{"notes.txt", "settings.json.old", "settings.json.20200101T000000Z.bak"}
	for _, n := range others {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("keep\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(target, []byte("keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := "settings.json.20190101T000000Z"
	if err := os.Symlink(target, filepath.Join(dir, link)); err != nil {
		t.Fatal(err)
	}

	res, err := Install(repo, InstallOptions{}, offerPrompter(true, nil))
	if err != nil {
		t.Fatal(err)
	}
	if line, ok := statusLineOf(t, settings); !ok || line["command"] == nil {
		t.Fatalf("precondition: the status line was wired; notes = %v", res.Notes)
	}

	copies := copiesIn(t, home)
	var regular []string
	for _, n := range copies {
		if fi, err := os.Lstat(filepath.Join(dir, n)); err == nil && fi.Mode().IsRegular() {
			regular = append(regular, n)
		}
	}
	if len(regular) != 10 {
		t.Errorf("copies kept = %d %v, want the newest 10", len(regular), regular)
	}
	for _, gone := range []string{"settings.json.20200101T000001Z", "settings.json.20200101T000001Z-2"} {
		if _, err := os.Lstat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
			t.Errorf("%s survived; it is among the two oldest", gone)
		}
	}
	if _, err := os.Lstat(filepath.Join(dir, "settings.json.20200101T000001Z-10")); err != nil {
		t.Errorf("the -10 copy was pruned before the -2 copy: %v", err)
	}
	for _, n := range append(others, link) {
		if _, err := os.Lstat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s, which is not a copy, was removed: %v", n, err)
		}
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("the symlink's target was touched: %v", err)
	}
}

// TestStatusLineBackupPruneFailureIsANote: a copy that cannot be pruned never
// fails the write it was made for — the write goes ahead and a note says what
// was left behind.
func TestStatusLineBackupPruneFailureIsANote(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(""))
	repo := installedRepo(t)
	var names []string
	for i := 10; i <= 20; i++ {
		names = append(names, "settings.json.20200101T0000"+itoa2(i)+"Z")
	}
	seedBackups(t, home, names...)
	prev := removeBackupCopy
	removeBackupCopy = func(*os.Root, string) error { return errors.New("operation not permitted") }
	t.Cleanup(func() { removeBackupCopy = prev })

	res, err := Install(repo, InstallOptions{}, offerPrompter(true, nil))
	if err != nil {
		t.Fatal(err)
	}
	if line, ok := statusLineOf(t, settings); !ok || line["command"] == nil {
		t.Fatalf("the write failed with the prune; notes = %v", res.Notes)
	}
	var note string
	for _, n := range res.Notes {
		if strings.Contains(n, abcdhome.Display("backups")) && strings.Contains(n, "could not be removed") {
			note = n
		}
	}
	if note == "" || strings.Contains(note, home) {
		t.Errorf("notes = %v, want one naming the copies left in the backups folder, in tilde form", res.Notes)
	}
}

func itoa2(i int) string {
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
