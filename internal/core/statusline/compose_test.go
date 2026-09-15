package statusline

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/gittest"
)

// composeRepo stands up a git checkout with the local-ephemeral tier, and
// returns the root as git names it (the temp area is reached through a
// symlink on macOS, and Compose's repository name is the base of the root git
// resolves).
func composeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "--initial-branch=main")
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	root := strings.TrimSpace(string(out))
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(mode.TierRelPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(gittest.Env(t),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// seedRecord lays a record with one open issue and two intents not yet
// shipped (a draft and a planned one) plus one shipped, in the shapes the
// ledger and intent readers accept.
func seedRecord(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		".abcd/work/issues/open/iss-1-a-seeded-issue.md":       "---\nschema_version: 1\nid: \"iss-1\"\nslug: \"a-seeded-issue\"\nseverity: \"minor\"\ncategory: \"observation\"\nsource: \"user-observation\"\nfound_during: \"manual-capture\"\n---\n\nA seeded issue.\n",
		".abcd/work/issues/resolved/iss-2-a-resolved-issue.md": "---\nschema_version: 1\nid: \"iss-2\"\nslug: \"a-resolved-issue\"\nseverity: \"minor\"\ncategory: \"observation\"\nsource: \"user-observation\"\nfound_during: \"manual-capture\"\n---\n\nA resolved issue.\n",
		".abcd/development/intents/drafts/itd-1-a-draft.md":    "---\nid: itd-1\nslug: a-draft\nspec_id: null\nkind: standalone\nsuggested_kind: null\nreclassification_history: []\nbuilds_on: []\nseverity: minor\n---\n\n# A draft\n\n## Press Release\n\n> A draft.\n",
		".abcd/development/intents/planned/itd-2-a-plan.md":    "---\nid: itd-2\nslug: a-plan\nspec_id: null\nkind: standalone\nsuggested_kind: null\nreclassification_history: []\nbuilds_on: []\nseverity: minor\n---\n\n# A plan\n\n## Press Release\n\n> A plan.\n",
		".abcd/development/intents/shipped/itd-3-a-ship.md":    "---\nid: itd-3\nslug: a-ship\nspec_id: null\nkind: standalone\nsuggested_kind: null\nreclassification_history: []\nbuilds_on: []\nseverity: minor\n---\n\n# A ship\n\n## Press Release\n\n> Shipped.\n",
	}
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func plainOf(t *testing.T, row Row, k ElementKey) string {
	t.Helper()
	el, ok := row.Element(k)
	if !ok {
		t.Fatalf("row has no %s element: %q", k, row.Plain())
	}
	return el.Plain
}

// TestComposeReadsTheThreeInputs is the composition contract: the badge from
// the mode store, the repository name and branch from the checkout, the counts
// from the record's folders (open issues; drafts plus planned intents), and
// the payload elements from the parsed payload.
func TestComposeReadsTheThreeInputs(t *testing.T) {
	root := composeRepo(t)
	seedRecord(t, root)
	if err := mode.SetAt(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	p, err := ParsePayload([]byte(fullPayloadJSON))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Compose(root, p, Defaults())
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if res.State != StateProductThinker {
		t.Errorf("State = %q, want %q", res.State, StateProductThinker)
	}
	if res.Row.Elements[0].Key != KeyPresence || res.Row.Elements[0].Plain != "waiting: product thinker" {
		t.Errorf("the badge is not element one: %+v", res.Row.Elements[0])
	}
	if got := plainOf(t, res.Row, KeyRepo); got != filepath.Base(root) {
		t.Errorf("repo = %q, want %q", got, filepath.Base(root))
	}
	if got := plainOf(t, res.Row, KeyBranch); got != "main" {
		t.Errorf("branch = %q, want main", got)
	}
	if got := plainOf(t, res.Row, KeyModel); got != "Opus" {
		t.Errorf("model = %q", got)
	}
	if got := plainOf(t, res.Row, KeyIssues); got != "iss 1" {
		t.Errorf("issues = %q, want \"iss 1\" (open only)", got)
	}
	if got := plainOf(t, res.Row, KeyIntents); got != "itd 2" {
		t.Errorf("intents = %q, want \"itd 2\" (drafts plus planned, never shipped)", got)
	}
	if len(res.Notes) != 0 {
		t.Errorf("unexpected notes: %v", res.Notes)
	}
}

// TestComposeWithoutARecordDropsTheCounts: a repository with no
// `.abcd/development` has nothing to count, and a zero it does not mean is
// not rendered (the render's own rule for Counts).
func TestComposeWithoutARecordDropsTheCounts(t *testing.T) {
	root := composeRepo(t)
	set := Defaults()
	res, err := Compose(root, Payload{}, set)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	for _, k := range []ElementKey{KeyIssues, KeyIntents} {
		if _, ok := res.Row.Element(k); ok {
			t.Errorf("%s rendered with no record to count: %q", k, res.Row.Plain())
		}
	}
	if res.State != StateManaged {
		t.Errorf("an absent store must read as managed, got %q", res.State)
	}
	// The caller's settings are not mutated to achieve it.
	if !set.Enabled(KeyIssues) || !set.Enabled(KeyIntents) {
		t.Error("Compose switched the caller's own settings off")
	}
	want := "abcd · " + filepath.Base(root) + " · main"
	if res.Row.Plain() != want {
		t.Errorf("Plain = %q, want %q", res.Row.Plain(), want)
	}
}

// TestComposeRendersTheShortShaOnADetachedHead: a detached HEAD has no branch
// name, and the short sha is the honest thing to show in its place.
func TestComposeRendersTheShortShaOnADetachedHead(t *testing.T) {
	root := composeRepo(t)
	if err := os.WriteFile(filepath.Join(root, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "f")
	git(t, root, "commit", "-q", "-m", "one")
	git(t, root, "checkout", "-q", "--detach")
	out, err := exec.Command("git", "-C", root, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(out))
	res, err := Compose(root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if got := plainOf(t, res.Row, KeyBranch); got != want {
		t.Errorf("branch on a detached HEAD = %q, want the short sha %q", got, want)
	}
}

// TestComposeRefusesAMalformedStore: a store somebody wrote a fourth word into
// is an error, not a quiet badge — reporting "nobody is waiting" over it
// would hide exactly the parked stop the badge exists to show.
func TestComposeRefusesAMalformedStore(t *testing.T) {
	root := composeRepo(t)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(mode.FileRelPath)), []byte("everyone\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Compose(root, Payload{}, Defaults()); err == nil {
		t.Fatal("a malformed mode store composed a row")
	}
}

// TestComposeNotesAnUnreadableRecordAndKeepsTheRow: a bucket the count cannot
// read — something that is not a real directory standing where it should be —
// drops the count it could not take, says so in a note, and leaves every other
// element standing. A symlinked bucket is refused rather than followed: the
// count is a guarded readdir, and a directory reached through a link a
// hostile checkout committed could count anything.
func TestComposeNotesAnUnreadableRecordAndKeepsTheRow(t *testing.T) {
	cases := []struct {
		name  string
		stand func(t *testing.T, drafts string)
	}{
		{"a file where the bucket should be", func(t *testing.T, drafts string) {
			if err := os.WriteFile(drafts, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"a symlinked bucket", func(t *testing.T, drafts string) {
			if err := os.Symlink(t.TempDir(), drafts); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := composeRepo(t)
			seedRecord(t, root)
			drafts := filepath.Join(root, ".abcd", "development", "intents", "drafts")
			if err := os.RemoveAll(drafts); err != nil {
				t.Fatal(err)
			}
			tc.stand(t, drafts)
			res, err := Compose(root, Payload{}, Defaults())
			if err != nil {
				t.Fatalf("Compose: %v", err)
			}
			if _, ok := res.Row.Element(KeyIntents); ok {
				t.Errorf("an unreadable intent store still rendered a count: %q", res.Row.Plain())
			}
			if got := plainOf(t, res.Row, KeyIssues); got != "iss 1" {
				t.Errorf("issues = %q; the ledger was readable", got)
			}
			if len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "intent") {
				t.Errorf("notes = %v, want one naming the intent store", res.Notes)
			}
			for _, n := range res.Notes {
				if strings.Contains(n, root) {
					t.Errorf("a note carries the absolute checkout path: %q", n)
				}
			}
		})
	}
}

// TestComposeCountsAreFolderCounts: the two counts are readdir counts, not
// parses. Folder membership is the record's own status signal (a record is
// open because it is in open/), so N files named iss-*.md in open/ count N
// whether or not any of them parses as a record, and the intents count is the
// drafts bucket plus the planned one. Anything that is not a regular file
// with the family's name — another name, a directory, a symlink — is not a
// record and is not counted. A status refresh runs this on every keystroke,
// and a count that loaded and parsed every record was measured at a second
// over a large ledger.
func TestComposeCountsAreFolderCounts(t *testing.T) {
	root := composeRepo(t)
	lay := func(rel, body string) {
		t.Helper()
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"1-a", "2-b", "3-c", "4-d", "5-e"} {
		lay(".abcd/work/issues/open/iss-"+n+".md", "not a record at all\n")
	}
	lay(".abcd/work/issues/open/README.md", "not an issue\n")
	lay(".abcd/work/issues/open/iss-6-f.txt", "wrong extension\n")
	lay(".abcd/work/issues/open/iss-7-g.md/inner", "a directory carrying the name\n")
	lay(".abcd/work/issues/resolved/iss-8-h.md", "resolved, not open\n")
	if err := os.Symlink(filepath.Join(root, ".abcd", "work", "issues", "open", "iss-1-a.md"),
		filepath.Join(root, ".abcd", "work", "issues", "open", "iss-9-i.md")); err != nil {
		t.Fatal(err)
	}
	lay(".abcd/development/intents/drafts/itd-1-a.md", "garbage\n")
	lay(".abcd/development/intents/drafts/itd-2-b.md", "garbage\n")
	lay(".abcd/development/intents/drafts/notes.md", "not an intent\n")
	lay(".abcd/development/intents/planned/itd-3-c.md", "garbage\n")
	lay(".abcd/development/intents/shipped/itd-4-d.md", "shipped, not counted\n")

	res, err := Compose(root, Payload{}, Defaults())
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if got := plainOf(t, res.Row, KeyIssues); got != "iss 5" {
		t.Errorf("issues = %q, want \"iss 5\" (five iss-*.md regular files in open/)", got)
	}
	if got := plainOf(t, res.Row, KeyIntents); got != "itd 3" {
		t.Errorf("intents = %q, want \"itd 3\" (two drafts plus one planned)", got)
	}
	if len(res.Notes) != 0 {
		t.Errorf("unexpected notes: %v", res.Notes)
	}
}
