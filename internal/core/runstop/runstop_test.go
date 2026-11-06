package runstop

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"errors"
	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"strconv"
)

// world is a fixture home and a sticky shared folder, each with the setting
// that names this account as the person.
func world(t *testing.T) (home, folder string) {
	t.Helper()
	base := t.TempDir()
	home = filepath.Join(base, "home")
	folder = filepath.Join(base, "shared")
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(folder, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(folder, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSetting(home, folder, ""); err != nil {
		t.Fatalf("WriteSetting: %v", err)
	}
	return home, folder
}

// TestNoStopIsNoStop: a home with no setting and no file reads as no stop and
// nothing refused, and creates nothing.
func TestNoStopIsNoStop(t *testing.T) {
	home := t.TempDir()
	got := readNoPin(home)
	if got.Stop != nil || len(got.Refused) != 0 {
		t.Fatalf("Read = %+v, want nothing", got)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Fatalf("a read created %v", entries)
	}
}

// TestSetIsReadAndLifted: the person's set is a stop in force in the folder,
// naming who and when from the file itself, and the lift removes it and is
// recorded in the person's own home.
func TestSetIsReadAndLifted(t *testing.T) {
	home, folder := world(t)
	now := time.Date(2026, 11, 4, 18, 0, 0, 0, time.UTC)
	set, err := Set(home, now)
	if err != nil {
		t.Fatal(err)
	}
	if want := stopPath(folder, os.Getuid()); set.Path != want {
		t.Fatalf("set at %q, want %q", set.Path, want)
	}
	got := readNoPin(home)
	if got.Stop == nil || len(got.Refused) != 0 {
		t.Fatalf("Read = %+v, want a stop", got)
	}
	if got.Stop.UID != os.Getuid() || got.Stop.SetBy == "" || got.Stop.SetAt.IsZero() {
		t.Fatalf("the stop names %+v", got.Stop)
	}
	fi, err := os.Lstat(set.Path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o044 != 0o044 {
		t.Fatalf("mode %v: another account could not read the stop", fi.Mode())
	}
	lifted, err := Lift(home, now.Add(time.Minute))
	if err != nil || !lifted {
		t.Fatalf("Lift = %v, %v", lifted, err)
	}
	if got := readNoPin(home); got.Stop != nil {
		t.Fatalf("after the lift Read = %+v", got)
	}
	log, err := os.ReadFile(abcdhome.Path(home, LiftLogFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(log, []byte(`"set"`)) || !bytes.Contains(log, []byte(`"lifted"`)) {
		t.Fatalf("the log holds %q", log)
	}
	if again, err := Lift(home, now); err != nil || again {
		t.Fatalf("a second lift = %v, %v; want false, nil", again, err)
	}
}

// TestNoFolderStaysInTheHome: with no folder chosen the stop lives in the
// account's own home and reaches this account's runs; every other part works.
func TestNoFolderStaysInTheHome(t *testing.T) {
	home := t.TempDir()
	set, err := Set(home, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if want := abcdhome.Path(home, HomeStopFile); set.Path != want {
		t.Fatalf("set at %q, want %q", set.Path, want)
	}
	if got := readNoPin(home); got.Stop == nil {
		t.Fatalf("Read = %+v, want a stop", got)
	}
}

// TestContentIsNeverTheRequest: a file holding anything at all, within its
// bound, is a request to stop and nothing more: nothing in it reaches the stop.
func TestContentIsNeverTheRequest(t *testing.T) {
	home, folder := world(t)
	planted := `{"set_by":"someone-else","set_at":"1999-01-01T00:00:00Z","lift":true}` + "\x1b[2J"
	if err := os.WriteFile(stopPath(folder, os.Getuid()), []byte(planted), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readNoPin(home)
	if got.Stop == nil {
		t.Fatalf("Read = %+v, want a stop", got)
	}
	if got.Stop.SetBy == "someone-else" || got.Stop.SetAt.Year() == 1999 {
		t.Fatalf("the file's text reached the stop: %+v", got.Stop)
	}
}

// refusedOnce asserts a refusal naming the path and no stop.
func refusedOnce(t *testing.T, got Reading, path string) {
	t.Helper()
	if got.Stop != nil {
		t.Fatalf("the refused file is a stop: %+v", got.Stop)
	}
	if len(got.Refused) != 1 || !strings.Contains(got.Refused[0], filepath.Base(path)) {
		t.Fatalf("Refused = %q, want one line naming %s", got.Refused, path)
	}
}

// TestSymlinkRefused: a symlink is refused and never followed, whatever it
// points at.
func TestSymlinkRefused(t *testing.T) {
	home, folder := world(t)
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := stopPath(folder, os.Getuid())
	if err := os.Symlink(target, p); err != nil {
		t.Fatal(err)
	}
	refusedOnce(t, readNoPin(home), p)
}

// TestFIFORefusedWithoutBlocking: a FIFO at the path neither blocks the read
// nor counts.
func TestFIFORefusedWithoutBlocking(t *testing.T) {
	home, folder := world(t)
	p := stopPath(folder, os.Getuid())
	if err := syscall.Mkfifo(p, 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan Reading, 1)
	go func() { done <- readNoPin(home) }()
	select {
	case got := <-done:
		refusedOnce(t, got, p)
	case <-time.After(10 * time.Second):
		t.Fatal("a FIFO blocked the read")
	}
}

// TestOversizedRefused: a file past the bound is refused, not read.
func TestOversizedRefused(t *testing.T) {
	home, folder := world(t)
	p := stopPath(folder, os.Getuid())
	if err := os.WriteFile(p, bytes.Repeat([]byte("x"), MaxStopBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	refusedOnce(t, readNoPin(home), p)
}

// TestAnotherAccountRefused: a file any account but the person's wrote is
// refused, naming the account that wrote it. A test cannot make a file another
// account owns, so it names a person who is not this account.
func TestAnotherAccountRefused(t *testing.T) {
	_, folder := world(t)
	p := stopPath(folder, os.Getuid())
	if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	stop, refused, _ := judge(p, os.Getuid()+1)
	if stop != nil || !strings.Contains(refused, "another account") {
		t.Fatalf("judge = %+v, %q", stop, refused)
	}
}

// TestAFolderAnotherAccountOwnsIsRefused: its owner could remove the stop.
func TestAFolderAnotherAccountOwnsIsRefused(t *testing.T) {
	_, folder := world(t)
	if err := checkFolder(folder, os.Getuid()+1); err == nil || !strings.Contains(err.Error(), "neither root nor") {
		t.Fatalf("checkFolder = %v", err)
	}
}

// TestAPlantedStopDirectoryIsRefused: one planted before the person chose the
// folder is refused at setup rather than holding their stops; found later, it is
// no stop and is named.
func TestAPlantedStopDirectoryIsRefused(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	folder := filepath.Join(base, "shared")
	for _, d := range []string{home, folder} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dir := filepath.Join(folder, stopDirName(os.Getuid()))
	if err := os.Symlink(t.TempDir(), dir); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSetting(home, folder, ""); err == nil {
		t.Fatal("setup accepted a planted directory")
	}
	if _, refused, _ := checkStopDir(folder, os.Getuid()); !strings.Contains(refused, "refused") {
		t.Fatalf("checkStopDir = %q", refused)
	}
	if err := os.Mkdir(filepath.Join(folder, stopDirName(os.Getuid()+1)), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, refused, _ := checkStopDir(folder, os.Getuid()+1); !strings.Contains(refused, "refused") {
		t.Fatalf("another account's directory: %q", refused)
	}
}

// TestUnstickyFolderRefused: a folder others can write without the sticky bit
// lets any account remove the person's stop, so setup refuses it and so does
// every read.
func TestUnstickyFolderRefused(t *testing.T) {
	home := t.TempDir()
	folder := filepath.Join(t.TempDir(), "open")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(folder, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSetting(home, folder, ""); err == nil || !strings.Contains(err.Error(), "sticky") {
		t.Fatalf("WriteSetting over an unsticky folder = %v", err)
	}
	// A folder made unsticky after setup is refused at the read.
	if err := os.Chmod(folder, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSetting(home, folder, ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(folder, 0o777); err != nil {
		t.Fatal(err)
	}
	got := readNoPin(home)
	if got.Stop != nil || len(got.Refused) != 1 || !strings.Contains(got.Refused[0], "sticky") {
		t.Fatalf("Read = %+v", got)
	}
}

// TestSymlinkedFolderRefused: a folder reached through a link is not the
// folder chosen.
func TestSymlinkedFolderRefused(t *testing.T) {
	home := t.TempDir()
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSetting(home, link, ""); err == nil {
		t.Fatal("a symlinked folder was accepted")
	}
}

// TestSetOverARefusedFileSaysSo: the person is never told a stop took hold
// while a file the runs refuse occupies its path.
func TestSetOverARefusedFileSaysSo(t *testing.T) {
	home, folder := world(t)
	p := stopPath(folder, os.Getuid())
	if err := os.Symlink("/nonexistent", p); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(home, time.Now()); err == nil || !strings.Contains(err.Error(), filepath.Base(p)) {
		t.Fatalf("Set over a symlink = %v", err)
	}
	if fi, err := os.Lstat(p); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the symlink was replaced: %v %v", fi, err)
	}
}

// TestSettingIsTheAccountsOwn: a setting others can write is not read as the
// person's word; the stop is refused loudly rather than read from another
// folder.
func TestSettingIsTheAccountsOwn(t *testing.T) {
	home, _ := world(t)
	p := abcdhome.Path(home, SettingFile)
	if err := os.Chmod(p, 0o666); err != nil {
		t.Fatal(err)
	}
	got := readNoPin(home)
	if got.Stop != nil || len(got.Refused) != 1 || !strings.Contains(got.Refused[0], SettingFile) {
		t.Fatalf("Read = %+v", got)
	}
}

// TestDigestPageRewrittenWhole: a page is written whole and readable by every
// account, a second write replacing the first; without a folder nothing is
// written.
func TestDigestPageRewrittenWhole(t *testing.T) {
	home, folder := world(t)
	day := time.Date(2026, 11, 4, 18, 0, 0, 0, time.UTC)
	p, err := WritePage(home, day, []byte("first\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(folder, DigestDirFor(os.Getuid()), "2026-11-04.md"); p != want {
		t.Fatalf("page at %q, want %q", p, want)
	}
	if _, err := WritePage(home, day, []byte("second\n")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil || string(got) != "second\n" {
		t.Fatalf("page holds %q, %v", got, err)
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm()&0o044 != 0o044 {
		t.Fatalf("page mode %v", fi.Mode())
	}
	if p, err := WritePage(t.TempDir(), day, []byte("x")); err != nil || p != "" {
		t.Fatalf("no folder: %q, %v", p, err)
	}
}

// TestAFolderBelowAnOpenParentIsRefused: whoever can rename a parent can move the
// whole folder away, stop and all.
func TestAFolderBelowAnOpenParentIsRefused(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "open")
	folder := filepath.Join(parent, "shared")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := checkFolder(folder, os.Getuid()); err == nil || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("a folder below a parent anyone can write: %v", err)
	}
	if err := os.Chmod(parent, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if err := checkFolder(folder, os.Getuid()); err != nil {
		t.Fatalf("a sticky parent keeps others from moving the folder: %v", err)
	}
}

// TestTheSuggestedFolderQualifies: what setup offers on this platform passes the
// check every run makes of it.
func TestTheSuggestedFolderQualifies(t *testing.T) {
	if _, err := os.Lstat(Suggested()); err != nil {
		t.Skipf("no %s here", Suggested())
	}
	if err := checkFolder(Suggested(), os.Getuid()); err != nil {
		t.Fatal(err)
	}
}

// TestLiftLeavesAPlantedDirectoryAlone: a link planted where the person's
// directory goes reaches no stop, and a lift does not remove what it points at.
func TestLiftLeavesAPlantedDirectoryAlone(t *testing.T) {
	home := t.TempDir()
	base := t.TempDir()
	folder := filepath.Join(base, "shared")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSetting(home, folder, ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(folder, stopDirName(os.Getuid()))); err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(base, "elsewhere")
	if err := os.Mkdir(elsewhere, 0o700); err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(elsewhere, StopFileName)
	if err := os.WriteFile(kept, []byte("mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(folder, stopDirName(os.Getuid()))); err != nil {
		t.Fatal(err)
	}
	if _, err := Lift(home, time.Now()); err == nil {
		t.Fatal("a lift through a planted link succeeded")
	}
	if _, err := os.Stat(kept); err != nil {
		t.Fatalf("the lift reached through the link: %v", err)
	}
}

// pinOf is the pin a run would take from the setting as it stands.
func pinOf(t *testing.T, home string) *Pin {
	t.Helper()
	s, ok, err := ReadSetting(home)
	if err != nil || !ok {
		t.Fatalf("the setting: %v, %v", ok, err)
	}
	return &Pin{Folder: s.Folder, Person: s.Person, PersonUID: s.PersonUID}
}

// TestAPinnedReadSeesTheStopWhateverTheSettingSays: a run reads the folder it
// started under, and a setting gone or changed since holds it and is said.
func TestAPinnedReadSeesTheStopWhateverTheSettingSays(t *testing.T) {
	home, _ := world(t)
	pin := pinOf(t, home)
	if got := ReadPinned(home, pin); got.Stop != nil || got.Held != "" {
		t.Fatalf("nothing set: %+v", got)
	}
	if _, err := Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	setting := abcdhome.Path(home, SettingFile)
	if err := os.Remove(setting); err != nil {
		t.Fatal(err)
	}
	got := ReadPinned(home, pin)
	if got.Stop == nil {
		t.Fatalf("the stop was not read with the setting gone: %+v", got)
	}
	if got.Held == "" {
		t.Fatalf("a setting gone during the run is not said: %+v", got)
	}
}

// TestAChangedSettingHoldsAPinnedRead: the setting naming another folder is no
// lift of the stop in the first, and holds the run.
func TestAChangedSettingHoldsAPinnedRead(t *testing.T) {
	home, _ := world(t)
	pin := pinOf(t, home)
	if _, err := Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeSettingFile(home, Setting{SchemaVersion: settingVersion, Folder: other, Person: pin.Person, PersonUID: pin.PersonUID}); err != nil {
		t.Fatal(err)
	}
	if got := ReadPinned(home, pin); got.Stop == nil || got.Held == "" {
		t.Fatalf("Read = %+v", got)
	}
}

// TestAnUnreadableSettingHoldsARead: a setting that cannot be read names an intent
// to be reached this read cannot honour, pinned or not.
func TestAnUnreadableSettingHoldsARead(t *testing.T) {
	home, _ := world(t)
	if err := os.Chmod(abcdhome.Path(home, SettingFile), 0o666); err != nil {
		t.Fatal(err)
	}
	if got := readNoPin(home); got.Held == "" {
		t.Fatalf("Read = %+v", got)
	}
}

// wellKnown stands a sticky folder in for the platform's shared folder.
func wellKnown(t *testing.T) string {
	t.Helper()
	folder := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(folder, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(folder, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	old := WellKnownFolder
	WellKnownFolder = func() string { return folder }
	t.Cleanup(func() { WellKnownFolder = old })
	return folder
}

// TestTheWellKnownFolderIsReadWithNoSetting: the platform's shared folder is read
// whatever the setting says, by the account that owns the stop's directory, under
// the same judgement as any stop.
func TestTheWellKnownFolderIsReadWithNoSetting(t *testing.T) {
	folder := wellKnown(t)
	personHome := t.TempDir()
	if _, err := WriteSetting(personHome, folder, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(personHome, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := readNoPin(t.TempDir()); got.Stop == nil || got.Stop.Path != stopPath(folder, os.Getuid()) {
		t.Fatalf("Read = %+v", got)
	}
	if lifted, err := Lift(personHome, time.Now()); err != nil || !lifted {
		t.Fatalf("Lift = %v, %v", lifted, err)
	}
	if err := os.WriteFile(stopPath(folder, os.Getuid()), bytes.Repeat([]byte("x"), MaxStopBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readNoPin(t.TempDir()); got.Stop != nil || len(got.Refused) != 1 {
		t.Fatalf("an oversized file: %+v", got)
	}
}

// TestLiftReachesTheWellKnownStopWithNoSetting: a stop every run still reads is
// one its person can lift, whatever has become of the setting.
func TestLiftReachesTheWellKnownStopWithNoSetting(t *testing.T) {
	folder := wellKnown(t)
	home := t.TempDir()
	if _, err := WriteSetting(home, folder, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(abcdhome.Path(home, SettingFile)); err != nil {
		t.Fatal(err)
	}
	if lifted, err := Lift(home, time.Now()); err != nil || !lifted {
		t.Fatalf("Lift = %v, %v", lifted, err)
	}
	if _, err := os.Lstat(stopPath(folder, os.Getuid())); !os.IsNotExist(err) {
		t.Fatalf("the stop stands: %v", err)
	}
}

// TestTheAccountHomeComesFromTheAccountDatabase: the environment does not move it.
func TestTheAccountHomeComesFromTheAccountDatabase(t *testing.T) {
	want, err := databaseHome()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(t.TempDir(), "elsewhere"))
	got, err := databaseHome()
	if err != nil || got != want || got == os.Getenv("HOME") {
		t.Fatalf("databaseHome = %q, %v, under HOME %q; want %q", got, err, os.Getenv("HOME"), want)
	}
}

// TestTheSettingDoesNotMoveFromUnderAStop: a stop stands where a worker reads it, so
// the folder is not moved away from it, and its lift still reaches it.
func TestTheSettingDoesNotMoveFromUnderAStop(t *testing.T) {
	home, folder := world(t)
	if _, err := Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSetting(home, other, ""); err == nil {
		t.Fatal("the setting moved from under a stop")
	}
	if s, _, _ := ReadSetting(home); s.Folder != folder {
		t.Fatalf("the setting names %q", s.Folder)
	}
	if _, err := WriteSetting(home, folder, ""); err != nil {
		t.Fatalf("choosing the same folder again: %v", err)
	}
	if lifted, err := Lift(home, time.Now()); err != nil || !lifted {
		t.Fatalf("Lift = %v, %v", lifted, err)
	}
	if _, err := WriteSetting(home, other, ""); err != nil {
		t.Fatalf("moving with no stop standing: %v", err)
	}
}

// TestAnUnknownHomeHoldsTheRead: no home is not no stop: the read is held, and the
// platform's shared folder is read all the same.
func TestAnUnknownHomeHoldsTheRead(t *testing.T) {
	if got := readNoPin(""); got.Held == "" {
		t.Fatalf("Read = %+v", got)
	}
	folder := wellKnown(t)
	personHome := t.TempDir()
	if _, err := WriteSetting(personHome, folder, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(personHome, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := readNoPin(""); got.Stop == nil || got.Held == "" {
		t.Fatalf("Read = %+v", got)
	}
}

// plantedByAnother leaves a stop in the platform's shared folder under another
// account's name.
func plantedByAnother(t *testing.T) (folder string, other int) {
	t.Helper()
	folder = wellKnown(t)
	other = os.Getuid() + 1
	if err := os.Mkdir(filepath.Join(folder, StopDirName+"-"+strconv.Itoa(other)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, StopDirName+"-"+strconv.Itoa(other), StopFileName), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return folder, other
}

// TestAnotherAccountsStopInThePlatformFolderIsNotHonoured: with no setting or pin
// naming that account, what it left is no stop and no refusal, so it neither halts a
// run nor stands in the way of a take-over.
func TestAnotherAccountsStopInThePlatformFolderIsNotHonoured(t *testing.T) {
	plantedByAnother(t)
	got := readNoPin(t.TempDir())
	if got.Stop != nil || len(got.Refused) != 0 || got.Held != "" {
		t.Fatalf("Read = %+v", got)
	}
	if len(got.Foreign) != 1 || !strings.Contains(got.Foreign[0], "not honoured") {
		t.Fatalf("what another account left goes unsaid: %+v", got)
	}
}

// TestThePlatformFolderIsReadForTheAccountThePinNames: the account a run started
// under is expected, whatever has become of the setting.
func TestThePlatformFolderIsReadForTheAccountThePinNames(t *testing.T) {
	folder, other := plantedByAnother(t)
	home := t.TempDir()
	pinned := Pin{Folder: filepath.Join(t.TempDir(), "elsewhere"), PersonUID: other}
	got := ReadPinned(home, &pinned)
	joined := strings.Join(got.Refused, " ")
	if !strings.Contains(joined, fsutil.DisplayPath(folder)) {
		t.Fatalf("the pin's account's folder was not judged: %+v", got)
	}
}

// TestLiftSaysWhatItCannotReach: a stop this account's lift cannot reach is named,
// so no one is told there is none.
func TestLiftSaysWhatItCannotReach(t *testing.T) {
	folder, _ := plantedByAnother(t)
	home := t.TempDir()
	if lifted, err := Lift(home, time.Now()); err != nil || lifted {
		t.Fatalf("Lift = %v, %v", lifted, err)
	}
	got := Unreached(home)
	if len(got) != 1 || !strings.Contains(got[0], fsutil.DisplayPath(folder)) {
		t.Fatalf("Unreached = %q", got)
	}
	if _, err := os.Lstat(filepath.Join(folder, StopDirName+"-"+strconv.Itoa(os.Getuid()+1), StopFileName)); err != nil {
		t.Fatalf("the lift removed what it could not reach: %v", err)
	}
}

// TestTheStopsDirectoryCarriesThePersonsNumber: one account's directory is not
// another's to contest.
func TestTheStopsDirectoryCarriesThePersonsNumber(t *testing.T) {
	want := StopDirName + "-" + strconv.Itoa(os.Getuid())
	if got := stopDirName(os.Getuid()); got != want {
		t.Fatalf("stopDirName = %q, want %q", got, want)
	}
	home, folder := world(t)
	if _, err := Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(folder, want, StopFileName)); err != nil {
		t.Fatal(err)
	}
}

// unreadableBy makes p unreadable to this account for the test.
func unreadableBy(t *testing.T, p string) {
	t.Helper()
	if os.Getuid() == 0 {
		t.Skip("root reads everything")
	}
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, fi.Mode().Perm()) })
}

// TestAStopThatCannotBeReadHoldsTheRead: a folder the run started under or the setting
// names, whose stop cannot be read, is not read as no stop.
func TestAStopThatCannotBeReadHoldsTheRead(t *testing.T) {
	for name, target := range map[string]func(folder string) string{
		"the stop file":      func(f string) string { return stopPath(f, os.Getuid()) },
		"the stop directory": func(f string) string { return filepath.Join(f, stopDirName(os.Getuid())) },
	} {
		t.Run(name, func(t *testing.T) {
			home, folder := world(t)
			if _, err := Set(home, time.Now()); err != nil {
				t.Fatal(err)
			}
			pin := pinOf(t, home)
			unreadableBy(t, target(folder))
			if got := ReadPinned(home, pin); got.Stop != nil || got.Held == "" {
				t.Fatalf("pinned: %+v", got)
			}
			if got := readNoPin(home); got.Stop != nil || got.Held == "" {
				t.Fatalf("by the setting: %+v", got)
			}
		})
	}
}

// TestAFolderThatFailsItsCheckHoldsTheRead: the folder a run started under going away
// or failing its check is not read as no stop.
func TestAFolderThatFailsItsCheckHoldsTheRead(t *testing.T) {
	home, folder := world(t)
	pin := pinOf(t, home)
	if err := os.Chmod(folder, 0o777); err != nil {
		t.Fatal(err)
	}
	if got := ReadPinned(home, pin); got.Held == "" || len(got.Refused) == 0 {
		t.Fatalf("Read = %+v", got)
	}
}

// TestWhatIsNoStopStaysARefusal: a wrong owner, a non-regular file and an oversized
// file are refused and hold nothing, and the platform's folder holds nothing.
func TestWhatIsNoStopStaysARefusal(t *testing.T) {
	home, folder := world(t)
	if err := os.Symlink("/nonexistent", stopPath(folder, os.Getuid())); err != nil {
		t.Fatal(err)
	}
	if got := readNoPin(home); got.Held != "" || len(got.Refused) != 1 {
		t.Fatalf("a link: %+v", got)
	}
	known := wellKnown(t)
	if err := os.Mkdir(filepath.Join(known, stopDirName(os.Getuid())), 0o755); err != nil {
		t.Fatal(err)
	}
	unreadableBy(t, filepath.Join(known, stopDirName(os.Getuid())))
	if got := readNoPin(t.TempDir()); got.Held != "" {
		t.Fatalf("the platform's folder held the read: %+v", got)
	}
}

// TestAPlantedLinkIsNotChangedThrough: what stands at the person's directory is judged
// before anything is changed, so a link does not have its target's mode changed.
func TestAPlantedLinkIsNotChangedThrough(t *testing.T) {
	base := t.TempDir()
	folder := filepath.Join(base, "shared")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(folder, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(folder, stopDirName(os.Getuid()))); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSetting(t.TempDir(), folder, ""); err == nil {
		t.Fatal("a setting over a planted link succeeded")
	}
	fi, err := os.Stat(real)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o700 {
		t.Fatalf("the link's target became %v", fi.Mode().Perm())
	}
}

// TestALinkAmongTheParentsIsHeldToTheRule: a link's containing directory is judged as
// any parent is, not only where the link leads.
func TestALinkAmongTheParentsIsHeldToTheRule(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(filepath.Join(real, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	open := filepath.Join(base, "open")
	if err := os.Mkdir(open, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(open, "link")); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(open, "link", "shared")
	if err := checkFolder(folder, os.Getuid()); err != nil {
		t.Fatalf("a link in a closed parent: %v", err)
	}
	if err := os.Chmod(open, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := checkFolder(folder, os.Getuid()); err == nil {
		t.Fatal("a link in a parent anyone can write passed")
	}
}

// TestANoticeNamesAnAccountOnlyWhereTheDirectoryIsItsOwn: a name alone proves nothing.
func TestANoticeNamesAnAccountOnlyWhereTheDirectoryIsItsOwn(t *testing.T) {
	folder, other := plantedByAnother(t)
	got := readNoPin(t.TempDir())
	if len(got.Foreign) != 1 || strings.Contains(got.Foreign[0], accountName(other)) {
		t.Fatalf("a directory another account did not make named it: %q", got.Foreign)
	}
	own := filepath.Join(folder, stopDirName(os.Getuid()))
	if err := os.Mkdir(own, 0o755); err != nil {
		t.Fatal(err)
	}
	lines := foreignLines([]foreignStop{{folder: folder, path: filepath.Join(own, StopFileName), dir: own, owner: os.Getuid()}})
	if len(lines) != 1 || !strings.Contains(lines[0], accountName(os.Getuid())) {
		t.Fatalf("a directory its account owns: %q", lines)
	}
}

// TestAFloodOfForeignDirectoriesFillsNothing: the notice is capped.
func TestAFloodOfForeignDirectoriesFillsNothing(t *testing.T) {
	folder := wellKnown(t)
	for i := 1; i <= 40; i++ {
		dir := filepath.Join(folder, StopDirName+"-"+strconv.Itoa(os.Getuid()+i))
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, StopFileName), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := readNoPin(t.TempDir()).Foreign
	if len(got) > maxForeignLines+1 || !strings.Contains(got[len(got)-1], "more") {
		t.Fatalf("the notice: %d lines, %q", len(got), got)
	}
}

// TestAPlantedNumberNamesItsRemedy: the refusal names the entry and what to do.
func TestAPlantedNumberNamesItsRemedy(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/nonexistent", filepath.Join(folder, stopDirName(os.Getuid()))); err != nil {
		t.Fatal(err)
	}
	_, err := WriteSetting(t.TempDir(), folder, "")
	if err == nil || !strings.Contains(err.Error(), stopDirName(os.Getuid())) || !strings.Contains(err.Error(), "administrator") {
		t.Fatalf("the refusal: %v", err)
	}
}

// TestANoticeIsSafeToPrintWhereverItGoes: what a notice says is made safe where it is
// composed, since it reaches JSON and a board as well as a terminal.
func TestANoticeIsSafeToPrintWhereverItGoes(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "x\x1b[31m")
	if err := os.Mkdir(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	line := foreignStop{folder: bad, path: filepath.Join(bad, "f"), dir: filepath.Join(bad, "d"), owner: os.Getuid()}.say("then")
	if strings.ContainsRune(line, 0x1b) {
		t.Fatalf("%q", line)
	}
}

// TestAStopDirectoryThatCannotBeChangedSaysWhy: a directory that is the person's own and
// cannot be changed is no planted entry, and its reason is kept.
func TestAStopDirectoryThatCannotBeChangedSaysWhy(t *testing.T) {
	old := chmodStopDir
	t.Cleanup(func() { chmodStopDir = old })
	chmodStopDir = func(*os.Root, string, os.FileMode) error { return syscall.EROFS }
	folder := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	err := ensureStopDir(folder, os.Getuid())
	if !errors.Is(err, syscall.EROFS) || strings.Contains(err.Error(), "administrator") {
		t.Fatalf("ensureStopDir = %v", err)
	}
}

// TestASecondSetKeepsTheStopThatStands: typing the stop again leaves it as it is, at the time
// it was set, and says so; the runs it has reached read it as one stop.
func TestASecondSetKeepsTheStopThatStands(t *testing.T) {
	home, _ := world(t)
	first, err := Set(home, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if first.Stood {
		t.Fatalf("a new stop says it stood: %+v", first)
	}
	time.Sleep(20 * time.Millisecond)
	again, err := Set(home, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !again.Stood || !again.SetAt.Equal(first.SetAt) {
		t.Fatalf("the second set: %+v, the first: %+v", again, first)
	}
	if r := ReadPinned(home, nil); r.Stop == nil || !r.Stop.SetAt.Equal(first.SetAt) {
		t.Fatalf("the stop in force moved: %+v, the first: %+v", r.Stop, first)
	}
}

// TestEachAccountHasItsOwnPageDirectory: a page written by one account leaves another's
// directory alone, so a second account running under the same stop has somewhere to write.
func TestEachAccountHasItsOwnPageDirectory(t *testing.T) {
	home, folder := world(t)
	if got, want := DigestDirFor(501), DigestDir+"-501"; got != want {
		t.Fatalf("the directory for 501 is %q, want %q", got, want)
	}
	if err := os.WriteFile(filepath.Join(folder, DigestDir), []byte("something else\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := WritePage(home, time.Date(2026, 11, 4, 18, 0, 0, 0, time.UTC), []byte("mine\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(folder, DigestDirFor(os.Getuid()), "2026-11-04.md"); p != want {
		t.Fatalf("page at %q, want %q", p, want)
	}
}

// TestAPlantedPageDirectoryIsRefused: a link or a file at the account's directory is not
// written through.
func TestAPlantedPageDirectoryIsRefused(t *testing.T) {
	day := time.Date(2026, 11, 4, 18, 0, 0, 0, time.UTC)
	t.Run("a file", func(t *testing.T) {
		home, folder := world(t)
		if err := os.WriteFile(filepath.Join(folder, DigestDirFor(os.Getuid())), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := WritePage(home, day, []byte("mine\n")); err == nil {
			t.Fatal("a page was written through a file")
		}
	})
	t.Run("a link", func(t *testing.T) {
		home, folder := world(t)
		elsewhere := t.TempDir()
		if err := os.Symlink(elsewhere, filepath.Join(folder, DigestDirFor(os.Getuid()))); err != nil {
			t.Fatal(err)
		}
		if _, err := WritePage(home, day, []byte("mine\n")); err == nil {
			t.Fatal("a page was written through a link")
		}
		if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
			t.Fatalf("the page reached through the link: %v", entries)
		}
	})
}
