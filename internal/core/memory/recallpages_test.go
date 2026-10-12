package memory

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// recallPageText is a typed page with a recall: list.
func recallPageText(summary string, recall ...string) string {
	return "---\nrecall: [" + strings.Join(recall, ", ") + "]\nsource:\n  class: session_memory\n  ingested_at: 2026-10-01\n---\n# " + summary + "\n\nThe body.\n"
}

func writeStoreFile(t *testing.T, dir, name, text string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, name), mode); err != nil {
		t.Fatal(err)
	}
}

// TestRecallPagesReadsTheIndexedPages: the recall's read yields each page's
// filename, its index line's summary, its body and its recall: list, for
// pages named <type>_<domain>_<slug>.md at the store's top level only.
func TestRecallPagesReadsTheIndexedPages(t *testing.T) {
	repo := t.TempDir()
	mem := filepath.Join(repo, ".abcd", "memory")
	writeStoreFile(t, mem, "fact_eng_auth-flow.md", recallPageText("Auth uses short-lived tokens", "auth", "login"), 0o644)
	writeStoreFile(t, mem, "fact_eng_silent.md", "---\nsource:\n  class: session_memory\n---\n# No recall list\n", 0o644)
	writeStoreFile(t, mem, "index.md", recallPageText("not a page", "auth"), 0o644)
	writeStoreFile(t, mem, "notes.md", recallPageText("not a typed name", "auth"), 0o644)
	writeStoreFile(t, filepath.Join(mem, "sources"), "fact_eng_nested.md", recallPageText("nested", "auth"), 0o644)

	pages, refused := ReadRecallPages(repo)
	if len(refused) != 0 {
		t.Fatalf("refused = %+v", refused)
	}
	var names []string
	for _, p := range pages {
		names = append(names, p.Filename)
	}
	if !slices.Equal(names, []string{"fact_eng_auth-flow.md", "fact_eng_silent.md"}) {
		t.Fatalf("pages = %v", names)
	}
	p := pages[0]
	if p.Summary != "Auth uses short-lived tokens" || !slices.Equal(p.Recall, []string{"auth", "login"}) ||
		!strings.Contains(p.Body, "The body.") || strings.Contains(p.Body, "recall:") {
		t.Fatalf("page = %+v", p)
	}
	if len(pages[1].Recall) != 0 {
		t.Fatalf("a page with no recall list read one: %+v", pages[1])
	}
	if got, refused := ReadRecallPages(t.TempDir()); len(got) != 0 || len(refused) != 0 {
		t.Fatalf("an absent store read %v %v", got, refused)
	}
}

// TestRecallPagesRefuseASymlinkedPage: a page committed as a symlink to a
// file outside the store is never followed — it is not listed as a page, so
// its target never reaches the recall — in either scope.
func TestRecallPagesRefuseASymlinkedPage(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte(recallPageText("LEAKED outside text", "auth")), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Join(repo, ".abcd", "memory"), abcdhome.Path(home, "memory")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		plantSymlink(t, outside, filepath.Join(dir, "fact_eng_auth-flow.md"))
	}
	workspace, _ := ReadRecallPages(repo)
	machine, _ := ReadMachineRecallPages(home)
	for _, p := range append(workspace, machine...) {
		if strings.Contains(p.Summary+p.Body, "LEAKED") {
			t.Fatalf("a symlinked page was followed: %+v", p)
		}
	}
}

// TestRecallPagesRefuseASymlinkedStore: the workspace store committed as a
// directory symlink, and a symlinked ~/.abcd.noindex or memory/, are refused
// whole and named, never walked.
func TestRecallPagesRefuseASymlinkedStore(t *testing.T) {
	target := t.TempDir()
	writeStoreFile(t, target, "fact_eng_auth-flow.md", recallPageText("LEAKED through the link", "auth"), 0o600)

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	plantSymlink(t, target, filepath.Join(repo, ".abcd", "memory"))
	pages, refused := ReadRecallPages(repo)
	var unsafe *UnsafeStorePathError
	if len(pages) != 0 || len(refused) != 1 || refused[0].File != ".abcd/memory/" || !errors.As(refused[0].Err, &unsafe) {
		t.Fatalf("a symlinked workspace store was read: pages %+v refused %+v", pages, refused)
	}

	for name, link := range map[string]func(home string){
		"memory": func(home string) {
			if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
				t.Fatal(err)
			}
			plantSymlink(t, target, abcdhome.Path(home, "memory"))
		},
		"abcd-home": func(home string) {
			real := filepath.Join(t.TempDir(), "dotfiles")
			if err := os.MkdirAll(filepath.Join(real, "memory"), 0o700); err != nil {
				t.Fatal(err)
			}
			writeStoreFile(t, filepath.Join(real, "memory"), "fact_eng_auth-flow.md", recallPageText("LEAKED through the link", "auth"), 0o600)
			plantSymlink(t, real, abcdhome.Path(home))
		},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			link(home)
			pages, refused := ReadMachineRecallPages(home)
			if len(pages) != 0 || len(refused) != 1 || !errors.Is(refused[0].Err, fsutil.ErrHomeScopeSymlinked) {
				t.Fatalf("a symlinked machine scope was read: pages %+v refused %+v", pages, refused)
			}
			if strings.Contains(refused[0].Err.Error(), home) {
				t.Fatalf("the refusal names the expanded home: %v", refused[0].Err)
			}
		})
	}
}

// TestMachineRecallPagesAreReadAsTheMachineFilesAre: a machine page is read
// as the machine memory.json is — a regular file this account owns that no
// one else can write — and one others can write is refused and named, while
// the scope's other pages are still read. An absent scope, or no home, reads
// nothing and creates nothing.
func TestMachineRecallPagesAreReadAsTheMachineFilesAre(t *testing.T) {
	home := t.TempDir()
	dir := abcdhome.Path(home, "memory")
	writeStoreFile(t, dir, "fact_eng_auth-flow.md", recallPageText("Kept to this account", "auth"), 0o600)
	writeStoreFile(t, dir, "fact_eng_auth-keys.md", recallPageText("Anyone can rewrite this", "auth"), 0o666)

	pages, refused := ReadMachineRecallPages(home)
	if len(pages) != 1 || pages[0].Filename != "fact_eng_auth-flow.md" {
		t.Fatalf("pages = %+v", pages)
	}
	if len(refused) != 1 || refused[0].File != abcdhome.Display("memory", "fact_eng_auth-keys.md") ||
		!strings.Contains(refused[0].Err.Error(), "writable by others") {
		t.Fatalf("refused = %+v", refused)
	}

	empty := t.TempDir()
	for _, h := range []string{empty, "", "relative/home"} {
		if pages, refused := ReadMachineRecallPages(h); len(pages) != 0 || len(refused) != 0 {
			t.Fatalf("home %q read %+v %+v", h, pages, refused)
		}
	}
	if _, err := os.Lstat(abcdhome.Path(empty)); !os.IsNotExist(err) {
		t.Fatalf("the read created %s: %v", abcdhome.Path(empty), err)
	}
}
