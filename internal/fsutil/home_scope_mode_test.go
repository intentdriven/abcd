//go:build unix

package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// abcdHomeAt lays out home/.abcd (and any directories below it named in rel's
// directory) at mode dirMode, with a 0600 declaration at rel holding body.
// Modes are set by chmod, so the umask cannot soften them.
func abcdHomeAt(t *testing.T, rel string, dirMode os.FileMode, body string) string {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, filepath.FromSlash(filepath.ToSlash(filepath.Dir(rel))))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	for d := dir; d != home; d = filepath.Dir(d) {
		if err := os.Chmod(d, dirMode); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

// A directory between home and a declaration that every account can write is
// refused, sticky or not: anyone could have renamed or hard-linked a file of
// the caller's shape in under the declaration's name, so the file's own mode
// and owner say nothing about who put it there (iss-2609290656480443). The
// refusal names the directory in tilde form and the chmod that repairs it:
// here the directory's group is shared too, so the repair takes the group's
// write bit with every account's.
func TestReadHomeDeclarationRefusesADirectoryEveryAccountCanWrite(t *testing.T) {
	restore := swapGroupMembersForTest(func(uint32) ([]uint32, bool) { return []uint32{uint32(os.Getuid()), uint32(os.Getuid()) + 1}, true })
	defer restore()
	for _, c := range []struct {
		name, rel, shown, repair string
		mode                     os.FileMode
	}{
		{"0777 ~/.abcd", ".abcd/trusted-roots", "~/.abcd", "chmod go-w", 0o777},
		{"sticky 1777 ~/.abcd", ".abcd/trusted-roots", "~/.abcd", "chmod go-w", 0o777 | os.ModeSticky},
		{"0703 ~/.abcd", ".abcd/trusted-roots", "~/.abcd", "chmod o-w", 0o703},
		{"0777 directory below ~/.abcd", ".abcd/sub/f", "~/.abcd", "chmod go-w", 0o777},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := abcdHomeAt(t, c.rel, c.mode, "/example\n")
			raw, refusal, err := ReadHomeDeclaration(home, c.rel, 1024)
			if refusal != DeclarationDirectoryExposed || !errors.Is(err, ErrHomeScopeExposed) || raw != nil {
				t.Fatalf("a declaration in a directory every account can write must be refused: refusal %d, err %v, raw %q", refusal, err, raw)
			}
			if !strings.Contains(err.Error(), c.shown) || !strings.Contains(err.Error(), c.repair) {
				t.Fatalf("the refusal must name the directory in tilde form and the repair: %v", err)
			}
		})
	}
}

// The deepest exposed directory is the one named: a sound ~/.abcd over an
// exposed ~/.abcd/sub names the level that is exposed.
func TestReadHomeDeclarationNamesTheExposedLevel(t *testing.T) {
	home := abcdHomeAt(t, ".abcd/sub/f", 0o700, "x\n")
	if err := os.Chmod(filepath.Join(home, ".abcd", "sub"), 0o777); err != nil {
		t.Fatal(err)
	}
	_, refusal, err := ReadHomeDeclaration(home, ".abcd/sub/f", 1024)
	if refusal != DeclarationDirectoryExposed || err == nil || !strings.Contains(err.Error(), "~/.abcd/sub ") {
		t.Fatalf("the exposed level must be the one named: refusal %d, err %v", refusal, err)
	}
}

// HomeDeclarationNames carries the exposed-directory refusal in the same
// clause ReadHomeDeclaration wrote, naming the directory and the chmod that
// repairs it, rather than falling through to the generic could-not-be-read
// wording: the trusted-roots and local-transcript-roots callers render this
// clause as their only diagnostic.
func TestHomeDeclarationNamesNamesTheExposedDirectory(t *testing.T) {
	restore := swapGroupMembersForTest(func(uint32) ([]uint32, bool) { return []uint32{uint32(os.Getuid())}, true })
	defer restore()
	home := abcdHomeAt(t, ".abcd/trusted-roots", 0o777, "/example/checkout\n")
	declared, ignored := HomeDeclarationNames(home, ".abcd/trusted-roots", 1024, "/example/checkout", false)
	if declared {
		t.Fatal("a declaration in a directory every account can write re-admitted the target")
	}
	if !strings.Contains(ignored, "~/.abcd") || !strings.Contains(ignored, "chmod o-w") || strings.Contains(ignored, "could not be read") {
		t.Fatalf("the ignored declaration must name the exposed directory and the repair: %q", ignored)
	}
}

// A directory another account owns is refused, since that account can replace
// anything in it; one root owns is not, since root can replace anything
// anywhere and refusing it would protect nothing. The owner is read from the
// descriptor of the directory opened, through a seam, because a test process
// cannot create a directory it does not own.
func TestReadHomeDeclarationRefusesADirectoryAnotherAccountOwns(t *testing.T) {
	home := abcdHomeAt(t, ".abcd/trusted-roots", 0o700, "/example\n")
	me := uint32(os.Getuid())

	restore := swapHomeScopeDirOwnerForTest(func(os.FileInfo) (uint32, bool) { return me + 1, true })
	_, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
	restore()
	if refusal != DeclarationDirectoryExposed || !errors.Is(err, ErrHomeScopeExposed) || !strings.Contains(err.Error(), "another account") {
		t.Fatalf("a ~/.abcd another account owns must be refused: refusal %d, err %v", refusal, err)
	}

	restore = swapHomeScopeDirOwnerForTest(func(os.FileInfo) (uint32, bool) { return 0, false })
	_, refusal, err = ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
	restore()
	if refusal != DeclarationDirectoryExposed {
		t.Fatalf("a ~/.abcd whose owner cannot be read must be refused: refusal %d, err %v", refusal, err)
	}

	restore = swapHomeScopeDirOwnerForTest(func(os.FileInfo) (uint32, bool) { return 0, true })
	raw, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
	restore()
	if refusal != DeclarationOK || err != nil || string(raw) != "/example\n" {
		t.Fatalf("a root-owned ~/.abcd must not be refused: refusal %d, err %v, raw %q", refusal, err, raw)
	}
}

// What stays read. Home itself is not judged, for HomeScopeLink's reason: it
// is the machine's layout. A declaration absent from an exposed directory is
// absent, as it is behind a symlinked one: nothing declared costs its owner
// nothing.
func TestReadHomeDeclarationLeavesWhatItDoesNotJudge(t *testing.T) {
	home := abcdHomeAt(t, ".abcd/trusted-roots", 0o700, "/example\n")
	if err := os.Chmod(home, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024); refusal != DeclarationOK || err != nil {
		t.Fatalf("home itself is not judged: refusal %d, err %v", refusal, err)
	}

	home = abcdHomeAt(t, ".abcd/other", 0o777, "x\n")
	if _, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024); refusal != DeclarationAbsent || !os.IsNotExist(err) {
		t.Fatalf("an absent declaration in an exposed ~/.abcd is absent: refusal %d, err %v", refusal, err)
	}
}

// A directory its group can write is read only when that group's sole member
// is the directory's owner (the product thinker's ruling of 2026-10-11 on
// iss-2609290703091174, the shape Debian's OpenSSH admits): under a
// user-private-group umask of 002 a ~/.abcd made by hand is 0775 and its group
// is its owner alone, and nobody else can write it. A group with any other
// member, or whose members cannot be read, is refused, naming the one chmod
// that repairs it. The membership is read through a seam, since a test cannot
// make a group.
func TestReadHomeDeclarationAdmitsAGroupWritableDirectoryOnlyForItsOwnersPrivateGroup(t *testing.T) {
	me := uint32(os.Getuid())
	alone := func(uint32) ([]uint32, bool) { return []uint32{me}, true }
	shared := func(uint32) ([]uint32, bool) { return []uint32{me, me + 1}, true }
	other := func(uint32) ([]uint32, bool) { return []uint32{me + 1}, true }
	empty := func(uint32) ([]uint32, bool) { return nil, true }
	unread := func(uint32) ([]uint32, bool) { return nil, false }
	for _, c := range []struct {
		name    string
		mode    os.FileMode
		members func(uint32) ([]uint32, bool)
		// repair is the chmod a refusal names; "" when the declaration is read.
		repair string
	}{
		{"0755", 0o755, shared, ""},
		{"0775, the owner's private group", 0o775, alone, ""},
		{"0775, a group shared with another account", 0o775, shared, "chmod g-w ~/.abcd"},
		{"0775, a group whose one member is another account", 0o775, other, "chmod g-w ~/.abcd"},
		{"0775, a group with no member", 0o775, empty, "chmod g-w ~/.abcd"},
		{"0775, a group whose members cannot be read", 0o775, unread, "chmod g-w ~/.abcd"},
		{"0777, the owner's private group", 0o777, alone, "chmod o-w ~/.abcd"},
		{"0777, a shared group", 0o777, shared, "chmod go-w ~/.abcd"},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := abcdHomeAt(t, ".abcd/trusted-roots", c.mode, "/example\n")
			restore := swapGroupMembersForTest(c.members)
			defer restore()
			raw, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
			if c.repair == "" {
				if refusal != DeclarationOK || err != nil || string(raw) != "/example\n" {
					t.Fatalf("want the declaration read: refusal %d, err %v, raw %q", refusal, err, raw)
				}
				return
			}
			if refusal != DeclarationDirectoryExposed || !errors.Is(err, ErrHomeScopeExposed) || raw != nil {
				t.Fatalf("want the declaration refused: refusal %d, err %v, raw %q", refusal, err, raw)
			}
			if !strings.Contains(err.Error(), "`"+c.repair+"`") {
				t.Fatalf("the refusal must name the one repair %q: %v", c.repair, err)
			}
		})
	}
}

// The production membership reading takes a group's members from the account
// files: every account whose primary group it is, and every name it lists,
// each as its uid. A listed name no account carries makes the membership
// unreadable, so the group is not taken to be its owner's alone.
func TestGroupMembersFromAccountFiles(t *testing.T) {
	passwd := []byte("root:x:0:0:root:/root:/bin/sh\n# a comment\nalice:x:1000:1000::/home/alice:/bin/sh\nbob:x:1001:1001::/home/bob:/bin/sh\ncarol:x:1002:1000::/home/carol:/bin/sh\n")
	group := []byte("root:x:0:\nalice:x:1000:\nbob:x:1001:\nstaff:x:50:alice,bob\nlone:x:60:alice\nghost:x:70:nobody-here\n")
	for _, c := range []struct {
		name string
		gid  uint32
		want []uint32
		ok   bool
	}{
		{"a primary group two accounts share", 1000, []uint32{1000, 1002}, true},
		{"a private group", 1001, []uint32{1001}, true},
		{"a listed group", 50, []uint32{1000, 1001}, true},
		{"a group listing its one member", 60, []uint32{1000}, true},
		{"a group naming no account", 70, nil, false},
		{"a group the file does not hold", 99, nil, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := membersFromAccountFiles(passwd, group, c.gid)
			if ok != c.ok || !slices.Equal(got, c.want) {
				t.Fatalf("members(%d) = %v, %v; want %v, %v", c.gid, got, ok, c.want, c.ok)
			}
		})
	}
}

// An NSS compat line (`+` or `-` first) in either account file pulls accounts
// or group members in from NIS or LDAP, which the files do not show, so a
// group read from files that carry one is unread whatever its line says
// (iss-2609290703091174): the group the owner seems alone in may have members
// the network adds.
func TestGroupMembersAreUnreadWhenTheAccountFilesDeferToTheNetwork(t *testing.T) {
	passwd := "root:x:0:0:root:/root:/bin/sh\nbob:x:1001:1001::/home/bob:/bin/sh\n"
	group := "root:x:0:\nbob:x:1001:\n"
	for _, c := range []struct{ name, passwd, group string }{
		{"a passwd line pulling every network account in", passwd + "+::::::\n", group},
		{"a passwd line pulling a netgroup in", passwd + "+@staff\n", group},
		{"a passwd line excluding an account", passwd + "-mallory\n", group},
		{"a group line pulling every network group in", passwd, group + "+:::\n"},
		{"a group line naming a network group", passwd, group + "+bob\n"},
		{"a group line excluding a group", passwd, group + "-wheel\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got, ok := membersFromAccountFiles([]byte(c.passwd), []byte(c.group), 1001); ok {
				t.Fatalf("members(1001) = %v, read; want unread, since the files defer to the network", got)
			}
		})
	}
	if got, ok := membersFromAccountFiles([]byte(passwd), []byte(group), 1001); !ok || !slices.Equal(got, []uint32{1001}) {
		t.Fatalf("without a compat line, members(1001) = %v, %v; want [1001], read", got, ok)
	}
}

// accountSourcesAt writes an nsswitch.conf holding nsswitch (none when
// nsswitch is absent), and passwd and group files in which gid is the private
// group of an account with uid, and returns them as accountSources.
func accountSourcesAt(t *testing.T, nsswitch *string, uid, gid uint32) accountSources {
	t.Helper()
	dir := t.TempDir()
	src := accountSources{
		nsswitch: []string{filepath.Join(dir, "etc-nsswitch.conf"), filepath.Join(dir, "usr-etc-nsswitch.conf")},
		passwd:   filepath.Join(dir, "passwd"),
		group:    filepath.Join(dir, "group"),
	}
	files := map[string]string{
		src.passwd: "root:x:0:0:root:/root:/bin/sh\nalice:x:" + strconv.FormatUint(uint64(uid), 10) + ":" + strconv.FormatUint(uint64(gid), 10) + "::/home/alice:/bin/sh\n",
		src.group:  "root:x:0:\nalice:x:" + strconv.FormatUint(uint64(gid), 10) + ":\n",
	}
	if nsswitch != nil {
		files[src.nsswitch[0]] = *nsswitch
	}
	for p, body := range files {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return src
}

// The account files are the machine's whole account list only when the name
// service switch reads them alone. A host whose nsswitch.conf names a
// directory service (sss, ldap, winbind, nis), systemd's account sources, or
// a merge of another source into a file's group, takes accounts and group
// members from where the files do not show, with no `+` line in either file
// to say so; its group members are unread, so a group-writable home there is
// refused, naming the chmod that repairs it. A host whose switch names the
// files alone, or that has no nsswitch.conf (glibc's default, compat, which
// the `+` check covers), still reads a private group as the owner's alone.
func TestGroupMembersAreUnreadWhenTheNameServiceSwitchNamesAnotherSource(t *testing.T) {
	const uid, gid = 1001, 1001
	for _, c := range []struct {
		name, conf string
		read       bool
	}{
		{"files sss, no compat line", "passwd: files sss\ngroup: files sss\n", false},
		{"the group line alone names ldap", "passwd: files\ngroup:  files ldap\n", false},
		{"the passwd line alone names winbind", "passwd: compat winbind\ngroup: compat\n", false},
		{"nis", "passwd: files nis\ngroup: files nis\n", false},
		{"systemd", "passwd: files systemd\ngroup: files systemd\n", false},
		{"a merge into the files' groups", "passwd: files\ngroup: files [SUCCESS=merge] files\n", false},
		{"a merge spelt with spaces", "passwd: files\ngroup: files [ success = merge ]\n", false},
		{"initgroups names sss", "passwd: files\ngroup: files\ninitgroups: files sss\n", false},
		{"an unterminated action", "passwd: files\ngroup: files [NOTFOUND=return\n", false},
		{"files alone", "# comment\npasswd: files # local only\ngroup: files\nhosts: files dns\n", true},
		{"compat, with a return action", "passwd: compat\ngroup: compat [NOTFOUND=return] files\nshadow: files sss\n", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := groupMembersFrom(accountSourcesAt(t, &c.conf, uid, gid), gid)
			if ok != c.read {
				t.Fatalf("members(%d) = %v, read %v; want read %v", gid, got, ok, c.read)
			}
			if ok && !slices.Equal(got, []uint32{uid}) {
				t.Fatalf("members(%d) = %v; want [%d]", gid, got, uid)
			}
		})
	}
	t.Run("no nsswitch.conf", func(t *testing.T) {
		if got, ok := groupMembersFrom(accountSourcesAt(t, nil, uid, gid), gid); !ok || !slices.Equal(got, []uint32{uid}) {
			t.Fatalf("members(%d) = %v, %v; want [%d], read, as glibc's default reads the files", gid, got, ok, uid)
		}
	})
	t.Run("the vendor nsswitch.conf when /etc has none", func(t *testing.T) {
		src := accountSourcesAt(t, nil, uid, gid)
		if err := os.WriteFile(src.nsswitch[1], []byte("passwd: files sss\ngroup: files sss\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, ok := groupMembersFrom(src, gid); ok {
			t.Fatalf("members(%d) = %v, read; want unread under the vendor switch naming sss", gid, got)
		}
	})
	t.Run("an nsswitch.conf that cannot be read", func(t *testing.T) {
		src := accountSourcesAt(t, nil, uid, gid)
		if err := os.Mkdir(src.nsswitch[0], 0o700); err != nil {
			t.Fatal(err)
		}
		if got, ok := groupMembersFrom(src, gid); ok {
			t.Fatalf("members(%d) = %v, read; want unread when the switch cannot be read", gid, got)
		}
	})
}

// End to end: a 0775 ~/.abcd whose group is its owner's private group in the
// account files is refused on a host whose switch names sss, and the refusal
// names `chmod g-w`; under a switch naming the files alone it is read.
func TestAGroupWritableHomeIsRefusedUnderADirectoryService(t *testing.T) {
	home := abcdHomeAt(t, ".abcd/trusted-roots", 0o775, "/example\n")
	st, err := os.Stat(filepath.Join(home, ".abcd"))
	if err != nil {
		t.Fatal(err)
	}
	gid := st.Sys().(*syscall.Stat_t).Gid
	for _, c := range []struct{ name, conf, repair string }{
		{"files sss", "passwd: files sss\ngroup: files sss\n", "chmod g-w ~/.abcd"},
		{"files alone", "passwd: files\ngroup: files\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			src := accountSourcesAt(t, &c.conf, uint32(os.Getuid()), gid)
			restore := swapGroupMembersForTest(func(g uint32) ([]uint32, bool) { return groupMembersFrom(src, g) })
			defer restore()
			raw, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
			if c.repair == "" {
				if refusal != DeclarationOK || err != nil {
					t.Fatalf("want the declaration read: refusal %d, err %v", refusal, err)
				}
				return
			}
			if refusal != DeclarationDirectoryExposed || raw != nil || !strings.Contains(err.Error(), "`"+c.repair+"`") {
				t.Fatalf("want the declaration refused naming %q: refusal %d, err %v", c.repair, refusal, err)
			}
		})
	}
}
