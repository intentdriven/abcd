package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
)

// library_class_test.go — step 6 of spc-2610100613109045, criterion A13: an
// item the person has not classed is never filed by default. At a terminal
// library add asks the one class question and proceeds with the answer; off
// one it refuses naming the flags; --private and --public are the person's
// answer, so neither is asked about; --confidential is gone with no alias
// (decision 9, adr-40). Nothing is written in either refusal.

// classRun runs `abcd library add` with every stream a real file, so the
// drawn door's terminal check is what decides. atTerminal stands each stream
// in for a terminal and answers the class question with answer ("" fails the
// test if the question is put at all). It returns the exit code and stderr.
func classRun(t *testing.T, atTerminal bool, answer string, args ...string) (int, string) {
	t.Helper()
	in, keys := pipeWith(t, "")
	stdout, stderr := tempStream(t, "stdout"), tempStream(t, "stderr")
	swapStream, swapPut := isTerminalStream, drawnPut
	t.Cleanup(func() { isTerminalStream, drawnPut = swapStream, swapPut })
	if atTerminal {
		t.Setenv("ABCD_ACCESSIBLE", "1")
		isTerminalStream = func(*os.File) bool { return true }
	} else {
		_ = keys.Close()
	}
	drawnPut = func(tm ask.Terminal, a question.Ask) ([]ask.Answer, error) {
		if answer == "" {
			t.Fatalf("the class question was put: %+v", a)
		}
		q := a.Questions[0]
		if _, err := fmt.Fprintln(keys, number(q, answer)); err != nil {
			return nil, err
		}
		return tm.Put(a)
	}
	cmd := NewRootCommand()
	cmd.SetIn(in)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(append([]string{"library", "add"}, args...))
	err := cmd.Execute()
	code := 0
	if err != nil {
		code = 1
		var c interface{ ExitCode() int }
		if errors.As(err, &c) {
			code = c.ExitCode()
		}
	}
	got, _ := os.ReadFile(stderr.Name())
	msg := string(got)
	if err != nil {
		msg += err.Error()
	}
	return code, msg
}

// classCorpus makes the default corpus under the temp HOME and returns it.
func classCorpus(t *testing.T) string {
	t.Helper()
	home, _ := sourceCheckout(t)
	if code, out, errs := runLibrary(t, "init"); code != 0 {
		t.Fatalf("init: %d %s%s", code, out, errs)
	}
	return abcdhome.Path(home, "sources")
}

// filedUnder reports the class folder key sits in, "" when it sits in none.
func filedUnder(t *testing.T, corpus, key string) string {
	t.Helper()
	var where []string
	for _, class := range []string{"confidential", "public"} {
		if _, err := os.Stat(filepath.Join(corpus, class, key)); err == nil {
			where = append(where, class)
		}
	}
	if len(where) > 1 {
		t.Fatalf("%s sits in both class folders", key)
	}
	return strings.Join(where, "")
}

// noEntry fails when sources.json names key.
func noEntry(t *testing.T, corpus, key string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(corpus, "sources.json"))
	if err != nil {
		return
	}
	var entries []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("sources.json: %v", err)
	}
	for _, e := range entries {
		if e.ID == key {
			t.Fatalf("a refused add wrote the entry %s", key)
		}
	}
}

func stubArgs(key string, extra ...string) []string {
	return append([]string{"--key", key, "--title", "An Open Survey Of Harbours", "--url", "https://example.com/" + key}, extra...)
}

// TestLibraryAddAsksTheClassAtATerminal: given no class at a terminal, the
// class question is drawn, naming both answers and what each means, and the
// answer files the item; deciding later files nothing.
func TestLibraryAddAsksTheClassAtATerminal(t *testing.T) {
	corpus := classCorpus(t)

	code, errs := classRun(t, true, "private", stubArgs("k2026a")...)
	if code != 0 {
		t.Fatalf("answered private: exit %d\n%s", code, errs)
	}
	for _, want := range []string{"private or public", "Private", "Public", "confidential/", "public/"} {
		if !strings.Contains(errs, want) {
			t.Errorf("the class question lacks %q:\n%s", want, errs)
		}
	}
	if got := filedUnder(t, corpus, "k2026a"); got != "confidential" {
		t.Errorf("answered private, filed under %q", got)
	}

	if code, errs := classRun(t, true, "public", stubArgs("k2026b")...); code != 0 {
		t.Fatalf("answered public: exit %d\n%s", code, errs)
	}
	if got := filedUnder(t, corpus, "k2026b"); got != "public" {
		t.Errorf("answered public, filed under %q", got)
	}

	code, errs = classRun(t, true, "later", stubArgs("k2026c")...)
	if code != 2 || !strings.Contains(errs, "--private") || !strings.Contains(errs, "--public") {
		t.Fatalf("decided later: exit %d, want 2 naming both flags\n%s", code, errs)
	}
	if got := filedUnder(t, corpus, "k2026c"); got != "" {
		t.Errorf("deciding later filed the item under %s", got)
	}
	noEntry(t, corpus, "k2026c")
}

// TestLibraryAddRefusesWithoutAClassOffATerminal: off a terminal (a pipe, CI,
// a host's shell) no class is a refusal with exit 2 naming the flags, and
// nothing is written.
func TestLibraryAddRefusesWithoutAClassOffATerminal(t *testing.T) {
	corpus := classCorpus(t)
	code, errs := classRun(t, false, "", stubArgs("k2026d")...)
	if code != 2 {
		t.Fatalf("no class off a terminal: exit %d, want 2\n%s", code, errs)
	}
	for _, want := range []string{"--private", "--public", "class", "nothing was written"} {
		if !strings.Contains(errs, want) {
			t.Errorf("the refusal lacks %q:\n%s", want, errs)
		}
	}
	if got := filedUnder(t, corpus, "k2026d"); got != "" {
		t.Errorf("the refusal filed the item under %s", got)
	}
	noEntry(t, corpus, "k2026d")
}

// TestLibraryAddPrivateIsNotAsked: --private is the person's answer, so it is
// not asked even at a terminal, and it files under confidential/.
func TestLibraryAddPrivateIsNotAsked(t *testing.T) {
	corpus := classCorpus(t)
	if code, errs := classRun(t, true, "", stubArgs("k2026e", "--private")...); code != 0 {
		t.Fatalf("--private: exit %d\n%s", code, errs)
	}
	if got := filedUnder(t, corpus, "k2026e"); got != "confidential" {
		t.Errorf("--private filed under %q", got)
	}
}

// TestLibraryAddPublicIsNotAsked: --public is the person's answer too.
func TestLibraryAddPublicIsNotAsked(t *testing.T) {
	corpus := classCorpus(t)
	if code, errs := classRun(t, true, "", stubArgs("k2026f", "--public")...); code != 0 {
		t.Fatalf("--public: exit %d\n%s", code, errs)
	}
	if got := filedUnder(t, corpus, "k2026f"); got != "public" {
		t.Errorf("--public filed under %q", got)
	}
	code, errs := classRun(t, true, "", stubArgs("k2026g", "--private", "--public")...)
	if code != 2 || !strings.Contains(errs, "exclusive") {
		t.Fatalf("--private with --public: exit %d\n%s", code, errs)
	}
	noEntry(t, corpus, "k2026g")
}

// TestConfidentialFlagIsUnknown: --confidential went with no alias (adr-40);
// it is refused as any unknown flag on a corpus verb is, with its text
// withheld, and nothing is written.
func TestConfidentialFlagIsUnknown(t *testing.T) {
	corpus := classCorpus(t)
	code, errs := classRun(t, true, "", stubArgs("k2026h", "--confidential")...)
	if code != 2 || !strings.Contains(errs, "not understood") {
		t.Fatalf("--confidential: exit %d, want the unknown-flag refusal\n%s", code, errs)
	}
	if got := filedUnder(t, corpus, "k2026h"); got != "" {
		t.Errorf("--confidential filed the item under %s", got)
	}
	noEntry(t, corpus, "k2026h")
}

// TestLibraryPageAsksTheClassThroughTheQuestionTool: the page half of A13. An
// agent adding an item the person has not classed asks through the host's
// question tool, unless the person already said, and passes the answer as the
// flag; the removed flag is named nowhere on the page.
func TestLibraryPageAsksTheClassThroughTheQuestionTool(t *testing.T) {
	page, ok := commandFileBodies(t)["library"]
	if !ok {
		t.Fatal("commands/library.md is missing")
	}
	if strings.Contains(page, "--confidential") {
		t.Error("commands/library.md names --confidential, which went with no alias")
	}
	_, section, found := strings.Cut(page, "### Register a source")
	if !found {
		t.Fatal("commands/library.md has no Register a source section")
	}
	if i := strings.Index(section, "\n### "); i >= 0 {
		section = section[:i]
	}
	flat := strings.Join(strings.Fields(section), " ")
	for _, want := range []string{
		"the host's question tool",
		"unless the person already said",
		"--private",
		"--public",
		"never guessed",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("the Register a source section does not say %q:\n%s", want, section)
		}
	}
}
