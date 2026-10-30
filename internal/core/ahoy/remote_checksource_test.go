package ahoy

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

const (
	sourceCommitSHA = "1111111111111111111111111111111111111111"
	sourcePRHeadSHA = "2222222222222222222222222222222222222222"
)

// run is one check run as GitHub's check-runs answer lists it.
func run(name string, appID int, slug string) string {
	return `{"name":"` + name + `","app":{"id":` + strconv.Itoa(appID) + `,"slug":"` + slug + `"}}`
}

// sourceRoutes serves the default branch's newest commit and the newest pull
// request's head, with the check runs and commit statuses reported on each.
// A nil list leaves that read unrouted, so it fails as GitHub's 404 does.
func sourceRoutes(r ghRoutes, commitRuns, headRuns, commitStatuses []string) ghRoutes {
	r["GET repos/"+auditRepo+"/commits?sha=main&per_page=3"] = ghRoute{Body: `[{"sha":"` + sourceCommitSHA + `"}]`}
	r["GET repos/"+auditRepo+"/pulls?state=all&base=main&sort=updated&direction=desc&per_page=3"] = ghRoute{Body: `[{"head":{"sha":"` + sourcePRHeadSHA + `"}}]`}
	runs := func(list []string) ghRoute {
		return ghRoute{Body: `{"total_count":` + strconv.Itoa(len(list)) + `,"check_runs":[` + strings.Join(list, ",") + `]}`}
	}
	statuses := func(list []string) ghRoute {
		var items []string
		for _, c := range list {
			items = append(items, `{"context":"`+c+`","state":"success"}`)
		}
		return ghRoute{Body: `{"total_count":` + strconv.Itoa(len(items)) + `,"statuses":[` + strings.Join(items, ",") + `]}`}
	}
	r["GET repos/"+auditRepo+"/commits/"+sourceCommitSHA+"/check-runs?per_page=100"] = runs(commitRuns)
	r["GET repos/"+auditRepo+"/commits/"+sourcePRHeadSHA+"/check-runs?per_page=100"] = runs(headRuns)
	r["GET repos/"+auditRepo+"/commits/"+sourceCommitSHA+"/status?per_page=100"] = statuses(commitStatuses)
	r["GET repos/"+auditRepo+"/commits/"+sourcePRHeadSHA+"/status?per_page=100"] = statuses(nil)
	return r
}

// configuredChecksRepo is dessau's repository with check, gitleaks and
// zizmor named as its required checks, which the dessau ruleset already
// requires from any source.
func configuredChecksRepo(t *testing.T) string {
	t.Helper()
	repo := dessauRepo(t)
	setConfigValue(t, repo, "github", "required_checks", []any{"check", "gitleaks", "zizmor"})
	return repo
}

func notesNaming(notes []string, parts ...string) bool {
	for _, n := range notes {
		all := true
		for _, p := range parts {
			all = all && strings.Contains(n, p)
		}
		if all {
			return true
		}
	}
	return false
}

// TestRemoteAuditPinsACheckOneAppReports: a required check that one app's
// check runs report on the recent commits is offered pinned to that app, by
// the id GitHub's answer names. A check required from any source then
// differs, because pinning it is a tightening the apply makes only on the yes.
func TestRemoteAuditPinsACheckOneAppReports(t *testing.T) {
	setupHermetic(t)
	routes := sourceRoutes(dessauRoutes(),
		[]string{run("check", 15368, "github-actions"), run("gitleaks", 15368, "github-actions")},
		[]string{run("check", 15368, "github-actions")}, nil)
	ghRoutedFake(t, routes)
	res, err := RemoteRead(configuredChecksRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	f := findingsBySetting(t, res)["required_status_checks"]
	if f.State != "differs" {
		t.Fatalf("required_status_checks %+v, want differs: both checks are required from any source", f)
	}
	if f.Target != "check (from github-actions), gitleaks (from github-actions), zizmor" {
		t.Errorf("target %q", f.Target)
	}
	for _, c := range []string{"check", "gitleaks"} {
		if app := res.Audit.RequiredCheckApps[c]; app.ID != 15368 {
			t.Errorf("%s pinned to %+v, want app 15368", c, app)
		}
	}
	if d := strings.Join(res.Audit.Differences(), "\n"); !strings.Contains(d, "check, gitleaks, zizmor -> check (from github-actions)") {
		t.Errorf("the differences the apply asks about do not show the pin: %q", d)
	}
}

// TestRemoteAuditLeavesAnUnclearSourceOnAnySource: a check two apps report,
// or one a commit status reports, keeps any source and is named; a check no
// recent run reported is left unpinned and named; neither is a difference.
func TestRemoteAuditLeavesAnUnclearSourceOnAnySource(t *testing.T) {
	for name, tc := range map[string]struct {
		commitRuns, headRuns, statuses []string
		note                           []string
	}{
		"two apps": {
			commitRuns: []string{run("check", 15368, "github-actions"), run("gitleaks", 15368, "github-actions")},
			headRuns:   []string{run("check", 99, "other-ci")},
			note:       []string{"check", "more than one source"},
		},
		"a commit status": {
			commitRuns: []string{run("check", 15368, "github-actions"), run("gitleaks", 15368, "github-actions")},
			statuses:   []string{"check"},
			note:       []string{"check", "more than one source"},
		},
		"no recent run": {
			commitRuns: []string{run("gitleaks", 15368, "github-actions")},
			note:       []string{"check", "no recent run"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			ghRoutedFake(t, sourceRoutes(dessauRoutes(), tc.commitRuns, tc.headRuns, tc.statuses))
			res, err := RemoteRead(configuredChecksRepo(t))
			if err != nil {
				t.Fatal(err)
			}
			if _, pinned := res.Audit.RequiredCheckApps["check"]; pinned {
				t.Errorf("check pinned to %+v, want it left on any source", res.Audit.RequiredCheckApps["check"])
			}
			if res.Audit.RequiredCheckApps["gitleaks"].ID != 15368 {
				t.Errorf("gitleaks not pinned: %+v", res.Audit.RequiredCheckApps)
			}
			if !notesNaming(res.Audit.Notes, tc.note...) {
				t.Errorf("no note naming %v in %q", tc.note, res.Audit.Notes)
			}
			f := findingsBySetting(t, res)["required_status_checks"]
			if f.Target != "check, gitleaks (from github-actions), zizmor" {
				t.Errorf("target %q", f.Target)
			}
		})
	}
}

// TestRemoteAuditPinsNothingFromAReadItCannotFinish: a check-run listing cut
// short, or a read that fails, could hide a second app that reports a check,
// so nothing is pinned and the audit says why.
func TestRemoteAuditPinsNothingFromAReadItCannotFinish(t *testing.T) {
	for name, edit := range map[string]func(ghRoutes){
		"cut short": func(r ghRoutes) {
			r["GET repos/"+auditRepo+"/commits/"+sourcePRHeadSHA+"/check-runs?per_page=100"] = ghRoute{Body: `{"total_count":150,"check_runs":[` + run("check", 15368, "github-actions") + `]}`}
		},
		"failed": func(r ghRoutes) {
			delete(r, "GET repos/"+auditRepo+"/commits/"+sourcePRHeadSHA+"/status?per_page=100")
		},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			routes := sourceRoutes(dessauRoutes(),
				[]string{run("check", 15368, "github-actions"), run("gitleaks", 15368, "github-actions")}, nil, nil)
			edit(routes)
			ghRoutedFake(t, routes)
			res, err := RemoteRead(configuredChecksRepo(t))
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Audit.RequiredCheckApps) != 0 {
				t.Errorf("pinned %+v from a read that did not finish", res.Audit.RequiredCheckApps)
			}
			if !notesNaming(res.Audit.Notes, "any source") {
				t.Errorf("no note saying the checks stay on any source: %q", res.Audit.Notes)
			}
			if f := findingsBySetting(t, res)["required_status_checks"]; f.State != "ok" {
				t.Errorf("required_status_checks %+v, want ok", f)
			}
		})
	}
}

// TestRemoteAuditKeepsAnExistingPin: a check already required from the app
// that reports it is ok, and one required from another app is left as it is
// and named, never re-pinned.
func TestRemoteAuditKeepsAnExistingPin(t *testing.T) {
	setupHermetic(t)
	routes := sourceRoutes(dessauRoutes(),
		[]string{run("check", 15368, "github-actions"), run("gitleaks", 15368, "github-actions")}, nil, nil)
	routes["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Body: `[{"type":"deletion"},{"type":"non_fast_forward"},
{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"check","integration_id":15368},{"context":"gitleaks","integration_id":42},{"context":"zizmor"}]}}]`}
	ghRoutedFake(t, routes)
	res, err := RemoteRead(configuredChecksRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	f := findingsBySetting(t, res)["required_status_checks"]
	if f.State != "ok" {
		t.Errorf("required_status_checks %+v, want ok", f)
	}
	if f.Current != "check (from app 15368), gitleaks (from app 42), zizmor" {
		t.Errorf("current %q", f.Current)
	}
	if !notesNaming(res.Audit.Notes, "gitleaks", "app 42", "left as it is") {
		t.Errorf("no note naming gitleaks's other pin: %q", res.Audit.Notes)
	}
}

// TestRemoteApplyPinsTheChecksItRequires: a new ruleset names each pinned
// check's app; an update of abcd's own ruleset gives an unpinned check its
// app and leaves a check pinned elsewhere as it is.
func TestRemoteApplyPinsTheChecksItRequires(t *testing.T) {
	checksOf := func(t *testing.T, body string) map[string]any {
		t.Helper()
		var doc struct {
			Rules []struct {
				Type       string `json:"type"`
				Parameters struct {
					Checks []map[string]any `json:"required_status_checks"`
				} `json:"parameters"`
			} `json:"rules"`
		}
		if err := json.Unmarshal([]byte(body), &doc); err != nil {
			t.Fatal(err)
		}
		out := map[string]any{}
		for _, r := range doc.Rules {
			for _, c := range r.Parameters.Checks {
				out[c["context"].(string)] = c["integration_id"]
			}
		}
		return out
	}
	runs := []string{run("check", 15368, "github-actions"), run("gitleaks", 15368, "github-actions")}

	t.Run("new ruleset", func(t *testing.T) {
		setupHermetic(t)
		routes := sourceRoutes(dessauRoutes(), runs, nil, nil)
		routes["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Body: `[]`}
		routes["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[]`}
		logPath, _ := ghRoutedFake(t, routes)
		if _, err := RemoteApply(configuredChecksRepo(t), confirmingPrompter{}, nil); err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		for _, c := range callsTo(t, logPath, "POST", "rulesets") {
			if strings.Contains(c, "abcd: default branch") {
				got = checksOf(t, stdinOf(c))
			}
		}
		for _, c := range []string{"check", "gitleaks"} {
			if got[c] != float64(15368) {
				t.Errorf("%s written with integration_id %v, want 15368 (all: %v)", c, got[c], got)
			}
		}
	})

	t.Run("update", func(t *testing.T) {
		setupHermetic(t)
		ruleset := `{"id":5,"name":"abcd: default branch","target":"branch","enforcement":"active",
"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],
"rules":[{"type":"deletion"},{"type":"non_fast_forward"},
{"type":"required_status_checks","parameters":{"do_not_enforce_on_create":false,"strict_required_status_checks_policy":true,
"required_status_checks":[{"context":"check"},{"context":"gitleaks","integration_id":42}]}}]}`
		routes := sourceRoutes(dessauRoutes(), runs, nil, nil)
		routes["GET repos/"+auditRepo+"/rules/branches/main"] = ghRoute{Body: `[{"type":"deletion"},{"type":"non_fast_forward"},
{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"check"},{"context":"gitleaks","integration_id":42}]}}]`}
		routes["GET repos/"+auditRepo+"/rulesets?per_page=100&includes_parents=false"] = ghRoute{Body: `[{"id":5,"name":"abcd: default branch","source_type":"Repository"}]`}
		routes["GET repos/"+auditRepo+"/rulesets/5"] = ghRoute{Body: ruleset}
		logPath, _ := ghRoutedFake(t, routes)
		if _, err := RemoteApply(configuredChecksRepo(t), confirmingPrompter{}, nil); err != nil {
			t.Fatal(err)
		}
		puts := callsTo(t, logPath, "PUT", "rulesets/5")
		if len(puts) != 1 {
			t.Fatalf("%d updates of abcd's ruleset, want 1", len(puts))
		}
		got := checksOf(t, stdinOf(puts[0]))
		if got["check"] != float64(15368) {
			t.Errorf("check updated with integration_id %v, want 15368", got["check"])
		}
		if got["gitleaks"] != float64(42) {
			t.Errorf("gitleaks updated with integration_id %v, want its own 42 kept", got["gitleaks"])
		}
	})
}
