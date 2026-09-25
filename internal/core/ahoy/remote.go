package ahoy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// RepoSettingsMirrorRelPath is the committed mirror of the repo-object settings
// abcd manages, repo-relative and slash-separated. It sits beside the branch-ruleset
// mirror, which covers RULESETS only: the security toggles live behind the separate
// repository-object API, so a settings change had no source of truth in the tree at
// all and every claim about it rested on a web console (iss-2608270512210664).
const RepoSettingsMirrorRelPath = ".abcd/work/rulesets/repo-settings.json"

// remoteAPITimeout bounds each `gh` call. A subprocess reading from the network
// with no bound is a verb that hangs a session indefinitely on a stalled connection,
// and this one is invoked from a commit-adjacent workflow.
const remoteAPITimeout = 30 * time.Second

// maxGHOutputBytes caps what is read back from `gh` (trust boundary): the response
// is remote-controlled, and an unbounded read into memory is the shape a hostile or
// broken endpoint exploits.
const maxGHOutputBytes = 1 << 20

// The two toggles this verb manages, as GitHub names them in the
// security_and_analysis object.
const (
	secretScanningKey = "secret_scanning"
	pushProtectionKey = "secret_scanning_push_protection"
)

// statusEnabled is the only value either toggle is driven to.
const statusEnabled = "enabled"

// githubHost is the API host every request names EXPLICITLY. Without it `gh api`
// picks its host from GH_HOST, or from whichever single host the caller's gh is
// authenticated to — so the github.com-only rule parseGitHubRemote enforces would
// constrain the PATH while the environment chose the ENDPOINT. An ambient GH_HOST
// (a repo direnv file, a CI job, an agent harness) would then send this verb's
// authenticated PATCH, with whatever credential gh holds for that host, to a
// machine the origin URL never named. The decision is made once, where the origin
// is validated, and applied where the request is built.
const githubHost = "github.com"

// RemoteSecurity is the pair of GitHub-native secret-scanning toggles, as observed
// or as desired. A toggle GitHub did not report reads "unknown", never "disabled":
// the two demand opposite responses, and reporting an unread state as off would
// send an operator to enable something that may already be on.
type RemoteSecurity struct {
	SecretScanning string `json:"secret_scanning"`
	PushProtection string `json:"secret_scanning_push_protection"`
}

// RemoteMergeHygiene is the repo-object settings that gate MERGE behaviour. abcd
// mirrors them and does not set them: they encode a maintainer's workflow, not a
// security posture, and a verb that drove them would be changing how the project
// merges on the strength of a default nobody chose.
//
// They are here because they had no source of truth in the tree at all
// (iss-2608270512210664): the branch-ruleset mirror beside this one covers
// RULESETS, and these live behind the separate repository-object API — so every
// claim a contribution document made about them rested on a web console. Each is a
// pointer, omitted when the response does not carry it, because "false" and "the
// API did not say" are different facts.
type RemoteMergeHygiene struct {
	DeleteBranchOnMerge *bool `json:"delete_branch_on_merge,omitempty"`
	AllowMergeCommit    *bool `json:"allow_merge_commit,omitempty"`
	AllowSquashMerge    *bool `json:"allow_squash_merge,omitempty"`
	AllowRebaseMerge    *bool `json:"allow_rebase_merge,omitempty"`
}

// RemoteResult is the outcome of a remote config read or apply.
type RemoteResult struct {
	// Repo is the owner/name the verb resolved from this repo's origin remote, and
	// the ONLY repository it acts on. It is reported so a caller can see which
	// repository a write reached before believing the rest of the result.
	Repo string `json:"repo"`
	// Status is one of: already_up_to_date | clean | pending | aborted | opted_out |
	// refused. "aborted" is a confirmation the caller declined; "refused" is a gate
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
}

// ownerRepoRe is the charset a GitHub owner or repository name may use. It is
// applied to BOTH segments before either reaches an API path, because the path is
// assembled by concatenation: a segment carrying a slash or a `..` would not merely
// name a different repository, it would address a different ENDPOINT — on a verb
// whose whole purpose is an authenticated write. Anchored, and deliberately
// narrower than GitHub's own acceptance: refusing a name abcd cannot vouch for
// costs one error message, and the alternative costs a request nobody intended.
var ownerRepoRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

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
	out, err := runGit(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", false
	}
	top := strings.TrimSpace(out)
	if top == "" {
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
	owner, name, err := parseGitHubRemote(url)
	if err != nil {
		return "", err
	}
	return owner + "/" + name, nil
}

// parseGitHubRemote splits a github.com remote URL into owner and repository. It
// accepts the two forms git itself writes — `https://github.com/owner/repo[.git]`
// and `git@github.com:owner/repo[.git]` — and refuses everything else.
func parseGitHubRemote(url string) (owner, name string, err error) {
	rest := ""
	switch {
	case strings.HasPrefix(url, "git@github.com:"):
		rest = strings.TrimPrefix(url, "git@github.com:")
	case strings.HasPrefix(url, "ssh://git@github.com/"):
		rest = strings.TrimPrefix(url, "ssh://git@github.com/")
	case strings.HasPrefix(url, "https://github.com/"):
		rest = strings.TrimPrefix(url, "https://github.com/")
	case strings.HasPrefix(url, "http://github.com/"):
		rest = strings.TrimPrefix(url, "http://github.com/")
	default:
		return "", "", fmt.Errorf("the origin remote is not a github.com URL, and this verb speaks only the GitHub API")
	}
	rest = strings.TrimSuffix(strings.TrimSuffix(strings.Trim(rest, "/"), ".git"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("the origin remote does not name exactly one owner and one repository")
	}
	owner, name = parts[0], parts[1]
	if !ownerRepoRe.MatchString(owner) || !ownerRepoRe.MatchString(name) {
		// Names the fault without echoing the segment: the URL is repo-controlled
		// input, and a refusal is output like any other.
		return "", "", fmt.Errorf("the origin remote's owner or repository name is not a plain GitHub name, " +
			"so it will not be used to build an API path")
	}
	return owner, name, nil
}

// RemoteRead reports the managed repo's GitHub-native secret-scanning state and
// what an apply would change. It writes NOTHING — not to the remote, not to the
// tree — so the bare invocation of the verb it backs holds abcd's convention that
// looking is never acting.
func RemoteRead(cwd string) (RemoteResult, error) {
	res, abs, done := remotePrepare(cwd)
	if done {
		return res, nil
	}
	observed, merge, err := ghSecurityState(abs, res.Repo)
	if err != nil {
		return refuseRemote(res, "could not read "+res.Repo+"'s security settings, so nothing is known about what would change: "+errText(err)), nil
	}
	res.Observed, res.Merge = observed, merge
	res.Changes = pendingChanges(observed)
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
func RemoteApply(cwd string, p Prompter) (RemoteResult, error) {
	if p == nil {
		p = RefusingPrompter{}
	}
	res, abs, done := remotePrepare(cwd)
	if done {
		return res, nil
	}
	observed, merge, err := ghSecurityState(abs, res.Repo)
	if err != nil {
		return refuseRemote(res, "could not read "+res.Repo+"'s security settings; nothing was changed, because a write over an unknown state is a guess: "+errText(err)), nil
	}
	res.Observed, res.Merge = observed, merge

	// The confirmation, asked ONLY when a remote write would actually follow: a
	// prompt over a change that is not going to happen teaches the answerer to say
	// yes without reading, which is exactly the reflex this gate exists to avoid.
	if pending := pendingChanges(observed); len(pending) > 0 {
		if !p.Confirm("Change GitHub settings on " + res.Repo + "? (" + strings.Join(pending, "; ") + ")") {
			res.Status = "aborted"
			res.Notes = append(res.Notes, "the change to "+res.Repo+" was declined, so nothing was written on GitHub "+
				"and no desired state was recorded")
			return res, nil
		}
	}

	// ORDERED, and the order is the contract: GitHub refuses push protection on a
	// repository whose secret scanning is disabled, so the reverse order enables
	// nothing and reports a failure the operator cannot act on. A failed step stops
	// the sequence for the same reason.
	for _, step := range []struct{ key, have string }{
		{secretScanningKey, observed.SecretScanning},
		{pushProtectionKey, observed.PushProtection},
	} {
		if step.have == statusEnabled {
			continue
		}
		if err := ghEnable(abs, res.Repo, step.key); err != nil {
			return refuseRemote(res, "could not enable "+step.key+" on "+res.Repo+
				"; the remaining steps were not attempted and no desired state was recorded: "+errText(err)), nil
		}
		res.Changes = append(res.Changes, step.key+": "+displayStatus(step.have)+" -> "+statusEnabled)
	}

	// The mirror is written only once every enable step this run needed RETURNED
	// SUCCESS — never over a partial apply, whose failed step returns above. It is
	// not a re-read: the claim it records is "abcd drove these toggles and GitHub
	// accepted", which is what a later verify diffs the live state against, and a
	// confirming GET here would only move the same trust one request along.
	wrote, err := writeRepoSettingsMirror(abs, res.Repo, merge)
	if err != nil {
		res.Status = "refused"
		res.Notes = append(res.Notes, "the remote state is correct but the desired-state mirror could not be written ("+
			RepoSettingsMirrorRelPath+"), so nothing in the tree records it: "+errText(err))
		return res, nil
	}
	if wrote {
		res.Writes = append(res.Writes, RepoSettingsMirrorRelPath)
	}
	res.Status = "already_up_to_date"
	if len(res.Changes) > 0 || wrote {
		res.Status = "clean"
	}
	return res, nil
}

// remotePrepare runs the three gates both verbs share and returns done=true when
// one of them settled the outcome. The gates are the adr-44 boundary: after this
// returns done=false, and only then, may a request leave the machine.
func remotePrepare(cwd string) (res RemoteResult, abs string, done bool) {
	res = RemoteResult{Desired: RemoteSecurity{SecretScanning: statusEnabled, PushProtection: statusEnabled}}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return refuseRemote(res, "could not resolve this directory: "+errText(err)), "", true
	}
	// Anchor EVERYTHING that follows at the working-tree root, before the first
	// question is asked. Run from a subdirectory, git answers the identity questions
	// about the enclosing repository while a cwd-rooted config read answers the
	// consent question about an empty directory — so the verb would resolve WHICH
	// repository to write from one place and WHETHER it may from another.
	top, ok := worktreeRoot(abs)
	if !ok {
		return refuseRemote(res, "could not resolve this repository's working-tree root, so abcd cannot tell "+
			"which repository it would be acting on"), "", true
	}
	abs = top
	det, err := Detect(abs)
	if err != nil {
		return refuseRemote(res, "could not classify this folder, so abcd will not act on a repository it cannot identify: "+errText(err)), "", true
	}
	if det.FolderKind != ManagedRepo {
		return refuseRemote(res, "this is not a repo abcd manages ("+string(det.FolderKind)+
			"); a remote write is never made on a folder that did not adopt abcd (run `abcd ahoy install` first)"), "", true
	}
	optedOut, answerable := nativeScanningOptedOut(abs)
	if !answerable {
		return refuseRemote(res, "this repo's .abcd/config.json is absent or cannot be read, so the opt-out cannot "+
			"be checked — and a config that does not answer is not consent"), "", true
	}
	if optedOut {
		res.Status = "opted_out"
		res.Notes = append(res.Notes, "this repo sets scan.native_secret_scanning to false, so abcd left GitHub's "+
			"secret-scanning toggles exactly as they are and contacted nothing")
		return res, "", true
	}
	repo, err := resolveGitHubRepo(abs)
	if err != nil {
		return refuseRemote(res, "could not resolve which GitHub repository this checkout belongs to: "+errText(err)), "", true
	}
	res.Repo = repo
	return res, abs, false
}

// refuseRemote records a loud refusal on the result. Every failure path in this
// file goes through it, so "abcd did not do this" is never expressed as an absent
// change a caller has to notice.
func refuseRemote(res RemoteResult, reason string) RemoteResult {
	res.Status = "refused"
	res.Notes = append(res.Notes, reason)
	return res
}

// pendingChanges names the toggles that are not yet in the desired state.
func pendingChanges(observed RemoteSecurity) []string {
	var out []string
	for _, step := range []struct{ key, have string }{
		{secretScanningKey, observed.SecretScanning},
		{pushProtectionKey, observed.PushProtection},
	} {
		if step.have != statusEnabled {
			out = append(out, step.key+": "+displayStatus(step.have)+" -> "+statusEnabled)
		}
	}
	return out
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
func ghSecurityState(cwd, repo string) (RemoteSecurity, RemoteMergeHygiene, error) {
	out, err := runGH(cwd, nil, "api", "--hostname", githubHost, "-H", "Accept: application/vnd.github+json", "repos/"+repo)
	if err != nil {
		return RemoteSecurity{}, RemoteMergeHygiene{}, err
	}
	var doc struct {
		SecurityAndAnalysis map[string]struct {
			Status string `json:"status"`
		} `json:"security_and_analysis"`
		DeleteBranchOnMerge *bool `json:"delete_branch_on_merge"`
		AllowMergeCommit    *bool `json:"allow_merge_commit"`
		AllowSquashMerge    *bool `json:"allow_squash_merge"`
		AllowRebaseMerge    *bool `json:"allow_rebase_merge"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return RemoteSecurity{}, RemoteMergeHygiene{}, fmt.Errorf("the API response could not be read as JSON")
	}
	return RemoteSecurity{
			SecretScanning: doc.SecurityAndAnalysis[secretScanningKey].Status,
			PushProtection: doc.SecurityAndAnalysis[pushProtectionKey].Status,
		}, RemoteMergeHygiene{
			DeleteBranchOnMerge: doc.DeleteBranchOnMerge,
			AllowMergeCommit:    doc.AllowMergeCommit,
			AllowSquashMerge:    doc.AllowSquashMerge,
			AllowRebaseMerge:    doc.AllowRebaseMerge,
		}, nil
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
	_, err = runGH(cwd, body, "api", "--hostname", githubHost, "--method", "PATCH",
		"-H", "Accept: application/vnd.github+json", "repos/"+repo, "--input", "-")
	return err
}

// runGH executes one `gh` subcommand under cwd and returns its bounded stdout.
//
// The call goes through `gh` rather than a raw HTTP client on purpose: it inherits
// the caller's own authenticated identity, so abcd never handles a token and a
// write is made by the person who invoked it — which is what adr-44's "no uninvited
// remote mutation" rests on. The environment is passed through unchanged for the
// same reason; scrubbing it would blind gh to the credentials that make the caller
// the actor.
func runGH(cwd string, stdin []byte, args ...string) ([]byte, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, errors.New("the GitHub CLI (gh) is not on PATH; abcd speaks to GitHub through it so that a " +
			"remote write is made by your own authenticated identity, never by a token abcd holds")
	}
	ctx, cancel := context.WithTimeout(context.Background(), remoteAPITimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &out, remaining: maxGHOutputBytes}
	cmd.Stderr = &limitedWriter{w: &errBuf, remaining: maxGHOutputBytes}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("gh did not answer within %s", remoteAPITimeout)
		}
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%w (%s)", err, msg)
	}
	return out.Bytes(), nil
}

// limitedWriter caps what a subprocess can put into memory. It DISCARDS the
// overflow rather than erroring the command: the cap is a memory bound, and turning
// a large-but-valid response into a failure would refuse a repository for being
// verbose.
type limitedWriter struct {
	w         io.Writer
	remaining int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.remaining <= 0 {
		return len(p), nil
	}
	if len(p) > l.remaining {
		if _, err := l.w.Write(p[:l.remaining]); err != nil {
			return 0, err
		}
		l.remaining = 0
		return len(p), nil
	}
	l.remaining -= len(p)
	return l.w.Write(p)
}

// repoSettingsMirror renders the desired state as the committed mirror's bytes:
// normalised, with no volatile ids or timestamps, so a re-run of an unchanged
// repository produces an identical file and a real change produces a reviewable
// diff. The same shape the branch-ruleset mirror beside it already holds to.
func repoSettingsMirror(repo string, merge RemoteMergeHygiene) []byte {
	doc := map[string]any{
		"repo": repo,
		// What abcd drives, and holds this repo to.
		"managed": map[string]any{
			"security_and_analysis": map[string]any{
				secretScanningKey: map[string]string{"status": statusEnabled},
				pushProtectionKey: map[string]string{"status": statusEnabled},
			},
		},
		// What abcd only records. Separated from "managed" so a reader can tell a
		// setting abcd will restore from one it merely saw: collapsing the two would
		// make the file read as a promise about settings nothing enforces.
		"observed": merge,
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
func writeRepoSettingsMirror(cwd, repo string, merge RemoteMergeHygiene) (bool, error) {
	want := repoSettingsMirror(repo, merge)
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

// GH runs one `gh` subcommand under cwd with the same bounds the remote apply
// uses: a timeout, a capped read, and the caller's own authenticated identity,
// so a forge write is made by the person who invoked the verb and abcd never
// holds a forge token. Callers pass `--hostname github.com` explicitly for the
// reason githubHost states.
func GH(cwd string, stdin []byte, args ...string) ([]byte, error) { return runGH(cwd, stdin, args...) }

// GitHubHost is the API host every forge request names explicitly.
const GitHubHost = githubHost
