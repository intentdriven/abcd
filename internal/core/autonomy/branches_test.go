package autonomy

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// testCopyKey is a copy key's shape, built at runtime so no key-shaped
// literal sits in source (principles/secret-shaped-fixtures-at-runtime.md).
var testCopyKey = strings.Repeat("0123456789abcdef", 2)

func testStore(t *testing.T) Store {
	t.Helper()
	root, home := managedCopy(t)
	st, err := openStore(home, root)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func names(bs []CreatedBranch) []string {
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = b.Name
	}
	return out
}

// TestBranchRecorderUnderLock: recordings racing each other keep every
// branch, since each re-reads the list inside the store's lock.
func TestBranchRecorderUnderLock(t *testing.T) {
	st := testStore(t)
	now := time.Now()
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- st.RecordBranches(testCopyKey, []BranchChange{{Kind: BranchCreated, Name: fmt.Sprintf("feat/r%d", i)}}, now)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := st.CreatedBranches(testCopyKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != n {
		t.Fatalf("racing recordings kept %d branches, want %d: %v", len(got), n, names(got))
	}
}

// TestCreatedBranchesCapped: the list keeps at most MaxCreatedBranches, the
// oldest dropped; a name already kept is not kept twice; a rename carries a
// record by whole name only; and another copy's key keeps its own list.
func TestCreatedBranchesCapped(t *testing.T) {
	st := testStore(t)
	now := time.Now()
	for i := range MaxCreatedBranches + 5 {
		if err := st.RecordBranches(testCopyKey, []BranchChange{{Kind: BranchCreated, Name: fmt.Sprintf("feat/c%03d", i)}}, now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := st.CreatedBranches(testCopyKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxCreatedBranches || got[0].Name != "feat/c005" {
		t.Fatalf("the list holds %d, first %q; want %d, first feat/c005", len(got), got[0].Name, MaxCreatedBranches)
	}
	if err := st.RecordBranches(testCopyKey, []BranchChange{{Kind: BranchCreated, Name: "feat/c010"}}, now); err != nil {
		t.Fatal(err)
	}
	if got, _ = st.CreatedBranches(testCopyKey); len(got) != MaxCreatedBranches {
		t.Fatalf("a name already kept was kept again: %d", len(got))
	}
	if err := st.RecordBranches(testCopyKey, []BranchChange{{Kind: BranchMoved, From: "feat/c01", Name: "feat/x"}, {Kind: BranchMoved, From: "feat/c011", Name: "feat/c011b"}}, now); err != nil {
		t.Fatal(err)
	}
	got, _ = st.CreatedBranches(testCopyKey)
	have := map[string]bool{}
	for _, n := range names(got) {
		have[n] = true
	}
	if have["feat/x"] || !have["feat/c011b"] || have["feat/c011"] {
		t.Fatalf("a rename must carry a record by its whole name only: %v", names(got))
	}
	other, err := st.CreatedBranches("fedcba9876543210fedcba9876543210")
	if err != nil || len(other) != 0 {
		t.Fatalf("another copy's list must be its own: %v %v", other, err)
	}
}

// TestRecordCreatedKeepsTheSession (iss-2610311406481214): a branch the guard
// records is kept with the session that created it, a rename carries the
// creator rather than taking the renamer, an id abcd does not key on keeps
// none, and a list written before sessions were kept reads with none.
func TestRecordCreatedKeepsTheSession(t *testing.T) {
	root, home := managedCopy(t)
	rec := func(session string, c BranchChange) {
		t.Helper()
		if err := RecordCreated(root, root, []BranchChange{c}, Options{Home: home, SessionID: session}); err != nil {
			t.Fatal(err)
		}
	}
	rec("sess-a", BranchChange{Kind: BranchCreated, Name: "feat/a"})
	rec("sess-b", BranchChange{Kind: BranchMoved, From: "feat/a", Name: "feat/a2"})
	rec("not a session", BranchChange{Kind: BranchCreated, Name: "feat/b"})
	rec("", BranchChange{Kind: BranchCreated, Name: "feat/c"})
	st, err := openStore(home, root)
	if err != nil {
		t.Fatal(err)
	}
	key, err := claim.CopyKey(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.CreatedBranches(key)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]string{}
	for _, b := range got {
		by[b.Name] = b.Session
	}
	want := map[string]string{"feat/a2": "sess-a", "feat/b": "", "feat/c": ""}
	if len(by) != len(want) {
		t.Fatalf("recorded %v, want %v", by, want)
	}
	for n, s := range want {
		if by[n] != s {
			t.Errorf("%s recorded with session %q, want %q", n, by[n], s)
		}
	}

	old := `{"schema_version":1,"branches":[{"name":"feat/old","at":"2026-10-01T00:00:00Z"}]}` + "\n"
	err = st.WithLock(func() error { return st.WriteFile(createdFile(testCopyKey), []byte(old)) })
	if err != nil {
		t.Fatal(err)
	}
	if got, err := st.CreatedBranches(testCopyKey); err != nil || len(got) != 1 || got[0].Session != "" {
		t.Fatalf("a list from before sessions were kept: %v, %v", got, err)
	}
}
