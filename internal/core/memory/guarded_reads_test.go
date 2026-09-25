package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The memory store sits inside the repo working tree — a trust boundary
// (ingest.go: maxMemoryPageBytes) — so every store read must refuse a
// committed symlink rather than follow it. These tests plant mode-120000
// leaves pointing at readable files OUTSIDE the store and assert the content
// never crosses: a follow would succeed and leak the target, so each test
// fails against a raw os.ReadFile and passes against the guarded primitive.

func plantSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func TestBareRefusesSymlinkedStoreFiles(t *testing.T) {
	root := t.TempDir()
	mem := filepath.Join(root, ".abcd", "memory")
	if err := os.MkdirAll(mem, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.md")
	if err := os.WriteFile(outside, []byte("- injected contradiction\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plantSymlink(t, outside, filepath.Join(mem, "contradictions.md"))

	status, err := Bare(root)
	if err != nil {
		t.Fatalf("Bare: %v", err)
	}
	if len(status.Contradictions) != 0 {
		t.Fatalf("a symlinked contradictions.md was followed into the status: %q", status.Contradictions)
	}
}

func TestLintSkipsSymlinkedTypedPage(t *testing.T) {
	root := t.TempDir()
	mem := filepath.Join(root, ".abcd", "memory")
	if err := os.MkdirAll(mem, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.md")
	page := "---\nsource: sha256:0000\n---\n\nInjected body.\n"
	if err := os.WriteFile(outside, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	plantSymlink(t, outside, filepath.Join(mem, "fact_eng_injected.md"))

	store, err := openStore(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, p := range store.typedPages() {
		if p.rel == "fact_eng_injected.md" {
			t.Fatal("a symlinked page was followed and classified as a typed memory page")
		}
	}
}

func TestLoadQuotationBudgetRefusesSymlinkedConfig(t *testing.T) {
	root := t.TempDir()
	mem := filepath.Join(root, ".abcd", "memory")
	if err := os.MkdirAll(mem, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.json")
	cfg := `{"quotation_budget": {"per_page_pct": 0.99}}`
	if err := os.WriteFile(outside, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	plantSymlink(t, outside, filepath.Join(mem, "config.json"))

	store, err := openStore(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got := loadQuotationBudget(store)
	if got.PerPagePct == 0.99 {
		t.Fatal("a symlinked config.json was followed into the quotation budget")
	}
	if def := defaultBudget(); got != def {
		t.Fatalf("refused config must fall back to the default budget: got %+v want %+v", got, def)
	}
}

func TestTriStateReadRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside.md")
	if err := os.WriteFile(outside, []byte("target"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "page.md")
	plantSymlink(t, outside, link)

	_, present, err := triStateRead(link)
	if err == nil {
		t.Fatalf("a symlinked page must refuse on the write path, got present=%v err=nil", present)
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("want the writer-contract refusal, got: %v", err)
	}

	// Absent stays the soft branch.
	if _, present, err := triStateRead(filepath.Join(dir, "absent.md")); err != nil || present {
		t.Fatalf("absent file must stay (\"\", false, nil), got present=%v err=%v", present, err)
	}
}

func TestLicenceProbesRefuseSymlinks(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside.txt")
	if err := os.WriteFile(outside, []byte(`{"license": "MIT"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	plantSymlink(t, outside, filepath.Join(src, "package.json"))
	if got := manifestLicence(src); got != "" {
		t.Fatalf("a symlinked package.json was followed: %q", got)
	}

	outsideLic := filepath.Join(dir, "outside-licence.txt")
	if err := os.WriteFile(outsideLic, []byte("SPDX-License-Identifier: MIT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plantSymlink(t, outsideLic, filepath.Join(src, "LICENSE"))
	if got := licenceFileLicence(src); got != "" {
		t.Fatalf("a symlinked LICENSE was followed: %q", got)
	}
}
