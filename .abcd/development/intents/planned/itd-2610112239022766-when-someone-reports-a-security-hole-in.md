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
target_release: next
---

# Private security reports stay private until their fix ships

## Press Release

> When someone reports a security hole in your project privately, abcd now keeps it private until the fix ships and then publishes it with you. abcd embargo brings each private security advisory from GitHub into a private record on your own machine that never enters the public ledger, records your verdict on it and who gave it (a vulnerability to fix, not abcd's problem, not a vulnerability, or fine to handle in public), closes on GitHub, with the explanation you approved, an advisory you judged not abcd's or not a vulnerability, checks every commit message, pull-request body and record you are about to make public for anything that would give the advisory away, and, once the release carrying the fixes is cut, shows you the publication plan with each patched version worked out from the release that holds the fix. On your one explicit yes it publishes a single advisory first, checks it went out as planned, then publishes the rest and confirms none is left in draft.
>
> "Reports came in through private reporting the week before our release," said Kira. "abcd turned them into records only my machine holds, closed the one that was not ours with the explanation I approved, stopped me when a commit message named one of them, and on release day showed me the patched version for each. I said yes once, watched the first advisory go live, and the rest followed."

## Why This Matters

A private security report is the one piece of work abcd's records cannot hold today. The issue ledger is public, so an advisory copied into it discloses the hole before the fix exists; left out of it, the advisory is named by no record, and nothing in abcd shows whether it was triaged, fixed or published. Two hand-runs did the work by hand: private records kept in an uncommitted worktree, fixes held on local branches, every advisory id searched for by hand before the push, and patched versions typed, one of them wrongly at first. Each of those hand steps is a way for an advisory to leak early or be published wrong, and this rung replaces the ones that leak or mislead with commands: the private record, the verdict, closing what is not a vulnerability to fix, the outbound check and the publication.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

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

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-09, product thinker (recorded on [iss-2610090703514921](../../../work/issues/open/iss-2610090703514921-a-drain-reads-only-the-issue-ledger-so.md)): private security advisories get a separate security-advisory run, distinct from the general drain, whose rule ([adr-2609291342092738](../../decisions/adrs/2609291342092738-a-drain-takes-an-issue-alone-only-when.md)) stays as it is; its design starts from a primary-source state-of-the-art check and its requirements from the hand-run; an advisory's record stays out of the public ledger until its fix lands; the fixes are held on local branches and land together just before the release that publishes the advisories.
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
- 2026-10-12, product thinker: this intent refines [itd-149](../drafts/itd-149-abcd-handles-inbound-security-advisories.md), the broader draft of the same loop, which stays a draft for the rungs beyond this one; it answers [iss-2610090703514921](../../../work/issues/open/iss-2610090703514921-a-drain-reads-only-the-issue-ledger-so.md).

## References

- [iss-2610090703514921](../../../work/issues/open/iss-2610090703514921-a-drain-reads-only-the-issue-ledger-so.md): the finding this intent answers; the change that ships it resolves it.
- [itd-149](../drafts/itd-149-abcd-handles-inbound-security-advisories.md) and its [2026-08-27 pilot note](../../research/notes/2026-08-27-security-advisory-handling-pilot.md): the broader, earlier draft of the same loop.
- [adr-44](../../decisions/adrs/0044-remote-mutation-and-caller-identity.md): a remote write happens only through the caller's own identity; publication follows it.
- [adr-2610031751065746](../../decisions/adrs/2610031751065746-the-worktree-store-lives-under-the.md): where the private records' worktree lives.
