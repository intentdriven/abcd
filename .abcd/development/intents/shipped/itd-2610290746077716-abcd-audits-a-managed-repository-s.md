---
id: itd-2610290746077716
slug: abcd-audits-a-managed-repository-s
spec_id: spc-2610290749113317
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
refines: [iss-2610121535283764]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd audits and applies a repository's GitHub settings

## Press Release

> A person running several repositories with abcd can see, in one read-only report, how each repository's GitHub settings differ from a safe baseline: one line per setting, saying what it is now and what it should be. Secret scanning off on a public repository shows up. So does a default branch anyone can force-push, or a required check that no workflow runs. When the person says yes, abcd turns on what is missing and writes down what it changed. It never changes whether a repository is public or private, and it never requires a check the repository's own workflows do not run. A setting the plan does not offer is listed as owed, so it applies once the repository goes public. A repository that wants a setting left alone says so in its own reviewed config.

## Why This Matters

Settings drift between repositories that abcd manages, and nothing in abcd notices. In October 2026 one managed public repository had secret scanning, push protection and Dependabot security updates off, no CODEOWNERS file, its wiki on and branches kept after merge, while abcd's own repository had all of them set. A new private repository on the Free plan could not turn on secret scanning at all, and nothing recorded that it was owed once the repository went public. Today `abcd ahoy --remote` covers two toggles, secret scanning and push protection. Everything else rests on someone remembering to open a web console. This was reported back by a managed repository (iss-2610121535283764, major, security), and it blocks the next release cut.

## Mechanism

We expect fewer exposed repositories, because drift becomes visible: every setting the baseline names is read and compared on every run, and one yes applies the fix. The claim is wrong if a managed repository whose audit reports clean still has a baseline setting off, or if an apply ever leaves a pull request unmergeable because a required check never reports.

## Scope Conditions

- Repositories on github.com whose origin remote names one owner and one repository, read and written through the person's own `gh` sign-in. abcd holds no token. <!-- cond: cond-2610290749117650 -->
- Repositories that adopted abcd (`abcd ahoy install`). The audit refuses any other folder, as `ahoy --remote` does today. <!-- cond: cond-2610290749117599 -->
- Owner on any GitHub plan. A setting the plan or the visibility does not offer is reported as owed, never as a failure. <!-- cond: cond-2610290749118129 -->
- Organization-wide settings are read where the sign-in can read them and are never written. A missing scope is named. <!-- cond: cond-2610290749116813 -->

## SOTA

- OpenSSF Allstar: a GitHub App that enforces Branch-Protection, a security policy and similar checks across an organization, opening issues or fixing settings. Mature and widely used, but it needs an App installed with organization-wide write access.
- github/safe-settings: a GitHub App that applies repository settings, branch protection and rulesets from a central admin repository. Mature for organizations that run their own App, and heavy for one person's repositories.
- OpenSSF Scorecard (CLI and Action): reads and scores a repository against the same checks, and writes nothing. Mature, and the source of this baseline's checks.
- The Terraform GitHub provider: declares settings as infrastructure code. Mature, but it needs a token held in state and a provider workflow.

Path taken: 3, bespoke on the GitHub API through the person's own `gh` sign-in. abcd holds no token and installs no App, which is what adr-44's "no uninvited remote mutation" rests on. The baseline itself is adopted from GitHub's hardening guidance and the Scorecard checks rather than invented.

## Acceptance Criteria

- Given a managed public repository with secret scanning, push protection and Dependabot security updates off, no CODEOWNERS file, the wiki on and branches kept after merge, when the person runs `abcd ahoy --remote`, then each difference is listed as one line naming the setting, its current value and its target, and nothing is changed on GitHub or in the tree.
- Given that same repository, when the person runs `abcd ahoy remote apply` and says yes, then each listed difference abcd can apply is applied. A second `abcd ahoy --remote` then reports no difference, only the advisory and kept settings, and a second apply changes nothing.
- Given an unanswered run of `abcd ahoy remote apply` (no terminal, no `--yes`), when it runs, then nothing is changed and it exits non-zero.
- Given a private repository on a plan that does not offer secret scanning or rulesets, when the person runs the audit and then the apply, then the apply turns on what the plan offers. The rest is listed as owed until the repository is public, and the mirror records it as owed.
- Given a repository whose declared visibility in `.abcd/config.json` differs from its real visibility, when the audit runs, then the mismatch is reported, and no command of abcd's changes the visibility.
- Given a repository whose workflows run the checks `build` and `lint` on pull requests, when the apply creates the default-branch ruleset, then it requires exactly those checks. A check that runs only on some paths, or not on the merge queue when a merge queue is on, is not required. A required check that no workflow reports is named in the audit.
- Given a repository whose `.abcd/config.json` lists a setting under `github.keep`, when the audit and the apply run, then that setting is reported as kept by the repository's choice and is never changed.
- Given a sign-in that lacks a scope a read needs (for example `admin:org` for the organization's Actions policy), when the audit runs, then it names the missing scope and the command that grants it, and reports that setting as unknown, not as off.
- Given workflows that use an action by tag rather than by commit SHA, when the audit runs, then requiring SHA pinning is reported as advisory, with the number of unpinned uses, and the apply does not turn it on.

## Decisions

- The baseline is taken from GitHub's own hardening guidance and the OpenSSF Scorecard checks, and matches abcd's own repository (`.abcd/work/rulesets/`):
  - security: secret scanning, push protection, Dependabot alerts and Dependabot security updates on. Sources: GitHub's "Quickstart for securing your repository", and Scorecard's Vulnerabilities and Dependency-Update-Tool checks.
  - repository: branches deleted after merge; wiki and projects off.
  - default branch: no deletion, no force-push, and the checks the workflows run. A pull-request rule requires a code owner's review, with an approval count of 0. No ruleset abcd writes has a bypass actor. Sources: Scorecard's Branch-Protection and Code-Review checks, and GitHub's "About rulesets".
  - release tags `v*`: no deletion, no update, no force-move.
  - CODEOWNERS covering the publish surface that exists in the tree: `.github/`, `.githooks/`, `.abcd/` config, build files, reported as advisory only, and never one person picked by abcd. Whether a managed repository keeps the file at all is settled by the maintainer-gate record (see below).
  - Actions: SHA pinning required, or an allow-list. The default workflow token is read-only and cannot approve pull requests. Sources: GitHub's "Security hardening for GitHub Actions", and Scorecard's Pinned-Dependencies and Token-Permissions checks.
  (Technical facilitator on delegation, 29 October 2026.)
- Branch deletion after merge, the wiki and projects are now set on a yes, where the ahoy brief said merge hygiene is mirrored and never set. The managed repository's report asks for them, and `github.keep` leaves any of them alone. The merge methods stay mirrored and are never set. (Technical facilitator on delegation, 29 October 2026.)
- The code-owner review rule has no bypass, because a solo owner does not need one, as checked on abcd's own repository on 29 October 2026. Its live "main review" ruleset (id 21045110) has no bypass actors, and the API reports that its owner can never bypass it. Yet the fifteen most recent merged pull requests that change owned paths (`hooks/`, `commands/`, `.github/`, `Makefile`) merged with no review at all. A pull request whose author is the only code owner of what it changes needs no code-owner approval. A bypass would be a weaker rule than abcd's own. A second writer's pull request does need the owner's approval, which is what the rule is for. (Technical facilitator on delegation, 29 October 2026; checked at the coordinator's request.)
- A maintainer is anyone with Write, Maintain or Admin access to the repository, admins included; Read and Triage are contributors. GitHub counts a pull-request approval only from Write access and above, so a required approval is already a maintainer's, and no team has to be kept for it. Whether a managed repository keeps a CODEOWNERS file at all belongs to iss-2610290730539347, not here. Until that is settled the audit reports a missing or partial CODEOWNERS as advisory and names the paths it should cover, and the apply does not write the file (iss-2610290755587946). The code-owner rule abcd applies keeps no bypass, as above, and an approval count of 0: with a count of 1 a repository with one maintainer could never merge without a bypass, which would break the half of the ruling that lets maintainers merge freely. That rule asks for a maintainer's approval only on the paths CODEOWNERS covers; requiring one for a contributor's pull request on every path is the maintainer-gate record's to enforce, not this intent's. (Product thinker, 29 October 2026, through the coordinator; this supersedes the Maintain-role reading given earlier the same day. The count was confirmed by the coordinator the same day.) <!-- record-lint: forward-looking -->
- abcd never edits a ruleset it did not create. It creates and updates its own rulesets, found by name, and rules from several rulesets add up on GitHub, so a person's own ruleset keeps working beside abcd's. (Technical facilitator on delegation, 29 October 2026.)
- SHA pinning and the read-only workflow token are applied only when the workflows already comply: every `uses:` is pinned to a commit, and every workflow declares its `permissions:`. Otherwise the apply would break a working CI, so both are reported as advisory. (Technical facilitator on delegation, 29 October 2026.)
- A repository with no CI gets no required check. The audit says that no workflow reports a check, so nothing blocks a merge. Proposing a starter workflow is outside this intent and is its own record, iss-2610290748300390. (Technical facilitator on delegation, 29 October 2026; agreed by the coordinator as a scope call.)

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-6083de2cef52 -->
Fidelity review — receipt rcp-6083de2cef52 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:a7a8b4b7a236080fd3c5fb3d436c002f2bc79425b9bcc63af3bd39ec23d9fa84
Input attestations: diff:1b89a7e123b84846417b7bc959c7b212c47858c4^1..1b89a7e123b84846417b7bc959c7b212c47858c4 (PR #1225)@sha256:3ad1904768ba8ea2abf342388c530afd5d657d56e1257038c0a4f1540a0867e8; tree:a584e1cdf8a28ef1bb54b501e23fd427573babf3 (origin/main; internal/core/ahoy and internal/core/launch/scaffold unchanged since the merge)@-;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: RemoteRead builds the whole audit and returns its differences without any write; the text render prints one 'setting: current -> target [state]' line per finding not at target; a routed-fake test on the managed repository's reported state asserts each listed setting, no PATCH/PUT/POST/DELETE, and an unchanged tree
  evidence: internal/core/ahoy/remote.go:195 — "res.Audit = auditSettings(in)"
  evidence: internal/surface/cli/cli.go:4668 — "line := fmt.Sprintf(" setting: %s: %s -> %s [%s]", f.Setting, f.Current, f.Target, f.State)"
  evidence: internal/core/ahoy/remote_baseline_test.go:265 — "func TestRemoteAuditListsEachDifferenceAndChangesNothing(t *testing.T) {"
  evidence: internal/core/ahoy/remote_baseline_test.go:315 — "if w := writeCalls(t, logPath); len(w) != 0 {"
  evidence: internal/core/ahoy/remote_baseline.go:811 — "f.Current, f.State, f.Note = "absent", findingAdvisory, "add a CODEOWNERS file covering these paths"+later"
- ac-2 — MET: applyDifferences writes every differs finding in GitHub's required order (toggles, one repository PATCH, abcd's own rulesets, Actions settings); the test applies on the reported state, then re-reads the post-apply state with zero differences, and a second apply makes no write and rewrites no file
  evidence: internal/core/ahoy/remote_baseline_apply.go:52 — "func applyDifferences(root, repo string, audit *SettingsAudit) (a applied, failure string) {"
  evidence: internal/core/ahoy/remote_baseline_apply_test.go:77 — "func TestRemoteApplyAppliesTheDifferencesThenAuditsClean(t *testing.T) {"
  evidence: internal/core/ahoy/remote_baseline_apply_test.go:146 — "if d := read.Audit.Differences(); len(d) != 0 {"
  evidence: internal/core/ahoy/remote_baseline_apply_test.go:158 — "if w := writeCalls(t, logPath); len(w) != 0 {"
- ac-3 — MET: the confirmation precedes every write and a decline returns status aborted before applyDifferences runs; the CLI maps aborted to exit 1, and core and CLI tests drive RefusingPrompter and an EOF run to aborted, no write, no mirror, non-zero exit
  evidence: internal/core/ahoy/remote.go:265 — "if !p.Confirm(termsafe.Sanitize("Change GitHub settings on " + res.Repo + "? (" + strings.Join(pending, "; ") + ")")) {"
  evidence: internal/core/ahoy/remote.go:266 — "res.Status = "aborted""
  evidence: internal/surface/cli/cli.go:4579 — "if res.Status == "refused" || res.Status == "aborted" {"
  evidence: internal/surface/cli/ahoy_remote_test.go:54 — "func TestAhoyRemoteApplyExitsNonZeroWhenItChangesNothing(t *testing.T) {"
- ac-4 — MET: on a private repository an absent scanning field and a 403/404 'upgrade' refusal of the rules reads make those settings owed; the apply writes only differs findings, and the mirror gains an owed list; the test asserts only the Dependabot writes land, no ruleset POST, and owed holds secret scanning, push protection, release tags and code-owner review
  evidence: internal/core/ahoy/remote_baseline.go:286 — "case status == "" && priv:"
  evidence: internal/core/ahoy/remote_baseline.go:541 — "if planRefusal(err) && priv {"
  evidence: internal/core/ahoy/remote.go:560 — "doc["owed"] = owed"
  evidence: internal/core/ahoy/remote_baseline_apply_test.go:169 — "func TestRemoteApplyOnAPrivateRepoAppliesWhatThePlanOffers(t *testing.T) {"
- ac-5 — MET: visibilityFinding compares the GitHub answer with repo.visibility and reports a mismatch as advisory (never differs, so never applied); no write body carries visibility, asserted by the apply test, and no repository-edit call exists in internal/
  evidence: internal/core/ahoy/remote_baseline.go:395 — "f.State, f.Note = findingAdvisory, "the repository is "+real+" but .abcd/config.json declares it "+declared+"
  evidence: internal/core/ahoy/remote_baseline_test.go:365 — "func TestRemoteAuditReportsAVisibilityMismatch(t *testing.T) {"
  evidence: internal/core/ahoy/remote_baseline_apply_test.go:131 — "if b := stdinOf(c); strings.Contains(b, "visibility") || strings.Contains(b, `"private"`) {"
- ac-6 — MET: scaffold.RequiredChecks keeps only pull_request workflows with no paths filter and, under a merge queue, with merge_group; the apply test (build/lint, a path-filtered docs workflow, a pr-only workflow under a queue) asserts the ruleset requires exactly build,lint; a required context no workflow reports is named in the finding's note
  evidence: internal/core/launch/scaffold/cichecks.go:149 — "case mergeQueue && !hasEvent(on, "merge_group"):"
  evidence: internal/core/ahoy/remote_baseline_apply_test.go:260 — "if strings.Join(got, ",") != "build,lint" {"
  evidence: internal/core/ahoy/remote_baseline.go:643 — "staleNote = "required but not reported by any workflow abcd can read: " + strings.Join(stale, ", ") +"
  evidence: internal/core/ahoy/remote_baseline_test.go:300 — "if n := got["required_status_checks"].Note; !strings.Contains(n, "zizmor") {"
- ac-7 — MET: every finding passes through add(), which turns a github.keep setting into kept; the apply writes only differs and excludes kept rules from abcd's default-branch ruleset; tests assert kept states, no kept setting in any write body, and an unknown keep key refused before GitHub is contacted
  evidence: internal/core/ahoy/remote_baseline.go:267 — "if in.ov.keep[f.Setting] {"
  evidence: internal/core/ahoy/remote_baseline_apply.go:153 — "if !kept(settingBranchDeletion) {"
  evidence: internal/core/ahoy/remote_baseline_apply_test.go:267 — "func TestRemoteApplyLeavesAKeptSettingAlone(t *testing.T) {"
  evidence: internal/core/ahoy/remote_baseline_test.go:391 — "func TestRemoteAuditHonoursKeepAndRefusesAnUnknownKey(t *testing.T) {"
- ac-8 — MET_WITH_CONCERNS: the organization Actions policy read on 403/404 is unknown with a note naming admin:org and the 'gh auth refresh -h github.com -s admin:org' command, tested; concern: only that one read names a scope and a granting command, every repository-level 403/404 names the missing admin role with no command, and a failed plan read is silently 'unknown' with no reason
  evidence: internal/core/ahoy/remote_baseline.go:979 — "f.Note = "reading the organization's Actions policy needs an org owner's sign-in with the admin:org scope " +"
  evidence: internal/core/ahoy/remote_baseline.go:980 — ""(gh auth refresh -h github.com -s admin:org); abcd never writes it""
  evidence: internal/core/ahoy/remote_baseline.go:459 — "return "the sign-in cannot read " + what + ": it needs admin rights on the repository""
  evidence: internal/core/ahoy/remote_baseline.go:478 — "if err != nil { return """
  evidence: internal/core/ahoy/remote_baseline_test.go:306 — "if n := got["org_actions_policy"].Note; !strings.Contains(n, "admin:org") {"
- ac-9 — MET_WITH_CONCERNS: with a tag-referenced action, actions_sha_pinning is advisory with 'N uses ... not pinned' and the apply writes only differs findings, tested; concern: the unread-workflow branch is checked before the count branch, so a repository that also uses a local action or has an unreadable workflow gets advisory with no count, and an Actions answer without sha_pinning_required gives unknown rather than advisory
  evidence: internal/core/ahoy/remote_baseline.go:925 — "f.Note = strconv.Itoa(facts.Unpinned) + " " + uses + " not pinned to a commit SHA in .github/workflows; " +"
  evidence: internal/core/ahoy/remote_baseline.go:913 — "case len(facts.Unread) > 0:"
  evidence: internal/core/ahoy/remote_baseline.go:911 — "case doc.SHAPinningRequired == nil:"
  evidence: internal/core/ahoy/remote_baseline_apply_test.go:127 — "if w := callsTo(t, logPath, "PUT", "actions/permissions"); len(w) != 0 {"

Gap audit:
- honoured:
  - one read-only report, one line per setting saying what it is now and what it should be
    evidence: internal/surface/cli/cli.go:4668 — "line := fmt.Sprintf(" setting: %s: %s -> %s [%s]", f.Setting, f.Current, f.Target, f.State)"
    evidence: internal/core/ahoy/remote_baseline_test.go:318 — "if after := treeSnapshot(t, repo); after != before {"
  - on a yes abcd turns on what is missing and writes down what it changed (managed block, owed list, per-ruleset mirrors through os.Root, content-keyed)
    evidence: internal/core/ahoy/remote.go:306 — "wrote, err := writeRepoSettingsMirror(abs, mirrorRepoName(res.Repo, obj.Owner.Type), managedSettings(audit, ran), owedSettings(audit), merge)"
    evidence: internal/core/ahoy/remote_baseline_apply.go:605 — "if have, err := fsutil.ReadGuardedInRoot(r, rel, maxAhoyFileBytes); err == nil && bytes.Equal(have, data) {"
  - never changes whether a repository is public or private
    evidence: internal/core/ahoy/remote_baseline.go:396 — ""; abcd never changes a repository's visibility: change it on GitHub, or correct repo.visibility""
  - never requires a check the repository's own workflows do not run, and names a required check no workflow reports
    evidence: internal/core/launch/scaffold/cichecks.go:124 — "func RequiredChecks(repoRoot, defaultBranch string, mergeQueue bool) (checks, notes []string) {"
    evidence: internal/core/ahoy/remote_baseline_apply.go:163 — "if differs(settingRequiredChecks) && len(audit.RequiredChecks) > 0 {"
  - a setting the plan does not offer is listed as owed
    evidence: internal/core/ahoy/remote_baseline_apply.go:569 — "func owedSettings(audit *SettingsAudit) []string {"
  - a repository asks for a setting to be left alone in its own reviewed config (github.keep), and a typo refuses
    evidence: internal/core/ahoy/remote_baseline.go:155 — "return ov, fmt.Errorf("github.keep names %q, which is not a baseline setting (the settings are: %s)","
    evidence: commands/ahoy.md:1344 — ""keep": ["has_wiki"],"
  - abcd's rulesets carry no bypass actor and abcd edits only rulesets it created, merging rather than loosening
    evidence: internal/core/ahoy/remote_baseline_apply.go:242 — ""bypass_actors": []any{},"
    evidence: internal/core/ahoy/remote_baseline_apply.go:275 — "if _, ours := rulesetMirrorName[rs.Name]; ours && rulesetIDRe.MatchString(rs.ID.String()) {"
  - no file abcd writes names a person: a person-owned repository is mirrored by name alone
    evidence: internal/core/ahoy/remote_baseline_apply.go:508 — "func mirrorRepoName(repo, ownerType string) string {"
    evidence: internal/core/ahoy/remote_baseline_apply_test.go:292 — "func TestRemoteApplyWritesNoLoginIntoAnyFile(t *testing.T) {"
- diverged:
  - a missing scope is named with the command that grants it: delivered for the organization Actions policy only; repository-level read failures name the admin role with no command, and a failed plan read is silent
    evidence: internal/core/ahoy/remote_baseline.go:459 — "return "the sign-in cannot read " + what + ": it needs admin rights on the repository""
    evidence: internal/core/ahoy/remote_baseline.go:478 — "if err != nil { return """
  - SHA pinning reported as advisory with the number of unpinned uses: the count is dropped when any workflow or local action went unread, because that branch is checked first
    evidence: internal/core/ahoy/remote_baseline.go:913 — "case len(facts.Unread) > 0:"
  - owed on any plan: a plan refusal is recognised as owed only on a private or internal repository and only when GitHub's 403/404 text contains 'upgrade'; any other wording reads as unknown
    evidence: internal/core/ahoy/remote_baseline.go:452 — "return isHTTP(err, 403, 404) && strings.Contains(strings.ToLower(err.Error()), "upgrade")"
    evidence: internal/core/ahoy/remote_baseline.go:236 — "func (o repoObject) private() bool { return o.Visibility == "private" || o.Visibility == "internal" }"
- missing: (none)

Scope-condition dispositions:
- cond-2610290749117650 — survived: the repository comes only from the origin remote parsed as a github.com owner/name, and every read and write goes through github.API, which runs the person's own gh sign-in; abcd holds no token
  evidence: internal/core/ahoy/remote.go:155 — "func resolveGitHubRepo(cwd string) (string, error) {"
  evidence: internal/core/github/github.go:186 — ""remote write is made by your own authenticated identity, never by a token abcd holds""
- cond-2610290749117599 — survived: remotePrepare, shared by the read and the apply, refuses any folder whose kind is not ManagedRepo before a request leaves the machine, and the CLI test exercises the refusal
  evidence: internal/core/ahoy/remote.go:392 — "if det.FolderKind != ManagedRepo {"
  evidence: internal/surface/cli/ahoy_remote_test.go:20 — "func TestAhoyRemoteRefusesAnUnmanagedFolderFromTheCLI(t *testing.T) {"
- cond-2610290749118129 — narrowed: an unoffered setting is never written and never fails the run, but 'owed' is inferred from the visibility and the refusal's wording, not from the plan; the plan is only displayed
  narrowing: owed is reported only for a private or internal repository, and for rulesets only when GitHub's 403/404 message contains 'upgrade'; a plan refusal worded otherwise, or one on a public repository, is reported unknown rather than owed (still never a failure)
  evidence: internal/core/ahoy/remote_baseline.go:452 — "return isHTTP(err, 403, 404) && strings.Contains(strings.ToLower(err.Error()), "upgrade")"
  evidence: internal/core/ahoy/remote_baseline.go:544 — "return all(findingUnknown, readFailure("the default branch's rules", err)), nil, nil"
- cond-2610290749116813 — survived: the only organization endpoints touched are GET reads (plan and Actions policy); the apply file holds no orgs/ path and the org policy finding is never differs, and a 403 names admin:org
  evidence: internal/core/ahoy/remote_baseline.go:974 — "out, err := github.API(root, nil, "orgs/"+owner+"/actions/permissions")"
  evidence: internal/core/ahoy/remote_baseline.go:998 — "f.Current, f.State, f.Note = "any action, by any ref", findingAdvisory, "an org owner can require SHA pinning or an allow-list; abcd never writes the organization's settings""
<!-- abcd-review-end receipt=rcp-6083de2cef52 -->

## Grounds

- pursued: it clears a major security finding that blocks the release cut, and makes drift between managed repositories visible; it is shown wrong if an audit reports clean while a baseline setting is off, or an apply leaves a pull request unmergeable
