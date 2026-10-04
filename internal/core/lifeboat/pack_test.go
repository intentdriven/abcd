package lifeboat

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
)

// okScan is a SecretScan that finds nothing — the default for pack tests not
// exercising the secret gate.
func okScan([]PlannedFile) error { return nil }

// packFixture builds a small git repo with a README and one ADR, so a pack has a
// root-commit SHA (for voyage) and a verbatim record to copy. Returns the repo.
func packFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", "# demo\n\nA project with a record.\n")
	write(".abcd/development/decisions/adrs/0001-example.md",
		"# 1. Example\n\n## Context\n\nWe decided.\n\n## Alternatives Considered\n\nOther things.\n")
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	run("add", "-A")
	run("commit", "-q", "-m", "root")
	return repo
}

// packInto packs repo into a fresh dest under a temp HOME (so voyage writes land
// in the test sandbox, never the real ~/.abcd.noindex). Returns the dest path.
func packInto(t *testing.T, repo string, scan SecretScan) (string, PackResult) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dest := filepath.Join(t.TempDir(), "lifeboat")
	res, err := Pack(repo, dest, scan)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	return dest, res
}

// TestPackWritesLifeboatAndLeavesSourceUnchanged is the headline contract: a pack
// materialises the lifeboat at dest and never touches the source tree.
func TestPackWritesLifeboatAndLeavesSourceUnchanged(t *testing.T) {
	repo := packFixture(t)
	before := treeHash(t, repo)

	dest, res := packInto(t, repo, okScan)

	if after := treeHash(t, repo); after != before {
		t.Errorf("pack mutated the source tree (hash %s -> %s)", before, after)
	}
	for _, rel := range []string{"_provenance.json", "coverage.json", "coverage.md", "docs/adrs/0001-example.md"} {
		if _, err := os.Stat(filepath.Join(dest, rel)); err != nil {
			t.Errorf("expected %s in the lifeboat: %v", rel, err)
		}
	}
	if res.FilesWritten == 0 || res.ManifestSHA256 == "" {
		t.Errorf("empty pack result: %+v", res)
	}
}

// TestPackProvenanceHashVerifies re-hashes the written tree with an independent
// implementation of adr-35 (sort by path, provenance excluded) and requires it to
// equal the provenance's manifest_sha256 — the integrity chain end to end.
func TestPackProvenanceHashVerifies(t *testing.T) {
	repo := packFixture(t)
	dest, _ := packInto(t, repo, okScan)

	var prov Provenance
	data, err := os.ReadFile(filepath.Join(dest, ProvenanceName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &prov); err != nil {
		t.Fatal(err)
	}

	type entry struct{ path, line string }
	var entries []entry
	err = filepath.Walk(dest, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dest, p)
		rel = filepath.ToSlash(rel)
		if rel == ProvenanceName {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		entries = append(entries, entry{rel, fmt.Sprintf("%x  %s\n", sha256.Sum256(b), rel)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	var buf strings.Builder
	for _, e := range entries {
		buf.WriteString(e.line)
	}
	want := fmt.Sprintf("%x", sha256.Sum256([]byte(buf.String())))
	if prov.ManifestSHA256 != want {
		t.Errorf("provenance manifest_sha256 %s does not verify against the written tree %s", prov.ManifestSHA256, want)
	}
}

// TestPackGateRefusesNonEmptyNonLifeboat: never overwrite a directory abcd did
// not produce.
func TestPackGateRefusesNonEmptyNonLifeboat(t *testing.T) {
	repo := packFixture(t)
	t.Setenv("HOME", t.TempDir())
	dest := t.TempDir() // exists, non-empty is arranged below
	if err := os.WriteFile(filepath.Join(dest, "keep.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Pack(repo, dest, okScan)
	if err == nil {
		t.Fatal("pack into a non-empty non-lifeboat dir must be refused")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Errorf("want an overwrite-refusal message, got: %v", err)
	}
	// The pre-existing file must survive an aborted pack.
	if _, err := os.Stat(filepath.Join(dest, "keep.txt")); err != nil {
		t.Errorf("refused pack destroyed the pre-existing directory: %v", err)
	}
}

// TestPackGateAllowsExistingLifeboatOverwrite: an abcd-produced directory (has a
// parseable _provenance.json) may be re-packed.
func TestPackGateAllowsExistingLifeboatOverwrite(t *testing.T) {
	repo := packFixture(t)
	t.Setenv("HOME", t.TempDir())
	dest := filepath.Join(t.TempDir(), "lb")
	if _, err := Pack(repo, dest, okScan); err != nil {
		t.Fatalf("first pack: %v", err)
	}
	// A stray file from the first pack must be gone after a clean re-pack.
	stray := filepath.Join(dest, "stray-not-in-plan.txt")
	if err := os.WriteFile(stray, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Pack(repo, dest, okScan); err != nil {
		t.Fatalf("re-pack over an abcd lifeboat must succeed: %v", err)
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("re-pack did not replace the prior lifeboat (stray file survived)")
	}
}

// TestPackGateRefusesSymlinkDest: a symlinked destination is refused.
func TestPackGateRefusesSymlinkDest(t *testing.T) {
	repo := packFixture(t)
	t.Setenv("HOME", t.TempDir())
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("cannot symlink: %v", err)
	}
	if _, err := Pack(repo, link, okScan); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Errorf("symlinked dest must be refused, got: %v", err)
	}
}

// TestPackGateRefusesOverlapWithSource covers dest == source, dest inside source,
// and dest an ancestor of source.
func TestPackGateRefusesOverlapWithSource(t *testing.T) {
	repo := packFixture(t)
	t.Setenv("HOME", t.TempDir())
	cases := map[string]string{
		"equal":    repo,
		"inside":   filepath.Join(repo, "out"),
		"ancestor": filepath.Dir(repo),
	}
	for name, dest := range cases {
		if _, err := Pack(repo, dest, okScan); err == nil || !strings.Contains(err.Error(), "overlaps the source") {
			t.Errorf("%s: dest overlapping source must be refused, got: %v", name, err)
		}
	}
}

// TestPathOverlapsCaseFolding proves that on a case-folding filesystem a
// differently-cased child still counts as overlapping the parent, so the pack
// overlap gate cannot be bypassed by re-casing a path segment.
//
// It asserts on pathOverlaps — the symbol the gate itself calls — and
// substitutes the fold predicate rather than skipping on a case-sensitive host,
// so the folding branch is proven wherever the suite runs. The unfolded branch
// states the other half: two genuinely distinct directories must still pass.
func TestPathOverlapsCaseFolding(t *testing.T) {
	parent := "/Users/x/repo"         // abcd-audit:allow
	child := "/Users/x/REPO/lifeboat" // abcd-audit:allow

	restore := caseFoldingFS
	t.Cleanup(func() { caseFoldingFS = restore })

	caseFoldingFS = func() bool { return true }
	if !pathOverlaps(child, parent) {
		t.Errorf("pathOverlaps(%q, %q) = false; overlap gate is bypassable via case", child, parent)
	}
	if !pathOverlaps(parent, child) {
		t.Errorf("pathOverlaps(%q, %q) = false; the overlap gate must answer the same either way round", parent, child)
	}

	caseFoldingFS = func() bool { return false }
	if pathOverlaps(child, parent) {
		t.Errorf("pathOverlaps(%q, %q) = true; on a case-sensitive filesystem these are distinct directories", child, parent)
	}
}

// TestRootSHARegexAcceptsSHA256 proves the voyage key regex accepts a 64-char
// SHA-256 root SHA, not just 40-char SHA-1 — otherwise SHA-256 repos never log a
// voyage.
func TestRootSHARegexAcceptsSHA256(t *testing.T) {
	sha1 := strings.Repeat("a", 40)
	sha256 := strings.Repeat("b", 64)
	if !rootSHARe.MatchString(sha1) {
		t.Errorf("40-char SHA-1 rejected")
	}
	if !rootSHARe.MatchString(sha256) {
		t.Errorf("64-char SHA-256 rejected")
	}
	if rootSHARe.MatchString(strings.Repeat("c", 50)) {
		t.Errorf("50-char string wrongly accepted")
	}
}

// TestPackGateRefusesDestThroughSymlinkIntoSource is the symlink-resolution
// guard: a destination reached through a symlinked parent that points into the
// source resolves into the source tree and must be refused — a purely lexical
// gate would miss it and write the lifeboat inside the source.
func TestPackGateRefusesDestThroughSymlinkIntoSource(t *testing.T) {
	repo := packFixture(t)
	t.Setenv("HOME", t.TempDir())
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(repo, link); err != nil {
		t.Skipf("cannot symlink: %v", err)
	}
	dest := filepath.Join(link, "lb") // link -> repo, so this resolves into the source
	if _, err := Pack(repo, dest, okScan); err == nil || !strings.Contains(err.Error(), "overlaps the source") {
		t.Errorf("dest through a symlink into the source must be refused, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(repo, "lb")); !os.IsNotExist(statErr) {
		t.Error("a refused pack created a directory inside the source")
	}
}

// TestSwapIntoPlaceRestoresPriorOnRenameFailure covers the rename-aside data-loss
// fix: if the staging→dest rename fails (here staging is absent), the prior
// lifeboat is restored intact rather than lost, and no backup is left behind.
func TestSwapIntoPlaceRestoresPriorOnRenameFailure(t *testing.T) {
	parent := t.TempDir()
	dest := filepath.Join(parent, "lb")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "marker"), []byte("prior"), 0o644); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(parent, "staging-does-not-exist")
	if err := swapIntoPlace(staging, dest); err == nil {
		t.Fatal("expected the swap to fail when staging is absent")
	}
	b, err := os.ReadFile(filepath.Join(dest, "marker"))
	if err != nil || string(b) != "prior" {
		t.Errorf("prior lifeboat not restored after a failed swap: %v / %q", err, b)
	}
	entries, _ := os.ReadDir(parent)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".abcd-prev-") {
			t.Errorf("backup directory left behind: %s", e.Name())
		}
	}
}

// TestPackGateRefusesInsideDotGit: a destination inside a .git directory is
// refused.
func TestPackGateRefusesInsideDotGit(t *testing.T) {
	repo := packFixture(t)
	t.Setenv("HOME", t.TempDir())
	dest := filepath.Join(t.TempDir(), "somerepo", ".git", "lifeboat")
	if _, err := Pack(repo, dest, okScan); err == nil || !strings.Contains(err.Error(), ".git") {
		t.Errorf("dest inside .git must be refused, got: %v", err)
	}
}

// TestPackRefusesOnSecretHardFailAndWritesNothing: a hard-fail from the injected
// scan refuses the whole pack and leaves no destination behind.
func TestPackRefusesOnSecretHardFailAndWritesNothing(t *testing.T) {
	repo := packFixture(t)
	t.Setenv("HOME", t.TempDir())
	dest := filepath.Join(t.TempDir(), "lb")
	badScan := func([]PlannedFile) error { return fmt.Errorf("1 hard-fail secret") }
	_, err := Pack(repo, dest, badScan)
	if err == nil || !strings.Contains(err.Error(), "hard-fail") {
		t.Fatalf("pack must refuse on a secret hard-fail, got: %v", err)
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Errorf("a refused pack left a destination behind: %v", statErr)
	}
}

// TestPackNilScanRefused: the secret scan is mandatory.
func TestPackNilScanRefused(t *testing.T) {
	repo := packFixture(t)
	if _, err := Pack(repo, filepath.Join(t.TempDir(), "lb"), nil); err == nil {
		t.Error("a nil scan must be refused (fail closed)")
	}
}

// TestPackStripsMarkerBlockFromCopiedRecord: a verbatim record carrying an abcd
// marker block has it stripped before it is written, and the provenance hash
// covers the stripped bytes (plan/pack parity).
func TestPackStripsMarkerBlockFromCopiedRecord(t *testing.T) {
	repo := packFixture(t)
	adr := filepath.Join(repo, ".abcd/development/decisions/adrs/0002-marked.md")
	content := "# 2. Marked\n\n<!-- BEGIN ABCD -->\nstale loader text\n<!-- END ABCD -->\n\n## Context\n\nBody.\n"
	if err := os.WriteFile(adr, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	dest, _ := packInto(t, repo, okScan)

	packed, err := os.ReadFile(filepath.Join(dest, "docs/adrs/0002-marked.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(packed, []byte("BEGIN ABCD")) {
		t.Errorf("packed record still carries the marker block:\n%s", packed)
	}
	if !bytes.Contains(packed, []byte("## Context")) {
		t.Errorf("marker strip damaged the record body:\n%s", packed)
	}
}

// TestValidRelPath is the write-path validator's table.
func TestValidRelPath(t *testing.T) {
	good := []string{"a.md", "docs/adrs/x.md", "brief/01-product/README.md"}
	bad := []string{"", "/abs", "../up", "a/../b", "a//b", "./a", "a/.", "with\x00nul", "ctrl\x01name"}
	for _, p := range good {
		if !validRelPath(p) {
			t.Errorf("validRelPath(%q) = false, want true", p)
		}
	}
	for _, p := range bad {
		if validRelPath(p) {
			t.Errorf("validRelPath(%q) = true, want false", p)
		}
	}
}

// TestNotPresentTreatsENOTDIRAsAbsent is the attack/behaviour test for the
// destination gate's ENOTDIR handling: when a prefix component of the destination
// path is a regular file, stat returns ENOTDIR, which means the destination cannot
// exist and is therefore "absent" (writable) — not an uninterpretable error that
// refused a valid destination. ErrNotExist is absent too; an unrelated error is not.
func TestNotPresentTreatsENOTDIRAsAbsent(t *testing.T) {
	if !notPresent(syscall.ENOTDIR) {
		t.Error("notPresent(ENOTDIR) = false; a file shadowing a path prefix means the dest is absent")
	}
	if !notPresent(os.ErrNotExist) {
		t.Error("notPresent(ErrNotExist) = false")
	}
	if notPresent(syscall.EACCES) {
		t.Error("notPresent(EACCES) = true; an unrelated stat error must not read as absent")
	}
}
