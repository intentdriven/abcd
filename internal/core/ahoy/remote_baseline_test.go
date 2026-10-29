package ahoy

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ghRoute is one answer of the routed gh fake: the body on stdout, and, when
// Exit is non-zero, Err on stderr as gh prints an HTTP failure.
type ghRoute struct {
	Body string
	Exit int
	Err  string
}

// ghRoutes is the routed gh fake's table, keyed "METHOD endpoint".
type ghRoutes map[string]ghRoute

// routeKeyRe flattens a route key into a file name, as the fake script does.
var routeKeyRe = regexp.MustCompile(`[^A-Za-z0-9]`)

// ghRoutedFake installs a stand-in `gh` at the front of PATH that answers each
// request by its method and endpoint, from routes, and logs one
// "ARGV …" / "STDIN …" pair per call like ghFake. A GET with no route fails as
// GitHub's 404 does; a write with no route succeeds. setRoutes replaces the
// table, so a test can serve the state an apply left behind.
func ghRoutedFake(t *testing.T, routes ghRoutes) (logPath string, setRoutes func(ghRoutes)) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	logPath = filepath.Join(dir, "gh.log")
	routeDir := filepath.Join(dir, "routes")
	setRoutes = func(rs ghRoutes) {
		if err := os.RemoveAll(routeDir); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(routeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for key, r := range rs {
			base := filepath.Join(routeDir, routeKeyRe.ReplaceAllString(key, "_"))
			write := func(ext, s string) {
				if err := os.WriteFile(base+ext, []byte(s), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			write(".body", r.Body)
			write(".err", r.Err)
			write(".exit", strconv.Itoa(r.Exit))
		}
	}
	setRoutes(routes)
	script := `#!/usr/bin/env bash
set -euo pipefail
stdin=$(cat)
{
  printf 'ARGV %s\n' "$*"
  printf 'STDIN %s\n' "$stdin"
} >>"` + logPath + `"
method=GET
endpoint=
shift # api
while [ $# -gt 0 ]; do
  case "$1" in
  --hostname|-H|--input|--jq) shift 2 ;;
  --method|-X) method=$2; shift 2 ;;
  -*) shift ;;
  *) endpoint=$1; shift ;;
  esac
done
key=$(printf '%s %s' "$method" "$endpoint" | LC_ALL=C sed 's/[^A-Za-z0-9]/_/g')
base="` + routeDir + `/$key"
if [ -f "$base.body" ]; then
  code=$(cat "$base.exit")
  cat "$base.body"
  if [ "$code" != "0" ]; then
    cat "$base.err" >&2
    exit "$code"
  fi
  exit 0
fi
if [ "$method" = "GET" ]; then
  echo "gh: Not Found (HTTP 404)" >&2
  exit 1
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath, setRoutes
}

const (
	auditRepo     = "example-org/example-repo"
	auditOrigin   = "https://github.com/" + auditRepo
	upgradeErr403 = "gh: Upgrade to GitHub Pro or make this repository public to enable this feature. (HTTP 403)"
)

// dessauRoutes is a public organization repository in the state the managed
// repository's report described: security features off, the wiki and projects
// on, branches kept after merge, a ruleset with deletion and force-push
// protection but no code-owner review, no tag ruleset, and a required check
// (zizmor) no workflow in the tree runs.
func dessauRoutes() ghRoutes {
	return ghRoutes{
		"GET repos/" + auditRepo: {Body: `{"visibility":"public","private":false,"default_branch":"main",
"has_wiki":true,"has_projects":true,"delete_branch_on_merge":false,
"allow_merge_commit":true,"allow_squash_merge":true,"allow_rebase_merge":true,
"owner":{"type":"Organization","login":"example-org"},
"security_and_analysis":{"secret_scanning":{"status":"disabled"},
"secret_scanning_push_protection":{"status":"disabled"},
"dependabot_security_updates":{"status":"disabled"}}}`},
		"GET repos/" + auditRepo + "/vulnerability-alerts": {Exit: 1, Err: "gh: Vulnerability alerts are disabled. (HTTP 404)"},
		"GET repos/" + auditRepo + "/rules/branches/main": {Body: `[{"type":"deletion"},{"type":"non_fast_forward"},
{"type":"pull_request","parameters":{"require_code_owner_review":false,"required_approving_review_count":0}},
{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"check"},{"context":"gitleaks"},{"context":"zizmor"}]}}]`},
		"GET repos/" + auditRepo + "/rulesets?targets=tag&includes_parents=true": {Body: `[]`},
		"GET repos/" + auditRepo + "/actions/permissions":                        {Body: `{"enabled":true,"allowed_actions":"all","sha_pinning_required":false}`},
		"GET repos/" + auditRepo + "/actions/permissions/workflow":               {Body: `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`},
		"GET repos/" + auditRepo + "/contents/.github/workflows?ref=main":        workflowsRoute(map[string]string{"ci.yml": dessauWorkflow}),
		"GET orgs/example-org":                     {Body: `{"plan":{"name":"free"}}`},
		"GET orgs/example-org/actions/permissions": {Exit: 1, Err: "gh: You must be an org admin or have the actions policies fine-grained permission. (HTTP 403)"},
	}
}

// workflowsRoute is GitHub's contents answer for a default branch whose
// .github/workflows holds exactly files, each with its git blob id.
func workflowsRoute(files map[string]string) ghRoute {
	var entries []string
	for name, body := range files {
		sum := sha1.Sum([]byte("blob " + strconv.Itoa(len(body)) + "\x00" + body))
		entries = append(entries, `{"name":"`+name+`","path":".github/workflows/`+name+`","type":"file","sha":"`+hex.EncodeToString(sum[:])+`"}`)
	}
	sort.Strings(entries)
	return ghRoute{Body: "[" + strings.Join(entries, ",") + "]"}
}

// writeTreeFile writes one file into the test repository.
func writeTreeFile(t *testing.T, repo, rel, body string) {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// dessauWorkflow runs check and gitleaks on every pull request, with an
// action referenced by tag rather than by commit.
const dessauWorkflow = `on:
  pull_request:
permissions:
  contents: read
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
  gitleaks:
    runs-on: ubuntu-latest
`

// setConfigValue sets one key of the test repository's .abcd/config.json.
func setConfigValue(t *testing.T, repo, section, key string, v any) {
	t.Helper()
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	setSub(cfg, section, key, v)
	if err := writeConfig(repo, cfg); err != nil {
		t.Fatal(err)
	}
}

// dessauRepo stands up the managed checkout the Dessau routes describe.
func dessauRepo(t *testing.T) string {
	t.Helper()
	repo := managedRepoWithOrigin(t, auditOrigin)
	setConfigValue(t, repo, "repo", "visibility", "public")
	writeTreeFile(t, repo, ".github/workflows/ci.yml", dessauWorkflow)
	return repo
}

// findingsBySetting indexes an audit's findings.
func findingsBySetting(t *testing.T, res RemoteResult) map[string]Finding {
	t.Helper()
	if res.Audit == nil {
		t.Fatalf("no audit on the result (status %s, notes %v)", res.Status, res.Notes)
	}
	out := map[string]Finding{}
	for _, f := range res.Audit.Findings {
		out[f.Setting] = f
	}
	return out
}

// wantStates checks each named setting's state.
func wantStates(t *testing.T, got map[string]Finding, want map[string]string) {
	t.Helper()
	for setting, state := range want {
		f, ok := got[setting]
		if !ok {
			t.Errorf("%s: no finding", setting)
			continue
		}
		if f.State != state {
			t.Errorf("%s: state %q (current %q, target %q, note %q), want %q", setting, f.State, f.Current, f.Target, f.Note, state)
		}
	}
}

// writeCalls returns the logged calls that write to GitHub.
func writeCalls(t *testing.T, logPath string) []string {
	t.Helper()
	var out []string
	for _, c := range ghCalls(t, logPath) {
		for _, m := range []string{"--method PATCH", "--method PUT", "--method POST", "--method DELETE"} {
			if strings.Contains(c, m) {
				out = append(out, c)
			}
		}
	}
	return out
}

// treeSnapshot is the repository's working-tree state as git sees it, plus
// every file's bytes under .abcd/work and .github.
func treeSnapshot(t *testing.T, repo string) string {
	t.Helper()
	var b strings.Builder
	for _, dir := range []string{".abcd/work", ".github"} {
		_ = filepath.Walk(filepath.Join(repo, dir), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			data, _ := os.ReadFile(p)
			b.WriteString(p + "\n" + string(data) + "\n")
			return nil
		})
	}
	return b.String()
}

// TestRemoteAuditListsEachDifferenceAndChangesNothing is the intent's first
// acceptance criterion: the read lists every baseline difference as setting,
// current and target, and writes nothing on GitHub or in the tree.
func TestRemoteAuditListsEachDifferenceAndChangesNothing(t *testing.T) {
	setupHermetic(t)
	logPath, _ := ghRoutedFake(t, dessauRoutes())
	repo := dessauRepo(t)
	before := treeSnapshot(t, repo)

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBySetting(t, res)
	wantStates(t, got, map[string]string{
		"secret_scanning":                 "differs",
		"secret_scanning_push_protection": "differs",
		"dependabot_alerts":               "differs",
		"dependabot_security_updates":     "differs",
		"delete_branch_on_merge":          "differs",
		"has_wiki":                        "differs",
		"has_projects":                    "differs",
		"default_branch_deletion":         "ok",
		"default_branch_force_push":       "ok",
		"required_status_checks":          "advisory",
		"code_owner_review":               "differs",
		"release_tags":                    "differs",
		"codeowners":                      "advisory",
		"actions_sha_pinning":             "advisory",
		"workflow_token":                  "ok",
		"org_actions_policy":              "unknown",
		"visibility":                      "ok",
	})
	for _, f := range res.Audit.Findings {
		if f.State == "differs" && (f.Current == "" || f.Target == "") {
			t.Errorf("%s differs but does not say from what to what: %+v", f.Setting, f)
		}
	}
	if n := got["required_status_checks"].Note; !strings.Contains(n, "zizmor") {
		t.Errorf("a required check no workflow runs is not named: %q", n)
	}
	if n := got["actions_sha_pinning"].Note; !strings.Contains(n, "1 ") {
		t.Errorf("the unpinned count is not given: %q", n)
	}
	if n := got["org_actions_policy"].Note; !strings.Contains(n, "admin:org") {
		t.Errorf("the missing scope is not named: %q", n)
	}
	if res.Audit.Plan != "free" || res.Audit.Visibility != "public" || res.Audit.DeclaredVisibility != "public" {
		t.Errorf("plan %q, visibility %q, declared %q", res.Audit.Plan, res.Audit.Visibility, res.Audit.DeclaredVisibility)
	}
	if res.Status != "pending" {
		t.Errorf("status %q, want pending", res.Status)
	}
	if w := writeCalls(t, logPath); len(w) != 0 {
		t.Errorf("the read wrote to GitHub:\n%s", strings.Join(w, "\n"))
	}
	if after := treeSnapshot(t, repo); after != before {
		t.Error("the read changed the tree")
	}
}

// TestRemoteAuditRecordsWhatThePlanDoesNotOfferAsOwed: on a private repository
// whose plan offers neither secret scanning nor rulesets, those settings are
// owed until the repository is public, never a difference to write.
func TestRemoteAuditRecordsWhatThePlanDoesNotOfferAsOwed(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo] = ghRoute{Body: `{"visibility":"private","private":true,"default_branch":"main",
"has_wiki":false,"has_projects":false,"delete_branch_on_merge":true,
"owner":{"type":"Organization","login":"example-org"},
"security_and_analysis":{"dependabot_security_updates":{"status":"disabled"}}}`}
	routes["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Exit: 1, Err: upgradeErr403}
	routes["GET repos/"+auditRepo+"/rulesets?targets=tag&includes_parents=true"] = ghRoute{Exit: 1, Err: upgradeErr403}
	ghRoutedFake(t, routes)
	repo := dessauRepo(t)
	setConfigValue(t, repo, "repo", "visibility", "private")

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBySetting(t, res)
	wantStates(t, got, map[string]string{
		"secret_scanning":                 "owed",
		"secret_scanning_push_protection": "owed",
		"default_branch_deletion":         "owed",
		"default_branch_force_push":       "owed",
		"required_status_checks":          "owed",
		"code_owner_review":               "owed",
		"release_tags":                    "owed",
		"dependabot_security_updates":     "differs",
		"dependabot_alerts":               "differs",
	})
	for _, c := range res.Changes {
		if strings.HasPrefix(c, "secret_scanning") {
			t.Errorf("an owed setting is listed as a change: %s", c)
		}
	}
}

// TestRemoteAuditReportsAVisibilityMismatch: the declared and the real
// visibility are compared and the difference reported, as advisory: no
// command of abcd's changes the visibility.
func TestRemoteAuditReportsAVisibilityMismatch(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo] = ghRoute{Body: strings.Replace(routes["GET repos/"+auditRepo].Body,
		`"visibility":"public","private":false`, `"visibility":"private","private":true`, 1)}
	ghRoutedFake(t, routes)
	repo := dessauRepo(t)

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	f := findingsBySetting(t, res)["visibility"]
	if f.State != "advisory" || f.Current != "private" || f.Target != "public" {
		t.Errorf("visibility finding %+v, want advisory private -> public", f)
	}
	for _, c := range res.Changes {
		if strings.Contains(c, "visibility") {
			t.Errorf("visibility is listed as a change abcd would make: %s", c)
		}
	}
}

// TestRemoteAuditHonoursKeepAndRefusesAnUnknownKey: a setting the repository
// keeps is reported as kept and never proposed; a key the baseline does not
// know refuses, naming it, so a typo cannot quietly keep nothing.
func TestRemoteAuditHonoursKeepAndRefusesAnUnknownKey(t *testing.T) {
	setupHermetic(t)
	logPath, _ := ghRoutedFake(t, dessauRoutes())
	repo := dessauRepo(t)
	setConfigValue(t, repo, "github", "keep", []any{"has_wiki", "secret_scanning_push_protection"})

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	wantStates(t, findingsBySetting(t, res), map[string]string{
		"has_wiki": "kept", "secret_scanning_push_protection": "kept",
		"has_projects": "differs", "secret_scanning": "differs",
	})
	for _, c := range res.Changes {
		if strings.HasPrefix(c, "has_wiki") || strings.HasPrefix(c, "secret_scanning_push_protection") {
			t.Errorf("a kept setting is listed as a change: %s", c)
		}
	}

	setConfigValue(t, repo, "github", "keep", []any{"has_wikki"})
	_ = os.Remove(logPath)
	res, err = RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "refused" || !strings.Contains(strings.Join(res.Notes, " "), "has_wikki") {
		t.Errorf("an unknown keep key gave status %q, notes %v; want refused naming it", res.Status, res.Notes)
	}
	if calls := ghCalls(t, logPath); len(calls) != 0 {
		t.Errorf("a refused config still contacted GitHub:\n%s", strings.Join(calls, "\n"))
	}
}

// TestRemoteAuditRequiresTheConfiguredChecks: github.required_checks replaces
// the checks derived from the workflows.
func TestRemoteAuditRequiresTheConfiguredChecks(t *testing.T) {
	setupHermetic(t)
	ghRoutedFake(t, dessauRoutes())
	repo := dessauRepo(t)
	setConfigValue(t, repo, "github", "required_checks", []any{"check", "gitleaks", "zizmor"})

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	if f := findingsBySetting(t, res)["required_status_checks"]; f.State != "ok" {
		t.Errorf("required_status_checks %+v, want ok once the config names zizmor", f)
	}
	if got := strings.Join(res.Audit.RequiredChecks, ","); got != "check,gitleaks,zizmor" {
		t.Errorf("required checks %q", got)
	}
}

// TestRemoteAuditOffersSHAPinningOnlyWhenEveryActionIsPinned: with every
// `uses:` pinned to a commit, requiring SHA pinning breaks nothing, so it is a
// difference the apply may make.
func TestRemoteAuditOffersSHAPinningOnlyWhenEveryActionIsPinned(t *testing.T) {
	setupHermetic(t)
	pinned := strings.Replace(dessauWorkflow,
		"actions/checkout@v4", "actions/checkout@0123456789abcdef0123456789abcdef01234567", 1)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo+"/contents/.github/workflows?ref=main"] = workflowsRoute(map[string]string{"ci.yml": pinned})
	ghRoutedFake(t, routes)
	repo := dessauRepo(t)
	writeTreeFile(t, repo, ".github/workflows/ci.yml", pinned)

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	wantStates(t, findingsBySetting(t, res), map[string]string{"actions_sha_pinning": "differs"})
}

// TestRemoteAuditKeepsTheScanningTogglesOfARepoThatOptedOut: the
// native-scanning opt-out keeps the two toggles it names, and the rest of the
// baseline is still audited.
func TestRemoteAuditKeepsTheScanningTogglesOfARepoThatOptedOut(t *testing.T) {
	setupHermetic(t)
	ghRoutedFake(t, dessauRoutes())
	repo := dessauRepo(t)
	writeNativeScanningOptOut(t, repo)

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	wantStates(t, findingsBySetting(t, res), map[string]string{
		"secret_scanning":                 "kept",
		"secret_scanning_push_protection": "kept",
		"has_wiki":                        "differs",
	})
	for _, c := range res.Changes {
		if strings.HasPrefix(c, "secret_scanning") {
			t.Errorf("an opted-out toggle is listed as a change: %s", c)
		}
	}
}

// TestRemoteAuditHoldsTheActionsTighteningsWhenAWorkflowWentUnread pins that
// a workflow abcd could not read, or a local action whose own steps it does
// not read, is never counted as compliant: SHA pinning and a read-only token
// stay advisory, naming what went unread, because requiring either over steps
// abcd has not seen could stop a CI that works today.
func TestRemoteAuditHoldsTheActionsTighteningsWhenAWorkflowWentUnread(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo+"/actions/permissions/workflow"] = ghRoute{Body: `{"default_workflow_permissions":"write","can_approve_pull_request_reviews":false}`}
	ghRoutedFake(t, routes)
	repo := dessauRepo(t)
	pinned := strings.Replace(dessauWorkflow,
		"actions/checkout@v4", "actions/checkout@0123456789abcdef0123456789abcdef01234567", 1)
	writeTreeFile(t, repo, ".github/workflows/ci.yml", pinned)
	writeTreeFile(t, repo, ".github/workflows/huge.yml", "on: push\n"+strings.Repeat("# padding\n", 30000))

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBySetting(t, res)
	wantStates(t, got, map[string]string{"actions_sha_pinning": "advisory", "workflow_token": "advisory"})
	for _, s := range []string{"actions_sha_pinning", "workflow_token"} {
		if !strings.Contains(got[s].Note, "huge.yml") {
			t.Errorf("%s does not name the unread workflow: %q", s, got[s].Note)
		}
	}

	writeTreeFile(t, repo, ".github/workflows/huge.yml", "on: push\npermissions: {}\njobs:\n  a:\n    steps:\n      - uses: ./.github/actions/setup\n")
	res, err = RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	got = findingsBySetting(t, res)
	wantStates(t, got, map[string]string{"actions_sha_pinning": "advisory"})
	if !strings.Contains(got["actions_sha_pinning"].Note, "./.github/actions/setup") {
		t.Errorf("the local action is not named: %q", got["actions_sha_pinning"].Note)
	}
}

// TestRemoteAuditDoesNotReadANonAdminsAlertsAnswerAsOff pins that the 404
// GitHub gives a sign-in that is not an admin of the repository is not taken
// as "Dependabot alerts are off": the endpoint answers only admins, so the
// setting is unknown, and an unknown is never applied.
func TestRemoteAuditDoesNotReadANonAdminsAlertsAnswerAsOff(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo] = ghRoute{Body: strings.Replace(routes["GET repos/"+auditRepo].Body,
		`"default_branch":"main",`, `"default_branch":"main","permissions":{"admin":false,"push":true},`, 1)}
	ghRoutedFake(t, routes)
	repo := dessauRepo(t)

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBySetting(t, res)
	wantStates(t, got, map[string]string{"dependabot_alerts": "unknown"})
	if !strings.Contains(got["dependabot_alerts"].Note, "admin") {
		t.Errorf("the note does not say the answer needs an admin: %q", got["dependabot_alerts"].Note)
	}
}

// TestRemoteAuditLeavesTheTogglesUnknownWhenTheAnswerOmitsThem pins that a
// public repository's answer without security_and_analysis, which GitHub
// shows only to an admin, makes both scanning toggles unknown rather than off,
// so the apply writes neither.
func TestRemoteAuditLeavesTheTogglesUnknownWhenTheAnswerOmitsThem(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	body := routes["GET repos/"+auditRepo].Body
	i := strings.Index(body, `,
"security_and_analysis"`)
	routes["GET repos/"+auditRepo] = ghRoute{Body: body[:i] + "}"}
	logPath, _ := ghRoutedFake(t, routes)
	repo := dessauRepo(t)

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantStates(t, findingsBySetting(t, res), map[string]string{
		"secret_scanning": "unknown", "secret_scanning_push_protection": "unknown",
	})
	for _, c := range writeCalls(t, logPath) {
		if strings.Contains(c, "secret_scanning") {
			t.Errorf("a toggle abcd could not read was written:\n%s", c)
		}
	}
}

// TestRemoteAuditHoldsWorkflowSettingsWhenTheCheckoutDiffersFromTheDefaultBranch
// pins that the settings drawn from the workflows are judged against the
// default branch's workflows on GitHub, not the checkout's: from a feature
// branch whose workflows differ, the required checks, SHA pinning and the
// workflow token are advisory and name the difference, and the apply writes
// no required check, since every pull request into the default branch would
// wait on a check only that branch's workflows report.
func TestRemoteAuditHoldsWorkflowSettingsWhenTheCheckoutDiffersFromTheDefaultBranch(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Body: `[{"type":"deletion"},{"type":"non_fast_forward"},
{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"check"}]}}]`}
	routes["GET repos/"+auditRepo+"/actions/permissions/workflow"] = ghRoute{Body: `{"default_workflow_permissions":"write","can_approve_pull_request_reviews":false}`}
	routes["GET repos/"+auditRepo+"/contents/.github/workflows?ref=main"] = workflowsRoute(map[string]string{"ci.yml": "on:\n  pull_request:\njobs:\n  check:\n    runs-on: x\n"})
	logPath, _ := ghRoutedFake(t, routes)
	repo := dessauRepo(t)

	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBySetting(t, res)
	wantStates(t, got, map[string]string{"required_status_checks": "advisory", "workflow_token": "advisory"})
	for _, s := range []string{"required_status_checks", "workflow_token"} {
		if !strings.Contains(got[s].Note, "ci.yml") || !strings.Contains(got[s].Note, "main") {
			t.Errorf("%s does not name the differing workflow and the default branch: %q", s, got[s].Note)
		}
	}

	if _, err := RemoteApply(repo, confirmingPrompter{}, nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range writeCalls(t, logPath) {
		if strings.Contains(stdinOf(c), "required_status_checks") {
			t.Errorf("a required check was written from a checkout that differs from the default branch:\n%s", c)
		}
		if strings.Contains(c, "actions/permissions/workflow") {
			t.Errorf("the workflow token was written from a checkout that differs from the default branch:\n%s", c)
		}
	}
}
