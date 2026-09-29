//go:build smoke

package evals

// The write-path smoke: the verbs that write a committed record run for real,
// against a scratch repository, through the built binary, and the test asserts
// what lands on disk (product thinker's ruling M11 of 2026-09-23, on
// iss-2608231120121681). A unit test that constructs a request by hand
// exercises a caller that does not exist in production; the CLI derives fields
// the hand-built request supplies (a capture's slug comes from its text), and
// on 2026-08-23 a change to the ledger's write path passed every gate while the
// verb refused every issue whose text carried a home path. Only the built
// binary answers whether the verb works, so this lane runs it.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// kebab is the slug grammar a record's filename and frontmatter must satisfy.
var kebab = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// scratchRepo makes an empty git repository and a fixture home, both under the
// test's own temporary directory, and returns them. The home's last segment is
// distinctive on purpose: the redactor reads it as the account name, and a
// common word there would be rewritten wherever it appears in the text.
func scratchRepo(t *testing.T) (repo, home string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo")
	home = filepath.Join(root, "hq7smokehome")
	for _, d := range []string{repo, home} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	initCmd := exec.Command("git", "init", "-q", ".")
	initCmd.Dir = repo
	initCmd.Env = append(os.Environ(), "HOME="+home, "GIT_CONFIG_NOSYSTEM=1")
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return repo, home
}

// runJSON runs a verb in repo under home with --json and decodes stdout-and-
// stderr's JSON object. The verbs print notices on stderr, so the object is
// cut from the first `{` of the combined output.
func runJSON(t *testing.T, repo, home string, into any, args ...string) {
	t.Helper()
	out, code := runIn(t, repo, []string{"HOME=" + home}, append(args, "--json")...)
	label := "abcd " + strings.Join(args, " ")
	if panicked(out) {
		t.Fatalf("`%s` panicked:\n%s", label, out)
	}
	if code != 0 {
		t.Fatalf("`%s` exit=%d, want 0\n%s", label, code, out)
	}
	i := strings.Index(out, "{")
	if i < 0 {
		t.Fatalf("`%s` printed no JSON object:\n%s", label, out)
	}
	if err := json.NewDecoder(strings.NewReader(out[i:])).Decode(into); err != nil {
		t.Fatalf("`%s` JSON: %v\n%s", label, err, out)
	}
}

// readRecord returns a record's bytes by its repo-relative path, failing the
// test when the verb reported a path that holds nothing.
func readRecord(t *testing.T, repo, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("the verb reported %s, and it is not on disk: %v", rel, err)
	}
	return string(b)
}

// TestCaptureWritesARecordTheReaderReadsBack runs the capture that broke on
// 2026-08-23: free text carrying a home path, with the slug derived from the
// text the way production derives it. The record must land under open/, its
// slug must be kebab-case in the filename and the frontmatter alike, the home
// path must not reach the disk, and the ledger's own reader must list it (a
// record the reader drops is invisible to every surface).
func TestCaptureWritesARecordTheReaderReadsBack(t *testing.T) {
	repo, home := scratchRepo(t)
	homePath := filepath.Join(home, ".local", "bin", "abcd")
	text := "the path entry is " + homePath + " and it moved after the update"

	var got struct {
		ID, Slug, Path, Status string
	}
	runJSON(t, repo, home, &got, "capture", text, "--severity", "minor", "--category", "bug")

	if got.Status != "open" || !strings.HasPrefix(got.ID, "iss-") {
		t.Fatalf("capture reported id=%q status=%q, want an iss- id in open", got.ID, got.Status)
	}
	if !kebab.MatchString(got.Slug) {
		t.Errorf("capture's slug %q is not kebab-case", got.Slug)
	}
	wantPath := ".abcd/work/issues/open/" + got.ID + "-" + got.Slug + ".md"
	if got.Path != wantPath {
		t.Errorf("capture reported path %q, want %q", got.Path, wantPath)
	}
	rec := readRecord(t, repo, got.Path)
	for _, want := range []string{`id: "` + got.ID + `"`, `slug: "` + got.Slug + `"`, `severity: "minor"`, `category: "bug"`} {
		if !strings.Contains(rec, want) {
			t.Errorf("the record on disk lacks %s:\n%s", want, rec)
		}
	}
	// The account segment is the part of a home path that identifies someone,
	// and it can leak without the path shape around it: a slug kebab-cased from
	// the raw text carries it into the filename, where no redactor sees a path.
	account := filepath.Base(home)
	if strings.Contains(got.Path, account) || strings.Contains(rec, account) {
		t.Errorf("the home path's account segment %q reached the committed record %s:\n%s", account, got.Path, rec)
	}

	var list struct {
		Issues []struct{ ID, Status string }
	}
	runJSON(t, repo, home, &list, "capture", "list", "--open")
	if len(list.Issues) != 1 || list.Issues[0].ID != got.ID {
		t.Errorf("capture list --open read back %+v, want exactly %s", list.Issues, got.ID)
	}
}

// TestResolveMovesTheRecordAndWritesItsResolution runs the ledger's other
// write: resolve moves the record from open/ to resolved/ carrying the note and
// the impact, and leaves nothing behind in open/.
func TestResolveMovesTheRecordAndWritesItsResolution(t *testing.T) {
	repo, home := scratchRepo(t)
	var captured struct{ ID, Path string }
	runJSON(t, repo, home, &captured, "capture", "the scratch widget refuses every input it is handed", "--severity", "minor")

	var resolved struct {
		ID, Path, Status string
	}
	runJSON(t, repo, home, &resolved, "capture", "resolve", captured.ID, "the widget accepts its input again",
		"--impact", "fix", "--grounds", "pursued: the widget accepts input; a refusal on the same input would show it wrong")

	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(captured.Path))); !os.IsNotExist(err) {
		t.Errorf("the open record %s is still on disk after resolve (stat err=%v)", captured.Path, err)
	}
	want := strings.Replace(captured.Path, "/open/", "/resolved/", 1)
	if resolved.Path != want {
		t.Errorf("resolve reported path %q, want %q", resolved.Path, want)
	}
	rec := readRecord(t, repo, want)
	for _, w := range []string{`id: "` + captured.ID + `"`, "the widget accepts its input again", "impact: fix"} {
		if !strings.Contains(rec, w) {
			t.Errorf("the resolved record lacks %q:\n%s", w, rec)
		}
	}
}

// TestDecideMintsTheDecisionRecord runs decide, the committed-tier writer of a
// decision record: the file it names exists, under the ADR store, and carries
// the title it was given.
func TestDecideMintsTheDecisionRecord(t *testing.T) {
	repo, home := scratchRepo(t)
	const title = "Scratch decisions live in one folder"
	var got struct{ ID, Slug, Title, Path string }
	runJSON(t, repo, home, &got, "decide", title)

	if !strings.HasPrefix(got.ID, "adr-") || !kebab.MatchString(got.Slug) {
		t.Fatalf("decide reported id=%q slug=%q", got.ID, got.Slug)
	}
	if !strings.HasPrefix(got.Path, ".abcd/development/decisions/adrs/") {
		t.Errorf("decide wrote %q, outside the ADR store", got.Path)
	}
	if rec := readRecord(t, repo, got.Path); !strings.Contains(rec, title) {
		t.Errorf("the decision record lacks its title %q:\n%s", title, rec)
	}
}
