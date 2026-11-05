// Package runstop is the stop a person sets from their own account, which every
// unattended run on the machine reads before its next move (itd-2611041549471712,
// dcn-2611041550139264).
//
// The person's seat and the run's seat may be two accounts on one machine, and
// no store abcd has can carry a word between them, since each is owner-only. So
// the person chooses a shared folder at setup, on a machine no other person
// shares, and the stop is a file there: the one file outside abcd's owner-only
// rule. Declining leaves a stop in the account's own home, which reaches only
// that account's runs.
//
// The file is an input from outside the reading account, and it is a request to
// stop and nothing more. It is opened once with O_NOFOLLOW and O_NONBLOCK
// (fsutil.OpenRegular) and judged on that same descriptor: a regular file, within
// MaxStopBytes, owned by the account the person named at setup. Who set it and
// when come from the descriptor. No byte of it is read, so nothing written there
// can reach a run, a record or a terminal. Anything else at the path is refused
// and named, and is no stop.
package runstop

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const (
	// StopDirName is the person's own directory in the shared folder,
	// StopFileName the stop inside it, and HomeStopFile the stop in the account's
	// own home where no folder was chosen.
	StopDirName  = "abcd-stop"
	StopFileName = "abcd-stop.json"
	HomeStopFile = "stop.json"
	// SettingFile holds the folder and the person's account, in abcd's home.
	SettingFile = "shared-folder.json"
	// LiftLogFile is the person's own account of each stop set and lifted, in
	// abcd's home.
	LiftLogFile = "stop-log.jsonl"
	// DigestDir holds the dated pages in the shared folder.
	DigestDir = "abcd-digest"
	// MaxStopBytes bounds the stop file: what the verb writes is a few dozen
	// bytes.
	MaxStopBytes = 4 << 10

	maxSettingBytes = 4 << 10
	settingVersion  = 1
	// sharedFileMode is what another account must be able to read.
	sharedFileMode os.FileMode = 0o644
	sharedDirMode  os.FileMode = 0o755
)

// Setting is the choice made at setup, in each account that takes part.
type Setting struct {
	SchemaVersion int `json:"schema_version"`
	// Folder is the shared folder, absolute.
	Folder string `json:"folder"`
	// Person and PersonUID name the account whose stop this account honours.
	Person    string `json:"person"`
	PersonUID int    `json:"person_uid"`
}

// Stop is a stop in force. Every field but Path comes from the file's
// descriptor, never its text.
type Stop struct {
	SetBy string    `json:"set_by"`
	UID   int       `json:"uid"`
	SetAt time.Time `json:"set_at"`
	Path  string    `json:"path"`
}

// Pin is the shared folder and the person a run started under.
type Pin struct {
	Folder    string `json:"folder"`
	Person    string `json:"person"`
	PersonUID int    `json:"person_uid"`
}

// Reading is what one read found: the stop in force, if any, and one line for
// each thing refused on the way, which is no stop. Held says why the read cannot
// be trusted to have seen a stop, which holds the run; "" when it can.
type Reading struct {
	Stop    *Stop
	Refused []string
	Held    string
}

// AccountHome is the home a stop is read from, and databaseHome the production
// value. A test names a fixture.
var AccountHome = databaseHome

// databaseHome is this account's home: the account database's entry, or for an
// account the database does not list, the platform's own answer. An error is a home
// that cannot be found, which every reader treats as unknown and never as no stop.
func databaseHome() (string, error) {
	if u, err := user.LookupId(strconv.Itoa(os.Getuid())); err == nil && filepath.IsAbs(u.HomeDir) {
		return u.HomeDir, nil
	}
	return os.UserHomeDir()
}

// WellKnownFolder is the platform's shared folder, read whatever the setting says;
// a test empties it.
var WellKnownFolder = Suggested

// PinOf is the pin a run takes from this account's setting; nil when none was
// chosen or it cannot be read.
func PinOf(home string) *Pin {
	s, ok, err := ReadSetting(home)
	if err != nil || !ok {
		return nil
	}
	return &Pin{Folder: s.Folder, Person: s.Person, PersonUID: s.PersonUID}
}

// Suggested is the folder setup offers: a real, sticky, shared directory on this
// platform.
func Suggested() string {
	if runtime.GOOS == "darwin" {
		return "/Users/Shared"
	}
	return "/var/tmp"
}

// stopPath is where the person's stop lives in the folder.
func stopPath(folder string) string { return filepath.Join(folder, StopDirName, StopFileName) }

// checkFolder refuses a folder the stop cannot rest on: one that is not a real
// directory, one others can write without the sticky bit, or one owned by any
// account but root and the person, and the same of each parent above it, which
// could move the whole folder: each lets another account move the person's stop
// away.
func checkFolder(folder string, person int) error {
	if !filepath.IsAbs(folder) || filepath.Clean(folder) != folder {
		return fmt.Errorf("the shared folder %q is not a clean absolute path", folder)
	}
	fi, err := os.Lstat(folder)
	if err != nil {
		return fmt.Errorf("the shared folder %s cannot be read: %w", fsutil.DisplayPath(folder), err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("the shared folder %s is not a real directory", fsutil.DisplayPath(folder))
	}
	if fi.Mode().Perm()&0o022 != 0 && fi.Mode()&os.ModeSticky == 0 {
		return fmt.Errorf("the shared folder %s can be written by other accounts and has no sticky bit, so any of them could remove a stop", fsutil.DisplayPath(folder))
	}
	if uid, ok := ownerOf(fi); !ok || (uid != 0 && uid != person) {
		return fmt.Errorf("the shared folder %s is owned by an account that is neither root nor the person's, which could remove a stop", fsutil.DisplayPath(folder))
	}
	return checkFolderParents(folder, person)
}

// checkFolderParents walks the folder's real parents, each held to the folder's
// own rule, since whoever can rename a parent can move the folder away.
func checkFolderParents(folder string, person int) error {
	dir, err := filepath.EvalSymlinks(filepath.Dir(folder))
	if err != nil {
		return fmt.Errorf("the parents of the shared folder %s cannot be read: %w", fsutil.DisplayPath(folder), err)
	}
	for {
		fi, err := os.Lstat(dir)
		if err != nil {
			return fmt.Errorf("the parents of the shared folder %s cannot be read: %w", fsutil.DisplayPath(folder), err)
		}
		uid, known := ownerOf(fi)
		open := fi.Mode().Perm()&0o022 != 0 && fi.Mode()&os.ModeSticky == 0
		if open || !known || (uid != 0 && uid != person) {
			return fmt.Errorf("the shared folder %s has a parent, %s, that another account can change, which could move a stop away", fsutil.DisplayPath(folder), fsutil.DisplayPath(dir))
		}
		next := filepath.Dir(dir)
		if next == dir {
			return nil
		}
		dir = next
	}
}

// checkStopDir judges the person's directory: ok is false when it is absent,
// which is no stop and nothing to refuse. Anything but a real directory the person
// owns and no one else can write is refused.
func checkStopDir(folder string, person int) (ok bool, refused string) {
	dir := filepath.Join(folder, StopDirName)
	fi, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return false, ""
	}
	if err != nil {
		return false, fmt.Sprintf("the stop directory %s cannot be read, so it holds no stop: %v", fsutil.DisplayPath(dir), err)
	}
	uid, known := ownerOf(fi)
	if !fi.IsDir() || !known || uid != person || fi.Mode().Perm()&0o022 != 0 {
		return false, fmt.Sprintf("the stop directory %s is refused: it is not a directory the person's account alone can write, so it holds no stop", fsutil.DisplayPath(dir))
	}
	return true, ""
}

// ensureStopDir makes the person's directory, refusing one that stands and is not
// theirs: planted before they chose the folder, it would otherwise hold every stop
// they set.
func ensureStopDir(folder string, person int) error {
	root, err := os.OpenRoot(folder)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Mkdir(StopDirName, sharedDirMode); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	if err := root.Chmod(StopDirName, sharedDirMode); err != nil {
		return fmt.Errorf("the stop directory in %s is not this account's: %w", fsutil.DisplayPath(folder), err)
	}
	if ok, refused := checkStopDir(folder, person); !ok {
		if refused == "" {
			refused = "the stop directory is gone after it was made"
		}
		return errors.New(refused)
	}
	return nil
}

// ReadSetting reads this account's setting: ok is false when none was chosen.
// It is the account's word, so it passes the guards every such file passes.
func ReadSetting(home string) (Setting, bool, error) {
	data, refusal, err := fsutil.ReadHomeDeclaration(home, abcdhome.Rel(SettingFile), maxSettingBytes)
	if refusal == fsutil.DeclarationAbsent {
		return Setting{}, false, nil
	}
	if err != nil {
		return Setting{}, false, fmt.Errorf("%s is not safely this account's: %w", abcdhome.Display(SettingFile), err)
	}
	var s Setting
	if err := json.Unmarshal(data, &s); err != nil || s.SchemaVersion != settingVersion || s.Folder == "" {
		return Setting{}, false, fmt.Errorf("%s does not parse as a shared-folder setting", abcdhome.Display(SettingFile))
	}
	return s, true, nil
}

// WriteSetting records the choice: the folder, and the account whose stop this
// one honours (this account when person is empty). The folder is judged first,
// and nothing is written when it fails.
func WriteSetting(home, folder, person string) (Setting, error) {
	var u *user.User
	var err error
	if person == "" {
		u, err = user.LookupId(strconv.Itoa(os.Getuid()))
	} else {
		u, err = user.Lookup(person)
	}
	if err != nil {
		return Setting{}, fmt.Errorf("the account %q cannot be found on this machine: %w", person, err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return Setting{}, fmt.Errorf("the account %q has no numeric id", u.Username)
	}
	if err := checkFolder(folder, uid); err != nil {
		return Setting{}, err
	}
	// A stop stands where every worker reads it, so the choice does not move from
	// under it: the person lifts it first.
	if was, chosen, err := ReadSetting(home); err == nil && chosen && (was.Folder != folder || was.PersonUID != uid) {
		if stop, _ := judge(stopPath(was.Folder), was.PersonUID); stop != nil {
			return Setting{}, fmt.Errorf("a stop stands in %s, so the shared folder does not move from under it; lift it with `abcd implement stop --lift` first", fsutil.DisplayPath(was.Folder))
		}
	}
	s := Setting{SchemaVersion: settingVersion, Folder: folder, Person: u.Username, PersonUID: uid}
	if uid == os.Getuid() {
		if err := ensureStopDir(folder, uid); err != nil {
			return Setting{}, err
		}
	}
	if err := writeSettingFile(home, s); err != nil {
		return Setting{}, err
	}
	return s, nil
}

func writeSettingFile(home string, s Setting) error {
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer root.Close()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(root, SettingFile, append(data, '\n'), abcdhome.FileMode)
}

// ReadPinned is the one read every run makes before a move. It never fails: what
// it cannot judge is refused and named, and is no stop, so a planted file can
// neither halt a run nor pass unsaid. It reads the account's own stop, the folder
// the run started under, the folder the setting names, and the platform's shared
// folder for an account it expects. A setting that cannot be read, or that has
// gone or changed since the run took pin, leaves Held saying so: the read cannot
// be trusted to have seen a stop, and the run does not take it for none.
func ReadPinned(home string, pin *Pin) Reading {
	var out Reading
	var read []string
	if pin != nil {
		out.readFolder(pin.Folder, pin.PersonUID)
		read = append(read, pin.Folder)
	}
	var s Setting
	var ok bool
	var err error
	if home == "" {
		out.Held = "this account's home cannot be found, so whether a stop holds cannot be told"
	} else {
		stop, refused := judge(abcdhome.Path(home, HomeStopFile), os.Getuid())
		if out.Stop == nil {
			out.Stop = stop
		}
		if refused != "" {
			out.Refused = append(out.Refused, refused)
		}
		s, ok, err = ReadSetting(home)
	}
	switch {
	case home == "":
	case err != nil:
		out.Refused = append(out.Refused, err.Error())
		out.Held = abcdhome.Display(SettingFile) + " cannot be read, so whether a stop holds cannot be told"
	case !ok && pin != nil:
		out.Held = "the shared-folder setting this run started under has gone from " + abcdhome.Display(SettingFile)
	case ok && pin != nil && (s.Folder != pin.Folder || s.PersonUID != pin.PersonUID):
		out.Held = "the shared-folder setting in " + abcdhome.Display(SettingFile) + " is not the one this run started under"
	}
	if ok && !slices.Contains(read, s.Folder) {
		out.readFolder(s.Folder, s.PersonUID)
		read = append(read, s.Folder)
	}
	// The platform's shared folder is read for an account a run has reason to
	// expect: this one, the one the setting names, the one the run started under.
	// Anything another account left there is neither a stop nor a refusal.
	if folder := WellKnownFolder(); folder != "" && !slices.Contains(read, folder) {
		if owner, present := stopDirOwnerOf(folder); present &&
			(owner == os.Getuid() || (ok && owner == s.PersonUID) || (pin != nil && owner == pin.PersonUID)) {
			out.readFolder(folder, owner)
		}
	}
	return out
}

// readFolder judges one folder's stop as the person's, naming what it refuses.
func (r *Reading) readFolder(folder string, person int) {
	if err := checkFolder(folder, person); err != nil {
		r.Refused = append(r.Refused, err.Error())
		return
	}
	present, refused := checkStopDir(folder, person)
	if refused != "" {
		r.Refused = append(r.Refused, refused)
	}
	if !present {
		return
	}
	stop, refused := judge(stopPath(folder), person)
	if refused != "" {
		r.Refused = append(r.Refused, refused)
	}
	if r.Stop == nil {
		r.Stop = stop
	}
}

// stopDirOwnerOf is stopDirOwner; a test stands in another account, which it cannot
// create (SwapStopDirOwnerForTest).
var stopDirOwnerOf = stopDirOwner

// SwapStopDirOwnerForTest stands in who owns a stop's directory, and returns what
// restores it.
func SwapStopDirOwnerForTest(f func(string) (int, bool)) func() {
	old := stopDirOwnerOf
	stopDirOwnerOf = f
	return func() { stopDirOwnerOf = old }
}

// Unreached names each stop that stands where this account's lift cannot reach it,
// so a person is never told there is none: one another account left in the
// platform's shared folder, which that account lifts.
func Unreached(home string) []string {
	folder := WellKnownFolder()
	if folder == "" {
		return nil
	}
	if s, ok, err := ReadSetting(home); err == nil && ok && s.Folder == folder {
		return nil // the lift itself judges the folder the setting names
	}
	owner, present := stopDirOwnerOf(folder)
	if !present || owner == os.Getuid() {
		return nil
	}
	p := stopPath(folder)
	if _, err := os.Lstat(p); err != nil {
		return nil
	}
	return []string{fmt.Sprintf("this lift cannot reach the stop file %s: the account %s holds its directory, and lifts it from there", fsutil.DisplayPath(p), accountName(owner))}
}

// stopDirOwner is who owns the stop's directory in folder, which in the platform's
// shared folder is the person where no setting says; present is false when there
// is none.
func stopDirOwner(folder string) (owner int, present bool) {
	fi, err := os.Lstat(filepath.Join(folder, StopDirName))
	if err != nil {
		return 0, false
	}
	return ownerOf(fi)
}

// judge is the file's whole judgement, on one descriptor: absent is nothing,
// and anything but a regular file within the bound owned by want is refused.
func judge(p string, want int) (*Stop, string) {
	f, fi, err := fsutil.OpenRegular(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ""
	}
	if err != nil {
		return nil, fmt.Sprintf("the stop file %s is refused: it is not a regular file abcd can open (%v), so it is no stop", fsutil.DisplayPath(p), err)
	}
	defer f.Close()
	uid, ok := ownerOf(fi)
	if !ok {
		return nil, fmt.Sprintf("the stop file %s is refused: its owner cannot be read, so it is no stop", fsutil.DisplayPath(p))
	}
	if uid != want {
		return nil, fmt.Sprintf("the stop file %s is refused: another account (%s) wrote it, not the person's, so it is no stop", fsutil.DisplayPath(p), accountName(uid))
	}
	if fi.Size() > MaxStopBytes {
		return nil, fmt.Sprintf("the stop file %s is refused: it holds %d bytes, over the %d a stop file may hold, so it is no stop", fsutil.DisplayPath(p), fi.Size(), MaxStopBytes)
	}
	return &Stop{SetBy: accountName(uid), UID: uid, SetAt: fi.ModTime().UTC(), Path: p}, ""
}

// ownerOf is the owner fi was read with: the descriptor's, for judge, so a file
// swapped after any check by path is judged as the file that was opened.
func ownerOf(fi os.FileInfo) (int, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int(st.Uid), true
}

// accountName is the account's name, or its number when it has none.
func accountName(uid int) string {
	if u, err := user.LookupId(strconv.Itoa(uid)); err == nil && u.Username != "" {
		return u.Username
	}
	return "uid " + strconv.Itoa(uid)
}

// target is where this account's stop lives, and the folder that holds it.
func target(home string) (dir, leaf string, err error) {
	s, ok, err := ReadSetting(home)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return abcdhome.Path(home), HomeStopFile, nil
	}
	if err := checkFolder(s.Folder, s.PersonUID); err != nil {
		return "", "", err
	}
	if s.PersonUID != os.Getuid() {
		return "", "", fmt.Errorf("this account is not the one the setting names (%s), so a stop written here would be refused by every run", s.Person)
	}
	if err := ensureStopDir(s.Folder, s.PersonUID); err != nil {
		return "", "", err
	}
	return filepath.Join(s.Folder, StopDirName), StopFileName, nil
}

// Set writes the stop from this account. A path a run would refuse is reported
// and left alone: the person is never told a stop took hold that no run honours.
func Set(home string, now time.Time) (Stop, error) {
	dir, leaf, err := target(home)
	if err != nil {
		return Stop{}, err
	}
	p := filepath.Join(dir, leaf)
	if _, refused := judge(p, os.Getuid()); refused != "" {
		return Stop{}, errors.New(refused)
	}
	if leaf == HomeStopFile {
		h, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
		if err != nil {
			return Stop{}, err
		}
		h.Close()
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Stop{}, err
	}
	defer root.Close()
	body, err := json.Marshal(struct {
		Request string    `json:"request"`
		At      time.Time `json:"at"`
	}{"stop", now.UTC()})
	if err != nil {
		return Stop{}, err
	}
	if err := fsutil.WriteFileAtomicInRoot(root, leaf, append(body, '\n'), sharedFileMode); err != nil {
		return Stop{}, err
	}
	if err := root.Chmod(leaf, sharedFileMode); err != nil {
		return Stop{}, err
	}
	stop, refused := judge(p, os.Getuid())
	if refused != "" {
		return Stop{}, errors.New(refused)
	}
	if stop == nil {
		return Stop{}, fmt.Errorf("the stop file %s is gone after it was written", fsutil.DisplayPath(p))
	}
	// The stop stands whether or not its account line can be written.
	_ = logEvent(home, "set", now)
	return *stop, nil
}

// Lift removes this account's stop: the lift is recorded first, so a lift with
// no record does not happen. A stop another account wrote is left alone and
// named. lifted is false when there was none.
func Lift(home string, now time.Time) (lifted bool, _ error) {
	type place struct{ dir, leaf string }
	places := []place{{abcdhome.Path(home), HomeStopFile}}
	s, ok, err := ReadSetting(home)
	if err != nil {
		return false, err
	}
	refusedDir := ""
	if ok {
		// A directory no run would read holds no stop, and what it points at is
		// not this lift's to remove.
		present, refused := checkStopDir(s.Folder, s.PersonUID)
		refusedDir = refused
		if present {
			places = append(places, place{filepath.Join(s.Folder, StopDirName), StopFileName})
		}
	}
	// The platform's shared folder is read whatever the setting says, so a stop
	// there is this account's to lift when the directory is its own.
	if folder := WellKnownFolder(); folder != "" && (!ok || folder != s.Folder) {
		if owner, present := stopDirOwnerOf(folder); present && owner == os.Getuid() {
			if yes, _ := checkStopDir(folder, owner); yes {
				places = append(places, place{filepath.Join(folder, StopDirName), StopFileName})
			}
		}
	}
	var found []place
	for _, pl := range places {
		p := filepath.Join(pl.dir, pl.leaf)
		fi, err := os.Lstat(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, err
		}
		if uid, ok := ownerOf(fi); !ok || uid != os.Getuid() {
			return false, fmt.Errorf("the stop file %s is not this account's, so it is left where it is", fsutil.DisplayPath(p))
		}
		found = append(found, pl)
	}
	if len(found) == 0 {
		if refusedDir != "" {
			return false, errors.New(refusedDir)
		}
		return false, nil
	}
	if err := logEvent(home, "lifted", now); err != nil {
		return false, fmt.Errorf("the lift could not be recorded, so the stop stands: %w", err)
	}
	for _, pl := range found {
		root, err := os.OpenRoot(pl.dir)
		if err != nil {
			return false, err
		}
		err = root.Remove(pl.leaf)
		root.Close()
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
	}
	return true, nil
}

// logEvent appends one line to the person's own account of the stop.
func logEvent(home, event string, now time.Time) error {
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer root.Close()
	line, err := json.Marshal(struct {
		Event string    `json:"event"`
		By    string    `json:"by"`
		At    time.Time `json:"at"`
	}{event, accountName(os.Getuid()), now.UTC()})
	if err != nil {
		return err
	}
	return fsutil.AppendLineIn(root, LiftLogFile, line, abcdhome.FileMode)
}

// WritePage writes one dated page whole into the shared folder, as this
// account, readable by every account on the machine, and returns its path; ""
// when no folder was chosen, which writes nothing.
func WritePage(home string, day time.Time, data []byte) (string, error) {
	s, ok, err := ReadSetting(home)
	if err != nil || !ok {
		return "", err
	}
	if err := checkFolder(s.Folder, s.PersonUID); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(s.Folder)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := root.Mkdir(DigestDir, sharedDirMode); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", err
	}
	fi, err := root.Lstat(DigestDir)
	if err != nil {
		return "", err
	}
	if uid, ok := ownerOf(fi); !ok || !fi.IsDir() || uid != os.Getuid() {
		return "", fmt.Errorf("%s in the shared folder is not a directory this account made", DigestDir)
	}
	rel := DigestDir + "/" + day.UTC().Format(time.DateOnly) + ".md"
	if err := fsutil.WriteFileAtomicInRoot(root, rel, data, sharedFileMode); err != nil {
		return "", err
	}
	if err := root.Chmod(rel, sharedFileMode); err != nil {
		return "", err
	}
	return filepath.Join(s.Folder, filepath.FromSlash(rel)), nil
}
