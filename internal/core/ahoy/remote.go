package ahoy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// RepoSettingsMirrorRelPath is the committed mirror of the repo-object settings
// abcd manages, repo-relative and slash-separated. It sits beside the branch-ruleset
// mirror, which covers RULESETS only: the security toggles live behind the separate
// repository-object API, so a settings change had no source of truth in the tree at
// all and every claim about it rested on a web console (iss-2608270512210664).
const RepoSettingsMirrorRelPath = ".abcd/work/rulesets/repo-settings.json"

// The two toggles this verb manages, as GitHub names them in the
// security_and_analysis object.
const (
	secretScanningKey = "secret_scanning"
	pushProtectionKey = "secret_scanning_push_protection"
)

// statusEnabled is the only value either toggle is driven to.
const statusEnabled = "enabled"

// RemoteSecurity is the pair of GitHub-native secret-scanning toggles, as observed
// or as desired. A toggle GitHub did not report reads "unknown", never "disabled":
// the two demand opposite responses, and reporting an unread state as off would
// send an operator to enable something that may already be on.
type RemoteSecurity struct {
	SecretScanning string `json:"secret_scanning"`
	PushProtection string `json:"secret_scanning_push_protection"`
}

// RemoteMergeHygiene is the repo-object settings that gate MERGE behaviour. abcd
// mirrors them. Of them it sets only delete_branch_on_merge, as part of the
// settings baseline and on a yes (itd-2610290746077716); the merge methods
// encode a maintainer's workflow, not a security posture, and a verb that drove
// them would be changing how the project merges on the strength of a default
// nobody chose.
//
// They are here because they had no source of truth in the tree at all
// (iss-2608270512210664): the branch-ruleset mirror beside this one covers
// RULESETS, and these live behind the separate repository-object API — so every
// claim a contribution document made about them rested on a web console. Each is a
// pointer, omitted when the response does not carry it, because "false" and "the
// API did not say" are different facts.
//
// ForkPRContributorApproval is the repository's fork pull-request approval
// policy: which outside contributors' pull-request workflows wait for the
// technical facilitator's approval before they run
// (first_time_contributors_new_to_github, first_time_contributors or
// all_external_contributors). A required check is
// matched by its name, so which workflows run on a pull request decides what
// may report that name; it is recorded for the same reason as the rest, and
// omitted where the repository has no such policy (the API answers 404). A
// read that fails otherwise is said in a note, and an apply then keeps the
// policy the mirror already records, so a failed request never reads as a
// changed policy.
type RemoteMergeHygiene struct {
	DeleteBranchOnMerge       *bool  `json:"delete_branch_on_merge,omitempty"`
	AllowMergeCommit          *bool  `json:"allow_merge_commit,omitempty"`
	AllowSquashMerge          *bool  `json:"allow_squash_merge,omitempty"`
	AllowRebaseMerge          *bool  `json:"allow_rebase_merge,omitempty"`
	ForkPRContributorApproval string `json:"fork_pr_contributor_approval,omitempty"`
}

// RemoteResult is the outcome of a remote config read or apply.
type RemoteResult struct {
	// Repo is the owner/name the verb resolved from this repo's origin remote, and
	// the ONLY repository it acts on. It is reported so a caller can see which
	// repository a write reached before believing the rest of the result.
	Repo string `json:"repo"`
	// Status is one of: already_up_to_date | clean | pending | aborted | refused.
	// A repo that opted out of native scanning has its two toggles reported as
	// kept (itd-2610290746077716) and the rest of its baseline audited. "aborted" is a confirmation the caller declined; "refused" is a gate
	// abcd itself closed. They are reported apart because only one of them is a
	// decision the caller can revisit by answering differently.
	Status string `json:"status"`
	// Observed is the state GitHub reported before anything was changed.
	Observed RemoteSecurity `json:"observed"`
	// Merge is the merge-hygiene snapshot read in the same request: mirrored, never
	// driven.
	Merge RemoteMergeHygiene `json:"merge_hygiene"`
	// Desired is the state abcd holds this repo to.
	Desired RemoteSecurity `json:"desired"`
	// Changes are the toggles an apply changed — or, from the read, would change.
	Changes []string `json:"changes,omitempty"`
	// Writes are the tree paths this run wrote.
	Writes []string `json:"writes,omitempty"`
	// Notes carry what abcd deliberately did not do, and why. A refusal that shows
	// up only as an unchanged toggle reads as a silent failure, so the reason
	// travels with the result.
	Notes []string `json:"notes,omitempty"`
	// Audit is the whole settings baseline as the read found it
	// (itd-2610290746077716); nil when a gate refused before anything was read.
	Audit *SettingsAudit `json:"audit,omitempty"`
}

// nativeScanningOptedOut reports an explicit `false` at scan.native_secret_scanning
// in this repo's config. Nil-is-unset, following ScanDeep: enable-by-default is the
// intended posture for a managed repo (both toggles are free on public
// repositories), so only a deliberate false opts out. A config that cannot be read
// is NOT an opt-out — but it is not consent either, which is why the caller refuses
// on it rather than proceeding.
func nativeScanningOptedOut(cwd string) (optedOut, answerable bool) {
	cfg, err := readConfig(cwd)
	if err != nil || cfg == nil {
		// ABSENT is not consent, and neither is unreadable. readConfig reports an
		// absent file as (nil, nil), and treating that as "no opt-out recorded" would
		// make a missing file authorise a remote write — the exact direction a consent
		// check must never fail in. Every other abcd verb may treat an absent config as
		// unset; this one holds a remote write behind it.
		return false, false
	}
	v, ok := boolVal(subMap(cfg, "scan"), "native_secret_scanning")
	return ok && !v, true
}

// worktreeRoot resolves the WORKING-TREE root for cwd. Every question this verb
// asks must be answered about the same repository, and they are not naturally: git
// resolves a repository by searching UPWARD, so `Detect` and the origin read find
// the enclosing repo from any subdirectory, while a config read rooted at cwd finds
// nothing there. Anchoring all of them here is what stops a run from a subdirectory
// resolving its identity from the repo and its CONSENT from an empty directory —
// and what keeps the os.Root the mirror is written through pointed at the tier the
// mirror belongs in rather than at a stray `.abcd` two levels down.
func worktreeRoot(cwd string) (string, bool) {
	top, err := gitutil.Toplevel(cwd)
	if err != nil {
		return "", false
	}
	return top, true
}

// resolveGitHubRepo returns the owner/name this repo's origin remote points at, or
// an error naming what could not be trusted. It is the answer to "which repository
// may this verb write to", so every unclear case is an error rather than a guess.
//
// Only github.com is accepted. The API this verb speaks is GitHub's, and sending a
// request derived from some other forge's URL would at best fail and at worst
// address a same-named repository on GitHub that the maintainer has nothing to do
// with.
func resolveGitHubRepo(cwd string) (string, error) {
	url := originURL(cwd)
	if url == "" {
		return "", errors.New("this repo has no origin remote, so there is no repository to act on " +
			"(abcd never picks one for you: `git remote add origin <url>` first)")
	}
	owner, name, err := github.ParseRemote(url)
	if err != nil {
		return "", err
	}
	return owner + "/" + name, nil
}

// RemoteRead reports the managed repo's GitHub-native secret-scanning state and
// what an apply would change. It writes NOTHING — not to the remote, not to the
// tree — so the bare invocation of the verb it backs holds abcd's convention that
// looking is never acting.
func RemoteRead(cwd string) (RemoteResult, error) {
	res, prep, done := remotePrepare(cwd)
	if done {
		return res, nil
	}
	abs := prep.root
	observed, merge, obj, forkErr, err := ghSecurityState(abs, res.Repo)
	if err != nil {
		res = refuseRemote(res, "could not read "+res.Repo+"'s security settings, so nothing is known about what would change: "+errText(err))
		// The read installs nothing (looking is never acting); it names the
		// verb that offers the install.
		var missing *tools.MissingError
		if errors.As(err, &missing) && missing.Explanation.Tool == "gh" {
			res.Notes = append(res.Notes, "abcd ahoy remote apply offers to install it, and runs the step only on a yes typed at a terminal")
		}
		return res, nil
	}
	res.Observed, res.Merge = observed, merge
	if forkErr != nil {
		res.Notes = append(res.Notes, "the fork pull-request approval policy could not be read, so it is not reported: "+errText(forkErr))
	}
	in := prep.audit
	in.repo, in.obj = res.Repo, obj
	res.Audit = auditSettings(in)
	res.Changes = res.Audit.Differences()
	res.Status = "already_up_to_date"
	if len(res.Changes) > 0 {
		res.Status = "pending"
	}
	return res, nil
}

// RemoteApply enables GitHub-native secret scanning and secret-scanning push
// protection on the managed repo this checkout points at, and mirrors the desired
// state into the tree so a later verify reads the same intent (itd-153).
//
// It answers to adr-44 and invariant 10: a remote write happens only through a
// dedicated verb the user invokes AND CONFIRMS. Four gates stand before any change
// leaves the machine — the folder must be a repo abcd manages, the repository must
// be the one this checkout's own origin names, the config must not carry the
// opt-out, and the caller must say yes to the specific toggles named — and every one
// of them refuses or aborts rather than guesses. A nil prompter is the RefusingPrompter,
// so a caller that forgot the seam gets no write by omission.
//
// The READ comes first and needs no ceremony (adr-44 rule 1: reading remote state is
// the default), which is also what lets the confirmation name exactly what would
// change. Then secret scanning, then push protection: GitHub refuses push protection
// on a repo whose secret scanning is off.
func RemoteApply(cwd string, p Prompter, confirmTool tools.Confirm) (RemoteResult, error) {
	if p == nil {
		p = RefusingPrompter{}
	}
	res, prep, done := remotePrepare(cwd)
	if done {
		return res, nil
	}
	abs := prep.root
	// The gh offer comes after the gates, so a verb that would refuse anyway
	// never asks to install anything, and before the read, which needs gh.
	offer, ok := OfferGH(abs, confirmTool)
	if !ok {
		res = refuseRemote(res, offer[0])
		res.Notes = append(res.Notes, offer[1:]...)
		return res, nil
	}
	res.Notes = append(res.Notes, offer...)
	observed, merge, obj, forkErr, err := ghSecurityState(abs, res.Repo)
	if err != nil {
		return refuseRemote(res, "could not read "+res.Repo+"'s security settings; nothing was changed, because a write over an unknown state is a guess: "+errText(err)), nil
	}
	if forkErr != nil {
		// The mirror records what was seen, so a policy left unread keeps the
		// one it already records rather than reading as removed.
		merge.ForkPRContributorApproval = mirroredForkPRApproval(abs)
		kept := "it records none"
		if merge.ForkPRContributorApproval != "" {
			kept = "it keeps " + merge.ForkPRContributorApproval
		}
		res.Notes = append(res.Notes, "the fork pull-request approval policy could not be read, so the mirror is not changed for it ("+kept+"): "+errText(forkErr))
	}
	res.Observed, res.Merge = observed, merge
	in := prep.audit
	in.repo, in.obj = res.Repo, obj
	audit := auditSettings(in)
	res.Audit = audit

	// The confirmation, asked ONLY when a remote write would actually follow: a
	// prompt over a change that is not going to happen teaches the answerer to say
	// yes without reading, which is exactly the reflex this gate exists to avoid.
	// It names every setting the apply would change, and nothing else is written.
	if pending := audit.Differences(); len(pending) > 0 {
		if !p.Confirm("Change GitHub settings on " + res.Repo + "? (" + strings.Join(pending, "; ") + ")") {
			res.Status = "aborted"
			res.Notes = append(res.Notes, "the change to "+res.Repo+" was declined, so nothing was written on GitHub "+
				"and no desired state was recorded")
			return res, nil
		}
	}

	// ORDERED, and a failed step stops the sequence (applyDifferences): GitHub
	// refuses push protection on a repository whose secret scanning is off, so the
	// reverse order enables nothing and reports a failure the operator cannot act
	// on.
	ran, failure := applyDifferences(abs, res.Repo, audit)
	res.Changes = ran.changes
	if failure != "" {
		return refuseRemote(res, failure+"; the remaining steps were not attempted and no desired state was recorded"+
			landedBeforeRefusal(res.Changes)), nil
	}

	// The mirrors are written only once every step this run needed RETURNED
	// SUCCESS — never over a partial apply, whose failed step returns above. They
	// are not a re-read: the claim they record is "abcd drove these settings and
	// GitHub accepted", which is what a later verify diffs the live state against,
	// and a confirming GET here would only move the same trust one request along.
	// The observed block records what GitHub holds after the run, so a setting
	// this run changed is recorded at its new value, not as it was read before.
	for _, c := range ran.changes {
		if strings.HasPrefix(c, settingDeleteBranch+":") {
			on := true
			merge.DeleteBranchOnMerge = &on
			res.Merge = merge
		}
	}
	rulesetWrites, err := writeRulesetMirrors(abs, ran.rulesets)
	res.Writes = append(res.Writes, rulesetWrites...)
	if err != nil {
		res.Status = "refused"
		res.Notes = append(res.Notes, "the remote state is correct but a ruleset mirror could not be written ("+
			rulesetMirrorDir+"): "+errText(err)+"; nothing in the tree records it"+landedBeforeRefusal(res.Changes))
		return res, nil
	}
	wrote, err := writeRepoSettingsMirror(abs, mirrorRepoName(res.Repo, obj.Owner.Type), managedSettings(audit, ran), owedSettings(audit), merge)
	if err != nil {
		res.Status = "refused"
		res.Notes = append(res.Notes, "the remote state is correct but the desired-state mirror could not be written ("+
			RepoSettingsMirrorRelPath+"): "+errText(err)+"; nothing in the tree records it"+landedBeforeRefusal(res.Changes))
		return res, nil
	}
	if wrote {
		res.Writes = append(res.Writes, RepoSettingsMirrorRelPath)
	}
	res.Status = "already_up_to_date"
	if len(res.Changes) > 0 || len(res.Writes) > 0 {
		res.Status = "clean"
	}
	return res, nil
}

// OfferGH is the explain-then-install mode for gh at a remote write (itd-63
// criterion 2; the product thinker's DQ3 ruling, 2026-09-29: offer to install
// gh on an explicit yes). With gh on PATH it says nothing. Otherwise it hands
// the registry's explanation to confirm, which the front door answers yes only
// for a person at a terminal, and runs the registry's step only on that yes.
//
// ok is true only when gh was installed AND verified; the notes then say what
// ran. Every other outcome (a no, no one asked, CI, a failed or unverified
// step) is ok=false with the reason first, then the explanation with the exact
// command, so the caller refuses loudly and reaches no gh call on a half-done
// install.
func OfferGH(guard string, confirm tools.Confirm) (notes []string, ok bool) {
	if onPath(guard, "gh") {
		return nil, true
	}
	res := newToolInstaller(guard).Install("gh", tools.GitHubSettings, confirm)
	if res.Ready && !res.Ran {
		return nil, true
	}
	if res.Ready {
		return []string{res.Summary() + "; the install does not sign gh in, so if GitHub refuses the request, run gh auth login"}, true
	}
	notes = []string{"the GitHub CLI (gh) is not on PATH: " + res.Summary()}
	if !res.Ran {
		notes = append(notes, tools.Explain("gh", tools.GitHubSettings).Lines()...)
	}
	return notes, false
}

// remotePrep is what the gates settled, for the verb that passed them.
type remotePrep struct {
	// root is the working-tree root every question was answered about.
	root string
	// audit carries the gates' answers the settings audit reads: the opt-out,
	// the github overrides and the declared visibility.
	audit auditInput
}

// remotePrepare runs the gates both verbs share and returns done=true when one
// of them settled the outcome. The gates are the adr-44 boundary: after this
// returns done=false, and only then, may a request leave the machine. The
// native-scanning opt-out is a gate no longer: it keeps the two toggles it
// names, and the caller decides what that means for it.
func remotePrepare(cwd string) (res RemoteResult, prep remotePrep, done bool) {
	res = RemoteResult{Desired: RemoteSecurity{SecretScanning: statusEnabled, PushProtection: statusEnabled}}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return refuseRemote(res, "could not resolve this directory: "+errText(err)), prep, true
	}
	// Anchor EVERYTHING that follows at the working-tree root, before the first
	// question is asked. Run from a subdirectory, git answers the identity questions
	// about the enclosing repository while a cwd-rooted config read answers the
	// consent question about an empty directory — so the verb would resolve WHICH
	// repository to write from one place and WHETHER it may from another.
	top, ok := worktreeRoot(abs)
	if !ok {
		return refuseRemote(res, "could not resolve this repository's working-tree root, so abcd cannot tell "+
			"which repository it would be acting on"), prep, true
	}
	abs = top
	det, err := Detect(abs)
	if err != nil {
		return refuseRemote(res, "could not classify this folder, so abcd will not act on a repository it cannot identify: "+errText(err)), prep, true
	}
	if rerr := det.RepoIdentity.RootErr(); det.RootSHA == "" && rerr != nil {
		// git not answering is not a folder outside abcd: say which it was,
		// and still make no remote write (iss-2610051434156079).
		return refuseRemote(res, "git gave no root commit for this repository, so abcd cannot tell whether it manages it: "+errText(rerr)), prep, true
	}
	if det.FolderKind != ManagedRepo {
		return refuseRemote(res, "this is not a repo abcd manages ("+string(det.FolderKind)+
			"); a remote write is never made on a folder that did not adopt abcd (run `abcd ahoy install` first)"), prep, true
	}
	optedOut, answerable := nativeScanningOptedOut(abs)
	if !answerable {
		return refuseRemote(res, "this repo's .abcd/config.json is absent or cannot be read, so the opt-out cannot "+
			"be checked — and a config that does not answer is not consent"), prep, true
	}
	cfg, err := readConfig(abs)
	if err != nil {
		return refuseRemote(res, "this repo's .abcd/config.json cannot be read: "+errText(err)), prep, true
	}
	ov, err := readGitHubOverrides(cfg)
	if err != nil {
		return refuseRemote(res, "this repo's .abcd/config.json has a github block abcd will not guess at: "+errText(err)), prep, true
	}
	repo, err := resolveGitHubRepo(abs)
	if err != nil {
		return refuseRemote(res, "could not resolve which GitHub repository this checkout belongs to: "+errText(err)), prep, true
	}
	res.Repo = repo
	declared, _ := stringVal(subMap(cfg, "repo"), "visibility")
	owner, _, _ := strings.Cut(repo, "/")
	prep.root = abs
	prep.audit = auditInput{root: abs, owner: owner, ov: ov, optedOut: optedOut, declared: declared}
	return res, prep, false
}

// refuseRemote records a loud refusal on the result. Every failure path in this
// file goes through it, so "abcd did not do this" is never expressed as an absent
// change a caller has to notice.
func refuseRemote(res RemoteResult, reason string) RemoteResult {
	res.Status = "refused"
	res.Notes = append(res.Notes, reason)
	return res
}

// landedBeforeRefusal words the toggles this run already changed on GitHub, for
// a refusal reached after one of them landed (iss-2610072347230409). A refusal
// that said only what failed would read as "the forge is untouched" when it is
// not, and the person would re-run or report against a state that has moved.
// Empty when nothing landed, so a refusal before the first toggle reads as before.
func landedBeforeRefusal(changes []string) string {
	if len(changes) == 0 {
		return ""
	}
	return "; this run had already landed on GitHub, and left in place: " + strings.Join(changes, "; ")
}

// displayStatus words a toggle GitHub did not report. "unknown" and "disabled" are
// different facts, and a report that flattened them would claim knowledge it has not
// got.
func displayStatus(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// ghSecurityState reads the repository object and returns its two toggles. A field
// the response does not carry stays empty, which the apply treats as "not enabled"
// and therefore writes — the safe direction: the write is idempotent on GitHub's
// side, whereas assuming enabled would leave a repo unprotected while reporting it
// covered.
//
// The fork pull-request approval policy is a second request, so it can fail
// on its own: forkErr says why it was left unread, and the rest stands.
func ghSecurityState(cwd, repo string) (sec RemoteSecurity, merge RemoteMergeHygiene, obj repoObject, forkErr, err error) {
	out, err := github.API(cwd, nil, "-H", "Accept: application/vnd.github+json", "repos/"+repo)
	if err != nil {
		return RemoteSecurity{}, RemoteMergeHygiene{}, repoObject{}, nil, err
	}
	if err := json.Unmarshal(out, &obj); err != nil {
		return RemoteSecurity{}, RemoteMergeHygiene{}, repoObject{}, nil, fmt.Errorf("the API response could not be read as JSON")
	}
	policy, forkErr := ghForkPRApproval(cwd, repo)
	return RemoteSecurity{
			SecretScanning: obj.SecurityAndAnalysis[secretScanningKey].Status,
			PushProtection: obj.SecurityAndAnalysis[pushProtectionKey].Status,
		}, RemoteMergeHygiene{
			DeleteBranchOnMerge:       obj.DeleteBranchOnMerge,
			AllowMergeCommit:          obj.AllowMergeCommit,
			AllowSquashMerge:          obj.AllowSquashMerge,
			AllowRebaseMerge:          obj.AllowRebaseMerge,
			ForkPRContributorApproval: policy,
		}, obj, forkErr, nil
}

// ghForkPRApproval reads the repository's fork pull-request approval policy
// from the Actions permissions API. A 404 is a repository with no such policy,
// "" with no error; any other failure, or an answer naming no policy, is an
// error, so an unread policy is never recorded as an absent one.
func ghForkPRApproval(cwd, repo string) (string, error) {
	out, err := github.API(cwd, nil, "-H", "Accept: application/vnd.github+json",
		"repos/"+repo+"/actions/permissions/fork-pr-contributor-approval")
	if err != nil {
		if strings.Contains(err.Error(), "HTTP 404") {
			return "", nil
		}
		return "", err
	}
	var doc struct {
		ApprovalPolicy string `json:"approval_policy"`
	}
	if json.Unmarshal(out, &doc) != nil {
		return "", errors.New("the API response could not be read as JSON")
	}
	if doc.ApprovalPolicy == "" {
		return "", errors.New("the API answered without naming a policy")
	}
	return doc.ApprovalPolicy, nil
}

// mirroredForkPRApproval is the fork pull-request approval policy the
// committed mirror records, or "" when it records none or cannot be read.
func mirroredForkPRApproval(cwd string) string {
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return ""
	}
	defer root.Close()
	have, err := fsutil.ReadGuardedInRoot(root, RepoSettingsMirrorRelPath, maxAhoyFileBytes)
	if err != nil {
		return ""
	}
	var doc struct {
		Observed RemoteMergeHygiene `json:"observed"`
	}
	if json.Unmarshal(have, &doc) != nil {
		return ""
	}
	return doc.Observed.ForkPRContributorApproval
}

// ghEnable PATCHes one toggle to enabled. The body goes in on STDIN as JSON rather
// than through `-f key[sub][sub]=value`: the bracket form's nesting rules have
// varied across gh releases, and a body that silently flattened would send a
// different request from the one this code reads as sending.
func ghEnable(cwd, repo, key string) error {
	body, err := json.Marshal(map[string]any{
		"security_and_analysis": map[string]any{key: map[string]string{"status": statusEnabled}},
	})
	if err != nil {
		return err
	}
	_, err = github.API(cwd, body, "--method", "PATCH",
		"-H", "Accept: application/vnd.github+json", "repos/"+repo, "--input", "-")
	return err
}

// repoSettingsMirror renders the desired state as the committed mirror's bytes:
// normalised, with no volatile ids or timestamps, so a re-run of an unchanged
// repository produces an identical file and a real change produces a reviewable
// diff. The same shape the branch-ruleset mirror beside it already holds to.
func repoSettingsMirror(repo string, managed map[string]any, owed []string, merge RemoteMergeHygiene) []byte {
	doc := map[string]any{
		"repo": repo,
		// What abcd drives, and holds this repo to (managedSettings).
		"managed": managed,
		// What abcd only records. Separated from "managed" so a reader can tell a
		// setting abcd will restore from one it merely saw: collapsing the two would
		// make the file read as a promise about settings nothing enforces.
		"observed": merge,
	}
	// What the plan or the visibility does not offer yet, so it applies once the
	// repository is public.
	if len(owed) > 0 {
		doc["owed"] = owed
	}
	// Indented and sorted (encoding/json sorts map keys), then newline-terminated:
	// this file is reviewed in a diff, not parsed by a human.
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil
	}
	return append(data, '\n')
}

// writeRepoSettingsMirror writes the mirror when its content would change, and
// reports whether it wrote. Content-keyed rather than unconditional: a verb that
// rewrote the file on every run would make "nothing changed" and "the change was
// reapplied" produce the same receipt.
//
// The write is CONTAINED — resolved through an os.Root opened at the repo — so a
// committed `.abcd` symlink cannot land the mirror outside the working tree while
// every surface reports the in-repo path.
func writeRepoSettingsMirror(cwd, repo string, managed map[string]any, owed []string, merge RemoteMergeHygiene) (bool, error) {
	want := repoSettingsMirror(repo, managed, owed, merge)
	if len(want) == 0 {
		return false, errors.New("the mirror could not be rendered")
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return false, err
	}
	defer root.Close()
	if have, err := fsutil.ReadGuardedInRoot(root, RepoSettingsMirrorRelPath, maxAhoyFileBytes); err == nil && bytes.Equal(have, want) {
		return false, nil
	}
	if dir := path.Dir(RepoSettingsMirrorRelPath); dir != "." {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return false, err
		}
	}
	if err := fsutil.WriteFileAtomicInRoot(root, RepoSettingsMirrorRelPath, want, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// GitHubRepo is the owner/name this checkout's origin remote names, under the
// same github.com-only, plain-name rules the remote apply writes behind. Every
// verb that writes to the forge resolves its repository here, so there is one
// answer to "which repository may this write reach".
func GitHubRepo(cwd string) (string, error) { return resolveGitHubRepo(cwd) }

// runGH is github.Run, kept under the name ahoy's own tests call it by.
func runGH(cwd string, stdin []byte, args ...string) ([]byte, error) {
	return github.Run(cwd, stdin, args...)
}
