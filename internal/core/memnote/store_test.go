//go:build unix

package memnote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const testRoot = "0123456789abcdef0123456789abcdef01234567"

// fixture sets a home and a repository, pins the root commit and the clock,
// and returns them.
func fixture(t *testing.T) (home, repo string) {
	t.Helper()
	home, repo = t.TempDir(), t.TempDir()
	prevRoot, prevNow := rootCommit, now
	rootCommit = func(string) string { return testRoot }
	now = func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { rootCommit, now = prevRoot, prevNow })
	return home, repo
}

func addReq(home, repo string) AddRequest {
	return AddRequest{Home: home, RepoRoot: repo, Title: "The tests need the race flag",
		Body: "Run the race-enabled tests before saying they pass.", Recall: []string{"test", "race"},
		By: "Example:model-1", Session: "s-0001"}
}

// tree lists every path under root, relative, sorted.
func tree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	_ = filepath.Walk(root, func(p string, _ os.FileInfo, err error) error {
		if err == nil {
			rel, _ := filepath.Rel(root, p)
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func machineFile(t *testing.T, home string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(abcdhome.Path(home, "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestWriteStampsTimeAndMintsAMemID(t *testing.T) {
	home, repo := fixture(t)
	n, err := Add(addReq(home, repo))
	if err != nil {
		t.Fatal(err)
	}
	if !ValidID(n.ID) || !strings.HasPrefix(n.ID, "mem-261011120000") {
		t.Errorf("id = %q, want a mem- id minted at the clock's second", n.ID)
	}
	if n.WrittenAt != "2026-10-11T12:00:00Z" {
		t.Errorf("written_at = %q, want the clock's", n.WrittenAt)
	}
	if n.WrittenBy != "Example:model-1" || n.Session == nil || *n.Session != "s-0001" || n.Origin != OriginAgent {
		t.Errorf("stamps = %+v", n)
	}
	if n.Root == nil || *n.Root != testRoot {
		t.Errorf("root = %v, want the repository's root commit", n.Root)
	}
	fi, err := os.Stat(abcdhome.Path(home, "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("the machine file's mode = %o, want 600", fi.Mode().Perm())
	}
	f, ok, err := ReadMachine(home)
	if err != nil || !ok || len(f.Notes) != 1 || f.Notes[0].ID != n.ID {
		t.Fatalf("read back = %+v, %v, %v", f, ok, err)
	}
}

// N7: a host with no session id is recorded as unknown, a null in the file.
func TestWriteWithoutASessionRecordsItUnknown(t *testing.T) {
	home, repo := fixture(t)
	req := addReq(home, repo)
	req.Session = ""
	n, err := Add(req)
	if err != nil {
		t.Fatal(err)
	}
	if n.Session != nil {
		t.Errorf("session = %q, want nil", *n.Session)
	}
	note := machineFile(t, home)["notes"].([]any)[0].(map[string]any)
	if v, ok := note["session"]; !ok || v != nil {
		t.Errorf("the file's session = %v (present %v), want null", v, ok)
	}
}

func TestASessionShapedLikeALinkIsRefused(t *testing.T) {
	home, repo := fixture(t)
	for _, s := range []string{"https://example.com/session/abc", "example.com", "a/b", "id with space"} {
		req := addReq(home, repo)
		req.Session = s
		_, err := Add(req)
		var fe *FieldError
		if !errors.As(err, &fe) || fe.Field != "session" {
			t.Errorf("session %q: Add = %v, want a session refusal", s, err)
		}
	}
	if _, err := os.Stat(abcdhome.Path(home)); !os.IsNotExist(err) {
		t.Errorf("a refused write created the home scope (stat err %v)", err)
	}
}

func TestEverywhereNoteCarriesNoRoot(t *testing.T) {
	home, repo := fixture(t)
	rootCommit = func(string) string { return "" }
	req := addReq(home, repo)
	if _, err := Add(req); !errors.Is(err, ErrNoRoot) {
		t.Fatalf("a tied note with no root commit: %v, want ErrNoRoot", err)
	}
	req.Everywhere = true
	n, err := Add(req)
	if err != nil {
		t.Fatal(err)
	}
	if n.Root != nil {
		t.Errorf("root = %q, want nil", *n.Root)
	}
	note := machineFile(t, home)["notes"].([]any)[0].(map[string]any)
	if v, ok := note["root"]; !ok || v != nil {
		t.Errorf("the file's root = %v (present %v), want null", v, ok)
	}
}

// N5: add writes the machine layer and nothing in the repository.
func TestWriteTouchesOnlyTheMachineLayer(t *testing.T) {
	home, repo := fixture(t)
	before := tree(t, repo)
	if _, err := Add(addReq(home, repo)); err != nil {
		t.Fatal(err)
	}
	if after := tree(t, repo); !equal(before, after) {
		t.Errorf("Add changed the repository: %v -> %v", before, after)
	}
	want := []string{".", ".abcd.noindex", ".abcd.noindex/memory.json"}
	if got := tree(t, home); !equal(got, want) {
		t.Errorf("home = %v, want %v", got, want)
	}
}

func equal(a, b []string) bool { return strings.Join(a, "\n") == strings.Join(b, "\n") }

func TestConcurrentWritesBothLand(t *testing.T) {
	home, repo := fixture(t)
	const n = 6
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := addReq(home, repo)
			req.Title = fmt.Sprintf("Lesson %d", i)
			_, errs[i] = Add(req)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("writer %d: %v", i, err)
		}
	}
	f, _, err := ReadMachine(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Notes) != n {
		t.Fatalf("%d notes landed, want %d", len(f.Notes), n)
	}
}

func TestRemoveRemovesOnlyThatNote(t *testing.T) {
	home, repo := fixture(t)
	a, err := Add(addReq(home, repo))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Add(addReq(home, repo))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Remove(home, a.ID); err != nil {
		t.Fatal(err)
	}
	f, _, _ := ReadMachine(home)
	if len(f.Notes) != 1 || f.Notes[0].ID != b.ID {
		t.Fatalf("after remove: %+v", f.Notes)
	}
	if _, err := Remove(home, a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("removing it again: %v, want ErrNotFound", err)
	}
}

// TestAbsentLayersCostNothingAndCreateNothing: with neither file, both reads
// are absent and nothing is created, in the home or the repository.
func TestAbsentLayersCostNothingAndCreateNothing(t *testing.T) {
	for _, withDir := range []bool{false, true} {
		home, repo := t.TempDir(), t.TempDir()
		if withDir {
			if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		hb, rb := tree(t, home), tree(t, repo)
		if _, ok, err := ReadMachine(home); ok || err != nil {
			t.Errorf("machine read with no file: present %v, err %v", ok, err)
		}
		if _, ok, err := ReadRepo(repo); ok || err != nil {
			t.Errorf("repository read with no file: present %v, err %v", ok, err)
		}
		if _, ok, err := ReadMachine(""); ok || err != nil {
			t.Errorf("machine read with no home: present %v, err %v", ok, err)
		}
		if !equal(hb, tree(t, home)) || !equal(rb, tree(t, repo)) {
			t.Error("a read created something")
		}
	}
}

func writeMachine(t *testing.T, home, body string) string {
	t.Helper()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	p := abcdhome.Path(home, "memory.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const validMachine = `{"schema_version":1,"notes":[]}`

// TestMachineLayerReadRefusesWhatRulesJSONRefuses mirrors the rules user
// layer's refusals: each is an error naming the file in tilde form, never the
// expanded home.
func TestMachineLayerReadRefusesWhatRulesJSONRefuses(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, home string)
		want  string
	}{
		{"symlinked home scope", func(t *testing.T, home string) {
			real := filepath.Join(home, "dotfiles")
			if err := os.MkdirAll(real, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(real, "memory.json"), []byte(validMachine), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, abcdhome.Path(home)); err != nil {
				t.Fatal(err)
			}
		}, abcdhome.Display() + " is a symlink"},
		{"foreign owner", func(t *testing.T, home string) {
			writeMachine(t, home, validMachine)
			t.Cleanup(fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) { return uint32(os.Getuid()) + 1, nil }))
		}, "not owned by this session's uid"},
		{"writable by others", func(t *testing.T, home string) {
			if err := os.Chmod(writeMachine(t, home, validMachine), 0o666); err != nil {
				t.Fatal(err)
			}
		}, "writable by others"},
		{"over 256 KiB", func(t *testing.T, home string) {
			writeMachine(t, home, validMachine+strings.Repeat(" ", maxFileBytes))
		}, "cap"},
		{"not a regular file", func(t *testing.T, home string) {
			if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(abcdhome.Path(home, "memory.json"), 0o600); err != nil {
				t.Skipf("mkfifo: %v", err)
			}
		}, "not a regular file"},
		{"symlinked leaf", func(t *testing.T, home string) {
			if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(home, "elsewhere.json")
			if err := os.WriteFile(target, []byte(validMachine), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, abcdhome.Path(home, "memory.json")); err != nil {
				t.Fatal(err)
			}
		}, "not a regular file"},
		{"malformed JSON", func(t *testing.T, home string) { writeMachine(t, home, "{ not json") }, "malformed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			tc.setup(t, home)
			_, _, err := ReadMachine(home)
			if err == nil {
				t.Fatal("the read was not refused")
			}
			msg := err.Error()
			if !strings.Contains(msg, abcdhome.Display()) || !strings.Contains(msg, tc.want) {
				t.Errorf("refusal = %q, want the tilde-form file and %q", msg, tc.want)
			}
			if strings.Contains(msg, home) {
				t.Errorf("the refusal carries the expanded home: %s", msg)
			}
		})
	}
}

// A refused machine file is never overwritten by a write.
func TestWriteRefusesOverAnUnreadableMachineFile(t *testing.T) {
	home, repo := fixture(t)
	p := writeMachine(t, home, "{ not json")
	if _, err := Add(addReq(home, repo)); err == nil {
		t.Fatal("Add wrote over a malformed machine file")
	}
	if data, _ := os.ReadFile(p); !bytes.Equal(data, []byte("{ not json")) {
		t.Errorf("the malformed file was changed: %q", data)
	}
}

func TestNoteSchemaRefusesEachMalformedField(t *testing.T) {
	note := func(edit string) string {
		fields := map[string]string{
			"id": `"mem-2610111200001234"`, "title": `"A title"`, "body": `"A body."`, "recall": `["test"]`,
			"root": `null`, "origin": `"agent"`, "written_by": `"Example:model-1"`,
			"written_at": `"2026-10-11T12:00:00Z"`, "session": `null`,
		}
		k, v, _ := strings.Cut(edit, "=")
		if k != "" {
			if v == "<absent>" {
				delete(fields, k)
			} else {
				fields[k] = v
			}
		}
		var parts []string
		for _, key := range []string{"id", "title", "body", "recall", "root", "origin", "written_by", "written_at", "session", "extra"} {
			if val, ok := fields[key]; ok {
				parts = append(parts, fmt.Sprintf("%q:%s", key, val))
			}
		}
		return `{"schema_version":1,"notes":[{` + strings.Join(parts, ",") + `}]}`
	}
	if _, err := parseFile([]byte(note("")), machineLayer); err != nil {
		t.Fatalf("the valid note was refused: %v", err)
	}
	long := strings.Repeat("x", MaxTitleChars+1)
	for _, edit := range []string{
		`id="iss-2610111200001234"`, `id="mem-1"`,
		`title=""`, `title="` + long + `"`, `title="two\nlines"`, `title="esc\u001b"`,
		`body=""`, `body="` + strings.Repeat("y", MaxBodyBytes+1) + `"`, `body="bidi‮"`,
		`recall=[]`, `recall=["a","b","c","d","e","f","g","h","i","j","k","l","m"]`, `recall=["--"]`, `recall=["x","X"]`, `recall=[" x"]`,
		`root=<absent>`, `root="nothex"`, `root=7`,
		`origin="person"`, `origin=<absent>`,
		`written_by="Example"`, `written_by="free text"`,
		`written_at="yesterday"`, `written_at="2026-10-11T13:00:00+01:00"`,
		`session="https://example.com/s"`, `session="a.b"`,
		`extra=1`,
	} {
		if _, err := parseFile([]byte(note(edit)), machineLayer); err == nil {
			t.Errorf("%s: accepted", edit)
		}
	}
	if _, err := parseFile([]byte(note("")), repoLayer); err == nil {
		t.Error("a root key in the repository layer was accepted")
	}
	if _, err := parseFile([]byte(note("root=<absent>")), repoLayer); err != nil {
		t.Errorf("a repository-layer note with no root was refused: %v", err)
	}
	for _, f := range []string{`{"schema_version":2,"notes":[]}`, `{"schema_version":1,"notes":[],"notes":[]}`, `{"schema_version":1,"notes":[]} {}`} {
		if _, err := parseFile([]byte(f), machineLayer); err == nil {
			t.Errorf("%s: accepted", f)
		}
	}
	two := strings.Replace(note(""), "}]}", "},"+strings.TrimSuffix(strings.TrimPrefix(note(""), `{"schema_version":1,"notes":[`), "]}")+"]}", 1)
	if _, err := parseFile([]byte(two), machineLayer); err == nil {
		t.Error("two notes sharing an id were accepted")
	}
}

// The id is minted through the shared seam, in the mem family.
func TestMintedIDIsTheMemFamily(t *testing.T) {
	id, err := recordid.Minter{}.Mint(Family)
	if err != nil || !ValidID(id) {
		t.Fatalf("Mint(%q) = %q, %v", Family, id, err)
	}
}

// The re-read under the lock goes through the directory the write writes
// through. A same-account rename of ~/.abcd.noindex between the write's walk and
// its lock, with an empty directory left at the name, must not make the write
// read the empty one and replace the old directory's notes with a one-note
// file (the class iss-2609290300313698 fixed for the credential index). The
// read through the descriptor finds the old file, and the shared guarded read
// (fsutil.ReadHomeDeclarationDenyingIn) then refuses it, because it confirms
// the owner by path as well and the path no longer reaches that file: so the
// write fails closed, and both directories are left as they were.
func TestWriteReadsUnderTheLockThroughTheDirectoryItWrites(t *testing.T) {
	home, repo := fixture(t)
	first, err := Add(addReq(home, repo))
	if err != nil {
		t.Fatal(err)
	}
	scope := abcdhome.Path(home)
	moved := scope + ".moved"
	prev := machineScopeOpened
	machineScopeOpened = func() {
		if err := os.Rename(scope, moved); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(scope, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { machineScopeOpened = prev })
	req := addReq(home, repo)
	req.Title = "A second lesson"
	_, addErr := Add(req)
	data, err := os.ReadFile(filepath.Join(moved, "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Notes []struct {
			ID string `json:"id"`
		} `json:"notes"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Notes) != 1 || f.Notes[0].ID != first.ID {
		t.Fatalf("the directory written through holds %d note(s), want only the first, untouched: %s", len(f.Notes), data)
	}
	if addErr == nil {
		t.Error("the write landed although the file it re-read is no longer the one at its path")
	}
	if _, err := os.Stat(filepath.Join(scope, "memory.json")); !os.IsNotExist(err) {
		t.Errorf("the directory now at the name gained a memory.json (stat err %v)", err)
	}
}
