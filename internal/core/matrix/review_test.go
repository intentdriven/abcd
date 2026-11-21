package matrix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/reading"
	"github.com/intentdriven/abcd/internal/gittest"
)

// Every reviewer in a model-matrix run is handed a self-contained request and
// nothing else (adr-2610161733400551, spc-2610161949559050 step 5): the
// normalised diff of one copy, assembled through the reading surface's include
// table and hashed manifest, under an opaque label, with commit trailers,
// author identities, branch names, pull-request text and run metadata left
// out.

const reviewRun = "mx-261121000000-cd34"

// builtCopy is a frozen commit and one copy's clone of it, worked the way a
// builder leaves it: its source changed, a file added and one deleted, a
// decision record naming its model, a commit carrying its trailer and its
// author on a branch named after its cell, and a file that wrote down where
// it ran.
func builtCopy(t *testing.T) (src *gittest.Repo, frozen, store, clone string) {
	t.Helper()
	src = gittest.NewRepo(t)
	src.Write("README.md", "# widget\n")
	src.Write("widget.go", "package widget\n\nfunc Size() int {\n\treturn 1\n}\n")
	src.Write("widget_test.go", "package widget\n\nimport \"testing\"\n\nfunc TestSize(t *testing.T) {}\n")
	src.Write(".abcd/work/CONTEXT.md", "orientation\n")
	src.Commit("frozen")
	frozen = src.Git("rev-parse", "HEAD")
	store = filepath.Join(t.TempDir(), "matrix-store")
	var err error
	clone, err = makeClone(src.Root(), store, reviewRun, frozen, TrialKey{Task: "widget", Cell: "gpt5-builder", Repeat: 1})
	if err != nil {
		t.Fatal(err)
	}
	c := gittest.OpenRepo(t, clone)
	c.Git("checkout", "-q", "-b", "build/"+reviewRun+"-gpt5-builder-lane-1")
	c.Write("widget.go", "package widget\n\nfunc Size() int {\n\treturn 2\n}\n")
	c.Write("extra.go", "package widget\n\n// built at "+clone+" for run "+reviewRun+"\nconst Where = \"here\"\n")
	c.Remove("README.md")
	c.Write(".abcd/work/decisions/dcn-1-built-by-gpt-5.md", "gpt-5 built this in cell gpt5-builder\n")
	c.Git("-c", "user.name=Gpt Five Builder", "-c", "user.email=gpt5-builder@example.invalid",
		"commit", "-q", "-am", "feat: widget\n\nAssisted-by: OpenAI:gpt-5")
	c.Git("add", "-A")
	c.Git("-c", "user.name=Gpt Five Builder", "-c", "user.email=gpt5-builder@example.invalid",
		"commit", "-q", "-m", "feat: extra\n\nAssisted-by: OpenAI:gpt-5")
	return src, frozen, store, clone
}

func assembleFor(t *testing.T, src *gittest.Repo, frozen, store, clone string) (ReviewRequest, ReviewManifest) {
	t.Helper()
	req, man, err := assembleReview(context.Background(), reviewInputs{
		RepoRoot: src.Root(), Frozen: frozen, Dir: clone, Label: "copy-0a1b2c3d",
		Run: reviewRun, Store: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	return req, man
}

// TestAReviewerRequestCarriesNoTrailerAuthorBranchOrRunMetadata: the step's
// first test. The request is the copy's diff from the frozen tree and nothing
// of how it was made: no trailer, author, branch, run id, cell, clone path or
// record of the run reaches it.
func TestAReviewerRequestCarriesNoTrailerAuthorBranchOrRunMetadata(t *testing.T) {
	src, frozen, store, clone := builtCopy(t)
	req, man := assembleFor(t, src, frozen, store, clone)
	data, err := encodeReviewRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, leak := range []string{
		"Assisted-by", "OpenAI", "gpt-5", "Gpt Five", "gpt5-builder@", "build/", "lane-1",
		reviewRun, "gpt5-builder", clone, store, filepath.Dir(store), ".abcd", "CONTEXT.md", "dcn-1",
	} {
		if strings.Contains(text, leak) {
			t.Errorf("the reviewer request carries %q:\n%s", leak, text)
		}
	}
	if req.Label != "copy-0a1b2c3d" {
		t.Fatalf("the request is under the opaque label it was given: %q", req.Label)
	}
	// The request names what it carries in the include table's own classes.
	var got []string
	for _, it := range req.Items {
		got = append(got, it.Key+" "+string(it.Kind)+" "+firstLine(it.Text))
	}
	want := []string{
		"item-001 " + string(reading.KindDoc) + " --- a/README.md",
		"item-002 " + string(reading.KindSource) + " --- /dev/null",
		"item-003 " + string(reading.KindSource) + " --- a/widget.go",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("items:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !strings.Contains(req.Items[2].Text, "-\treturn 1\n+\treturn 2\n") {
		t.Fatalf("the source item is the change's diff:\n%s", req.Items[2].Text)
	}
	if !strings.Contains(req.Items[1].Text, "// built at "+redactedMark+" for run "+redactedMark+"\n") {
		t.Fatalf("where the copy ran is withheld from what it wrote:\n%s", req.Items[1].Text)
	}
	if man.Redactions != 2 {
		t.Fatalf("the manifest counts the two withheld mentions of the run: %d", man.Redactions)
	}
	// The ask names no model, so the reviewer guesses the author unprompted.
	if strings.Contains(req.Ask, "gpt") || req.Ask != reviewAsk {
		t.Fatalf("the ask is the fixed one: %q", req.Ask)
	}
}

// TestTheReviewManifestHashesWhatWasPassedAndNamesWhatWasNot: the manifest is
// the run's own account of the request, never handed to the reviewer: the
// include table's version, each item's path, class and hash, the request's
// hash, the ask's hash, and every changed path withheld with its reason.
func TestTheReviewManifestHashesWhatWasPassedAndNamesWhatWasNot(t *testing.T) {
	src, frozen, store, clone := builtCopy(t)
	req, man := assembleFor(t, src, frozen, store, clone)
	if man.Assembler != reading.AssemblerVersion() {
		t.Fatalf("the manifest pins the include table it was assembled through: %q", man.Assembler)
	}
	data, _ := encodeReviewRequest(req)
	if sum := sha256.Sum256(data); man.RequestSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("the manifest's request hash is the hash of the request as encoded")
	}
	if sum := sha256.Sum256([]byte(reviewAsk)); man.AskSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("the manifest pins the ask by hash")
	}
	if len(man.Items) != len(req.Items) {
		t.Fatalf("one manifest item per request item: %d, %d", len(man.Items), len(req.Items))
	}
	paths := map[string]string{"item-001": "README.md", "item-002": "extra.go", "item-003": "widget.go"}
	changes := map[string]string{"item-001": changeDeleted, "item-002": changeAdded, "item-003": changeModified}
	for i, it := range man.Items {
		sum := sha256.Sum256([]byte(req.Items[i].Text))
		if it.Key != req.Items[i].Key || it.Kind != req.Items[i].Kind || it.SHA256 != hex.EncodeToString(sum[:]) ||
			it.Bytes != len(req.Items[i].Text) || it.Path != paths[it.Key] || it.Change != changes[it.Key] {
			t.Errorf("manifest item %d = %+v", i, it)
		}
	}
	withheld := map[string]string{}
	for _, w := range man.Withheld {
		withheld[w.Path] = w.Reason
	}
	if withheld[".abcd/work/decisions/dcn-1-built-by-gpt-5.md"] != withheldByTable {
		t.Fatalf("a record the copy wrote is withheld by the include table, and named: %v", man.Withheld)
	}
	if len(man.Withheld) != 1 {
		t.Fatalf("only the record is withheld: %v", man.Withheld)
	}
	// The manifest is no part of the request.
	text := string(data)
	if strings.Contains(text, "dcn-1") || strings.Contains(text, man.RequestSHA256) {
		t.Fatal("the request carries the manifest's account of it")
	}
}

// TestACopyThatChangedNothingTheTableAdmitsIsNotReviewed: a reviewer is never
// asked to judge an empty request.
func TestACopyThatChangedNothingTheTableAdmitsIsNotReviewed(t *testing.T) {
	src := gittest.NewRepo(t)
	src.Write("widget.go", "package widget\n")
	src.Commit("frozen")
	frozen := src.Git("rev-parse", "HEAD")
	store := filepath.Join(t.TempDir(), "matrix-store")
	clone, err := makeClone(src.Root(), store, reviewRun, frozen, TrialKey{Task: "widget", Cell: "a", Repeat: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(clone, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clone, ".abcd", "x.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, man, err := assembleReview(context.Background(), reviewInputs{RepoRoot: src.Root(), Frozen: frozen, Dir: clone,
		Label: "copy-00000000", Run: reviewRun, Store: store})
	if err == nil || !strings.Contains(err.Error(), "nothing the include table admits") {
		t.Fatalf("a copy with nothing to review is refused: %v (%+v)", err, man)
	}
	if strings.Contains(err.Error(), clone) || strings.Contains(err.Error(), src.Root()) {
		t.Fatalf("the refusal names no machine path: %v", err)
	}
}

// TestAReviewRefusalNamesNoMachinePath: inputs the request cannot be built
// from are refused in words that name no place on this machine.
func TestAReviewRefusalNamesNoMachinePath(t *testing.T) {
	src, frozen, store, clone := builtCopy(t)
	missing := strings.Repeat("0", 40)
	for _, in := range []reviewInputs{
		{RepoRoot: src.Root(), Frozen: missing, Dir: clone, Label: "copy-0a1b2c3d", Run: reviewRun, Store: store},
		{RepoRoot: src.Root(), Frozen: frozen, Dir: filepath.Join(clone, "absent"), Label: "copy-0a1b2c3d", Run: reviewRun, Store: store},
		{RepoRoot: src.Root(), Frozen: frozen, Dir: clone, Label: "gpt-5", Run: reviewRun, Store: store},
		{RepoRoot: "relative", Frozen: frozen, Dir: clone, Label: "copy-0a1b2c3d", Run: reviewRun, Store: store},
	} {
		_, _, err := assembleReview(context.Background(), in)
		if err == nil {
			t.Fatalf("%+v was assembled", in)
		}
		for _, p := range []string{src.Root(), clone, store, os.TempDir()} {
			if strings.Contains(err.Error(), p) {
				t.Fatalf("the refusal names a machine path: %v", err)
			}
		}
	}
}

// TestAHugeFlatDirectoryIsRefusedAtTheBoundMidListing: the copy controls its
// tree, so a directory of millions of entries is no reason to hold millions
// in memory. A directory is listed in batches, counted as it is listed, and
// the walk stops at the entry bound inside the directory, never after
// listing it whole.
func TestAHugeFlatDirectoryIsRefusedAtTheBoundMidListing(t *testing.T) {
	dir := t.TempDir()
	flat := filepath.Join(dir, "flat")
	if err := os.Mkdir(flat, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range 60 {
		if err := os.WriteFile(filepath.Join(flat, fmt.Sprintf("f%03d.go", i)), []byte("package f\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const bound, batch = 20, 4
	_, listed, err := walkCopyTree(context.Background(), dir, nil, bound, batch)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("more than %d entries", bound)) {
		t.Fatalf("a tree past the entry bound is refused: %v", err)
	}
	if listed > bound+batch {
		t.Fatalf("%d entries of a 60-entry directory were listed before the walk refused at %d", listed, bound)
	}
	if strings.Contains(err.Error(), dir) {
		t.Fatalf("the refusal names no machine path: %v", err)
	}
	// Within the bound, the same tree is read whole.
	tree, _, err := walkCopyTree(context.Background(), dir, nil, 100, batch)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 60 || tree["flat/f059.go"].data == nil {
		t.Fatalf("the tree within its bound is read whole: %d entries", len(tree))
	}
}

// TestEveryChangedPathTravelsOrIsWithheldWithAReason: no changed path leaves
// both the request and the manifest's withheld list. An empty file added or
// deleted travels as its header, and a change of line endings alone, which a
// review reads as no change, is withheld and says so.
func TestEveryChangedPathTravelsOrIsWithheldWithAReason(t *testing.T) {
	src := gittest.NewRepo(t)
	src.Write("widget.go", "package widget\n")
	src.Write("gone.go", "")
	src.Write("crlf.go", "package widget\n\nconst A = 1\n")
	src.Commit("frozen")
	frozen := src.Git("rev-parse", "HEAD")
	store := filepath.Join(t.TempDir(), "matrix-store")
	clone, err := makeClone(src.Root(), store, reviewRun, frozen, TrialKey{Task: "widget", Cell: "a", Repeat: 1})
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"widget.go": "package widget\n\nconst B = 2\n",
		"empty.go":  "",
		"crlf.go":   "package widget\r\n\r\nconst A = 1\r\n",
	} {
		if err := os.WriteFile(filepath.Join(clone, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(clone, "gone.go")); err != nil {
		t.Fatal(err)
	}
	req, man := assembleFor(t, src, frozen, store, clone)
	travelled := map[string]string{}
	for i, it := range man.Items {
		travelled[it.Path] = it.Change
		if it.Path == "empty.go" && req.Items[i].Text != "--- /dev/null\n+++ b/empty.go\n" {
			t.Errorf("an empty file added travels as its header: %q", req.Items[i].Text)
		}
		if it.Path == "gone.go" && req.Items[i].Text != "--- a/gone.go\n+++ /dev/null\n" {
			t.Errorf("an empty file deleted travels as its header: %q", req.Items[i].Text)
		}
	}
	withheld := map[string]string{}
	for _, w := range man.Withheld {
		withheld[w.Path] = w.Reason
	}
	if travelled["empty.go"] != changeAdded || travelled["gone.go"] != changeDeleted {
		t.Errorf("the empty files travel as added and deleted: %v", travelled)
	}
	if withheld["crlf.go"] != withheldLineEndings {
		t.Errorf("a change of line endings alone is withheld and says so: %v", man.Withheld)
	}
	for _, p := range []string{"widget.go", "empty.go", "gone.go", "crlf.go"} {
		_, t1 := travelled[p]
		_, t2 := withheld[p]
		if t1 == t2 {
			t.Errorf("%s is in the request %v and withheld %v: a changed path is in exactly one", p, t1, t2)
		}
	}
}

// TestAReviewGitCutOffByTheCallersDeadlineSaysSo: git cut off by the
// caller's deadline is not git failing to answer within its own.
func TestAReviewGitCutOffByTheCallersDeadlineSaysSo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := reviewGit(ctx, t.TempDir(), 1<<10, "version")
	if err == nil || strings.Contains(err.Error(), reviewGitTimeout.String()) || !strings.Contains(err.Error(), "review's deadline") {
		t.Fatalf("git cut off by the caller's deadline: %v", err)
	}
}

func TestReviewLabelsAreOpaqueAndDistinct(t *testing.T) {
	re := regexp.MustCompile(`^copy-[0-9a-f]{8}$`)
	seen := map[string]bool{}
	labels, err := mintLabels(64)
	if err != nil || len(labels) != 64 {
		t.Fatalf("64 labels: %d %v", len(labels), err)
	}
	for _, l := range labels {
		if !re.MatchString(l) || seen[l] {
			t.Fatalf("label %q is not an opaque, distinct one", l)
		}
		seen[l] = true
	}
}

// TestTheReviewRequestDecodesToItsOwnShape: the request's encoding is the
// fields it declares and no others, so nothing rides along unseen.
func TestTheReviewRequestDecodesToItsOwnShape(t *testing.T) {
	src, frozen, store, clone := builtCopy(t)
	req, _ := assembleFor(t, src, frozen, store, clone)
	data, _ := encodeReviewRequest(req)
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range top {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if strings.Join(keys, ",") != "_type,ask,items,label,schema_version" {
		t.Fatalf("the request's fields: %v", keys)
	}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
