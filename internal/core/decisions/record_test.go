package decisions

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// git runs git in dir with an identity under the shared hermetic environment
// (gittest.Env), so neither an ambient GIT_DIR nor the developer's own settings
// (a merge driver, a signing key) can reach the test.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(gittest.Env(t),
		"GIT_AUTHOR_NAME=Example", "GIT_AUTHOR_EMAIL=example@example.com",
		"GIT_COMMITTER_NAME=Example", "GIT_COMMITTER_EMAIL=example@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// sharedTierRepo is a git repository with one commit holding the shared tier,
// so `git status` afterwards names exactly what a recording added.
func sharedTierRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Setenv("HOME", t.TempDir())
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q")
	if err := os.MkdirAll(filepath.Join(root, ".abcd", "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "work", "CONTEXT.md"), []byte("# Context\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "base")
	return root
}

func status(t *testing.T, root string) []string {
	t.Helper()
	out := strings.TrimRight(git(t, root, "status", "--porcelain", "--untracked-files=all"), "\n")
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

var recordPathRe = regexp.MustCompile(`^\.abcd/work/decisions/dcn-[0-9]{16}-[a-z0-9-]+\.md$`)

// TestRecordWritesOneFileAndNothingElse is A1: recording a decision adds one
// file, named by an adr-45 id, in the folder, and changes no other path, the
// mint lock included. The record reads back through the folder's own reader.
func TestRecordWritesOneFileAndNothingElse(t *testing.T) {
	root := sharedTierRepo(t)

	d, err := Record(root, "Decisions are one flat file each")
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if !recordPathRe.MatchString(d.Path) {
		t.Errorf("record path %q is not a dcn record in the folder", d.Path)
	}
	if got := status(t, root); len(got) != 1 || got[0] != "?? "+d.Path {
		t.Fatalf("git status after one recording = %q, want exactly one new path, %s", got, d.Path)
	}
	if !strings.HasPrefix(d.Text, d.Date+" — Decisions are one flat file each") {
		t.Errorf("body = %q, want it to open with %s — and the text", d.Text, d.Date)
	}
	listed, skipped, err := List(root)
	if err != nil || len(skipped) != 0 || len(listed) != 1 || listed[0] != d {
		t.Errorf("List after Record = %+v, %+v, %v; want exactly %+v", listed, skipped, err, d)
	}
}

// TestRecordResolvesTheCheckoutRoot: a recording from a subdirectory lands in
// the checkout's folder, and outside every repository it refuses and writes
// nothing.
func TestRecordResolvesTheCheckoutRoot(t *testing.T) {
	root := sharedTierRepo(t)
	sub := filepath.Join(root, "internal", "core")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := Record(sub, "Recorded from a package directory")
	if err != nil {
		t.Fatalf("Record from a subdirectory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(d.Path))); err != nil {
		t.Errorf("the record is not in the checkout's folder: %v", err)
	}

	plain := t.TempDir()
	if _, err := Record(plain, "Recorded outside any repository"); err == nil {
		t.Error("Record outside a repository succeeded; it must refuse")
	}
	if _, err := os.Stat(filepath.Join(plain, ".abcd")); err == nil {
		t.Error("a refused recording laid a folder outside the repository")
	}
}

// TestTwoBranchesRecordingMergeCleanly is A4: two branches cut from one base
// each record one decision, and both merge into a third branch with no conflict,
// in a repository with no .gitattributes, so no merge attribute or driver is
// doing the work.
func TestTwoBranchesRecordingMergeCleanly(t *testing.T) {
	root := sharedTierRepo(t)
	if _, err := os.Stat(filepath.Join(root, ".gitattributes")); err == nil {
		t.Fatal("the fixture must carry no .gitattributes")
	}

	var paths []string
	for _, branch := range []string{"left", "right"} {
		git(t, root, "checkout", "-q", "-b", branch, "main")
		d, err := Record(root, "A decision recorded on the "+branch+" branch")
		if err != nil {
			t.Fatalf("Record on %s: %v", branch, err)
		}
		paths = append(paths, d.Path)
		git(t, root, "add", "-A")
		git(t, root, "commit", "-q", "-m", "record on "+branch)
	}

	git(t, root, "checkout", "-q", "-b", "integration", "main")
	for _, branch := range []string{"left", "right"} {
		// git fails the helper on a non-zero exit, a conflict included.
		git(t, root, "merge", "--no-ff", "-q", "-m", "merge "+branch, branch)
	}
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			t.Errorf("%s is missing after both merges: %v", p, err)
		}
		if got := strings.TrimSpace(git(t, root, "check-attr", "merge", "--", p)); got != p+": merge: unspecified" {
			t.Errorf("git check-attr merge %s = %q, want the attribute unspecified", p, got)
		}
	}
	listed, _, err := List(root)
	if err != nil || len(listed) != 2 {
		t.Errorf("List after both merges = %d records, %v; want 2", len(listed), err)
	}
}

// TestListingIsDateOrderedAndDerived is A2 for the listing: records written out
// of id order list newest first, in both forms, each with its id and full text,
// and reading the listing changes no file.
func TestListingIsDateOrderedAndDerived(t *testing.T) {
	root := sharedTierRepo(t)
	writeRecord(t, root, "dcn-2610141200005555-the-third.md", record("dcn-2610141200005555", "the-third", "2026-10-14", "the third"))
	writeRecord(t, root, "dcn-2609010000001111-the-first.md", record("dcn-2609010000001111", "the-first", "2026-09-01", "the first"))
	writeRecord(t, root, "dcn-2610010930002222-the-second.md", record("dcn-2610010930002222", "the-second", "2026-10-01", "the second"))
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "three decisions")

	l, err := ReadListing(root)
	if err != nil {
		t.Fatalf("ReadListing: %v", err)
	}
	want := []string{"dcn-2610141200005555", "dcn-2610010930002222", "dcn-2609010000001111"}

	text := l.Text()
	last := -1
	for _, id := range want {
		i := strings.Index(text, id+"\n")
		if i < 0 || i < last {
			t.Fatalf("the text listing does not hold %s after the one before it:\n%s", id, text)
		}
		last = i
	}
	if !strings.Contains(text, "dcn-2609010000001111\n2026-09-01 — the first\n") {
		t.Errorf("a block is not the id and then the body:\n%s", text)
	}

	raw, err := json.Marshal(l.Entries())
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("the JSON form is not an array of objects: %v\n%s", err, raw)
	}
	if len(got) != len(want) {
		t.Fatalf("JSON listing holds %d records, want %d: %s", len(got), len(want), raw)
	}
	for i, id := range want {
		if got[i]["id"] != id {
			t.Errorf("JSON listing[%d] = %s, want %s (newest first)", i, got[i]["id"], id)
		}
	}
	for i, id := range want {
		e := got[i]
		if e["id"] != id {
			t.Errorf("JSON entry %d is %s, want %s", i, e["id"], id)
		}
		for _, k := range []string{"id", "date", "slug", "path", "text"} {
			if e[k] == "" {
				t.Errorf("JSON entry %s has no %s: %v", id, k, e)
			}
		}
	}
	if s := status(t, root); len(s) != 0 {
		t.Errorf("reading the listing changed the tree: %q", s)
	}
	if strings.Contains(text, "DECISIONS.md") {
		t.Errorf("with no frozen log the listing names one:\n%s", text)
	}
}

// TestListingNamesAFrozenLog: where a managed repository kept a single log
// before the folder, the listing ends with one line naming it as earlier
// history it does not list, and reads nothing from it.
func TestListingNamesAFrozenLog(t *testing.T) {
	root := sharedTierRepo(t)
	frozen := filepath.Join(root, ".abcd", "work", "DECISIONS.md")
	if err := os.WriteFile(frozen, []byte("# DECISIONS\n\n- 2026-01-02 — an old entry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Record(root, "A decision recorded after the log froze"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	l, err := ReadListing(root)
	if err != nil {
		t.Fatalf("ReadListing: %v", err)
	}
	text := l.Text()
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	tail := lines[len(lines)-1]
	if !strings.Contains(tail, ".abcd/work/DECISIONS.md") || !strings.Contains(tail, "does not list") {
		t.Errorf("the listing's last line does not name the frozen log as unlisted history: %q\n%s", tail, text)
	}
	if strings.Contains(text, "an old entry") {
		t.Errorf("the listing read entries out of the frozen log:\n%s", text)
	}
	if len(l.Entries()) != 1 {
		t.Errorf("the JSON form holds %d records, want only the folder's one", len(l.Entries()))
	}
}

// TestRecordRefusesWithoutTheSharedTier: a repository with no shared tier has
// nowhere to commit a decision, so the writer refuses naming `ahoy install` and
// lays no directory.
func TestRecordRefusesWithoutTheSharedTier(t *testing.T) {
	root := sharedTierRepo(t)
	if err := os.RemoveAll(filepath.Join(root, ".abcd")); err != nil {
		t.Fatal(err)
	}
	_, err := Record(root, "A decision with nowhere to go")
	if err == nil {
		t.Fatal("Record without the shared tier succeeded; it must refuse")
	}
	if !errors.Is(err, ErrNoSharedTier) || !strings.Contains(err.Error(), "ahoy install") {
		t.Errorf("the refusal is not ErrNoSharedTier naming `ahoy install`: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".abcd")); err == nil {
		t.Error("a refused recording laid .abcd/ anyway")
	}
}

// TestRecordRefusesAnUnslugableText: a text with nothing slug-able cannot name
// its file, and neither can an empty one; both refuse with nothing written.
func TestRecordRefusesAnUnslugableText(t *testing.T) {
	root := sharedTierRepo(t)
	for _, text := range []string{"///", "  ", ""} {
		if d, err := Record(root, text); err == nil {
			t.Errorf("Record(%q) wrote %+v; want a refusal", text, d)
		}
	}
	if _, err := Record(root, "—…!"); !errors.Is(err, ErrNoSlug) {
		t.Errorf("Record over punctuation alone = %v; want ErrNoSlug", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(RelDir))); err == nil {
		t.Error("a refused recording laid the decisions folder")
	}
	if s := status(t, root); len(s) != 0 {
		t.Errorf("a refused recording changed the tree: %q", s)
	}
}

// TestRecordRefusesADegradedScanner: the text is committed prose and names the
// file, so a recording under a silently weakened detector refuses rather than
// writing.
func TestRecordRefusesADegradedScanner(t *testing.T) {
	root := sharedTierRepo(t)
	cfg := filepath.Join(root, ".abcd", "config")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "pii.json"), []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Record(root, "A decision under a broken scanner")
	if err == nil || !strings.Contains(err.Error(), "degraded") {
		t.Fatalf("Record under a degraded scanner = %v; want a refusal naming it degraded", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(RelDir))); err == nil {
		t.Error("a refused recording laid the decisions folder")
	}
}
