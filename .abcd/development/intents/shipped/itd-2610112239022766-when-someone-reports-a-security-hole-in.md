---
id: itd-2610112239022766
slug: when-someone-reports-a-security-hole-in
spec_id: spc-2610120557432880
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
refines: [itd-149]
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Private security reports stay private until their fix ships

## Press Release

> When someone reports a security hole in your project privately, abcd now keeps it private until the fix ships and then publishes it with you. abcd embargo brings each private security advisory from GitHub into a private record on your own machine that never enters the public ledger, records your verdict on it and who gave it (a vulnerability to fix, not abcd's problem, not a vulnerability, or fine to handle in public), closes on GitHub, with the explanation you approved, an advisory you judged not abcd's or not a vulnerability, checks every commit message, pull-request body and record you are about to make public for anything that would give the advisory away, and, once the release carrying the fixes is cut, shows you the publication plan with each patched version worked out from the release that holds the fix. On your one explicit yes it publishes a single advisory first, checks it went out as planned, then publishes the rest and confirms none is left in draft.
>
> "Reports came in through private reporting the week before our release," said Kira. "abcd turned them into records only my machine holds, closed the one that was not ours with the explanation I approved, stopped me when a commit message named one of them, and on release day showed me the patched version for each. I said yes once, watched the first advisory go live, and the rest followed."

## Why This Matters

A private security report is the one piece of work abcd's records cannot hold today. The issue ledger is public, so an advisory copied into it discloses the hole before the fix exists; left out of it, the advisory is named by no record, and nothing in abcd shows whether it was triaged, fixed or published. Two hand-runs did the work by hand: private records kept in an uncommitted worktree, fixes held on local branches, every advisory id searched for by hand before the push, and patched versions typed, one of them wrongly at first. Each of those hand steps is a way for an advisory to leak early or be published wrong, and this rung replaces the ones that leak or mislead with commands: the private record, the verdict, closing what is not a vulnerability to fix, the outbound check and the publication.

## Mechanism

We expect a private advisory to stay unnamed in public until its fix ships, and to be published with the right patched version, because the three hand steps that leaked or misled become commands: the advisory's record lives only in the machine's own abcd home and never on a committed path, so no push can carry it; every commit range, pull-request body and record bound for the public passes `abcd embargo check`, which refuses the advisory's id, its reserved CVE id, its title and the store's location; and the patched version is computed from the first published release that holds the fix (a release tag origin holds, with a GitHub release that is neither a draft nor a prerelease) instead of being typed. The claim fails if an advisory's id or title reaches a pushed commit, a pull-request body or a public record before publication although the check was run over it, or if a published advisory names a patched version other than the first published release holding its fix.

## Scope Conditions

- Holds for a repository hosted on github.com with private vulnerability reporting, where the person's own `gh` is signed in as an account allowed to read and edit the repository's security advisories. <!-- cond: cond-2610120557436663 -->
- Holds while one machine handles a release's advisories: the private records live in that machine's abcd home and are not shared with other machines. <!-- cond: cond-2610120557430932 -->
- Holds for releases tagged `vMAJOR.MINOR.PATCH`, where a fix counts as released once a release tag contains its commit. <!-- cond: cond-2610120557431051 -->

## Acceptance Criteria

- **Given** a repository with private security advisories waiting on GitHub (reported and not yet accepted, or accepted as drafts), **when** the person runs `abcd embargo intake`, **then** each advisory not seen before becomes a private record on the person's machine, linked to its advisory, with high severity recorded as major, medium and low as minor, and critical as critical, and nothing is written to the project's public ledger or committed anywhere.
- **Given** intake has run before, **when** it runs again, **then** an advisory already held is not recorded twice, and an advisory GitHub gives no severity is listed for the person to set on GitHub rather than recorded with a guessed one.
- **Given** a private record, **when** the person records a verdict with `abcd embargo triage` (vulnerability, not abcd's, not a vulnerability, or public track), **then** the verdict is kept with who decided it and when, and an unknown verdict, or one missing its reason, is refused with nothing written.
- **Given** a verdict of "not abcd's" or "not a vulnerability", **when** it is recorded, **then** abcd drafts the closing explanation for a person to approve, and a "not abcd's" draft says why the problem would exist without abcd and why abcd cannot help without changing how the repository works.
- **Given** a drafted closing explanation, **when** the person runs `abcd embargo close` on that advisory, **then** without the person's approval of the draft nothing is posted and the advisory is left as it is on GitHub; with their approval of exactly the text shown to them, abcd closes the advisory on GitHub with that text, reads the advisory back to confirm it is closed with it, and records when it was closed; and an approval of any other text posts nothing.
- **Given** an advisory that is not yet published, **when** `abcd embargo check` runs over a range of commits, a pull-request body or a record about to be made public, **then** it refuses any advisory id, any CVE id reserved for an unpublished advisory, an unpublished advisory's title, and the location of the private records, naming where each was found without repeating it; and text holding none of them passes.
- **Given** a cut release that contains fixes for advisories judged vulnerabilities, **when** the person runs `abcd embargo publish`, **then** it shows the plan (each advisory, its patched version taken from the first release tag that contains its fix, and its affected range in GitHub's form) and changes nothing on GitHub.
- **Given** an advisory whose fix is in no release tag yet, or whose record is not resolved in the release, **when** the plan is shown, **then** that advisory is left out and the plan says why.
- **Given** the plan, **when** the person approves exactly that plan, **then** abcd publishes one advisory first, reads it back to confirm it went out as planned, publishes the rest, and confirms none of them is still a draft; without the approval, or with an approval for a different plan, nothing is published.
- **Given** a person reading abcd's command page for `embargo`, **when** they look up what it does, **then** it says in plain words what the command does on its own and what stays the person's act in this release: fixing and landing the fixes, requesting a CVE, and putting redacted records back after publication; and that abcd closes or publishes an advisory only on the person's own approval of what it showed them.

## SOTA

Surveyed 2026-10-11 against primary sources (the state-of-the-art decision below lists them):

- **GitHub's repository security advisories** (REST API, publishing, temporary private forks): mature and the only channel the reports arrive by. Adopted as the floor: every read and write is that API, called through the person's own `gh`. Its temporary private forks are not adopted, because they skip CI, branch protection and the merge queue; fixes are made on local, unpushed branches instead.
- **Published project processes** (curl, Go, Kubernetes, Rust) and **the guides they follow** (the OpenSSF vulnerability-disclosure guide for maintainers, the CERT Guide to Coordinated Vulnerability Disclosure): mature practice, not tools. Adopted: four triage verdicts, a short time from push to publication, the patched version computed from the first release tag holding the fix, an outbound check over every public artefact, and one human approval per act with a canary first.
- **No existing tool** keeps a private advisory's record out of a public, committed ledger. Path taken: 2, GitHub's API as the native floor behind one `gh` runner seam, with the private store, the verdicts and the outbound check bespoke to abcd's record model.

## Open Questions

None open. Both questions the draft carried were answered by the product thinker on 2026-10-12 (see Decisions).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-bb21ac30f744 -->
Fidelity review — receipt rcp-bb21ac30f744 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:0e480950bfe8c1c3cf86e5868b1153b593355a36eaabd9de511af05468ed80c0
Input attestations: tree:frozen copy of the branch head (embargo-audit-8c8730a3a); spec steps landed d41392a2b@-; decision:.abcd/work/decisions/dcn-2610282341088544-step-11-of-spc-2610120557432880-the-live.md@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Intake lists triage and draft advisories through the person's gh and mints each new one as a record in a detached store worktree under the abcd home. The advisory link is kept in the store's state, and severity maps critical to critical, high to major, and medium and low to minor. A test proves the checkout's status, index and HEAD are unchanged, and intake ran live on 16, 17 and 23 October.
  evidence: internal/core/embargo/intake.go:18 — "var intakeStates = []string{"triage", "draft"}"
  evidence: internal/core/embargo/intake.go:68 — "return "major", true"
  evidence: internal/core/embargo/intake.go:268 — "entry := Advisory{GHSAID: a.GHSAID, IssueID: res.ID, Title: a.Summary, Severity: severity, ForgeState: a.State}"
  evidence: internal/core/embargo/store.go:314 — "gitutil.Run(s.Checkout, "worktree", "add", "--detach", "--", s.Root, base)"
  evidence: internal/core/embargo/intake_test.go:175 — "func TestIntakeLeavesPublicLedgerUntouched(t *testing.T) {"
  evidence: .abcd/work/decisions/dcn-2610282341088544-step-11-of-spc-2610120557432880-the-live.md:6 — "abcd embargo intake ran here on 16, 17 and 23 October"
- ac-2 — MET: On a re-run, an advisory already held is refreshed and listed under held, not minted again. An advisory with a null or unknown severity goes to needs_severity with no record and is kept as pending, so the check still covers it. Both behaviours are tested, and the re-run path was exercised by the three live intakes.
  evidence: internal/core/embargo/intake.go:201 — "refreshHeld(&st.Advisories[i], a)"
  evidence: internal/core/embargo/intake.go:207 — "res.NeedsSeverity = append(res.NeedsSeverity, id)"
  evidence: internal/core/embargo/intake_test.go:201 — "func TestIntakeIsIdempotent(t *testing.T) {"
  evidence: internal/core/embargo/intake_test.go:240 — "func TestIntakeListsNullSeverityAndMintsNothing(t *testing.T) {"
- ac-3 — MET: Triage accepts the four verdicts and stores who decided, a decided_at stamped from the clock, and the reason. Every request is validated before the store is opened, so an unknown verdict, a missing --by or a missing reason is refused, and snapshot tests prove nothing was written. Live triage verdicts are in the store according to dcn-2610282341088544.
  evidence: internal/core/embargo/triage.go:103 — "return "", refuse("the verdict is not one of "+strings.Join(Verdicts, ", ")+"; nothing was written", "name one of the four verdicts")"
  evidence: internal/core/embargo/triage.go:100 — "verdict needs its reason; nothing was written"
  evidence: internal/core/embargo/triage.go:203 — "a.Verdict, a.DecidedBy, a.DecidedAt = &verdict, &by, &now"
  evidence: internal/core/embargo/triage_test.go:172 — "func TestTriageRefusesUnknownVerdictWritingNothing(t *testing.T) {"
  evidence: internal/core/embargo/triage_test.go:184 — "func TestTriageRefusesMissingBy(t *testing.T) {"
- ac-4 — MET: Both closing verdicts write a draft to the store. The not-abcds draft has both halves of the rule under their own headings, and the verdict is refused unless both halves are given.
  evidence: internal/core/embargo/triage.go:156 — "draft = closingDraft(req.Verdict, reason, strings.TrimSpace(req.ExistsWithout), strings.TrimSpace(req.CannotHelp))"
  evidence: internal/core/embargo/closing.go:33 — ""Why the problem would exist without abcd:\n\n" + existsWithout"
  evidence: internal/core/embargo/closing.go:34 — ""Why abcd cannot help without changing how the repository works:\n\n" + cannotHelp"
  evidence: internal/core/embargo/triage_test.go:229 — "func TestClosingDraftNamesBothHalves(t *testing.T) {"
  evidence: internal/core/embargo/triage_test.go:208 — "func TestNotAbcdsNeedsBothHalves(t *testing.T) {"
- ac-5 — MET_WITH_CONCERNS: The code does what the criterion asks. Without --approve it sends only a GET. The digest covers the advisory id, the exact text and the description as read, so a different digest sends nothing. On a match one PATCH closes the advisory, a read-back must show it closed with the text, and closed_at is then stamped. Concerns: (1) the GitHub-side half has been tested only against a fake gh. The one live close ran on a record whose verdict does not close the advisory, so neither the show step nor the approved close has run against GitHub. (2) Putting the explanation at the top of the description is still waiting on the product thinker's confirmation.
  evidence: internal/core/embargo/close.go:225 — "if req.Approve == "" {"
  evidence: internal/core/embargo/close.go:228 — "if req.Approve != digest {"
  evidence: internal/core/embargo/close.go:244 — "stateHeld, textHeld := back.State == "closed", approvedTextHeld(back.description(), text)"
  evidence: internal/core/embargo/close.go:250 — "recordForgeEnd(&st.Advisories[i], "closed", now)"
  evidence: internal/core/embargo/close.go:86 — "spec's design and is pending the product thinker's confirmation"
  evidence: internal/core/embargo/close_test.go:121 — "func TestCloseWrongDigestSendsNothing(t *testing.T) {"
  evidence: .abcd/work/decisions/dcn-2610282341088544-step-11-of-spc-2610120557432880-the-live.md:6 — "Its show-and-approve path has no live run yet"
- ac-6 — MET_WITH_CONCERNS: The check reads commit ranges, files or stdin, and record ids. It refuses advisory ids, reserved CVE ids, titles and the store's path forms, and each finding gives its source and line with the match starred out. Clean text passes, and a range check ran live before two release tags. Concern: some unpublished advisories' titles are not refused, which narrows the promise. These are titles shorter than 12 characters, titles of closed but unpublished advisories, and titles of advisories moved to the public track. The command page discloses the first two but not the third.
  evidence: internal/core/embargo/check.go:29 — "KindAdvisoryID = "advisory_id""
  evidence: internal/core/embargo/check.go:56 — "const minTitleRunes = 12"
  evidence: internal/core/embargo/check.go:645 — "if (a.ClosedAt == nil || !dropClosedTitles) && (a.Verdict == nil || *a.Verdict != VerdictPublicTrack) {"
  evidence: internal/core/embargo/check.go:780 — "masked[i] = '*'"
  evidence: internal/core/embargo/check_test.go:178 — "func TestCheckRefusesStorePathForms(t *testing.T) {"
  evidence: internal/core/embargo/check_test.go:419 — "func TestCheckClean(t *testing.T) {"
  evidence: commands/embargo.md:139 — "its title (not once the advisory is closed, since its reporter chose it and"
- ac-7 — MET_WITH_CONCERNS: The plan sends only reads: GETs, an ls-remote and the release lookups. For each entry it shows the advisory, the fix commit, the first tag, the patched version and an affected range checked against GitHub's grammar. It is tested and ran live on 17 October. Concern: the patched version comes from the first published release that holds the fix, not the first release tag. A tag counts only when origin holds it and GitHub has a release for it that is neither a draft nor a prerelease. When an earlier tag is skipped, the patched version is not the first release tag holding the fix, which is the case the Mechanism names as falsifying.
  evidence: internal/core/embargo/plan.go:286 — "patched := patchedVersion(first)"
  evidence: internal/core/embargo/versions.go:94 — "func affectedRange(existing *string, patched string) (string, error) {"
  evidence: internal/core/embargo/releases.go:14 — "tag in this checkout counts only when origin holds the tag at the same"
  evidence: internal/core/embargo/plan_test.go:184 — "func TestFirstReleaseTagContainingFix(t *testing.T) {"
  evidence: internal/core/embargo/plan_test.go:463 — "func TestPlanSendsOnlyGets(t *testing.T) {"
  evidence: .abcd/work/decisions/dcn-2610282341088544-step-11-of-spc-2610120557432880-the-live.md:6 — "abcd embargo publish showed a live plan on 17 October"
- ac-8 — MET_WITH_CONCERNS: An advisory whose fix is in no release tag is left out with the reason 'no release tag', and one with no resolved record anywhere is left out with 'not resolved in < release>'. Concern: this diverges from the literal criterion. An advisory whose record is NOT resolved in the release is still planned when its private record is resolved in the store with a full fix commit (store_evidence). This is deliberate, since the security rule keeps the record out of every committed tree, but it contradicts the wording.
  evidence: internal/core/embargo/plan.go:582 — "return "", "its fix is in no release tag yet", nil"
  evidence: internal/core/embargo/plan.go:276 — "} else if fix, storeRec, why = s.storeFixCommit(a.IssueID); why != "" {"
  evidence: internal/core/embargo/plan_test.go:260 — "func TestUnresolvedRecordLeftOut(t *testing.T) {"
  evidence: internal/core/embargo/plan_test.go:565 — "func TestStoreRecordedFixPlans(t *testing.T) {"
- ac-9 — MET_WITH_CONCERNS: Publish rebuilds the plan and sends nothing if the digest does not match. It then publishes the advisories in id order, the first as a canary: PATCH the versions, GET, PATCH to published, GET. It stops at the first mismatch, and finally checks that the draft and triage listings name none of the planned advisories. All of this is tested against a fake gh. Concern: the GitHub-side half rests only on tests. The decision record gives a live plan but no live approved publication, so the canary and no-draft-left checks have never met the real API.
  evidence: internal/core/embargo/publish.go:103 — "if req.Approve != plan.Digest {"
  evidence: internal/core/embargo/publish.go:126 — "if why := publishOne(s, st.Repo, e); why != "" {"
  evidence: internal/core/embargo/publish.go:143 — "for _, state := range []string{"draft", "triage"} {"
  evidence: internal/core/embargo/publish_test.go:175 — "func TestCanaryFirstThenRest(t *testing.T) {"
  evidence: internal/core/embargo/publish_test.go:217 — "func TestCanaryMismatchStops(t *testing.T) {"
- ac-10 — MET: The page has a 'What stays the person's act' section. It lists fixing and landing the fixes, requesting a CVE, putting back redacted records and restoring records, and says abcd closes or publishes only on the person's approval of what it showed them. A test asserts each phrase.
  evidence: commands/embargo.md:291 — "## What stays the person's act"
  evidence: commands/embargo.md:302 — "- requesting a CVE for an advisory, on GitHub;"
  evidence: commands/embargo.md:307 — "abcd closes or publishes an advisory only on the person's own approval of what"
  evidence: internal/surface/cli/embargo_page_test.go:16 — "func TestEmbargoPageNamesThePersonsActs(t *testing.T) {"

Gap audit:
- honoured:
  - Private advisories become records only this machine holds, never in the public ledger
    evidence: internal/core/embargo/store.go:314 — ""worktree", "add", "--detach""
    evidence: internal/core/embargo/intake_test.go:175 — "func TestIntakeLeavesPublicLedgerUntouched(t *testing.T) {"
  - Each verdict is recorded with who gave it and when
    evidence: internal/core/embargo/triage.go:203 — "a.Verdict, a.DecidedBy, a.DecidedAt = &verdict, &by, &now"
  - Outbound check refuses ids, reserved CVE ids, titles and the store location
    evidence: internal/core/embargo/check.go:29 — "KindAdvisoryID = "advisory_id""
  - One explicit yes publishes a canary first, then the rest, then confirms none is left in draft
    evidence: internal/core/embargo/publish.go:143 — "for _, state := range []string{"draft", "triage"} {"
- diverged:
  - The patched version is worked out from the release holding the fix: delivered as the first PUBLISHED release (on origin, with a GitHub release that is neither a draft nor a prerelease), not the first tag
    evidence: internal/core/embargo/releases.go:14 — "tag in this checkout counts only when origin holds the tag at the same"
  - An advisory not resolved in the release is left out: delivered as planned when the store's private record is resolved with a full fix commit
    evidence: internal/core/embargo/plan.go:276 — "s.storeFixCommit(a.IssueID)"
  - Closing posts the approved explanation: delivered at the head of the advisory description because GitHub has no advisory comment, and the placement still waits on the product thinker's confirmation
    evidence: internal/core/embargo/close.go:86 — "spec's design and is pending the product thinker's confirmation"
  - The check refuses an unpublished advisory's title: titles under 12 characters, closed advisories' titles and public-track titles are not refused in a range or file check
    evidence: internal/core/embargo/check.go:645 — "if (a.ClosedAt == nil || !dropClosedTitles) && (a.Verdict == nil || *a.Verdict != VerdictPublicTrack) {"
- missing:
  - A live demonstration of closing an advisory on GitHub with the explanation the person approved; only a refusal on a record whose verdict does not close the advisory has run live
    evidence: .abcd/work/decisions/dcn-2610282341088544-step-11-of-spc-2610120557432880-the-live.md:6 — "Its show-and-approve path has no live run yet"
  - A live demonstration of an approved publication (canary goes live, then the rest); the live record shows a plan only
    evidence: .abcd/work/decisions/dcn-2610282341088544-step-11-of-spc-2610120557432880-the-live.md:6 — "abcd embargo publish showed a live plan on 17 October"

Scope-condition dispositions:
- cond-2610120557436663 — survived: Every request is the person's own gh against the fixed github.com host, origin is parsed by a github.com-only parser, and intake, triage, check and plan ran live on this github.com repository.
  evidence: internal/core/github/github.go:37 — "const Host = "github.com""
  evidence: internal/core/github/github.go:165 — "append([]string{"api", "--hostname", Host}, args...)"
- cond-2610120557430932 — survived: The store is a detached worktree under this machine's abcd home and is never committed or pushed. The check reads only stores under this home, so nothing assumes the records are shared with another machine.
  evidence: commands/embargo.md:13 — "machine's worktree store (`~/.abcd.noindex/worktrees/<root-sha>/embargo/`),"
  evidence: internal/core/embargo/store.go:314 — "gitutil.Run(s.Checkout, "worktree", "add", "--detach", "--", s.Root, base)"
- cond-2610120557431051 — narrowed: Release tags are still strict vMAJOR.MINOR.PATCH, but a tag containing the fix does not on its own make the fix released. The tag must be on origin at the same object, and GitHub must report a release for it that is neither a draft nor a prerelease.
  narrowing: A fix counts as released only once a vMAJOR.MINOR.PATCH tag that contains its commit is on origin at the same object and GitHub reports a release for that tag that is neither a draft nor a prerelease. A tag held only locally, or one whose release was never published, does not count.
  evidence: internal/core/embargo/releases.go:14 — "tag in this checkout counts only when origin holds the tag at the same"
  evidence: commands/embargo.md:201 — "**What counts as a published release.** A `vMAJOR.MINOR.PATCH` tag counts only"
<!-- abcd-review-end receipt=rcp-bb21ac30f744 -->

## Decisions

- 2026-10-09, product thinker (recorded on [iss-2610090703514921](../../../work/issues/resolved/iss-2610090703514921-a-drain-reads-only-the-issue-ledger-so.md)): private security advisories get a separate security-advisory run, distinct from the general drain, whose rule ([adr-2609291342092738](../../decisions/adrs/2609291342092738-a-drain-takes-an-issue-alone-only-when.md)) stays as it is; its design starts from a primary-source state-of-the-art check and its requirements from the hand-run; an advisory's record stays out of the public ledger until its fix lands; the fixes are held on local branches and land together just before the release that publishes the advisories.
- 2026-10-11, product thinker: the security-advisory run is built before this release.
- 2026-10-11, product thinker: it is its own command, named `embargo`, not a mode of `drain`.
- 2026-10-11, product thinker: every security finding is judged by [abcd-fixes-what-it-adds](../../principles/abcd-fixes-what-it-adds.md); the "not abcd's" verdict is that rule's closing route.
- 2026-10-11, state of the art (sources: GitHub's repository security advisories REST API, its publishing and temporary-private-fork documentation; the OpenSSF vulnerability-disclosure guide for maintainers; the CERT Guide to Coordinated Vulnerability Disclosure; the curl, Go, Kubernetes and Rust security processes): fixes are made on local unpushed branches rather than GitHub's temporary private forks, which skip CI, branch protection and the merge queue; the time from push to publication is kept short (curl: no more than 48 hours); triage ends in one of four verdicts; a CVE is requested once the fix is ready, with a person confirming; the patched version is computed from the first release tag containing the fix and checked against GitHub's range rules; an outbound check covers every public artefact; publication takes one human approval per release, one advisory first as a canary, and a final check that no drafts remain; redacted records are restored after publication with a three-way check; and a broken embargo (a leak, a second report, exploitation) goes straight to landing and publication.
- 2026-10-11, agent, under [script-first-mvp](../../principles/script-first-mvp.md): this release ships the smallest working rung (intake, triage, check, publish). The two hand-runs were the script stage that found the contract, so this rung is the Go absorption of what they proved. Running fix lanes, landing the held branches with redaction, requesting a CVE, restoring redacted records, closing advisories on GitHub, and the broken state are later rungs, each its own record once this one ships. (Closing advisories on GitHub moved into this rung by the product thinker's ruling of 2026-10-12, below.)
- 2026-10-11, agent, settled without a question: GitHub's critical severity maps to the ledger's critical. The ruling mapped high, medium and low; the ledger's own critical is the one defensible answer for the level above high.
- 2026-10-11, agent, settled without a question: an advisory with no severity is listed for the person and not recorded, because a guessed severity would decide whether the release cut's open-findings guard counts it.
- 2026-10-11, agent: the three scope conditions are proposals for the interview, not the product thinker's answers. The Mechanism is left for the product thinker.
- 2026-10-12, product thinker: the first rung is adopted as drafted, with what it leaves out, with one change: after a "not abcd's" or "not a vulnerability" verdict, abcd itself posts the closing explanation on the private advisory on GitHub (the advisory updated to closed with the approved text), on the person's explicit approval of the drafted text. Closing is in this rung; the other later rungs stay out.
- 2026-10-12, product thinker: the release waits for this rung; it targets the next release.
- 2026-10-12, product thinker: this intent refines [itd-149](../drafts/itd-149-abcd-handles-inbound-security-advisories.md), the broader draft of the same loop, which stays a draft for the rungs beyond this one; it answers [iss-2610090703514921](../../../work/issues/resolved/iss-2610090703514921-a-drain-reads-only-the-issue-ledger-so.md).
- 2026-10-28, technical facilitator, on the product thinker's delegation to decide what blocks an intent so it can be built: the Mechanism, the one prompt left unanswered (readiness failed on it), is worded from the press release, Why This Matters and criteria 1, 6 and 7, and stays open to the product thinker's review.

## References

- [iss-2610090703514921](../../../work/issues/resolved/iss-2610090703514921-a-drain-reads-only-the-issue-ledger-so.md): the finding this intent answers; the change that ships it resolves it.
- [itd-149](../drafts/itd-149-abcd-handles-inbound-security-advisories.md) and its [2026-08-27 pilot note](../../research/notes/2026-08-27-security-advisory-handling-pilot.md): the broader, earlier draft of the same loop.
- [adr-44](../../decisions/adrs/0044-remote-mutation-and-caller-identity.md): a remote write happens only through the caller's own identity; publication follows it.
- [adr-2610031751065746](../../decisions/adrs/2610031751065746-the-worktree-store-lives-under-the.md): where the private records' worktree lives.
