package ahoy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The settings audit (itd-2610290746077716): every setting of the baseline the
// intent's Decisions name, read from GitHub and the tree, and compared. The
// baseline is GitHub's own hardening guidance and the OpenSSF Scorecard checks
// (Branch-Protection, Code-Review, Pinned-Dependencies, Token-Permissions,
// Vulnerabilities, Dependency-Update-Tool), and matches abcd's own repository's
// rulesets mirror. The read writes nothing anywhere.

// The states a Finding can be in.
const (
	// findingOK is a setting already at its target.
	findingOK = "ok"
	// findingDiffers is a setting the apply can set to its target.
	findingDiffers = "differs"
	// findingAdvisory is a difference abcd reports and never applies, with the
	// reason: applying it could break a repository that works today, or it is
	// not abcd's to change.
	findingAdvisory = "advisory"
	// findingOwed is a setting the plan or the visibility does not offer yet.
	findingOwed = "owed"
	// findingKept is a setting the repository asked abcd to leave alone.
	findingKept = "kept"
	// findingUnknown is a setting abcd could not read. It is never reported as
	// off and never applied: a write over an unknown state is a guess.
	findingUnknown = "unknown"
)

// The baseline's setting keys, in report order. They are the keys
// `github.keep` in .abcd/config.json accepts.
const (
	settingVisibility       = "visibility"
	settingSecretScanning   = secretScanningKey
	settingPushProtection   = pushProtectionKey
	settingDependabotAlerts = "dependabot_alerts"
	settingDependabotFixes  = "dependabot_security_updates"
	settingDeleteBranch     = "delete_branch_on_merge"
	settingWiki             = "has_wiki"
	settingProjects         = "has_projects"
	settingBranchDeletion   = "default_branch_deletion"
	settingBranchForcePush  = "default_branch_force_push"
	settingRequiredChecks   = "required_status_checks"
	settingCodeOwnerReview  = "code_owner_review"
	settingReleaseTags      = "release_tags"
	settingCodeowners       = "codeowners"
	settingSHAPinning       = "actions_sha_pinning"
	settingWorkflowToken    = "workflow_token"
	settingOrgActions       = "org_actions_policy"
)

// baselineSettings is every key, in the order the audit reports them.
var baselineSettings = []string{
	settingVisibility, settingSecretScanning, settingPushProtection,
	settingDependabotAlerts, settingDependabotFixes,
	settingDeleteBranch, settingWiki, settingProjects,
	settingBranchDeletion, settingBranchForcePush, settingRequiredChecks, settingCodeOwnerReview,
	settingReleaseTags, settingCodeowners,
	settingSHAPinning, settingWorkflowToken, settingOrgActions,
}

// Finding is one baseline setting as the audit found it.
type Finding struct {
	Setting string `json:"setting"`
	Current string `json:"current"`
	Target  string `json:"target"`
	// State is one of ok | differs | advisory | owed | kept | unknown.
	State string `json:"state"`
	// Note says why the state is what it is, whenever it is not plain ok.
	Note string `json:"note,omitempty"`
}

// SettingsAudit is the whole audit of one repository.
type SettingsAudit struct {
	// Visibility is what GitHub reports; DeclaredVisibility is repo.visibility
	// in .abcd/config.json. Either is empty when it could not be read.
	Visibility         string `json:"visibility,omitempty"`
	DeclaredVisibility string `json:"declared_visibility,omitempty"`
	// Plan is the owner's GitHub plan, empty when the sign-in cannot see it.
	Plan          string `json:"plan,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
	// RequiredChecks is the check names the baseline requires on the default
	// branch: github.required_checks when set, else those every pull request's
	// workflows report.
	RequiredChecks []string  `json:"required_checks,omitempty"`
	Findings       []Finding `json:"findings"`
	// Notes say what the audit left out of its reading, and why.
	Notes []string `json:"notes,omitempty"`
}

// Differences returns the findings the apply can act on, as
// "setting: current -> target" lines.
func (a *SettingsAudit) Differences() []string {
	if a == nil {
		return nil
	}
	var out []string
	for _, f := range a.Findings {
		if f.State == findingDiffers {
			out = append(out, f.Setting+": "+f.Current+" -> "+f.Target)
		}
	}
	return out
}

// githubOverrides is the `github` block of .abcd/config.json: version-
// controlled and reviewed like every other key of it.
type githubOverrides struct {
	keep           map[string]bool
	requiredChecks []string
	requiredSet    bool
}

// requiredCheckNameRe is the allowlist for a configured check name: the
// scaffold's own, so a name can only ever be inert text in a request body.
var requiredCheckNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()/+-]{0,99}$`)

// readGitHubOverrides validates the `github` block. Every fault refuses: a key
// abcd does not read, a keep entry the baseline does not name, a malformed
// check name. A typo that silently kept nothing would read as consent.
func readGitHubOverrides(cfg map[string]any) (githubOverrides, error) {
	ov := githubOverrides{keep: map[string]bool{}}
	raw, ok := cfg["github"]
	if !ok {
		return ov, nil
	}
	block, ok := raw.(map[string]any)
	if !ok {
		return ov, errors.New(`.abcd/config.json's "github" is not an object`)
	}
	for k, v := range block {
		switch k {
		case "keep":
			items, err := stringList(v, "github.keep")
			if err != nil {
				return ov, err
			}
			for _, s := range items {
				if !inSet(s, baselineSettings) {
					return ov, fmt.Errorf("github.keep names %q, which is not a baseline setting (the settings are: %s)",
						s, strings.Join(baselineSettings, ", "))
				}
				ov.keep[s] = true
			}
		case "required_checks":
			items, err := stringList(v, "github.required_checks")
			if err != nil {
				return ov, err
			}
			for _, s := range items {
				if !requiredCheckNameRe.MatchString(s) {
					return ov, fmt.Errorf("github.required_checks holds %q, which is not a plain check name", s)
				}
			}
			ov.requiredChecks, ov.requiredSet = dedupeSorted(items), true
		default:
			return ov, fmt.Errorf("github.%s is not a key abcd reads (it reads github.keep and github.required_checks)", k)
		}
	}
	return ov, nil
}

// stringList reads a JSON array of strings.
func stringList(v any, key string) ([]string, error) {
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s is not a list", key)
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s holds an entry that is not a string", key)
		}
		out = append(out, s)
	}
	return out, nil
}

func dedupeSorted(in []string) []string {
	set := map[string]bool{}
	for _, s := range in {
		set[s] = true
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// repoObject is the part of `GET /repos/{owner}/{repo}` the remote verbs read.
// A pointer is a field the answer may not carry: "false" and "not said" are
// different facts.
type repoObject struct {
	Visibility          string `json:"visibility"`
	DefaultBranch       string `json:"default_branch"`
	HasWiki             *bool  `json:"has_wiki"`
	HasProjects         *bool  `json:"has_projects"`
	DeleteBranchOnMerge *bool  `json:"delete_branch_on_merge"`
	AllowMergeCommit    *bool  `json:"allow_merge_commit"`
	AllowSquashMerge    *bool  `json:"allow_squash_merge"`
	AllowRebaseMerge    *bool  `json:"allow_rebase_merge"`
	Owner               struct {
		Type string `json:"type"`
	} `json:"owner"`
	// Permissions is what the sign-in may do here; GitHub sends it to an
	// authenticated caller.
	Permissions struct {
		Admin *bool `json:"admin"`
	} `json:"permissions"`
	SecurityAndAnalysis map[string]struct {
		Status string `json:"status"`
	} `json:"security_and_analysis"`
}

// private reports whether GitHub said the repository is not public. An
// unread visibility is not private: the existing toggles' safe direction is to
// write, and a refusal from GitHub then says so.
func (o repoObject) private() bool { return o.Visibility == "private" || o.Visibility == "internal" }

// branchNameRe is the branch names put into an API path: plain segments
// joined by slashes, no `..`, nothing to escape.
var branchNameRe = regexp.MustCompile(`^[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)*$`)

// rulesetIDRe is a ruleset id as it goes into an API path.
var rulesetIDRe = regexp.MustCompile(`^[0-9]{1,15}$`)

// maxTagRulesets bounds how many tag rulesets are read in detail.
const maxTagRulesets = 20

// releaseTagProbe is the ref a release-tag ruleset must cover.
const releaseTagProbe = "refs/tags/v1.0.0"

// auditInput is what the audit needs that the gates already settled.
type auditInput struct {
	root, repo, owner string
	obj               repoObject
	ov                githubOverrides
	optedOut          bool
	declared          string
}

// auditSettings reads and compares the whole baseline. Every read goes
// through github.API, under its timeout and output bound; a read that fails
// leaves its settings unknown, owed, or (for the toggles abcd always managed)
// as before.
func auditSettings(in auditInput) *SettingsAudit {
	a := &SettingsAudit{Visibility: in.obj.Visibility, DeclaredVisibility: in.declared, DefaultBranch: in.obj.DefaultBranch}
	add := func(f Finding) {
		if in.ov.keep[f.Setting] {
			f.State, f.Note = findingKept, "kept as it is by github.keep in .abcd/config.json"
		}
		a.Findings = append(a.Findings, f)
	}
	priv := in.obj.private()
	owedNote := "not offered for a private repository on this plan; owed until the repository is public"

	add(visibilityFinding(in.obj.Visibility, in.declared))
	a.Plan = ghPlan(in.root, in.owner, in.obj.Owner.Type)

	for _, key := range []string{settingSecretScanning, settingPushProtection} {
		status := in.obj.SecurityAndAnalysis[key].Status
		f := Finding{Setting: key, Current: displayStatus(status), Target: statusEnabled}
		switch {
		case in.optedOut:
			f.State, f.Note = findingKept, "kept as it is: scan.native_secret_scanning is false in .abcd/config.json"
		case status == statusEnabled:
			f.State = findingOK
		case status == "" && priv:
			f.State, f.Note = findingOwed, owedNote
		case status == "":
			// GitHub shows security_and_analysis only to an admin, so an answer
			// without it says nothing about the toggle, and an unknown is
			// never applied.
			f.State, f.Note = findingUnknown, "GitHub's answer did not carry it; it shows these toggles only to an admin of the repository"
		default:
			f.State = findingDiffers
		}
		if in.optedOut {
			// The opt-out's own reason is the one worth reading.
			a.Findings = append(a.Findings, f)
			continue
		}
		add(f)
	}

	add(ghDependabotAlerts(in.root, in.repo, in.obj.Permissions.Admin))
	add(statusFinding(settingDependabotFixes, in.obj.SecurityAndAnalysis[settingDependabotFixes].Status))
	add(boolFinding(settingDeleteBranch, in.obj.DeleteBranchOnMerge, true))
	add(boolFinding(settingWiki, in.obj.HasWiki, false))
	add(boolFinding(settingProjects, in.obj.HasProjects, false))

	rules, checks, notes := ghBranchRules(in, priv)
	a.RequiredChecks, a.Notes = checks, append(a.Notes, notes...)
	for _, f := range rules {
		add(f)
	}
	add(ghReleaseTags(in.root, in.repo, priv))
	add(codeownersFinding(in.root))

	facts := readWorkflowFacts(in.root)
	add(ghSHAPinning(in.root, in.repo, facts))
	add(ghWorkflowToken(in.root, in.repo, facts))
	if in.obj.Owner.Type == "Organization" {
		add(ghOrgActions(in.root, in.owner))
	}
	holdWorkflowSettings(a, in)
	return a
}

// holdWorkflowSettings turns each setting drawn from this checkout's
// workflows from differs to advisory when those workflows are not the default
// branch's on GitHub (workflowDrift). Checks the repository declares in
// github.required_checks are its own statement, not drawn from the workflows,
// and are not held.
func holdWorkflowSettings(a *SettingsAudit, in auditInput) {
	drift := ""
	checked := false
	for i, f := range a.Findings {
		drawn := f.Setting == settingSHAPinning || f.Setting == settingWorkflowToken ||
			(f.Setting == settingRequiredChecks && !in.ov.requiredSet)
		if !drawn || f.State != findingDiffers {
			continue
		}
		if !checked {
			drift, checked = workflowDrift(in.root, in.repo, in.obj.DefaultBranch), true
		}
		if drift == "" {
			return
		}
		a.Findings[i].State, a.Findings[i].Note = findingAdvisory, drift
	}
}

// visibilityFinding compares the declared and the real visibility. A mismatch
// is advisory: no command of abcd's changes a repository's visibility.
func visibilityFinding(real, declared string) Finding {
	f := Finding{Setting: settingVisibility, Current: orUnknown(real), Target: orUnknown(declared)}
	switch {
	case real == "":
		f.State, f.Note = findingUnknown, "GitHub's answer did not say the repository's visibility"
	case declared == "":
		f.State, f.Note = findingUnknown, "repo.visibility is not declared in .abcd/config.json"
	case real == declared:
		f.State = findingOK
	default:
		f.State, f.Note = findingAdvisory, "the repository is "+real+" but .abcd/config.json declares it "+declared+
			"; abcd never changes a repository's visibility: change it on GitHub, or correct repo.visibility"
	}
	return f
}

// statusFinding judges a security_and_analysis toggle abcd drives to enabled.
func statusFinding(setting, status string) Finding {
	f := Finding{Setting: setting, Current: displayStatus(status), Target: statusEnabled}
	switch status {
	case statusEnabled:
		f.State = findingOK
	case "":
		f.State, f.Note = findingUnknown, "GitHub's answer did not carry this setting"
	default:
		f.State = findingDiffers
	}
	return f
}

// boolFinding judges a repository-object flag against its target.
func boolFinding(setting string, have *bool, want bool) Finding {
	f := Finding{Setting: setting, Current: "unknown", Target: strconv.FormatBool(want)}
	switch {
	case have == nil:
		f.State, f.Note = findingUnknown, "GitHub's answer did not carry this setting"
	case *have == want:
		f.Current, f.State = strconv.FormatBool(*have), findingOK
	default:
		f.Current, f.State = strconv.FormatBool(*have), findingDiffers
	}
	return f
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// isHTTP reports whether err is gh's report of one of the given statuses.
func isHTTP(err error, codes ...int) bool {
	if err == nil {
		return false
	}
	for _, c := range codes {
		if strings.Contains(err.Error(), "HTTP "+strconv.Itoa(c)) {
			return true
		}
	}
	return false
}

// planRefusal reports a GitHub refusal that names the plan: the 403 a feature
// the plan does not offer for a private repository answers with.
func planRefusal(err error) bool {
	return isHTTP(err, 403, 404) && strings.Contains(strings.ToLower(err.Error()), "upgrade")
}

// readFailure words a failed read. A 403 or 404 on a settings endpoint is the
// sign-in's reach, not the setting's state, so it says what rights are missing.
func readFailure(what string, err error) string {
	if isHTTP(err, 403, 404) {
		return "the sign-in cannot read " + what + ": it needs admin rights on the repository"
	}
	return "could not read " + what + ": " + errText(err)
}

// ghPlan is the owner's plan name, or "" when the sign-in cannot see it: an
// organization's plan is shown to its members, and a person's only to that
// person.
func ghPlan(root, owner, ownerType string) string {
	var path string
	switch ownerType {
	case "Organization":
		path = "orgs/" + owner
	case "User":
		path = "user"
	default:
		return ""
	}
	out, err := github.API(root, nil, path)
	if err != nil {
		return ""
	}
	var doc struct {
		Login string `json:"login"`
		Plan  struct {
			Name string `json:"name"`
		} `json:"plan"`
	}
	if json.Unmarshal(out, &doc) != nil {
		return ""
	}
	if ownerType == "User" && !strings.EqualFold(doc.Login, owner) {
		return ""
	}
	return doc.Plan.Name
}

// ghDependabotAlerts reads whether Dependabot alerts are on: GitHub answers
// 204 when they are and 404 when they are not.
func ghDependabotAlerts(root, repo string, admin *bool) Finding {
	f := Finding{Setting: settingDependabotAlerts, Target: statusEnabled}
	_, err := github.API(root, nil, "repos/"+repo+"/vulnerability-alerts")
	switch {
	case err == nil:
		f.Current, f.State = statusEnabled, findingOK
	case isHTTP(err, 404) && admin != nil && !*admin:
		// The endpoint answers only an admin of the repository, so a
		// non-admin's 404 says nothing about whether alerts are on.
		f.Current, f.State, f.Note = "unknown", findingUnknown,
			"GitHub answers whether Dependabot alerts are on only to an admin of the repository, and this sign-in is not one"
	case isHTTP(err, 404):
		f.Current, f.State = "disabled", findingDiffers
	default:
		f.Current, f.State, f.Note = "unknown", findingUnknown, readFailure("whether Dependabot alerts are on", err)
	}
	return f
}

// effectiveRule is one rule of `GET /repos/{repo}/rules/branches/{branch}`,
// which merges every active ruleset that targets the branch.
type effectiveRule struct {
	Type       string          `json:"type"`
	Parameters json.RawMessage `json:"parameters"`
}

// ghBranchRules reads the default branch's effective rules and judges the four
// branch settings, with the required checks the baseline asks for.
func ghBranchRules(in auditInput, priv bool) (findings []Finding, checks, notes []string) {
	names := []string{settingBranchDeletion, settingBranchForcePush, settingRequiredChecks, settingCodeOwnerReview}
	all := func(state, note string) []Finding {
		out := make([]Finding, 0, len(names))
		for _, n := range names {
			out = append(out, Finding{Setting: n, Current: "unknown", Target: branchTarget(n), State: state, Note: note})
		}
		return out
	}
	branch := in.obj.DefaultBranch
	if !branchNameRe.MatchString(branch) || strings.Contains(branch, "..") {
		return all(findingUnknown, "GitHub's answer did not name a default branch abcd can put into a request"), nil, nil
	}
	out, err := github.API(in.root, nil, "repos/"+in.repo+"/rules/branches/"+branch)
	if err != nil {
		if planRefusal(err) && priv {
			return all(findingOwed, "rulesets are not offered for a private repository on this plan; owed until the repository is public"), nil, nil
		}
		return all(findingUnknown, readFailure("the default branch's rules", err)), nil, nil
	}
	var rules []effectiveRule
	if json.Unmarshal(out, &rules) != nil {
		return all(findingUnknown, "the default branch's rules could not be read as JSON"), nil, nil
	}
	var deletion, forcePush, codeOwner, queue bool
	current := map[string]bool{}
	for _, r := range rules {
		switch r.Type {
		case "deletion":
			deletion = true
		case "non_fast_forward":
			forcePush = true
		case "merge_queue":
			queue = true
		case "pull_request":
			var p struct {
				RequireCodeOwnerReview bool `json:"require_code_owner_review"`
			}
			if json.Unmarshal(r.Parameters, &p) == nil && p.RequireCodeOwnerReview {
				codeOwner = true
			}
		case "required_status_checks":
			var p struct {
				Checks []struct {
					Context string `json:"context"`
				} `json:"required_status_checks"`
			}
			if json.Unmarshal(r.Parameters, &p) == nil {
				for _, c := range p.Checks {
					current[c.Context] = true
				}
			}
		}
	}
	protect := func(setting string, on bool) Finding {
		f := Finding{Setting: setting, Current: "allowed", Target: branchTarget(setting), State: findingDiffers}
		if on {
			f.Current, f.State = "blocked", findingOK
		}
		return f
	}
	findings = append(findings, protect(settingBranchDeletion, deletion), protect(settingBranchForcePush, forcePush))

	if in.ov.requiredSet {
		checks = in.ov.requiredChecks
	} else {
		checks, notes = scaffold.RequiredChecks(in.root, branch, queue)
	}
	findings = append(findings, requiredChecksFinding(checks, current))

	review := Finding{Setting: settingCodeOwnerReview, Current: "not required", Target: branchTarget(settingCodeOwnerReview), State: findingDiffers}
	if codeOwner {
		review.Current, review.State = "required", findingOK
	}
	return append(findings, review), checks, notes
}

// branchTarget words each branch setting's target.
func branchTarget(setting string) string {
	switch setting {
	case settingBranchDeletion, settingBranchForcePush:
		return "blocked"
	case settingRequiredChecks:
		return "the checks every pull request's workflows report"
	case settingCodeOwnerReview:
		return "required"
	}
	return ""
}

// requiredChecksFinding compares the required contexts with the checks the
// baseline asks for. A required context no workflow is known to report is
// named, because a pull request waits on it until it reports; abcd does not
// remove it, since it is not abcd's to remove.
func requiredChecksFinding(want []string, current map[string]bool) Finding {
	have := make([]string, 0, len(current))
	for c := range current {
		have = append(have, c)
	}
	sort.Strings(have)
	f := Finding{Setting: settingRequiredChecks, Current: listOrNone(have), Target: listOrNone(want)}
	wantSet := map[string]bool{}
	var missing []string
	for _, c := range want {
		wantSet[c] = true
		if !current[c] {
			missing = append(missing, c)
		}
	}
	var stale []string
	for _, c := range have {
		if !wantSet[c] {
			stale = append(stale, c)
		}
	}
	staleNote := ""
	if len(stale) > 0 {
		staleNote = "required but not reported by any workflow abcd can read: " + strings.Join(stale, ", ") +
			"; a pull request waits on a required check until it reports, unless a matrix or expression-named job reports it"
	}
	switch {
	case len(want) == 0 && len(have) == 0:
		f.State, f.Note = findingAdvisory, "no workflow reports a check on every pull request, so none is required "+
			"(a starter workflow is iss-2610290748300390)"
	case len(missing) > 0:
		f.State, f.Note = findingDiffers, staleNote
	case len(stale) > 0:
		f.State, f.Note = findingAdvisory, staleNote
	default:
		f.State = findingOK
	}
	return f
}

func listOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

// ghReleaseTags judges whether the `v*` release tags are protected from
// deletion, update and force-move, by the active tag rulesets that cover them.
func ghReleaseTags(root, repo string, priv bool) Finding {
	f := Finding{Setting: settingReleaseTags, Current: "unprotected", Target: "no deletion, update or force-move of v* tags"}
	out, err := github.API(root, nil, "repos/"+repo+"/rulesets?targets=tag&includes_parents=true")
	if err != nil {
		if planRefusal(err) && priv {
			f.Current, f.State, f.Note = "unknown", findingOwed, "rulesets are not offered for a private repository on this plan; owed until the repository is public"
			return f
		}
		f.Current, f.State, f.Note = "unknown", findingUnknown, readFailure("the tag rulesets", err)
		return f
	}
	var list []struct {
		ID          json.Number `json:"id"`
		Target      string      `json:"target"`
		Enforcement string      `json:"enforcement"`
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	dec.UseNumber()
	if dec.Decode(&list) != nil {
		f.Current, f.State, f.Note = "unknown", findingUnknown, "the tag rulesets could not be read as JSON"
		return f
	}
	have := map[string]bool{}
	read := 0
	for _, rs := range list {
		if rs.Target != "tag" || rs.Enforcement != "active" || !rulesetIDRe.MatchString(rs.ID.String()) {
			continue
		}
		if read++; read > maxTagRulesets {
			break
		}
		detail, err := github.API(root, nil, "repos/"+repo+"/rulesets/"+rs.ID.String()+"?includes_parents=true")
		if err != nil {
			f.Current, f.State, f.Note = "unknown", findingUnknown, readFailure("a tag ruleset", err)
			return f
		}
		var doc struct {
			Conditions struct {
				RefName struct {
					Include []string `json:"include"`
					Exclude []string `json:"exclude"`
				} `json:"ref_name"`
			} `json:"conditions"`
			Rules []struct {
				Type string `json:"type"`
			} `json:"rules"`
		}
		if json.Unmarshal(detail, &doc) != nil || !refCovered(doc.Conditions.RefName.Include, doc.Conditions.RefName.Exclude, releaseTagProbe) {
			continue
		}
		for _, r := range doc.Rules {
			have[r.Type] = true
		}
	}
	var missing []string
	for _, r := range []string{"deletion", "update", "non_fast_forward"} {
		if !have[r] {
			missing = append(missing, r)
		}
	}
	switch {
	case len(missing) == 0:
		f.Current, f.State = "protected", findingOK
	case len(missing) < 3:
		f.Current, f.State = "unprotected from "+strings.Join(missing, ", "), findingDiffers
	default:
		f.State = findingDiffers
	}
	return f
}

// refCovered reports whether a ruleset's ref conditions cover ref.
func refCovered(include, exclude []string, ref string) bool {
	match := func(pats []string) bool {
		for _, p := range pats {
			if p == "~ALL" || refGlob(p).MatchString(ref) {
				return true
			}
		}
		return false
	}
	return match(include) && !match(exclude)
}

// refGlob compiles a ruleset ref pattern: `**` matches any run of characters,
// `*` any run but `/`.
func refGlob(p string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString("[^/]*")
		case p[i] == '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// codeownersLocations are where GitHub looks for CODEOWNERS, in its order.
var codeownersLocations = []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}

// publishSurface is the paths a CODEOWNERS file must cover when they exist:
// the files that decide what CI runs and what a release publishes. A trailing
// slash is a directory.
var publishSurface = []string{
	".github/", ".githooks/", ".abcd/config.json", ".abcd/rules.json",
	"Makefile", "go.mod", "go.sum", "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
	"pyproject.toml", "Cargo.toml", "Cargo.lock",
}

// codeownersFinding judges the tree's CODEOWNERS against the publish surface.
// It is advisory either way: abcd does not write the file until who it must
// name is defined (iss-2610290755587946).
func codeownersFinding(root string) Finding {
	var surface []string
	for _, p := range publishSurface {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(p, "/")))); err == nil && info.IsDir() == strings.HasSuffix(p, "/") {
			surface = append(surface, p)
		}
	}
	f := Finding{Setting: settingCodeowners, Target: "covers " + listOrNone(surface)}
	if len(surface) == 0 {
		f.Current, f.State = "nothing to cover", findingOK
		return f
	}
	var data []byte
	for _, loc := range codeownersLocations {
		if b, err := fsutil.ReadGuarded(filepath.Join(root, filepath.FromSlash(loc)), maxAhoyFileBytes); err == nil {
			data = b
			break
		}
	}
	const later = "; abcd does not write it, because whether a managed repository keeps one is not yet decided (iss-2610290755587946)"
	if data == nil {
		f.Current, f.State, f.Note = "absent", findingAdvisory, "add a CODEOWNERS file covering these paths"+later
		return f
	}
	var uncovered []string
	for _, p := range surface {
		probe := p
		if strings.HasSuffix(p, "/") {
			probe = p + "x"
		}
		if !codeownersCovers(string(data), probe) {
			uncovered = append(uncovered, p)
		}
	}
	if len(uncovered) == 0 {
		f.Current, f.State = "covers the publish surface", findingOK
		return f
	}
	f.Current, f.State, f.Note = "does not cover "+strings.Join(uncovered, ", "), findingAdvisory, "add owners for these paths"+later
	return f
}

// codeownersCovers reports whether the CODEOWNERS text gives path at least one
// owner. The last matching line wins, as GitHub reads it, and a matching line
// with no owner removes ownership. The pattern syntax is read as the common
// gitignore subset: `*` and `**`, a leading `/` anchoring at the root, a
// trailing `/` naming a directory.
func codeownersCovers(text, path string) bool {
	owned := false
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || !codeownersMatch(fields[0], path) {
			continue
		}
		owned = len(fields) > 1
	}
	return owned
}

// codeownersMatch matches one CODEOWNERS pattern against a repo-relative path.
func codeownersMatch(pattern, path string) bool {
	anchored := strings.HasPrefix(pattern, "/")
	pat := strings.TrimPrefix(pattern, "/")
	if pat == "*" || pat == "**" || pat == "**/*" {
		return true
	}
	dir := strings.HasSuffix(pat, "/")
	pat = strings.TrimSuffix(strings.TrimSuffix(pat, "**"), "/")
	if pat == "" {
		return true
	}
	re := refGlob(pat)
	segs := strings.Split(path, "/")
	// A pattern matches the path itself or any directory above it, so a
	// directory pattern covers what it holds.
	for i := 1; i <= len(segs); i++ {
		if dir && i == len(segs) {
			break
		}
		prefix := strings.Join(segs[:i], "/")
		if re.MatchString(prefix) {
			return true
		}
		// An unanchored pattern with no slash matches at any depth.
		if !anchored && !strings.Contains(pat, "/") && re.MatchString(segs[i-1]) {
			return true
		}
	}
	return false
}

// ghSHAPinning judges the repository's Actions policy. An allow-list or a SHA
// requirement is the target; requiring SHA pinning is a difference the apply
// may make only when every workflow already pins every action, because
// otherwise the policy would stop a working CI.
func ghSHAPinning(root, repo string, facts workflowFacts) Finding {
	f := Finding{Setting: settingSHAPinning, Current: "unknown", Target: "actions pinned to a commit SHA, or an allow-list"}
	out, err := github.API(root, nil, "repos/"+repo+"/actions/permissions")
	if err != nil {
		f.State, f.Note = findingUnknown, readFailure("the repository's Actions policy", err)
		return f
	}
	var doc struct {
		Enabled            *bool  `json:"enabled"`
		AllowedActions     string `json:"allowed_actions"`
		SHAPinningRequired *bool  `json:"sha_pinning_required"`
	}
	if json.Unmarshal(out, &doc) != nil || doc.Enabled == nil {
		f.State, f.Note = findingUnknown, "GitHub's answer did not carry the Actions policy"
		return f
	}
	switch {
	case !*doc.Enabled:
		f.Current, f.State = "Actions off", findingOK
	case doc.SHAPinningRequired != nil && *doc.SHAPinningRequired:
		f.Current, f.State = "SHA pinning required", findingOK
	case doc.AllowedActions == "local_only" || doc.AllowedActions == "selected":
		f.Current, f.State = "allow-list ("+doc.AllowedActions+")", findingOK
	case doc.SHAPinningRequired == nil:
		f.Current, f.State, f.Note = "any action", findingUnknown, "GitHub's answer did not say whether SHA pinning is required"
	case len(facts.Unread) > 0:
		f.Current, f.State = "any action, by any ref", findingAdvisory
		f.Note = "abcd could not read " + strings.Join(facts.Unread, ", ") + ", so it cannot tell whether every " +
			"action is pinned; the apply does not require pinning over steps it has not seen"
	case facts.Unpinned == 0:
		f.Current, f.State = "any action, by any ref", findingDiffers
	default:
		uses := "uses of an action are"
		if facts.Unpinned == 1 {
			uses = "use of an action is"
		}
		f.Current, f.State = "any action, by any ref", findingAdvisory
		f.Note = strconv.Itoa(facts.Unpinned) + " " + uses + " not pinned to a commit SHA in .github/workflows; " +
			"pin them, and the apply can then require it"
	}
	return f
}

// ghWorkflowToken judges the default GITHUB_TOKEN: read-only, and unable to
// approve pull requests. Making a write default read-only is a difference the
// apply may make only when every workflow declares its own permissions.
func ghWorkflowToken(root, repo string, facts workflowFacts) Finding {
	f := Finding{Setting: settingWorkflowToken, Current: "unknown", Target: "read, cannot approve pull requests"}
	out, err := github.API(root, nil, "repos/"+repo+"/actions/permissions/workflow")
	if err != nil {
		f.State, f.Note = findingUnknown, readFailure("the workflow token's default permissions", err)
		return f
	}
	var doc struct {
		Default    string `json:"default_workflow_permissions"`
		CanApprove *bool  `json:"can_approve_pull_request_reviews"`
	}
	if json.Unmarshal(out, &doc) != nil || doc.Default == "" || doc.CanApprove == nil {
		f.State, f.Note = findingUnknown, "GitHub's answer did not carry the workflow token's permissions"
		return f
	}
	approves := "cannot approve pull requests"
	if *doc.CanApprove {
		approves = "can approve pull requests"
	}
	f.Current = doc.Default + ", " + approves
	switch {
	case doc.Default == "read" && !*doc.CanApprove:
		f.State = findingOK
	case doc.Default != "read" && len(facts.Unread) > 0:
		f.State, f.Note = findingAdvisory, "write is the default and abcd could not read "+strings.Join(facts.Unread, ", ")+
			", so a read-only default could stop steps it has not seen"
	case doc.Default == "read" || len(facts.NoPermissions) == 0:
		f.State = findingDiffers
	default:
		f.State, f.Note = findingAdvisory, "write is the default and these workflows declare no permissions, so a "+
			"read-only default could stop them: "+strings.Join(facts.NoPermissions, ", ")+
			"; declare permissions in each, and the apply can then set it"
	}
	return f
}

// ghOrgActions reads the organization's Actions policy. It is reported and
// never written: an organization's settings are an org owner's to change.
func ghOrgActions(root, owner string) Finding {
	f := Finding{Setting: settingOrgActions, Current: "unknown", Target: "actions pinned to a commit SHA, or an allow-list"}
	out, err := github.API(root, nil, "orgs/"+owner+"/actions/permissions")
	if err != nil {
		f.State = findingUnknown
		f.Note = "could not read the organization's Actions policy: " + errText(err)
		if isHTTP(err, 403, 404) {
			f.Note = "reading the organization's Actions policy needs an org owner's sign-in with the admin:org scope " +
				"(gh auth refresh -h github.com -s admin:org); abcd never writes it"
		}
		return f
	}
	var doc struct {
		AllowedActions     string `json:"allowed_actions"`
		SHAPinningRequired *bool  `json:"sha_pinning_required"`
	}
	if json.Unmarshal(out, &doc) != nil || doc.AllowedActions == "" {
		f.State, f.Note = findingUnknown, "GitHub's answer did not carry the organization's Actions policy"
		return f
	}
	switch {
	case doc.SHAPinningRequired != nil && *doc.SHAPinningRequired:
		f.Current, f.State = "SHA pinning required", findingOK
	case doc.AllowedActions != "all":
		f.Current, f.State = "allow-list ("+doc.AllowedActions+")", findingOK
	default:
		f.Current, f.State, f.Note = "any action, by any ref", findingAdvisory, "an org owner can require SHA pinning or an allow-list; abcd never writes the organization's settings"
	}
	return f
}
