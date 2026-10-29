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

<!-- abcd-review: OWED receipt=rcp-6083de2cef52 -->
Fidelity review OWED (receipt rcp-6083de2cef52).
<!-- abcd-review-end receipt=rcp-6083de2cef52 -->

## Grounds

- pursued: it clears a major security finding that blocks the release cut, and makes drift between managed repositories visible; it is shown wrong if an audit reports clean while a baseline setting is off, or an apply leaves a pull request unmergeable
