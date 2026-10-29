package ahoy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// callsTo returns the logged calls whose argv names method and contains
// endpoint, in order.
func callsTo(t *testing.T, logPath, method, endpoint string) []string {
	t.Helper()
	var out []string
	for _, c := range ghCalls(t, logPath) {
		argv, _, _ := strings.Cut(c, " | STDIN ")
		if strings.Contains(argv, "--method "+method) && strings.Contains(argv, endpoint) {
			out = append(out, c)
		}
	}
	return out
}

// stdinOf is the request body a logged call sent.
func stdinOf(call string) string {
	_, body, _ := strings.Cut(call, " | STDIN ")
	return body
}

// indexOfCall is the position of the first logged call containing all of
// parts, or -1.
func indexOfCall(t *testing.T, logPath string, parts ...string) int {
	t.Helper()
	for i, c := range ghCalls(t, logPath) {
		all := true
		for _, p := range parts {
			if !strings.Contains(c, p) {
				all = false
			}
		}
		if all {
			return i
		}
	}
	return -1
}

// postApplyDessauRoutes is the Dessau repository after an apply: what GitHub
// would answer once every difference abcd can apply has landed.
func postApplyDessauRoutes() ghRoutes {
	r := dessauRoutes()
	r["GET repos/"+auditRepo] = ghRoute{Body: `{"visibility":"public","private":false,"default_branch":"main",
"has_wiki":false,"has_projects":false,"delete_branch_on_merge":true,
"allow_merge_commit":true,"allow_squash_merge":true,"allow_rebase_merge":true,
"owner":{"type":"Organization","login":"example-org"},
"security_and_analysis":{"secret_scanning":{"status":"enabled"},
"secret_scanning_push_protection":{"status":"enabled"},
"dependabot_security_updates":{"status":"enabled"}}}`}
	r["GET repos/"+auditRepo+"/vulnerability-alerts"] = ghRoute{}
	r["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Body: `[{"type":"deletion"},{"type":"non_fast_forward"},
{"type":"pull_request","parameters":{"require_code_owner_review":true,"required_approving_review_count":0}},
{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"check"},{"context":"gitleaks"},{"context":"zizmor"}]}}]`}
	r["GET repos/"+auditRepo+"/rulesets?targets=tag&includes_parents=true"] = ghRoute{Body: `[{"id":7,"name":"abcd: release tags","target":"tag","enforcement":"active"}]`}
	r["GET repos/"+auditRepo+"/rulesets/7?includes_parents=true"] = ghRoute{Body: `{"conditions":{"ref_name":{"include":["refs/tags/v*"],"exclude":[]}},
"rules":[{"type":"deletion"},{"type":"update"},{"type":"non_fast_forward"}]}`}
	r["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[{"id":5,"name":"abcd: default branch"},{"id":6,"name":"abcd: code-owner review"},{"id":7,"name":"abcd: release tags"}]`}
	return r
}

// TestRemoteApplyAppliesTheDifferencesThenAuditsClean is the intent's second
// acceptance criterion: an apply after a yes applies every difference abcd can
// apply, in the order GitHub requires; a second audit then reports none; and
// a second apply makes no write and rewrites no file. No write's body names
// the repository's visibility.
func TestRemoteApplyAppliesTheDifferencesThenAuditsClean(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[]`}
	logPath, setRoutes := ghRoutedFake(t, routes)
	repo := dessauRepo(t)

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("status %q, notes %v", res.Status, res.Notes)
	}
	repoPath := "repos/" + auditRepo
	if i, j := indexOfCall(t, logPath, "PATCH", `"secret_scanning":`), indexOfCall(t, logPath, "PATCH", "secret_scanning_push_protection"); i < 0 || j < 0 || i > j {
		t.Errorf("secret scanning (%d) must be enabled before push protection (%d)", i, j)
	}
	if i, j := indexOfCall(t, logPath, "PUT", "vulnerability-alerts"), indexOfCall(t, logPath, "PUT", "automated-security-fixes"); i < 0 || j < 0 || i > j {
		t.Errorf("Dependabot alerts (%d) must be enabled before security updates (%d)", i, j)
	}
	var hygiene map[string]any
	for _, c := range callsTo(t, logPath, "PATCH", repoPath) {
		if strings.Contains(c, "has_wiki") {
			if err := json.Unmarshal([]byte(stdinOf(c)), &hygiene); err != nil {
				t.Fatal(err)
			}
		}
	}
	if hygiene["has_wiki"] != false || hygiene["has_projects"] != false || hygiene["delete_branch_on_merge"] != true {
		t.Errorf("repository settings PATCH = %v", hygiene)
	}
	// The repository's own ruleset already blocks deletion and force-push and
	// requires check and gitleaks, so abcd adds only the two rulesets it lacks.
	posts := callsTo(t, logPath, "POST", repoPath+"/rulesets")
	if len(posts) != 2 {
		t.Fatalf("ruleset creations = %d, want 2:\n%s", len(posts), strings.Join(posts, "\n"))
	}
	for _, c := range posts {
		var body struct {
			Name        string           `json:"name"`
			BypassActor []map[string]any `json:"bypass_actors"`
		}
		if err := json.Unmarshal([]byte(stdinOf(c)), &body); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(body.Name, "abcd: ") || len(body.BypassActor) != 0 {
			t.Errorf("ruleset %q has bypass actors %v; abcd's rulesets have none", body.Name, body.BypassActor)
		}
	}
	if w := callsTo(t, logPath, "PUT", "actions/permissions"); len(w) != 0 {
		t.Errorf("the Actions policy was written while an action is unpinned:\n%s", strings.Join(w, "\n"))
	}
	for _, c := range writeCalls(t, logPath) {
		if b := stdinOf(c); strings.Contains(b, "visibility") || strings.Contains(b, `"private"`) {
			t.Errorf("a write names the repository's visibility:\n%s", c)
		}
	}
	for _, name := range []string{"abcd-code-owner-review.json", "abcd-release-tags.json"} {
		if _, err := os.Stat(filepath.Join(repo, ".abcd", "work", "rulesets", name)); err != nil {
			t.Errorf("ruleset mirror %s: %v", name, err)
		}
	}

	setRoutes(postApplyDessauRoutes())
	read, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	if d := read.Audit.Differences(); len(d) != 0 {
		t.Errorf("after the apply the audit still lists differences: %v", d)
	}
	_ = os.Remove(logPath)
	before := treeSnapshot(t, repo)
	again, err := RemoteApply(repo, RefusingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != "already_up_to_date" {
		t.Errorf("second apply status %q, notes %v", again.Status, again.Notes)
	}
	if w := writeCalls(t, logPath); len(w) != 0 {
		t.Errorf("the second apply wrote:\n%s", strings.Join(w, "\n"))
	}
	if treeSnapshot(t, repo) != before {
		t.Error("the second apply rewrote a file")
	}
}

// TestRemoteApplyOnAPrivateRepoAppliesWhatThePlanOffers: the plan does not
// offer secret scanning or rulesets for this private repository, so the apply
// writes neither, applies the rest, and records the rest as owed.
func TestRemoteApplyOnAPrivateRepoAppliesWhatThePlanOffers(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo] = ghRoute{Body: `{"visibility":"private","private":true,"default_branch":"main",
"has_wiki":false,"has_projects":false,"delete_branch_on_merge":true,
"owner":{"type":"Organization","login":"example-org"},
"security_and_analysis":{"dependabot_security_updates":{"status":"disabled"}}}`}
	routes["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Exit: 1, Err: upgradeErr403}
	routes["GET repos/"+auditRepo+"/rulesets?targets=tag&includes_parents=true"] = ghRoute{Exit: 1, Err: upgradeErr403}
	logPath, _ := ghRoutedFake(t, routes)
	repo := dessauRepo(t)
	setConfigValue(t, repo, "repo", "visibility", "private")

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("status %q, notes %v", res.Status, res.Notes)
	}
	if w := callsTo(t, logPath, "PATCH", "repos/"+auditRepo); len(w) != 0 {
		t.Errorf("an owed or settled setting was written:\n%s", strings.Join(w, "\n"))
	}
	if w := callsTo(t, logPath, "POST", "rulesets"); len(w) != 0 {
		t.Errorf("a ruleset was created where the plan offers none:\n%s", strings.Join(w, "\n"))
	}
	if len(callsTo(t, logPath, "PUT", "vulnerability-alerts")) != 1 || len(callsTo(t, logPath, "PUT", "automated-security-fixes")) != 1 {
		t.Errorf("the Dependabot settings the plan offers were not applied:\n%s", strings.Join(writeCalls(t, logPath), "\n"))
	}
	var m struct {
		Owed []string `json:"owed"`
	}
	if err := json.Unmarshal([]byte(mirror(t, repo)), &m); err != nil {
		t.Fatal(err)
	}
	owed := strings.Join(m.Owed, ",")
	for _, want := range []string{"secret_scanning", "secret_scanning_push_protection", "release_tags", "code_owner_review"} {
		if !strings.Contains(owed, want) {
			t.Errorf("the mirror does not record %s as owed: %v", want, m.Owed)
		}
	}
}

// TestRemoteApplyRequiresExactlyTheChecksEveryPullRequestRuns: under a merge
// queue, a path-filtered workflow and one that does not run on the queue are
// left out, so abcd's default-branch ruleset requires only build and lint.
func TestRemoteApplyRequiresExactlyTheChecksEveryPullRequestRuns(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Body: `[{"type":"merge_queue"}]`}
	routes["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[]`}
	workflows := map[string]string{
		"ci.yml":   "on:\n  pull_request:\n  merge_group:\npermissions: {}\njobs:\n  build:\n    runs-on: x\n  lint:\n    runs-on: x\n",
		"docs.yml": "on:\n  pull_request:\n    paths: [docs/**]\n  merge_group:\njobs:\n  docs:\n    runs-on: x\n",
		"pr.yml":   "on: [pull_request]\njobs:\n  pr-only:\n    runs-on: x\n",
	}
	routes["GET repos/"+auditRepo+"/contents/.github/workflows?ref=main"] = workflowsRoute(workflows)
	logPath, _ := ghRoutedFake(t, routes)
	repo := dessauRepo(t)
	for name, body := range workflows {
		writeTreeFile(t, repo, ".github/workflows/"+name, body)
	}

	if _, err := RemoteApply(repo, confirmingPrompter{}, nil); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Rules []struct {
			Type       string `json:"type"`
			Parameters struct {
				Checks []struct {
					Context string `json:"context"`
				} `json:"required_status_checks"`
			} `json:"parameters"`
		} `json:"rules"`
	}
	for _, c := range callsTo(t, logPath, "POST", "rulesets") {
		if strings.Contains(c, "abcd: default branch") {
			if err := json.Unmarshal([]byte(stdinOf(c)), &body); err != nil {
				t.Fatal(err)
			}
		}
	}
	var got []string
	for _, r := range body.Rules {
		if r.Type == "required_status_checks" {
			for _, c := range r.Parameters.Checks {
				got = append(got, c.Context)
			}
		}
	}
	if strings.Join(got, ",") != "build,lint" {
		t.Errorf("abcd's ruleset requires %v, want [build lint]", got)
	}
}

// TestRemoteApplyLeavesAKeptSettingAlone: github.keep takes a setting out of
// every write.
func TestRemoteApplyLeavesAKeptSettingAlone(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[]`}
	logPath, _ := ghRoutedFake(t, routes)
	repo := dessauRepo(t)
	setConfigValue(t, repo, "github", "keep", []any{"has_wiki", "release_tags"})

	if _, err := RemoteApply(repo, confirmingPrompter{}, nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range writeCalls(t, logPath) {
		if strings.Contains(stdinOf(c), "has_wiki") || strings.Contains(stdinOf(c), "abcd: release tags") {
			t.Errorf("a kept setting was written:\n%s", c)
		}
	}
	if indexOfCall(t, logPath, "PATCH", "has_projects") < 0 {
		t.Error("the settings not kept were not applied")
	}
}

// TestRemoteApplyWritesNoLoginIntoAnyFile is the spec's no-login requirement:
// a repository owned by a person is mirrored by its name alone, and a bypass
// actor on a ruleset abcd updates is mirrored by type and id, so nothing abcd
// writes into the tree names a person.
func TestRemoteApplyWritesNoLoginIntoAnyFile(t *testing.T) {
	setupHermetic(t)
	const login = "zq-owner-7x"
	const userRepo = login + "/example-repo"
	routes := ghRoutes{}
	for k, v := range dessauRoutes() {
		routes[strings.Replace(k, auditRepo, userRepo, 1)] = v
	}
	delete(routes, "GET orgs/example-org")
	delete(routes, "GET orgs/example-org/actions/permissions")
	routes["GET repos/"+userRepo] = ghRoute{Body: strings.Replace(dessauRoutes()["GET repos/"+auditRepo].Body,
		`"owner":{"type":"Organization","login":"example-org"}`, `"owner":{"type":"User","login":"`+login+`"}`, 1)}
	routes["GET repos/"+userRepo+"/rules/branches/main"] = ghRoute{Body: `[]`}
	routes["GET user"] = ghRoute{Body: `{"login":"` + login + `","plan":{"name":"free"}}`}
	routes["GET repos/"+userRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[{"id":5,"name":"abcd: default branch"}]`}
	routes["GET repos/"+userRepo+"/rulesets/5"] = ghRoute{Body: `{"id":5,"name":"abcd: default branch","target":"branch",
"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"rules":[{"type":"deletion"}],
"bypass_actors":[{"actor_id":4242,"actor_type":"Integration","bypass_mode":"always"}]}`}
	logPath, _ := ghRoutedFake(t, routes)
	repo := managedRepoWithOrigin(t, "https://github.com/"+userRepo)
	setConfigValue(t, repo, "repo", "visibility", "public")
	writeTreeFile(t, repo, ".github/workflows/ci.yml", dessauWorkflow)

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("status %q, notes %v", res.Status, res.Notes)
	}
	if len(callsTo(t, logPath, "PUT", "repos/"+userRepo+"/rulesets/5")) != 1 {
		t.Errorf("abcd's existing ruleset was not updated in place:\n%s", strings.Join(writeCalls(t, logPath), "\n"))
	}
	_ = filepath.Walk(filepath.Join(repo, ".abcd", "work"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, _ := os.ReadFile(p)
		if strings.Contains(string(data), login) {
			t.Errorf("%s names the owner's login:\n%s", p, data)
		}
		return nil
	})
	data, err := os.ReadFile(filepath.Join(repo, ".abcd", "work", "rulesets", "abcd-default-branch.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"actor_id": 4242`) || !strings.Contains(string(data), `"actor_type": "Integration"`) {
		t.Errorf("the kept bypass actor is not mirrored by type and id:\n%s", data)
	}
}

// strictDefaultBranchRoutes serves a repository whose own "abcd: default
// branch" ruleset (id 5) is stricter than abcd's baseline in every way an
// update could lose: an extra rule, a pinned check source, a strict policy, a
// second ref and a check no workflow reports. Only gitleaks is missing, so the
// required checks differ and the apply updates that ruleset.
func strictDefaultBranchRoutes(ruleset string) ghRoutes {
	r := dessauRoutes()
	r["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Body: `[{"type":"deletion"},{"type":"non_fast_forward"},
{"type":"required_signatures"},
{"type":"pull_request","parameters":{"require_code_owner_review":true,"required_approving_review_count":0}},
{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"check","integration_id":15368},{"context":"extra"}]}}]`}
	r["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[{"id":5,"name":"abcd: default branch","source_type":"Repository"}]`}
	r["GET repos/"+auditRepo+"/rulesets/5"] = ghRoute{Body: ruleset}
	return r
}

const strictDefaultBranchRuleset = `{"id":5,"name":"abcd: default branch","target":"branch","enforcement":"active",
"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH","refs/heads/release/*"],"exclude":[]}},
"bypass_actors":[],
"rules":[{"type":"deletion"},{"type":"non_fast_forward"},{"type":"required_signatures"},
{"type":"required_status_checks","parameters":{"do_not_enforce_on_create":false,"strict_required_status_checks_policy":true,
"required_status_checks":[{"context":"check","integration_id":15368},{"context":"extra"}]}}]}`

// TestRemoteApplyMergesIntoAStricterRulesetOfItsOwnName pins that an update of
// a ruleset bearing abcd's name never leaves the repository less protected: a
// PUT replaces a ruleset's rules wholesale, so abcd starts from what the
// ruleset holds and only adds. Every existing rule, the pinned check source,
// the strict policy, the extra ref and the stale check all survive; a kept
// setting's existing rule is not stripped; and gitleaks is added.
func TestRemoteApplyMergesIntoAStricterRulesetOfItsOwnName(t *testing.T) {
	setupHermetic(t)
	logPath, _ := ghRoutedFake(t, strictDefaultBranchRoutes(strictDefaultBranchRuleset))
	repo := dessauRepo(t)
	setConfigValue(t, repo, "github", "keep", []any{"default_branch_force_push"})

	if _, err := RemoteApply(repo, confirmingPrompter{}, nil); err != nil {
		t.Fatal(err)
	}
	puts := callsTo(t, logPath, "PUT", "repos/"+auditRepo+"/rulesets/5")
	if len(puts) != 1 {
		t.Fatalf("abcd's ruleset was not updated once:\n%s", strings.Join(writeCalls(t, logPath), "\n"))
	}
	var body struct {
		Enforcement string `json:"enforcement"`
		Conditions  struct {
			RefName struct {
				Include []string `json:"include"`
			} `json:"ref_name"`
		} `json:"conditions"`
		Rules []struct {
			Type       string `json:"type"`
			Parameters struct {
				Strict *bool `json:"strict_required_status_checks_policy"`
				Checks []struct {
					Context       string `json:"context"`
					IntegrationID *int   `json:"integration_id"`
				} `json:"required_status_checks"`
			} `json:"parameters"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(stdinOf(puts[0])), &body); err != nil {
		t.Fatalf("PUT body: %v\n%s", err, stdinOf(puts[0]))
	}
	types := map[string]bool{}
	checks := map[string]*int{}
	strict := false
	for _, r := range body.Rules {
		types[r.Type] = true
		if r.Type == "required_status_checks" {
			strict = r.Parameters.Strict != nil && *r.Parameters.Strict
			for _, c := range r.Parameters.Checks {
				checks[c.Context] = c.IntegrationID
			}
		}
	}
	for _, want := range []string{"deletion", "non_fast_forward", "required_signatures", "required_status_checks"} {
		if !types[want] {
			t.Errorf("the update dropped the %s rule: %s", want, stdinOf(puts[0]))
		}
	}
	for _, want := range []string{"check", "extra", "gitleaks"} {
		if _, ok := checks[want]; !ok {
			t.Errorf("the update does not require %s: %s", want, stdinOf(puts[0]))
		}
	}
	if id := checks["check"]; id == nil || *id != 15368 {
		t.Errorf("the update dropped check's pinned source: %s", stdinOf(puts[0]))
	}
	if !strict {
		t.Errorf("the update relaxed the strict policy: %s", stdinOf(puts[0]))
	}
	if inc := strings.Join(body.Conditions.RefName.Include, ","); !strings.Contains(inc, "refs/heads/release/*") || !strings.Contains(inc, "~DEFAULT_BRANCH") {
		t.Errorf("the update narrowed the refs to %q", inc)
	}
	if body.Enforcement != "active" {
		t.Errorf("enforcement %q, want active", body.Enforcement)
	}
}

// TestRemoteApplyRefusesToUpdateARulesetWhoseBypassItCannotRead pins that a
// ruleset answer with no bypass_actors key (the field GitHub leaves out for a
// caller who may not see it) is never read as "no bypass" and written back as
// one: the update refuses and changes nothing of that ruleset.
func TestRemoteApplyRefusesToUpdateARulesetWhoseBypassItCannotRead(t *testing.T) {
	setupHermetic(t)
	noBypass := strings.Replace(strictDefaultBranchRuleset, `"bypass_actors":[],`, "", 1)
	logPath, _ := ghRoutedFake(t, strictDefaultBranchRoutes(noBypass))
	repo := dessauRepo(t)

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if puts := callsTo(t, logPath, "PUT", "rulesets/5"); len(puts) != 0 {
		t.Errorf("the ruleset was rewritten without its bypass list:\n%s", strings.Join(puts, "\n"))
	}
	if !strings.Contains(fmt.Sprint(err, res.Notes), "bypass") {
		t.Errorf("the refusal does not say the bypass list was missing: %v %v", err, res.Notes)
	}
}

// TestRemoteApplyUpdatesOnlyARulesetTheRepositoryItselfHolds pins that a
// ruleset of abcd's name inherited from the organization is not abcd's to
// update through the repository: it is skipped, and abcd's own is created.
func TestRemoteApplyUpdatesOnlyARulesetTheRepositoryItselfHolds(t *testing.T) {
	setupHermetic(t)
	routes := strictDefaultBranchRoutes(strictDefaultBranchRuleset)
	routes["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[{"id":5,"name":"abcd: default branch","source_type":"Organization"}]`}
	logPath, _ := ghRoutedFake(t, routes)
	repo := dessauRepo(t)

	if _, err := RemoteApply(repo, confirmingPrompter{}, nil); err != nil {
		t.Fatal(err)
	}
	if puts := callsTo(t, logPath, "PUT", "rulesets/5"); len(puts) != 0 {
		t.Errorf("an organization's ruleset was updated:\n%s", strings.Join(puts, "\n"))
	}
	if posts := callsTo(t, logPath, "POST", "repos/"+auditRepo+"/rulesets"); len(posts) == 0 {
		t.Error("abcd's own default-branch ruleset was not created")
	}
}

// TestRemoteApplyRefusesWhenTheRulesetListMayBeCutShort pins that a listing
// that fills the one page abcd reads is not taken as the whole list: abcd
// could not tell whether its own ruleset is on a later page, and creating it
// again would leave two.
func TestRemoteApplyRefusesWhenTheRulesetListMayBeCutShort(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	var list []string
	for i := 1; i <= 100; i++ {
		list = append(list, fmt.Sprintf(`{"id":%d,"name":"other %d","source_type":"Repository"}`, 1000+i, i))
	}
	routes["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: "[" + strings.Join(list, ",") + "]"}
	logPath, _ := ghRoutedFake(t, routes)
	repo := dessauRepo(t)

	res, _ := RemoteApply(repo, confirmingPrompter{}, nil)
	if posts := callsTo(t, logPath, "POST", "rulesets"); len(posts) != 0 {
		t.Errorf("a ruleset was created from a list that may be cut short:\n%s", strings.Join(posts, "\n"))
	}
	if !strings.Contains(strings.Join(res.Notes, "\n"), "100") {
		t.Errorf("the refusal does not say the list may be cut short: %v", res.Notes)
	}
}

// questionRecorder records the confirmation it is asked, and declines.
type questionRecorder struct{ asked *[]string }

func (q questionRecorder) Confirm(text string) bool                     { *q.asked = append(*q.asked, text); return false }
func (questionRecorder) Prompt(_ string, _ []string, def string) string { return def }

// TestRemoteApplySanitisesTheConfirmationItAsks pins that the question put to
// the person carries no terminal control sequence from what the API answered:
// a required-check context is text a ruleset's author chose.
func TestRemoteApplySanitisesTheConfirmationItAsks(t *testing.T) {
	setupHermetic(t)
	routes := dessauRoutes()
	routes["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Body: `[{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"check\u001b[2J"}]}}]`}
	ghRoutedFake(t, routes)
	repo := dessauRepo(t)
	var asked []string

	if _, err := RemoteApply(repo, questionRecorder{&asked}, nil); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 {
		t.Fatalf("asked %d questions, want 1", len(asked))
	}
	if strings.ContainsRune(asked[0], 0x1b) {
		t.Errorf("the confirmation carries an escape sequence: %q", asked[0])
	}
}

// TestRemoteApplyRefusesToMergeIntoARulesetItCannotReadWhole pins that the
// merge fails closed: an existing ruleset whose answer abcd cannot read whole
// (rules or conditions malformed or missing, a bypass actor or a check list
// of the wrong shape), whose target is not the one abcd's ruleset of that
// name has, or whose excludes may still leave the target out, is not updated,
// because a merge over what was not read would drop it.
func TestRemoteApplyRefusesToMergeIntoARulesetItCannotReadWhole(t *testing.T) {
	cases := map[string][2]string{
		"rules not a list":       {`"rules":[`, `"rules":{"x":[`},
		"no conditions":          {`"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH","refs/heads/release/*"],"exclude":[]}},`, ``},
		"bypass not an object":   {`"bypass_actors":[]`, `"bypass_actors":[1]`},
		"checks not a list":      {`"required_status_checks":[{"context":"check","integration_id":15368},{"context":"extra"}]}`, `"required_status_checks":"check"}`},
		"another target":         {`"target":"branch"`, `"target":"tag"`},
		"an exclude still there": {`"exclude":[]`, `"exclude":["refs/heads/main"]`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			body := strictDefaultBranchRuleset
			if !strings.Contains(body, c[0]) {
				t.Fatalf("fixture has no %q", c[0])
			}
			body = strings.Replace(body, c[0], c[1], 1)
			if name == "rules not a list" {
				body = strings.TrimSuffix(strings.TrimSpace(body), "}") + "}}"
			}
			logPath, _ := ghRoutedFake(t, strictDefaultBranchRoutes(body))
			repo := dessauRepo(t)

			res, _ := RemoteApply(repo, confirmingPrompter{}, nil)
			if puts := callsTo(t, logPath, "PUT", "rulesets/5"); len(puts) != 0 {
				t.Errorf("the ruleset was rewritten from an answer abcd could not read whole:\n%s", strings.Join(puts, "\n"))
			}
			if !strings.Contains(strings.Join(res.Notes, "\n"), "nothing of it was changed") {
				t.Errorf("the refusal is not reported: %v", res.Notes)
			}
		})
	}
}
