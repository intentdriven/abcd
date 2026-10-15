package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/gittest"
)

// embargo_surface_test.go — step 10 of spc-2610120557432880: the front door of
// `abcd embargo`. The core's behaviour is held in internal/core/embargo; these
// tests hold what the surface adds: each sub-verb's refusals exit 2 with the
// remedy and write nothing, and each --json document has its shape. Advisory
// ids are of the GHSA-xxxx-xxxx-xxxx placeholder form
// (examples-use-reserved-identifiers).

const surfaceGHSA = "GHSA-xxxx-xxxx-xxxx"

// embargoSurfaceRepo is a checkout whose origin names an example GitHub
// repository, under a temp HOME, with a stand-in `gh` on PATH: the triage
// listing answers surfaceGHSA once the test has created the file named by
// ghTriage, every other listing an empty page, and every read of one
// advisory answers surfaceGHSA in triage. The working directory is the
// checkout.
func embargoSurfaceRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/.gitkeep", "")
	r.Write("README.md", "a project\n")
	r.Commit("initial")
	r.Git("remote", "add", "origin", "https://github.com/example/example.git")
	bin := t.TempDir()
	advisory := `{"ghsa_id":"` + surfaceGHSA + `","cve_id":null,"summary":"a sample report","description":"the report",` +
		`"severity":"high","state":"triage","vulnerabilities":[]}`
	script := "#!/bin/sh\nfor a in \"$@\"; do case \"$a\" in\n" +
		"  state=triage) if [ -f '" + filepath.Join(bin, "triage-on") + "' ]; then echo '[" + advisory + "]'; else echo '[]'; fi; exit 0 ;;\n" +
		"  state=*) echo '[]'; exit 0 ;;\nesac; done\necho '" + advisory + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(ghTriageEnv, filepath.Join(bin, "triage-on"))
	t.Chdir(r.Root())
	return r
}

const ghTriageEnv = "ABCD_TEST_EMBARGO_TRIAGE_ON"

// seedEmbargo runs intake with surfaceGHSA waiting in triage, so the store
// holds its private record.
func seedEmbargo(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(os.Getenv(ghTriageEnv), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, stderr := runEmbargo(t, "", "embargo", "intake"); code != 0 || !strings.Contains(out, "1 new private record") {
		t.Fatalf("intake: exit %d, %s%s", code, out, stderr)
	}
}

func runEmbargo(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func jsonKeys(t *testing.T, doc string) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(doc), &m); err != nil {
		t.Fatalf("not one JSON object: %v\n%s", err, doc)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestEmbargoSurfaceRefusals: each sub-verb refuses a request it cannot act
// on at exit 2, saying what to do, and writes nothing.
func TestEmbargoSurfaceRefusals(t *testing.T) {
	r := embargoSurfaceRepo(t)
	store, ok := storepath.Path(os.Getenv("HOME"), r.Root())
	if !ok {
		t.Fatal("the fixture repository has no root commit to key the embargo store on")
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"embargo", "triage", surfaceGHSA, "vulnerability", "--by", "Alice"}, "abcd embargo intake"},
		{[]string{"embargo", "close", surfaceGHSA}, "abcd embargo intake"},
		{[]string{"embargo", "publish"}, "abcd embargo intake"},
		{[]string{"embargo", "check"}, "nothing to check"},
		{[]string{"embargo", "check", "--range", "nonsense"}, "<a>..<b>"},
		{[]string{"embargo", "triage", surfaceGHSA}, "accepts 2 arg"},
		{[]string{"embargo", "unknown"}, "unknown"},
	} {
		code, _, stderr := runEmbargo(t, "", tc.args...)
		if code != 2 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: exit %d, stderr %q; want exit 2 saying %q", tc.args, code, stderr, tc.want)
		}
	}
	if _, err := os.Lstat(store); err == nil {
		t.Fatal("a refused sub-verb made the embargo store")
	}

	seedEmbargo(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"embargo", "triage", surfaceGHSA, "maybe", "--by", "Alice"}, "verdict"},
		{[]string{"embargo", "triage", surfaceGHSA, "not-a-vulnerability", "--by", "Alice"}, "--reason"},
		{[]string{"embargo", "triage", surfaceGHSA, "vulnerability"}, "--by"},
		{[]string{"embargo", "close", surfaceGHSA}, "verdict"},
		{[]string{"embargo", "publish", "--release", "v1"}, "vMAJOR.MINOR.PATCH"},
	} {
		code, _, stderr := runEmbargo(t, "", tc.args...)
		if code != 2 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: exit %d, stderr %q; want exit 2 saying %q", tc.args, code, stderr, tc.want)
		}
	}

	t.Chdir(t.TempDir())
	if code, _, stderr := runEmbargo(t, "", "embargo", "intake"); code != 2 || !strings.Contains(stderr, "nothing written") {
		t.Errorf("intake outside a checkout: exit %d, stderr %q", code, stderr)
	}
}

// TestEmbargoJSONShapes: every sub-verb's --json is one document on stdout of
// the core's shape, a check finding included (exit 2, findings in the
// document), and no document repeats a matched advisory id.
func TestEmbargoJSONShapes(t *testing.T) {
	r := embargoSurfaceRepo(t)

	code, out, stderr := runEmbargo(t, "", "embargo", "intake", "--json")
	if code != 0 {
		t.Fatalf("intake: exit %d, %s", code, stderr)
	}
	if got := strings.Join(jsonKeys(t, out), ","); got != "held,intake_at,minted,needs_severity,repo,store,store_created" {
		t.Errorf("intake keys %s", got)
	}
	if strings.Contains(out, os.Getenv("HOME")) {
		t.Errorf("intake's document names the home directory: %s", out)
	}

	code, out, _ = runEmbargo(t, "fix: tighten the parser\n", "embargo", "check", "--file", "-", "--json")
	if code != 0 {
		t.Fatalf("clean check: exit %d", code)
	}
	if got := strings.Join(jsonKeys(t, out), ","); got != "allowed,commits,files,findings,intake_at,intake_stale,no_store,under_embargo" {
		t.Errorf("check keys %s", got)
	}

	seedEmbargo(t)
	code, out, _ = runEmbargo(t, "fixes "+surfaceGHSA+"\n", "embargo", "check", "--file", "-", "--json")
	var res embargo.CheckResult
	if err := json.Unmarshal([]byte(out), &res); err != nil || code != 2 || len(res.Findings) != 1 ||
		res.Findings[0].Kind != embargo.KindAdvisoryID || strings.Contains(out, "xxxx") {
		t.Fatalf("check finding: exit %d, %v, %s", code, err, out)
	}
	code, out, _ = runEmbargo(t, "fixes "+surfaceGHSA+"\n", "embargo", "check", "--file", "-")
	if code != 2 || !strings.Contains(out, "REFUSED, 1 finding") || strings.Contains(out, "xxxx") {
		t.Fatalf("check text: exit %d, %s", code, out)
	}

	code, out, stderr = runEmbargo(t, "", "embargo", "triage", surfaceGHSA, "not-a-vulnerability", "--by", "Alice",
		"--reason", "the behaviour is documented", "--json")
	if code != 0 {
		t.Fatalf("triage: exit %d, %s%s", code, stderr, out)
	}
	for _, k := range []string{"decided_at", "decided_by", "draft", "draft_path", "ghsa_id", "issue_id", "record_status", "verdict"} {
		if !strings.Contains(strings.Join(jsonKeys(t, out), ","), k) {
			t.Errorf("triage document has no %s: %s", k, out)
		}
	}

	code, out, stderr = runEmbargo(t, "", "embargo", "close", surfaceGHSA, "--json")
	if code != 0 {
		t.Fatalf("close: exit %d, %s", code, stderr)
	}
	var cr embargo.CloseResult
	if err := json.Unmarshal([]byte(out), &cr); err != nil || cr.Closed || cr.Digest == "" ||
		!strings.Contains(cr.ApproveWith, "--approve "+cr.Digest) || cr.Text == "" {
		t.Fatalf("close document %v %s", err, out)
	}
	code, out, _ = runEmbargo(t, "", "embargo", "close", surfaceGHSA)
	if code != 0 || !strings.Contains(out, "nothing was sent") || !strings.Contains(out, "approve with: ") {
		t.Fatalf("close text: exit %d, %s", code, out)
	}

	r.Git("tag", "v0.1.0")
	code, out, stderr = runEmbargo(t, "", "embargo", "publish", "--json")
	if code != 0 {
		t.Fatalf("publish: exit %d, %s", code, stderr)
	}
	if got := strings.Join(jsonKeys(t, out), ","); got != "approved,no_draft_left,plan,published" {
		t.Errorf("publish keys %s", got)
	}
	var pr embargo.PublishResult
	if err := json.Unmarshal([]byte(out), &pr); err != nil || pr.Approved || pr.Plan.Release != "v0.1.0" || pr.Plan.Entries == nil || pr.Plan.LeftOut == nil {
		t.Fatalf("publish document %v %s", err, out)
	}
}

// TestEmbargoCountsReadAsEnglish: a count of advisories is spelled as one.
func TestEmbargoCountsReadAsEnglish(t *testing.T) {
	for n, want := range map[int]string{0: "0 advisories", 1: "1 advisory", 6: "6 advisories"} {
		if got := advisoryCount(n); got != want {
			t.Errorf("advisoryCount(%d) = %q, want %q", n, got, want)
		}
	}
}
