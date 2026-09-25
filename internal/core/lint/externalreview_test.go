package lint_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The external-review gate (iss-281) is a required check whose logic lives in
// one run: script with no checkout, so these tests execute that script, read
// out of the committed workflow, against a fake gh. The fake reproduces the two
// gh behaviours the defects turned on: `--paginate --jq` applies the jq filter
// to each page separately (and refuses `--slurp` beside `--jq`), and a failed
// API call exits non-zero with its error on stderr.

const fakeGH = `set -euo pipefail
[ "$1" = api ] || { echo "fake gh: only api is faked" >&2; exit 99; }
shift
path=""; expr=""; paginate=0; slurp=0
while [ $# -gt 0 ]; do
  case "$1" in
    --jq) expr="$2"; shift 2 ;;
    --paginate) paginate=1; shift ;;
    --slurp) slurp=1; shift ;;
    -*) shift ;;
    *) path="$1"; shift ;;
  esac
done
if [ "$slurp" = 1 ] && [ -n "$expr" ]; then
  echo "the '--slurp' option is not supported with '--jq' or '--template'" >&2; exit 1
fi
case "$path" in
  */collaborators/*/permission)
    u="${path#*/collaborators/}"; u="${u%/permission}"
    if [ -f "$FAKE_DIR/fail" ] && grep -qx "$u" "$FAKE_DIR/fail"; then
      echo "gh: Resource not accessible by integration (HTTP 403)" >&2; exit 1
    fi
    role="$(awk -v u="$u" '$1 == u { print $2 }' "$FAKE_DIR/roles")"
    printf '{"role_name":"%s"}' "${role:-none}" | jq -r "${expr:-.}" ;;
  */reviews*)
    [ "$paginate" = 1 ] || { echo "fake gh: reviews fetched without --paginate" >&2; exit 98; }
    if [ "$slurp" = 1 ]; then jq -s . "$FAKE_DIR"/page*.json
    else for p in "$FAKE_DIR"/page*.json; do jq -r "${expr:-.}" "$p"; done
    fi ;;
  *) echo "fake gh: unexpected path $path" >&2; exit 97 ;;
esac
`

type review struct{ user, state, at string }

// runExternalReview runs the committed gate script for an external author's
// pull request with the given review pages and roles, returning its output
// and exit code.
func runExternalReview(t *testing.T, pages [][]review, roles map[string]string, fail []string) (string, int) {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("jq is required to fake gh's --jq and is missing on a CI runner")
		}
		t.Skip("jq not on PATH; the fake gh needs it to apply --jq per page")
	}
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/external-review.yml"
	script := stepScript(t, findStep(t, readRepoFile(t, root, rel), rel, "external-review:"))

	dir := t.TempDir()
	for i, page := range pages {
		var items []string
		for _, r := range page {
			items = append(items, `{"user":{"login":"`+r.user+`"},"state":"`+r.state+`","submitted_at":"`+r.at+`"}`)
		}
		name := filepath.Join(dir, "page"+string(rune('1'+i))+".json")
		if err := os.WriteFile(name, []byte("["+strings.Join(items, ",")+"]"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var lines []string
	for u, r := range roles {
		lines = append(lines, u+" "+r)
	}
	if err := os.WriteFile(filepath.Join(dir, "roles"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(fail) > 0 {
		if err := os.WriteFile(filepath.Join(dir, "fail"), []byte(strings.Join(fail, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bin := t.TempDir()
	fakeTool(t, bin, "gh", fakeGH)
	return runScript(t, script, dir, bin, map[string]string{
		"FAKE_DIR": dir, "GH_TOKEN": "fake", "EVENT_NAME": "pull_request_target",
		"REPO": "example/repo", "PR_NUMBER": "7", "PR_AUTHOR": "outsider", "PR_AUTHOR_TYPE": "User",
	})
}

var collaborators = map[string]string{"alice": "write", "bob": "maintain", "carol": "triage", "outsider": "read"}

// TestExternalReviewTallySpansPages is iss-301. `gh api --paginate --jq` runs
// the filter per page, so the "latest state per reviewer" reduction only ever
// saw one page: a page-1 APPROVED survived the same reviewer's later page-2
// CHANGES_REQUESTED, and an approval repeated across pages counted twice. The
// reduction must span every page.
func TestExternalReviewTallySpansPages(t *testing.T) {
	t.Run("a later change request on page two withdraws a page-one approval", func(t *testing.T) {
		out, code := runExternalReview(t, [][]review{
			{{"alice", "APPROVED", "2026-01-01T00:00:00Z"}, {"bob", "APPROVED", "2026-01-01T01:00:00Z"}},
			{{"alice", "CHANGES_REQUESTED", "2026-01-02T00:00:00Z"}},
		}, collaborators, nil)
		if code == 0 {
			t.Errorf("the gate passed with one standing approval (alice withdrew hers on page two):\n%s", out)
		}
		if strings.Contains(out, "counted: alice") {
			t.Errorf("alice's superseded approval was counted:\n%s", out)
		}
	})
	t.Run("an approval repeated across pages counts once", func(t *testing.T) {
		out, code := runExternalReview(t, [][]review{
			{{"alice", "APPROVED", "2026-01-01T00:00:00Z"}},
			{{"alice", "APPROVED", "2026-01-03T00:00:00Z"}, {"carol", "COMMENTED", "2026-01-03T01:00:00Z"}},
		}, collaborators, nil)
		if code == 0 {
			t.Errorf("the gate passed on one reviewer's approval counted twice:\n%s", out)
		}
		if n := strings.Count(out, "counted: alice"); n != 1 {
			t.Errorf("alice counted %d times, want once:\n%s", n, out)
		}
	})
	t.Run("two standing approvals across pages pass", func(t *testing.T) {
		out, code := runExternalReview(t, [][]review{
			{{"alice", "CHANGES_REQUESTED", "2026-01-01T00:00:00Z"}, {"bob", "APPROVED", "2026-01-01T01:00:00Z"}},
			{{"alice", "APPROVED", "2026-01-02T00:00:00Z"}},
		}, collaborators, nil)
		if code != 0 {
			t.Errorf("the gate refused two standing invited-collaborator approvals:\n%s", out)
		}
	})
}
