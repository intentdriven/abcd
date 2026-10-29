package ahoy

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The settings apply (itd-2610290746077716, step 2): every finding the audit
// marked differs, written in the order GitHub requires, after one confirmation
// that names each of them. A finding in any other state is never written:
// advisory could break a repository that works today, owed is not offered,
// kept is the repository's choice, unknown would be a write over a guess. No
// request body names the repository's visibility.

// The rulesets abcd creates, found again by these names. abcd never edits a
// ruleset it did not create; GitHub adds the rules of every active ruleset up,
// so a person's own ruleset keeps working beside these.
const (
	rulesetDefaultBranch = "abcd: default branch"
	rulesetCodeOwner     = "abcd: code-owner review"
	rulesetReleaseTags   = "abcd: release tags"
)

// rulesetMirrorDir is where abcd mirrors the rulesets it applies, beside the
// repository-settings mirror.
const rulesetMirrorDir = ".abcd/work/rulesets"

// rulesetMirrorName maps an abcd ruleset to its mirror file.
var rulesetMirrorName = map[string]string{
	rulesetDefaultBranch: "abcd-default-branch.json",
	rulesetCodeOwner:     "abcd-code-owner-review.json",
	rulesetReleaseTags:   "abcd-release-tags.json",
}

// applied is what one apply run did: the change lines it landed, and the
// bodies of the rulesets it wrote, keyed by ruleset name, for the mirror.
type applied struct {
	changes  []string
	rulesets map[string]map[string]any
}

// applyDifferences writes every differs finding. It stops at the first failed
// step and returns why; what landed before it is in a.changes.
func applyDifferences(root, repo string, audit *SettingsAudit) (a applied, failure string) {
	a.rulesets = map[string]map[string]any{}
	find := map[string]Finding{}
	for _, f := range audit.Findings {
		find[f.Setting] = f
	}
	differs := func(s string) bool { return find[s].State == findingDiffers }
	landed := func(s string) { a.changes = append(a.changes, s+": "+find[s].Current+" -> "+find[s].Target) }
	fail := func(s string, err error) (applied, string) {
		return a, "could not set " + s + " on " + repo + ": " + errText(err)
	}

	// Ordered: GitHub refuses push protection while secret scanning is off, and
	// security updates while Dependabot alerts are off.
	for _, key := range []string{settingSecretScanning, settingPushProtection} {
		if differs(key) {
			if err := ghEnable(root, repo, key); err != nil {
				return fail(key, err)
			}
			landed(key)
		}
	}
	for _, step := range []struct{ setting, endpoint string }{
		{settingDependabotAlerts, "/vulnerability-alerts"},
		{settingDependabotFixes, "/automated-security-fixes"},
	} {
		if differs(step.setting) {
			if _, err := github.API(root, nil, "--method", "PUT", "repos/"+repo+step.endpoint); err != nil {
				return fail(step.setting, err)
			}
			landed(step.setting)
		}
	}

	// The repository object's flags in one PATCH, holding only the keys that
	// differ, so nothing else on the object is touched.
	patch := map[string]any{}
	var flags []string
	for _, f := range []struct {
		setting string
		want    bool
	}{{settingDeleteBranch, true}, {settingWiki, false}, {settingProjects, false}} {
		if differs(f.setting) {
			patch[f.setting] = f.want
			flags = append(flags, f.setting)
		}
	}
	if len(patch) > 0 {
		if err := ghWrite(root, "PATCH", "repos/"+repo, patch); err != nil {
			return fail(strings.Join(flags, ", "), err)
		}
		for _, s := range flags {
			landed(s)
		}
	}

	if failure := applyRulesets(root, repo, audit, find, &a); failure != "" {
		return a, failure
	}

	if differs(settingSHAPinning) {
		// Read back the policy at the moment of the write, so the PUT keeps
		// every field it does not drive exactly as it stands.
		out, err := github.API(root, nil, "repos/"+repo+"/actions/permissions")
		if err != nil {
			return fail(settingSHAPinning, err)
		}
		var cur map[string]any
		if json.Unmarshal(out, &cur) != nil || cur["enabled"] == nil || cur["allowed_actions"] == nil {
			return fail(settingSHAPinning, errors.New("the Actions policy could not be read back before the write"))
		}
		body := map[string]any{"enabled": cur["enabled"], "allowed_actions": cur["allowed_actions"], "sha_pinning_required": true}
		if err := ghWrite(root, "PUT", "repos/"+repo+"/actions/permissions", body); err != nil {
			return fail(settingSHAPinning, err)
		}
		landed(settingSHAPinning)
	}
	if differs(settingWorkflowToken) {
		body := map[string]any{"default_workflow_permissions": "read", "can_approve_pull_request_reviews": false}
		if err := ghWrite(root, "PUT", "repos/"+repo+"/actions/permissions/workflow", body); err != nil {
			return fail(settingWorkflowToken, err)
		}
		landed(settingWorkflowToken)
	}
	return a, ""
}

// applyRulesets creates or updates abcd's own rulesets for the branch and tag
// findings that differ.
func applyRulesets(root, repo string, audit *SettingsAudit, find map[string]Finding, a *applied) string {
	differs := func(s string) bool { return find[s].State == findingDiffers }
	kept := func(s string) bool { return find[s].State == findingKept }
	type want struct {
		name     string
		settings []string
		body     map[string]any
	}
	var wants []want
	if differs(settingBranchDeletion) || differs(settingBranchForcePush) || differs(settingRequiredChecks) {
		var rules []any
		var settings []string
		if !kept(settingBranchDeletion) {
			rules, settings = append(rules, map[string]any{"type": "deletion"}), append(settings, settingBranchDeletion)
		}
		if !kept(settingBranchForcePush) {
			rules, settings = append(rules, map[string]any{"type": "non_fast_forward"}), append(settings, settingBranchForcePush)
		}
		// Checks are written only when the audit found them to differ: an
		// advisory set (one that names a check no workflow reports, or drawn from
		// workflows that are not the default branch's) is never applied alongside
		// a branch protection.
		if differs(settingRequiredChecks) && len(audit.RequiredChecks) > 0 {
			contexts := make([]any, 0, len(audit.RequiredChecks))
			for _, c := range audit.RequiredChecks {
				contexts = append(contexts, map[string]any{"context": c})
			}
			rules = append(rules, map[string]any{"type": "required_status_checks", "parameters": map[string]any{
				"do_not_enforce_on_create":             false,
				"required_status_checks":               contexts,
				"strict_required_status_checks_policy": false,
			}})
			settings = append(settings, settingRequiredChecks)
		}
		wants = append(wants, want{rulesetDefaultBranch, settings, rulesetBody(rulesetDefaultBranch, "branch", "~DEFAULT_BRANCH", rules)})
	}
	if differs(settingCodeOwnerReview) {
		// allowed_merge_methods is left out on purpose: setting it would change
		// which merges the repository allows.
		rules := []any{map[string]any{"type": "pull_request", "parameters": map[string]any{
			"dismiss_stale_reviews_on_push":     false,
			"require_code_owner_review":         true,
			"require_last_push_approval":        false,
			"required_approving_review_count":   0,
			"required_review_thread_resolution": false,
		}}}
		wants = append(wants, want{rulesetCodeOwner, []string{settingCodeOwnerReview}, rulesetBody(rulesetCodeOwner, "branch", "~DEFAULT_BRANCH", rules)})
	}
	if differs(settingReleaseTags) {
		rules := []any{map[string]any{"type": "deletion"}, map[string]any{"type": "update"}, map[string]any{"type": "non_fast_forward"}}
		wants = append(wants, want{rulesetReleaseTags, []string{settingReleaseTags}, rulesetBody(rulesetReleaseTags, "tag", "refs/tags/v*", rules)})
	}
	if len(wants) == 0 {
		return ""
	}
	existing, err := ghRulesetIDs(root, repo)
	if err != nil {
		return "could not list " + repo + "'s rulesets, so abcd cannot tell its own from a person's: " + errText(err)
	}
	for _, w := range wants {
		var werr error
		if id, ok := existing[w.name]; ok {
			// Update in place by merging into what the ruleset holds now. A PUT
			// replaces a ruleset's rules wholesale, so a body built from the
			// baseline alone would drop any stricter rule, parameter, ref or
			// bypass someone gave a ruleset bearing abcd's name: abcd only adds.
			cur, err := ghRulesetForUpdate(root, repo, id)
			if err != nil {
				return "could not read abcd's ruleset " + w.name + " before updating it, and nothing of it was changed: " + errText(err)
			}
			merged, err := mergeRuleset(cur, w.body)
			if err != nil {
				return "abcd's ruleset " + w.name + " was not updated, and nothing of it was changed: " + err.Error()
			}
			w.body = merged
			werr = ghWrite(root, "PUT", "repos/"+repo+"/rulesets/"+id, w.body)
		} else {
			werr = ghWrite(root, "POST", "repos/"+repo+"/rulesets", w.body)
		}
		if werr != nil {
			return "could not apply the ruleset " + w.name + " on " + repo + ": " + errText(werr)
		}
		a.rulesets[w.name] = w.body
		for _, s := range w.settings {
			if differs(s) {
				a.changes = append(a.changes, s+": "+find[s].Current+" -> "+find[s].Target)
			}
		}
	}
	return ""
}

// rulesetBody is an abcd ruleset as it is sent and mirrored: the normalised
// shape of the rulesets mirror (name, target, enforcement, conditions, bypass
// actors, rules), with no bypass actor.
func rulesetBody(name, target, ref string, rules []any) map[string]any {
	return map[string]any{
		"name":          name,
		"target":        target,
		"enforcement":   "active",
		"conditions":    map[string]any{"ref_name": map[string]any{"include": []any{ref}, "exclude": []any{}}},
		"bypass_actors": []any{},
		"rules":         rules,
	}
}

// ghRulesetIDs lists the repository's own rulesets and returns abcd's, by name.
// Rulesets inherited from the organization are left out: they are not the
// repository's to update. A listing that fills the one page read may be cut
// short, so it is refused rather than read as the whole list: abcd could not
// tell whether its own ruleset is on a later page.
func ghRulesetIDs(root, repo string) (map[string]string, error) {
	out, err := github.API(root, nil, "repos/"+repo+"/rulesets?per_page=100&includes_parents=false")
	if err != nil {
		return nil, err
	}
	var list []struct {
		ID         json.Number `json:"id"`
		Name       string      `json:"name"`
		SourceType string      `json:"source_type"`
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.UseNumber()
	if dec.Decode(&list) != nil {
		return nil, errors.New("the ruleset list could not be read as JSON")
	}
	if len(list) >= 100 {
		return nil, errors.New("the repository holds 100 rulesets or more, more than the one page abcd reads, so it cannot tell whether its own are among them")
	}
	ids := map[string]string{}
	for _, rs := range list {
		if rs.SourceType != "" && rs.SourceType != "Repository" {
			continue
		}
		if _, ours := rulesetMirrorName[rs.Name]; ours && rulesetIDRe.MatchString(rs.ID.String()) {
			ids[rs.Name] = rs.ID.String()
		}
	}
	return ids, nil
}

// currentRuleset is what an existing ruleset holds that an update must keep:
// its target, its conditions, its rules as GitHub returned them, and its
// bypass actors reduced to the three fields the mirror keeps (an actor is
// recorded by type and id, never by a name or login).
type currentRuleset struct {
	target     string
	conditions map[string]any
	rules      []any
	bypass     []any
}

// ghRulesetForUpdate reads one ruleset before abcd updates it, and refuses an
// answer it cannot read whole: the merge keeps only what it read, so a part
// read as empty would be dropped by the PUT. An answer with no bypass_actors
// list is refused, never read as "no bypass": GitHub leaves the field out for
// a caller who may not see it.
func ghRulesetForUpdate(root, repo, id string) (currentRuleset, error) {
	out, err := github.API(root, nil, "repos/"+repo+"/rulesets/"+id)
	if err != nil {
		return currentRuleset{}, err
	}
	var doc map[string]any
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.UseNumber()
	if dec.Decode(&doc) != nil {
		return currentRuleset{}, errors.New("the ruleset could not be read as JSON")
	}
	actors, ok := doc["bypass_actors"].([]any)
	if !ok {
		return currentRuleset{}, errors.New("its answer carries no bypass_actors list, so abcd cannot keep the bypass it holds")
	}
	cur := currentRuleset{bypass: []any{}}
	cur.target, _ = doc["target"].(string)
	if cur.conditions, ok = doc["conditions"].(map[string]any); !ok {
		return currentRuleset{}, errors.New("its answer carries no conditions, so abcd cannot keep the refs it covers")
	}
	if refName, present := cur.conditions["ref_name"]; present {
		m, ok := refName.(map[string]any)
		if !ok || !refList(m["include"]) || !refList(m["exclude"]) {
			return currentRuleset{}, errors.New("its ref conditions are not lists of refs")
		}
	}
	if cur.rules, ok = doc["rules"].([]any); !ok {
		return currentRuleset{}, errors.New("its answer carries no list of rules, so abcd cannot keep them")
	}
	for _, r := range cur.rules {
		m, ok := r.(map[string]any)
		if !ok {
			return currentRuleset{}, errors.New("one of its rules is not an object")
		}
		if _, ok := m["type"].(string); !ok {
			return currentRuleset{}, errors.New("one of its rules names no type")
		}
		params, present := m["parameters"]
		if !present {
			continue
		}
		pm, ok := params.(map[string]any)
		if !ok {
			return currentRuleset{}, errors.New("the parameters of its " + m["type"].(string) + " rule are not an object")
		}
		if checks, present := pm["required_status_checks"]; present {
			list, ok := checks.([]any)
			if !ok {
				return currentRuleset{}, errors.New("its required checks are not a list")
			}
			for _, c := range list {
				if cm, ok := c.(map[string]any); !ok || cm["context"] == nil {
					return currentRuleset{}, errors.New("one of its required checks names no context")
				}
			}
		}
	}
	for _, a := range actors {
		m, ok := a.(map[string]any)
		if !ok {
			return currentRuleset{}, errors.New("one of its bypass actors is not an object")
		}
		cur.bypass = append(cur.bypass, map[string]any{
			"actor_id":    m["actor_id"],
			"actor_type":  m["actor_type"],
			"bypass_mode": m["bypass_mode"],
		})
	}
	return cur, nil
}

// refList reports whether v is absent or a list of refs (strings).
func refList(v any) bool {
	if v == nil {
		return true
	}
	_, err := stringList(v, "refs")
	return err == nil
}

// mergeRuleset is the body that updates an existing ruleset to abcd's
// baseline without loosening anything it holds. Every existing rule, with
// its parameters, and every bypass actor stays; a rule type abcd wants and
// the ruleset lacks is added; a required check abcd wants is added beside the
// ones already required, each keeping its source; a pull-request rule gains
// the code-owner review and keeps its other parameters; and the target ref is
// added to the refs the ruleset already covers. It refuses a ruleset whose
// target is not the one abcd's ruleset of that name has, and one that still
// excludes refs, since abcd cannot tell whether an exclude leaves its target
// out.
func mergeRuleset(cur currentRuleset, want map[string]any) (map[string]any, error) {
	if cur.target != want["target"] {
		return nil, errors.New("its target is " + strconv.Quote(cur.target) + ", not " + strconv.Quote(want["target"].(string)))
	}
	out := map[string]any{
		"name":          want["name"],
		"target":        want["target"],
		"enforcement":   "active",
		"bypass_actors": cur.bypass,
	}
	wantRef := want["conditions"].(map[string]any)["ref_name"].(map[string]any)["include"].([]any)
	conditions := mergeConditions(cur.conditions, wantRef)
	if exclude := conditions["ref_name"].(map[string]any)["exclude"].([]any); len(exclude) > 0 {
		return nil, errors.New("it excludes refs abcd cannot match against its target, so it may not cover it")
	}
	out["conditions"] = conditions
	rules := append([]any{}, cur.rules...)
	for _, wr := range want["rules"].([]any) {
		w := wr.(map[string]any)
		i := ruleIndex(rules, w["type"])
		if i < 0 {
			rules = append(rules, w)
			continue
		}
		have := rules[i].(map[string]any)
		params, _ := have["parameters"].(map[string]any)
		wantParams, _ := w["parameters"].(map[string]any)
		switch w["type"] {
		case "required_status_checks":
			if params == nil {
				have["parameters"] = wantParams
				continue
			}
			checks, _ := params["required_status_checks"].([]any)
			for _, c := range wantParams["required_status_checks"].([]any) {
				if !hasCheck(checks, c.(map[string]any)["context"]) {
					checks = append(checks, c)
				}
			}
			params["required_status_checks"] = checks
		case "pull_request":
			if params == nil {
				have["parameters"] = wantParams
				continue
			}
			params["require_code_owner_review"] = true
		}
	}
	out["rules"] = rules
	return out, nil
}

// mergeConditions adds refs to the refs a ruleset already includes, and takes
// them out of its excludes, keeping every other condition it has.
func mergeConditions(cur map[string]any, refs []any) map[string]any {
	out := map[string]any{}
	for k, v := range cur {
		out[k] = v
	}
	refName, _ := out["ref_name"].(map[string]any)
	include, _ := refName["include"].([]any)
	exclude, _ := refName["exclude"].([]any)
	for _, r := range refs {
		if !containsValue(include, r) {
			include = append(include, r)
		}
	}
	kept := []any{}
	for _, e := range exclude {
		if !containsValue(refs, e) {
			kept = append(kept, e)
		}
	}
	if include == nil {
		include = []any{}
	}
	out["ref_name"] = map[string]any{"include": include, "exclude": kept}
	return out
}

func ruleIndex(rules []any, typ any) int {
	for i, r := range rules {
		if m, ok := r.(map[string]any); ok && m["type"] == typ {
			return i
		}
	}
	return -1
}

func hasCheck(checks []any, context any) bool {
	for _, c := range checks {
		if m, ok := c.(map[string]any); ok && m["context"] == context {
			return true
		}
	}
	return false
}

func containsValue(list []any, v any) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ghWrite sends one JSON body to GitHub, on stdin, as ghEnable does.
func ghWrite(root, method, endpoint string, body map[string]any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = github.API(root, data, "--method", method, "-H", "Accept: application/vnd.github+json", endpoint, "--input", "-")
	return err
}

// mirrorRepoName is the repository as the mirror records it: owner/name for an
// organization's repository, and the name alone otherwise, because a person's
// owner segment is their login, and no file abcd writes names a person.
func mirrorRepoName(repo, ownerType string) string {
	if ownerType == "Organization" {
		return repo
	}
	_, name, _ := strings.Cut(repo, "/")
	return name
}

// managedSettings is the managed block of the settings mirror: every baseline
// setting abcd now holds the repository to, at its target. A setting kept,
// owed, advisory or unknown is not held, and is not in it.
func managedSettings(audit *SettingsAudit, a applied) map[string]any {
	landed := map[string]bool{}
	for _, c := range a.changes {
		s, _, _ := strings.Cut(c, ":")
		landed[s] = true
	}
	held := func(s string) bool {
		for _, f := range audit.Findings {
			if f.Setting == s {
				return f.State == findingOK || (f.State == findingDiffers && landed[s])
			}
		}
		return false
	}
	managed := map[string]any{}
	sec := map[string]any{}
	for _, key := range []string{settingSecretScanning, settingPushProtection, settingDependabotFixes} {
		if held(key) {
			sec[key] = map[string]string{"status": statusEnabled}
		}
	}
	if len(sec) > 0 {
		managed["security_and_analysis"] = sec
	}
	if held(settingDependabotAlerts) {
		managed[settingDependabotAlerts] = statusEnabled
	}
	repoFlags := map[string]any{}
	for _, f := range []struct {
		setting string
		want    bool
	}{{settingDeleteBranch, true}, {settingWiki, false}, {settingProjects, false}} {
		if held(f.setting) {
			repoFlags[f.setting] = f.want
		}
	}
	if len(repoFlags) > 0 {
		managed["repository"] = repoFlags
	}
	if held(settingSHAPinning) {
		managed[settingSHAPinning] = true
	}
	if held(settingWorkflowToken) {
		managed[settingWorkflowToken] = map[string]any{"default_workflow_permissions": "read", "can_approve_pull_request_reviews": false}
	}
	return managed
}

// owedSettings names the settings owed until the plan or the visibility
// offers them, so they apply once the repository is public.
func owedSettings(audit *SettingsAudit) []string {
	var out []string
	for _, f := range audit.Findings {
		if f.State == findingOwed {
			out = append(out, f.Setting)
		}
	}
	return out
}

// writeRulesetMirrors writes the mirror of each ruleset this run applied,
// content-keyed through an os.Root, and returns the paths it wrote.
func writeRulesetMirrors(root string, rulesets map[string]map[string]any) ([]string, error) {
	if len(rulesets) == 0 {
		return nil, nil
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err := r.MkdirAll(rulesetMirrorDir, 0o755); err != nil {
		return nil, err
	}
	var wrote []string
	for _, name := range []string{rulesetDefaultBranch, rulesetCodeOwner, rulesetReleaseTags} {
		body, ok := rulesets[name]
		if !ok {
			continue
		}
		data, err := json.MarshalIndent(body, "", "  ")
		if err != nil {
			return wrote, err
		}
		data = append(data, '\n')
		rel := path.Join(rulesetMirrorDir, rulesetMirrorName[name])
		if have, err := fsutil.ReadGuardedInRoot(r, rel, maxAhoyFileBytes); err == nil && bytes.Equal(have, data) {
			continue
		}
		if err := fsutil.WriteFileAtomicInRoot(r, rel, data, 0o644); err != nil {
			return wrote, err
		}
		wrote = append(wrote, rel)
	}
	return wrote, nil
}
