//go:build linux

package recall

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/gittest"
)

// openRecorder is the filesystem seam that records every open under the
// trees it watches: an inotify instance with an IN_OPEN watch on every
// directory below each root, so a file or a directory opened by anything in
// the process — a primitive, a library, a git child — is recorded, whatever
// path the code took to reach it.
type openRecorder struct {
	fd   int
	dirs map[int32]string
}

func recordOpens(t *testing.T, roots ...string) *openRecorder {
	t.Helper()
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC | syscall.IN_NONBLOCK)
	if err != nil {
		t.Skipf("inotify unavailable: %v", err)
	}
	t.Cleanup(func() { syscall.Close(fd) })
	r := &openRecorder{fd: fd, dirs: map[int32]string{}}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return err
			}
			wd, err := syscall.InotifyAddWatch(fd, p, syscall.IN_OPEN|syscall.IN_ONLYDIR|syscall.IN_DONT_FOLLOW)
			if err != nil {
				return err
			}
			r.dirs[int32(wd)] = p
			return nil
		})
		if err != nil {
			t.Fatalf("watching %s: %v", root, err)
		}
	}
	// The walk that placed the watches opened each directory it listed;
	// those opens are the recorder's own, so they are drained unread.
	r.opened(t)
	return r
}

// opened drains the events queued so far and returns each path opened, once.
func (r *openRecorder) opened(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	buf := make([]byte, 64<<10)
	for {
		n, err := syscall.Read(r.fd, buf)
		if errors.Is(err, syscall.EAGAIN) {
			break
		}
		if err != nil {
			t.Fatalf("reading inotify events: %v", err)
		}
		for off := 0; off+syscall.SizeofInotifyEvent <= n; {
			wd := int32(binary.NativeEndian.Uint32(buf[off:]))
			mask := binary.NativeEndian.Uint32(buf[off+4:])
			nameLen := int(binary.NativeEndian.Uint32(buf[off+12:]))
			name := strings.TrimRight(string(buf[off+syscall.SizeofInotifyEvent:off+syscall.SizeofInotifyEvent+nameLen]), "\x00")
			off += syscall.SizeofInotifyEvent + nameLen
			if mask&syscall.IN_Q_OVERFLOW != 0 {
				t.Fatal("the inotify queue overflowed, so opens went unrecorded")
			}
			dir, ok := r.dirs[wd]
			if !ok || mask&syscall.IN_OPEN == 0 {
				continue
			}
			p := dir
			if name != "" {
				p = filepath.Join(dir, name)
			}
			seen[p] = true
		}
	}
	var out []string
	for p := range seen {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}

// TestRecallOpensOnlyTheDeclaredPlaces (P5): with a host memory directory
// full of matching text under the home, the recall — every note and page in
// reach read, then matched — opens only the four places it declares (the two
// memory.json layers and the two page scopes) and the directories on the way
// to them, plus the repository's own git data when a personal note asks git
// for the root commit. Nothing under the host's memory directory is opened.
func TestRecallOpensOnlyTheDeclaredPlaces(t *testing.T) {
	home := t.TempDir()
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "a repository\n")
	repo.Commit("root")
	root := repo.Root()
	tied := strings.TrimSpace(repo.Git("rev-list", "--max-parents=0", "HEAD"))

	machineNotes := abcdhome.Path(home, "memory.json")
	machinePages := abcdhome.Path(home, "memory")
	projectNotes := filepath.Join(root, ".abcd", "memory.json")
	projectPages := filepath.Join(root, ".abcd", "memory")
	for _, dir := range []string{machinePages, projectPages} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, machineNotes, `{"schema_version": 1, "notes": [`+
		noteJSON("mem-2610111200000001", "Auth tokens expire hourly", "Refresh before the hour.", "auth")+`, `+
		tiedNoteJSON("mem-2610111200000002", "This repo's auth lives in one package", "Look there first.", tied, "auth")+`]}`)
	writeFile(t, projectNotes, `{"schema_version": 1, "notes": [`+projectNoteJSON("mem-2610111200000003", "Auth is reviewed twice", "Ask for a second reviewer.", "auth")+`]}`)
	writePage(t, machinePages, "fact_eng_auth-keys.md", "# Auth keys live in the keychain\n", "auth")
	writePage(t, projectPages, "fact_eng_auth-flow.md", "# Auth uses short-lived tokens\n", "auth")

	// The host's own memory directory, full of text that matches: a reader
	// that scanned the home, or followed the host's layout, would recall it.
	host := filepath.Join(home, ".claude", "projects", "-repo", "memory")
	if err := os.MkdirAll(host, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(host, "MEMORY.md"), "- auth: the host's own lesson about auth\n")
	writePage(t, host, "fact_eng_auth-host.md", "# The host's auth page\n", "auth")
	writeFile(t, filepath.Join(home, ".claude", "memory.json"), `{"schema_version": 1, "notes": []}`)

	rec := recordOpens(t, home, root)
	items, refused := ReadItems(home, root)
	res := Recall(Request{Items: items, Prompt: "the auth login is broken", Budget: rules.InjectionBudgetBytes})
	opened := rec.opened(t)

	declared := []string{machineNotes, machinePages, projectNotes, projectPages}
	allowed := func(p string) bool {
		for _, d := range declared {
			if p == d || filepath.Dir(p) == d && (d == machinePages || d == projectPages) || strings.HasPrefix(d, p+string(filepath.Separator)) {
				return true
			}
		}
		// A personal note tied to this repository asks git for its root
		// commit, and git reads the repository's own data to answer.
		return strings.HasPrefix(p, filepath.Join(root, ".git")+string(filepath.Separator)) || p == filepath.Join(root, ".git")
	}
	if len(opened) == 0 {
		t.Fatal("the recorder saw no open at all, so it proves nothing")
	}
	for _, p := range opened {
		if p == host || strings.HasPrefix(p, filepath.Join(home, ".claude")) {
			t.Errorf("the recall opened %s, under the host's own files", p)
			continue
		}
		if !allowed(p) {
			t.Errorf("the recall opened %s, which is not one of the places it declares", p)
		}
	}
	for _, d := range declared {
		if !slices.Contains(opened, d) {
			t.Errorf("the recorder did not see %s opened, so it is not watching what the recall reads", d)
		}
	}

	// And the recall read what the four places hold, and nothing of the host's.
	if len(refused) != 0 {
		t.Errorf("refused = %+v", refused)
	}
	want := []string{"mem-2610111200000001", "mem-2610111200000002", "mem-2610111200000003", "page:fact_eng_auth-flow.md", "page:fact_eng_auth-keys.md"}
	if got := slices.Sorted(slices.Values(res.Recalled)); !slices.Equal(got, want) {
		t.Errorf("recalled %v, want %v", got, want)
	}
	if strings.Contains(res.Text, "host's") {
		t.Errorf("the host's memory came back:\n%s", res.Text)
	}
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
