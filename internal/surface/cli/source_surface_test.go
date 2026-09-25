package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sourceCheckout stands the process in a fresh repository under a temp HOME, so
// the default corpus (~/.abcd/sources) is a temp path and never the real one, and
// gives corpus commits a fixture identity.
func sourceCheckout(t *testing.T) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_AUTHOR_NAME", "Alice Example")
	t.Setenv("GIT_AUTHOR_EMAIL", "alice@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Alice Example")
	t.Setenv("GIT_COMMITTER_EMAIL", "alice@example.com")
	repo = filepath.Join(home, "repo")
	gitInitAt(t, repo)
	writeRel(t, repo, ".gitignore", ".abcd/.work.local/\n")
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "base")
	t.Chdir(repo)
	return home, repo
}

func runSource(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Run(append([]string{"source"}, args...), &out, &errb)
	return code, out.String(), errb.String()
}

// TestSourceNoCorpusSaysSoOnEveryVerb is AC6 at the front door: with no corpus,
// every verb says so on one line and exits with the distinct no-corpus code, and
// the guard's refresh mode says so and exits 0. Nothing creates the corpus.
func TestSourceNoCorpusSaysSoOnEveryVerb(t *testing.T) {
	home, _ := sourceCheckout(t)
	for _, args := range [][]string{
		{},
		{"ledger", "--list"},
		{"sync-banlist"},
		{"cite-check", "-"},
		{"declassify", "conf2026a"},
		{"add", "--key", "k2026x", "--title", "Some Title", "--public", "--url", "https://example.com/x"},
	} {
		code, stdout, stderr := runSource(t, args...)
		if code != 3 {
			t.Errorf("%v: exit %d, want 3\n%s%s", args, code, stdout, stderr)
		}
		if !strings.Contains(stdout+stderr, "no sources corpus") {
			t.Errorf("%v: does not say the corpus is absent\n%s%s", args, stdout, stderr)
		}
		noHomePath(t, home, stdout+stderr)
	}
	code, stdout, stderr := runSource(t, "sync-banlist", "--refresh")
	if code != 0 || strings.Count(stdout+stderr, "\n") != 1 || !strings.Contains(stderr, "no sources corpus") {
		t.Fatalf("refresh with no corpus: exit %d\n%s%s", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "sources")); !os.IsNotExist(err) {
		t.Fatal("a verb created the corpus")
	}
}

// TestSourceEndToEnd drives the whole personal cycle through the CLI: init, a
// confidential and a public add, a ledger line, a refused flip, the banlist sync,
// a cite-check that names the key only, declassification, and the flip it opens.
func TestSourceEndToEnd(t *testing.T) {
	home, repo := sourceCheckout(t)
	src := t.TempDir()
	notes := filepath.Join(src, "notes.md")
	if err := os.WriteFile(notes, []byte("the body\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	meta := filepath.Join(src, "meta.json")
	if err := os.WriteFile(meta, []byte(`{"title":"Quiet Harbour Working Notes","aliases":["harbourwatch"]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if code, out, errs := runSource(t, "init"); code != 0 {
		t.Fatalf("init: %d %s%s", code, out, errs)
	}
	code, out, errs := runSource(t, "add", notes, "--key", "conf2026a", "--meta", meta, "--confidential", "--json")
	if code != 0 {
		t.Fatalf("add confidential: %d %s%s", code, out, errs)
	}
	if strings.Contains(out+errs, "Harbour") {
		t.Fatalf("add echoes the confidential title:\n%s%s", out, errs)
	}
	if code, out, errs := runSource(t, "add", "--key", "survey2026", "--title", "An Open Survey", "--public", "--url", "https://example.com/survey"); code != 0 {
		t.Fatalf("add public: %d %s%s", code, out, errs)
	}
	if code, _, errs := runSource(t, "add", notes, "--key", "x2026", "--title", "No Class Given"); code != 2 || !strings.Contains(errs, "class") {
		t.Fatalf("add with no class: %d %s", code, errs)
	}

	code, out, errs = runSource(t, "ledger", "--decision", "adr-41", "--claim", "append-only", "--source", "conf2026a", "--influence", "supports", "--json")
	if code != 0 {
		t.Fatalf("ledger: %d %s%s", code, out, errs)
	}
	var line struct {
		Line   int `json:"line"`
		Record struct {
			Repo          string `json:"repo"`
			CitedPublicly bool   `json:"cited_publicly"`
		} `json:"record"`
	}
	if err := json.Unmarshal([]byte(out), &line); err != nil || line.Line != 1 || line.Record.CitedPublicly || len(line.Record.Repo) != 12 {
		t.Fatalf("ledger output: %v %s", err, out)
	}
	if code, out, errs := runSource(t, "ledger", "--flip", "1"); code != 2 || !strings.Contains(out+errs, "gate 1") {
		t.Fatalf("flip of an unpermitted line: %d %s%s", code, out, errs)
	}

	if code, out, errs := runSource(t, "sync-banlist"); code != 0 || !strings.Contains(out, "1 confidential source") {
		t.Fatalf("sync-banlist: %d %s%s", code, out, errs)
	}
	store, err := os.ReadFile(filepath.Join(repo, ".abcd", ".work.local", "private-names.txt"))
	if err != nil || !strings.Contains(string(store), "sources/conf2026a/title") {
		t.Fatalf("store: %v\n%s", err, store)
	}

	doc := filepath.Join(src, "draft.md")
	if err := os.WriteFile(doc, []byte("We follow the quiet harbour working notes here.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errs = runSource(t, "cite-check", doc)
	if code != 1 || !strings.Contains(out, "conf2026a") || strings.Contains(strings.ToLower(out+errs), "harbour") {
		t.Fatalf("cite-check: %d\n%s%s", code, out, errs)
	}
	var stdin bytes.Buffer
	stdin.WriteString("an unrelated sentence\n")
	if code, out, errs := runSourceStdin(t, &stdin, "cite-check", "-"); code != 0 || !strings.Contains(out, "clean") {
		t.Fatalf("cite-check clean: %d %s%s", code, out, errs)
	}

	if code, out, errs := runSource(t, "declassify", "conf2026a"); code != 0 {
		t.Fatalf("declassify: %d %s%s", code, out, errs)
	}
	if code, out, errs := runSource(t, "sync-banlist", "--refresh"); code != 0 {
		t.Fatalf("refresh: %d %s%s", code, out, errs)
	}
	store, _ = os.ReadFile(filepath.Join(repo, ".abcd", ".work.local", "private-names.txt"))
	if strings.Contains(string(store), "sources/conf2026a") {
		t.Fatalf("declassified key survived the refresh:\n%s", store)
	}
	if code, out, errs := runSource(t, "ledger", "--flip", "1"); code != 0 {
		t.Fatalf("flip after declassification: %d %s%s", code, out, errs)
	}
	code, out, errs = runSource(t, "--json")
	if code != 0 || !strings.Contains(out, `"present": true`) {
		t.Fatalf("status: %d %s%s", code, out, errs)
	}
	noHomePath(t, home, out+errs)
}

func runSourceStdin(t *testing.T, stdin *bytes.Buffer, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	root := NewRootCommand()
	root.SetArgs(append([]string{"source"}, args...))
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetIn(stdin)
	err := root.Execute()
	code := 0
	if err != nil {
		code = 1
		if c, ok := err.(interface{ ExitCode() int }); ok {
			code = c.ExitCode()
		}
		errb.WriteString(err.Error())
	}
	return code, out.String(), errb.String()
}

// confidentialCorpus makes the default corpus under the temp HOME and adds one
// confidential source, so a sync has something to project.
func confidentialCorpus(t *testing.T) {
	t.Helper()
	src := t.TempDir()
	notes := filepath.Join(src, "notes.md")
	if err := os.WriteFile(notes, []byte("the body\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	meta := filepath.Join(src, "meta.json")
	if err := os.WriteFile(meta, []byte(`{"title":"Quiet Harbour Working Notes","aliases":["harbourwatch"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := runSource(t, "init"); code != 0 {
		t.Fatalf("init: %d %s%s", code, out, errs)
	}
	if code, out, errs := runSource(t, "add", notes, "--key", "conf2026a", "--meta", meta, "--confidential"); code != 0 {
		t.Fatalf("add confidential: %d %s%s", code, out, errs)
	}
}

// TestSourceRefreshNeverCreatesTheStore: the guard's refresh updates a private store
// that already exists and never creates one (iss-2609252007426016). Creating the
// store writes every confidential title into the repository's local tier, which is
// the by-hand sync's act, not a side effect of a commit. The refresh says so on one
// line, exits 0, and leaves no tier behind.
func TestSourceRefreshNeverCreatesTheStore(t *testing.T) {
	home, repo := sourceCheckout(t)
	confidentialCorpus(t)
	code, stdout, stderr := runSource(t, "sync-banlist", "--refresh")
	if code != 0 {
		t.Fatalf("refresh with no store: exit %d\n%s%s", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", ".work.local")); !os.IsNotExist(err) {
		t.Fatalf("the refresh created the local tier or the store (%v)", err)
	}
	if strings.Count(stdout+stderr, "\n") != 1 || !strings.Contains(stderr, "abcd source sync-banlist") {
		t.Fatalf("the refresh does not say, on one line, that it created nothing\n%s%s", stdout, stderr)
	}
	noHomePath(t, home, stdout+stderr)

	// The by-hand sync is what creates it, and a refresh then updates it.
	if code, out, errs := runSource(t, "sync-banlist"); code != 0 {
		t.Fatalf("by-hand sync: %d %s%s", code, out, errs)
	}
	if code, out, errs := runSource(t, "sync-banlist", "--refresh"); code != 0 || !strings.Contains(out, "1 confidential source") {
		t.Fatalf("refresh of an existing store: %d %s%s", code, out, errs)
	}
}
