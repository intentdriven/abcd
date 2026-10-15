package cli

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// library_test.go — step 5 of spc-2610100613109045: the memory and source
// sub-trees are one person's verb, library (itd-2610090831227812, P4). Every
// sub-verb keeps its name (decision 7), bare library renders both halves
// (decision 8), and the two old names are unknown commands with no alias
// (adr-40), their notes naming what replaced them.

// TestLibraryCarriesEveryMemoryAndSourceSubVerb holds the merge whole: each
// sub-verb memory and source registered is a sub-verb of library under the same
// name, and nothing else is.
func TestLibraryCarriesEveryMemoryAndSourceSubVerb(t *testing.T) {
	root := NewRootCommand()
	var lib []string
	found := false
	for _, c := range root.Commands() {
		if c.Name() != "library" {
			continue
		}
		found = true
		for _, sub := range c.Commands() {
			lib = append(lib, sub.Name())
		}
	}
	if !found {
		t.Fatal("the binary registers no library verb")
	}
	sort.Strings(lib)
	want := []string{
		// memory's sub-verbs
		"ask", "ingest", "lint",
		// source's sub-verbs
		"add", "cite-check", "declassify", "init", "ledger", "sync-banlist",
	}
	sort.Strings(want)
	if strings.Join(lib, " ") != strings.Join(want, " ") {
		t.Errorf("library's sub-verbs = %v, want %v", lib, want)
	}
}

// TestSourceIsAnUnknownCommand is the old-names rule for step 5: the retired
// spelling refuses as an unknown command, the note names library, and nothing
// calls the binary stale for not knowing a name that was retired. The old
// memory verb's name is not retired any more: the verb split gave it to the
// agents' memory-notes verb (itd-2610091918433290), which
// memory_test.go covers.
func TestSourceIsAnUnknownCommand(t *testing.T) {
	for _, old := range []string{"source"} {
		t.Run(old, func(t *testing.T) {
			root := stalePluginRoot(t)
			setExecutable(t, filepath.Join(root, "abcd"))
			code, stdout, stderr := runMain(t, old)
			if code != 2 || stdout != "" {
				t.Fatalf("`abcd %s` exit = %d, stdout %q; want exit 2 and an empty stdout", old, code, stdout)
			}
			if !strings.Contains(stderr, "unknown command") {
				t.Errorf("`abcd %s` is not refused as an unknown command:\n%s", old, stderr)
			}
			if !strings.Contains(stderr, "abcd library") {
				t.Errorf("the refusal does not name the library verb that replaced it:\n%s", stderr)
			}
			for _, not := range []string{"predates", "make build", "abcd update"} {
				if strings.Contains(stderr, not) {
					t.Errorf("the refusal names %q:\n%s", not, stderr)
				}
			}
		})
	}
}

// libraryJSON is the bare --json envelope: each half present, or a note saying
// why it could not be read.
type libraryJSON struct {
	Corpus     json.RawMessage `json:"corpus"`
	CorpusNote string          `json:"corpus_note"`
	Memory     json.RawMessage `json:"memory"`
	MemoryNote string          `json:"memory_note"`
}

// TestBareLibraryRendersBothHalves is decision 8: bare library renders the
// corpus and the checkout's memory store, an unreadable half is one line naming
// what would create it, and the verb refuses only when neither half reads. Bare
// source exited 3 with no corpus; bare library inside a checkout renders.
func TestBareLibraryRendersBothHalves(t *testing.T) {
	t.Run("no corpus inside a checkout renders the memory half and a corpus line", func(t *testing.T) {
		home, _ := sourceCheckout(t)
		code, stdout, stderr := runLibrary(t)
		if code != 0 {
			t.Fatalf("bare library with no corpus inside a checkout exited %d, want 0\n%s%s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, "memory") || !strings.Contains(stdout, "page") {
			t.Errorf("the memory half is not rendered:\n%s", stdout)
		}
		if !strings.Contains(stdout, "no sources corpus") || !strings.Contains(stdout, "abcd library init") {
			t.Errorf("the corpus half is not one line naming `abcd library init`:\n%s", stdout)
		}
		noHomePath(t, home, stdout+stderr)

		code, stdout, stderr = runLibrary(t, "--json")
		if code != 0 {
			t.Fatalf("bare library --json exited %d\n%s%s", code, stdout, stderr)
		}
		var env libraryJSON
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("not JSON: %v\n%s", err, stdout)
		}
		if string(env.Corpus) != "null" || !strings.Contains(env.CorpusNote, "abcd library init") {
			t.Errorf("corpus = %s, corpus_note = %q; want null and a note naming `abcd library init`", env.Corpus, env.CorpusNote)
		}
		if len(env.Memory) == 0 || string(env.Memory) == "null" || env.MemoryNote != "" {
			t.Errorf("memory = %s, memory_note = %q; want the store's status and no note", env.Memory, env.MemoryNote)
		}
	})

	t.Run("a corpus outside a checkout renders the corpus half and a memory line", func(t *testing.T) {
		_, repo := sourceCheckout(t)
		if code, out, errb := runLibrary(t, "init"); code != 0 {
			t.Fatalf("library init: exit %d\n%s%s", code, out, errb)
		}
		t.Chdir(filepath.Dir(repo)) // the temp HOME: no checkout above it
		code, stdout, stderr := runLibrary(t)
		if code != 0 {
			t.Fatalf("bare library with a corpus outside a checkout exited %d, want 0\n%s%s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, "corpus") || !strings.Contains(stdout, "0 confidential, 0 public") {
			t.Errorf("the corpus half is not rendered:\n%s", stdout)
		}
		if !strings.Contains(stdout, "memory") || !strings.Contains(stdout, "checkout") {
			t.Errorf("the memory half is not one line naming the checkout it needs:\n%s", stdout)
		}
	})

	t.Run("neither half refuses", func(t *testing.T) {
		_, repo := sourceCheckout(t)
		t.Chdir(filepath.Dir(repo))
		code, stdout, stderr := runLibrary(t)
		if code == 0 {
			t.Fatalf("bare library with neither half readable exited 0:\n%s%s", stdout, stderr)
		}
		if stdout != "" {
			t.Errorf("a refusal wrote to stdout: %q", stdout)
		}
		for _, want := range []string{"no sources corpus", "abcd library init", "repository"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("the refusal does not name %q:\n%s", want, stderr)
			}
		}
	})
}
