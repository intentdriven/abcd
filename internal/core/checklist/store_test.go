package checklist_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/intentdriven/abcd/internal/fsutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// newRepo returns a checkout root holding the local tier, the one place the
// store may write; the store never creates the tier itself.
func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(checklist.TierRelPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func abs(root, rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// plant writes a file the way a shell or an editor would: straight to the path,
// past the verb's checks.
func plant(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func marshal(t *testing.T, c checklist.Checklist) []byte {
	t.Helper()
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// token is a GitHub-PAT-shaped string built at run time, so no secret-shaped
// literal is ever committed (principle secret-shaped-fixtures-at-runtime).
func token() string { return "ghp_" + testsecret.Synthetic(2610230805, 36) }

// assertNoEcho fails when text carries the token, its prefix, or any run of
// four of its characters: a redaction fingerprint (head, stars, tail) is a part
// of the secret too (invariant 24).
func assertNoEcho(t *testing.T, what, text, tok string) {
	t.Helper()
	if strings.Contains(text, tok[:3]) {
		t.Errorf("%s repeats the secret's prefix: %s", what, text)
	}
	for i := 0; i+4 <= len(tok); i++ {
		if strings.Contains(text, tok[i:i+4]) {
			t.Errorf("%s repeats part of the secret: %s", what, text)
			return
		}
	}
}

func TestWriteThenReadRoundTrips(t *testing.T) {
	root := newRepo(t)
	want := sample()
	if err := checklist.Write(root, want); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := checklist.Read(root)
	if got.Source != checklist.FromFile {
		t.Fatalf("want the checklist itself, got %s: %s", got.Source, got.Reason)
	}
	if !reflect.DeepEqual(got.Checklist, want) {
		t.Fatalf("round trip changed the checklist:\n got %+v\nwant %+v", got.Checklist, want)
	}
	if got.Reason == "" {
		t.Error("the result does not say why the checklist was used")
	}
	file := readFile(t, abs(root, checklist.FileRelPath))
	if lg := readFile(t, abs(root, checklist.LastGoodRelPath)); !bytes.Equal(file, lg) {
		t.Error("last-good is not the bytes the write just produced")
	}
}

func TestWriteRefusesWithoutTheLocalTier(t *testing.T) {
	root := t.TempDir()
	err := checklist.Write(root, sample())
	if !errors.Is(err, checklist.ErrNoLocalTier) {
		t.Fatalf("want ErrNoLocalTier, got %v", err)
	}
	if _, err := os.Stat(abs(root, checklist.TierRelPath)); !os.IsNotExist(err) {
		t.Error("the write created the local tier")
	}
}

func TestTornTempFileIsIgnoredAndRemovedOnTheNextWrite(t *testing.T) {
	root := newRepo(t)
	tier := abs(root, checklist.TierRelPath)

	// Before any checklist exists, a torn temp file is not a checklist.
	torn := filepath.Join(tier, ".abcd-tmp-1234567")
	plant(t, torn, []byte(`{"schema":"abcd/checklist/v1","updated_at":"2026-10-23T08:00:00Z","items":[{"id":"wr`))
	if got := checklist.Read(root); got.Source != checklist.FromNothing {
		t.Fatalf("a torn temp file was read as %s: %s", got.Source, got.Reason)
	}

	if err := checklist.Write(root, sample()); err != nil {
		t.Fatal(err)
	}
	plant(t, torn, []byte(`{"schema":"abcd/checklist/v1","items":[`))
	if got := checklist.Read(root); got.Source != checklist.FromFile || !reflect.DeepEqual(got.Checklist, sample()) {
		t.Fatalf("a torn temp file beside the checklist changed the read: %s, %s", got.Source, got.Reason)
	}

	// A writer killed a while ago left the torn file; a writer running now owns
	// the fresh one, and the next write must leave that one alone.
	old := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(torn, old, old); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(tier, ".abcd-tmp-7654321")
	plant(t, fresh, []byte(`{"sch`))

	if err := checklist.Write(root, sample()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(torn); !os.IsNotExist(err) {
		t.Error("the next write left the torn temp file in place")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("the next write removed a temp file a running writer may own: %v", err)
	}
}

func TestWriteRefusesASchemaViolationNamingTheField(t *testing.T) {
	root := newRepo(t)
	c := sample()
	c.Items[2].State = "paused"
	err := checklist.Write(root, c)
	var fe *checklist.FieldError
	if !errors.As(err, &fe) || fe.Field != "items[2].state" {
		t.Fatalf("want a refusal naming items[2].state, got %v", err)
	}
	if !strings.Contains(err.Error(), "items[2].state") {
		t.Errorf("the message does not name the field: %v", err)
	}
	for _, rel := range []string{checklist.FileRelPath, checklist.LastGoodRelPath} {
		if _, err := os.Stat(abs(root, rel)); !os.IsNotExist(err) {
			t.Errorf("a refused write left %s behind", rel)
		}
	}
}

func TestWriteRefusesASecretInEveryStringFieldKind(t *testing.T) {
	tok := token()
	cases := []struct {
		field string
		// scanned is false where the schema refuses the value before the
		// scanner sees it; the refusal must still name the field and not echo.
		scanned bool
		mutate  func(c *checklist.Checklist)
	}{
		{"updated_by", true, func(c *checklist.Checklist) { c.UpdatedBy = tok }},
		{"items[0].title", true, func(c *checklist.Checklist) { c.Items[0].Title = "renew " + tok + " tonight" }},
		{"items[0].promise.trigger", true, func(c *checklist.Checklist) { c.Items[0].Promise.Trigger = "token=" + tok }},
		{"items[0].promise.act", true, func(c *checklist.Checklist) { c.Items[0].Promise.Act = "export GH_TOKEN=" + tok }},
		{"items[0].owner_refs[2]", true, func(c *checklist.Checklist) { c.Items[0].OwnerRefs[2] = "branch:" + tok }},
		{"items[0].waits_on", false, func(c *checklist.Checklist) { c.Items[0].WaitsOn = "slot:" + tok }},
		{"items[0].id", false, func(c *checklist.Checklist) { c.Items[0].ID = tok }},
		{"items[0].state", false, func(c *checklist.Checklist) { c.Items[0].State = checklist.State(tok) }},
		{"items[0].next_look", false, func(c *checklist.Checklist) { c.Items[0].NextLook = tok }},
		{"items[0].credential.kind", false, func(c *checklist.Checklist) {
			c.Items[0].Credential.Kind = checklist.CredentialKind(tok)
		}},
		{"items[0].credential.scope[0]", false, func(c *checklist.Checklist) {
			c.Items[0].Credential.Scope[0] = checklist.CredentialScope(tok)
		}},
		{"items[0].credential.expiry", false, func(c *checklist.Checklist) { c.Items[0].Credential.Expiry = tok }},
	}
	for _, tc := range cases {
		root := newRepo(t)
		if err := checklist.Write(root, sample()); err != nil {
			t.Fatal(err)
		}
		before := readFile(t, abs(root, checklist.FileRelPath))

		c := sample()
		tc.mutate(&c)
		err := checklist.Write(root, c)
		if err == nil {
			t.Errorf("%s: a secret was written", tc.field)
			continue
		}
		assertNoEcho(t, tc.field+" refusal", err.Error(), tok)
		if !strings.Contains(err.Error(), tc.field) {
			t.Errorf("%s: the refusal does not name the field: %v", tc.field, err)
		}
		if tc.scanned {
			var se *checklist.SecretError
			if !errors.As(err, &se) || se.Field != tc.field || se.Kind != "token:github_pat" {
				t.Errorf("%s: want a *SecretError naming the field and token:github_pat, got %v", tc.field, err)
			} else if !strings.Contains(err.Error(), "token:github_pat") {
				t.Errorf("%s: the message does not name the pattern kind: %v", tc.field, err)
			}
		}
		for _, rel := range []string{checklist.FileRelPath, checklist.LastGoodRelPath} {
			if got := readFile(t, abs(root, rel)); !bytes.Equal(got, before) {
				t.Errorf("%s: the refused write changed %s", tc.field, rel)
			}
		}
	}
}

func TestLastGoodComesOnlyFromACheckedWrite(t *testing.T) {
	root := newRepo(t)
	a := sample()
	if err := checklist.Write(root, a); err != nil {
		t.Fatal(err)
	}
	aBytes := readFile(t, abs(root, checklist.LastGoodRelPath))

	// A valid, clean file written past the verb is not promoted: only a verb
	// write produces last-good, and it writes its own bytes, not the old file.
	edited := sample()
	edited.Items[0].Title = "edited by hand"
	plant(t, abs(root, checklist.FileRelPath), marshal(t, edited))
	if lg := readFile(t, abs(root, checklist.LastGoodRelPath)); !bytes.Equal(lg, aBytes) {
		t.Fatal("a hand edit moved last-good")
	}

	d := sample()
	d.Items[0].Title = "written by the verb"
	if err := checklist.Write(root, d); err != nil {
		t.Fatal(err)
	}
	lg := readFile(t, abs(root, checklist.LastGoodRelPath))
	if !bytes.Equal(lg, readFile(t, abs(root, checklist.FileRelPath))) {
		t.Fatal("last-good is not the file the verb just wrote")
	}
	if strings.Contains(string(lg), "edited by hand") {
		t.Fatal("the hand-edited file was promoted to last-good")
	}
}

func TestHandEditedSecretNeverBecomesLastGoodAndTheReaderFallsBack(t *testing.T) {
	tok := token()
	root := newRepo(t)
	if err := checklist.Write(root, sample()); err != nil {
		t.Fatal(err)
	}
	good := readFile(t, abs(root, checklist.LastGoodRelPath))

	bad := sample()
	bad.Items[1].Title = "paste " + tok
	plant(t, abs(root, checklist.FileRelPath), marshal(t, bad))

	got := checklist.Read(root)
	if got.Source != checklist.FromLastGood {
		t.Fatalf("want last-good, got %s: %s", got.Source, got.Reason)
	}
	if !reflect.DeepEqual(got.Checklist, sample()) {
		t.Fatalf("the fallback is not the last good checklist: %+v", got.Checklist)
	}
	for _, want := range []string{"items[1].title", "token:github_pat"} {
		if !strings.Contains(got.Reason, want) {
			t.Errorf("the reason does not name %s: %s", want, got.Reason)
		}
	}
	assertNoEcho(t, "the reason", got.Reason, tok)
	if lg := readFile(t, abs(root, checklist.LastGoodRelPath)); !bytes.Equal(lg, good) {
		t.Fatal("reading a secret-bearing file changed last-good")
	}
}

func TestUnreadableFileFallsBackWithoutEchoing(t *testing.T) {
	tok := token()
	docs := map[string]string{
		"corrupt":       `{"schema":"abcd/checklist/v1","items":[`,
		"unknown field": `{"` + tok + `":1,"schema":"abcd/checklist/v1"}`,
		"repeated key":  `{"schema":"abcd/checklist/v1","` + tok + `":1,"` + tok + `":2}`,
		"schema":        strings.Replace(string(marshal(t, sample())), `"state":"done"`, `"state":"`+tok+`"`, 1),
	}
	for name, doc := range docs {
		root := newRepo(t)
		if err := checklist.Write(root, sample()); err != nil {
			t.Fatal(err)
		}
		plant(t, abs(root, checklist.FileRelPath), []byte(doc))
		got := checklist.Read(root)
		if got.Source != checklist.FromLastGood {
			t.Errorf("%s: want last-good, got %s: %s", name, got.Source, got.Reason)
			continue
		}
		if !strings.Contains(got.Reason, "checklist.json") {
			t.Errorf("%s: the reason does not say which file failed: %s", name, got.Reason)
		}
		assertNoEcho(t, name+" reason", got.Reason, tok)
	}
}

func TestBothBadReadsNothingWithAReason(t *testing.T) {
	tok := token()
	root := newRepo(t)
	if got := checklist.Read(root); got.Source != checklist.FromNothing || got.Reason == "" {
		t.Fatalf("an empty tier: want nothing with a reason, got %s: %q", got.Source, got.Reason)
	}

	bad := sample()
	bad.UpdatedBy = tok
	plant(t, abs(root, checklist.FileRelPath), marshal(t, bad))
	// last-good is checked on read too: a secret planted there is refused.
	plant(t, abs(root, checklist.LastGoodRelPath), marshal(t, bad))
	got := checklist.Read(root)
	if got.Source != checklist.FromNothing {
		t.Fatalf("want nothing, got %s: %s", got.Source, got.Reason)
	}
	if !reflect.DeepEqual(got.Checklist, checklist.Checklist{}) {
		t.Errorf("a result of nothing carries a checklist: %+v", got.Checklist)
	}
	for _, want := range []string{"checklist.json", "checklist.last-good.json", "updated_by"} {
		if !strings.Contains(got.Reason, want) {
			t.Errorf("the reason does not name %s: %s", want, got.Reason)
		}
	}
	assertNoEcho(t, "the reason", got.Reason, tok)

	plant(t, abs(root, checklist.LastGoodRelPath), []byte("{"))
	if got := checklist.Read(root); got.Source != checklist.FromNothing ||
		!strings.Contains(got.Reason, "checklist.last-good.json") {
		t.Errorf("a corrupt last-good: want nothing naming it, got %s: %s", got.Source, got.Reason)
	}
}

// TestConcurrentUpdatesKeepEveryItem: a session's verb, the stop hook and
// sub-agents sharing one worktree write one checklist, so each read-change-
// write holds the checklist's lock across the whole cycle, and no writer's
// item is lost to another's write.
func TestConcurrentUpdatesKeepEveryItem(t *testing.T) {
	root := newRepo(t)
	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- checklist.Update(root, func(res checklist.Result) (checklist.Checklist, error) {
				c := res.Checklist
				if res.Source == checklist.FromNothing {
					c = checklist.Checklist{Schema: checklist.Schema}
				}
				c.UpdatedAt = "2026-10-23T08:00:00Z"
				c.UpdatedBy = "6f1c2a9e-3b4d-4e5f-8a7b-0c1d2e3f4a5b"
				// Widen the window between the read and the write, so an
				// unlocked cycle loses items every run, not by chance.
				time.Sleep(20 * time.Millisecond)
				c.Items = append(c.Items, checklist.Item{
					ID: fmt.Sprintf("item-%d", i), Title: fmt.Sprintf("Item %d", i), State: checklist.Working,
				})
				return c, nil
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got := checklist.Read(root)
	if got.Source != checklist.FromFile || len(got.Checklist.Items) != writers {
		t.Fatalf("want %d items from the file, got %d from %s: %s", writers, len(got.Checklist.Items), got.Source, got.Reason)
	}
}

// TestUpdateRefusesWhileAnotherWriterHoldsTheLock: a writer that cannot take
// the lock within the wait writes nothing and says why, never building on a
// list another writer is changing.
func TestUpdateRefusesWhileAnotherWriterHoldsTheLock(t *testing.T) {
	root := newRepo(t)
	defer checklist.SetLockTimeout(50 * time.Millisecond)()
	held := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = fsutil.WithFileLock(abs(root, checklist.LockRelPath), time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	called := false
	err := checklist.Update(root, func(checklist.Result) (checklist.Checklist, error) {
		called = true
		return sample(), nil
	})
	close(release)
	if !errors.Is(err, checklist.ErrLockHeld) {
		t.Fatalf("want ErrLockHeld, got %v", err)
	}
	if called {
		t.Fatal("the change ran without the lock")
	}
	if _, err := os.Stat(abs(root, checklist.FileRelPath)); !os.IsNotExist(err) {
		t.Fatalf("a refused update wrote the checklist: %v", err)
	}
}
