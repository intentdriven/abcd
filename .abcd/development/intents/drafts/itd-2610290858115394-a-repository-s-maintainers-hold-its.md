---
id: itd-2610290858115394
slug: a-repository-s-maintainers-hold-its
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610150602547271]
related_issues: [iss-2610290730539347]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A repository's maintainers hold its quality gate: a contributor's change waits for a maintainer's approval, and only a maintainer starts a release, adopts an intent or makes private work public

## Press Release

> **In a repository abcd manages, the people with write access are its maintainers, and abcd keeps every consequential act in their hands, on a free GitHub plan or a paid one, with no team to set up.** A maintainer is anyone with Write, Maintain or Admin access to the repository; anyone with Read or Triage access is a contributor. A contributor's change merges only once a maintainer has approved it. A maintainer's own change merges without approval while the repository's technical facilitator is set to autonomous, and waits for approval while it isn't. Only a maintainer starts a release, creates a version tag, adopts an intent, or makes a private lab or idea public. When a contributor tries one of those acts, abcd stops and says why before any work is wasted. abcd reads who is a maintainer from GitHub each time. Where GitHub cannot stop someone with full control of the project from switching a protection off, abcd checks those settings on every run and reports any change. It proposes the settings a repository needs and never changes them itself.
>
> "I gave two newcomers triage access and kept write access for the three of us who maintain the project," said Kira, an open-source project lead. "Their pull requests wait for one of us; ours go straight through when the facilitator is abcd. When someone switched a protection off on GitHub, abcd told me at the next run."
>
> "I tried to start a release out of curiosity," said Henry, a new hire with triage access. "abcd said only a maintainer can start one, and said who counts as a maintainer here. Nothing was half done."

## Why This Matters

abcd has no notion of who may do what in a repository. Today any account with write access can start a release cut, create the next version's tag before the release does (which breaks that release), move an intent into planned, or publish private material into the committed record. The organisation's default gives every member write access, so a second member joining becomes able to do all of it at once. The code-owners file that held a few paths named one person, which the product thinker's rule that no person is named in tracked files no longer allows. GitHub's own roles don't match the rule the product thinker set (an admin counts as a maintainer, and a maintainer's own change waits unless the facilitator is autonomous), and the stronger GitHub controls need a paid plan. abcd must work the same on both.

## Mechanism

We expect write access to be enough to define the maintainer, with no team, because GitHub already lets only accounts with write access merge, and counts approvals only from such accounts ([about protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)). So on any plan "merged" already means "a maintainer let it in". What GitHub cannot hold (a maintainer's own change waiting while the facilitator is a person, and a contributor's change needing an explicit approval before abcd arms its merge) sits in abcd's own merge path, which itd-2610150602547271 already makes conditional on the facilitator's mode. A gated act completed by an account without write access, or a protection switched off without abcd reporting it at the next run, shows the claim wrong.

## Scope Conditions

- The repository is on GitHub, on any plan; the core stays host-agnostic, and another forge supplies its own role lookup.
- Who is a maintainer is read live from the forge for the token's own login, never from git's configured name or email, and never from a list in the repository.
- abcd enforces only in its own paths (its verbs, its merge path, its guard). A maintainer acting directly on the GitHub website is bound by GitHub's settings alone.
- abcd proposes settings changes and never applies them: organisation and repository settings change only on the forge website, by a person.

## Acceptance Criteria

- **Given** an account with Read or Triage access, **when** it runs a gated act (start a release cut, reserve a release, promote a private lab or ideate verdict to public, adopt an intent, lift an embargo), **then** abcd refuses before writing anything, names the act, says that it needs Write, Maintain or Admin access, and names the GitHub setting that is the real boundary.
- **Given** an account with Admin access, **when** it runs a gated act, **then** abcd treats it as a maintainer.
- **Given** an account whose repository role is a custom role, **when** abcd reads its role, **then** it is judged by the custom role's base role; **given** a pull request author who is not a collaborator at all (a fork's author, for whom the forge reports no role), **then** abcd treats them as a contributor.
- **Given** the forge requires no approval on its own (the live setting since the code-owners file was removed), **when** a contributor's pull request reaches abcd's merge path, **then** abcd itself holds it until a maintainer has approved; the forge only guarantees that whoever merges has write access.
- **Given** a contributor's pull request, **when** abcd's merge path is asked to queue it or arm auto-merge, **then** it does so only once a maintainer has approved the pull request, whatever the facilitator's mode.
- **Given** a maintainer's own pull request, **when** the facilitator's mode is autonomous, **then** abcd's merge path queues it with no approval asked, subject to the review-receipt condition the facilitator's merge gate already sets for autonomous mode; **when** the mode is a person, **then** abcd arms nothing until the technical facilitator has approved, and then arms it.
- **Given** the forge cannot be reached, **when** a gated act's check runs, **then** abcd says the caller's role is unverified and never reports it as verified; a point where abcd itself is the enforcement (the merge path) arms nothing.
- **Given** the live settings differ from what the repository declares (a base permission above Read, a version-tag rule that lets any writer create tags, a required check whose source app is not pinned, a protection switched off since the last run), **when** abcd runs its settings check, **then** it reports each difference with the exact setting and the forge page that changes it, and changes nothing.
- **Given** the docs check's ban on the word "maintainer", **when** this intent ships, **then** the ban is lifted and "maintainer" is defined in the glossary as an account with Write, Maintain or Admin access.

## Decisions

The product thinker decided on 2026-10-29:

- A maintainer is anyone with Write, Maintain or Admin access to the repository; a contributor has Read or Triage. Admins are maintainers. This replaces two earlier rulings the same day: that an admin is not a maintainer unless a maintainer allows it, and that a maintainer is only GitHub's Maintain role.
- No team holds the maintainers. The owners team made that morning is deleted, and so is the code-owners file. The accepted consequence is that on GitHub no change needs an approval from anyone in particular; only maintainers can merge.
- A contributor's change needs a maintainer's approval. A maintainer's own change merges without approval only while the technical facilitator is autonomous. The product thinker never has to approve.
- Only a maintainer can create a version tag, accepting that the automatic release then needs its own GitHub App credential as the tag rule's one bypass, a kind of setup turned down once before (adr-2610281718537125).
- Adopting an intent (draft to planned) is a maintainer-only act.
- When a contributor tries a maintainer-only act, abcd refuses and says why; GitHub stays the real boundary.
- abcd works the same on free and paid plans. Stronger protection a paid plan offers is proposed, never required.
- The word "maintainer" returns to abcd's vocabulary for this role.

Technical choices with one defensible state-of-the-art answer, recorded rather than asked:

- The role is read from the forge's `role_name` for the caller, never the legacy `permission` field, which reports Maintain as write ([collaborators API](https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2022-11-28#get-repository-permissions-for-a-user)), and it is keyed on the token's login.
- A client-side check is a courtesy and says so: enforcement belongs on the server ([Pro Git: hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks)). abcd's wording names the GitHub setting that is the boundary, as the reservation page already does.
- Version tags: a tag ruleset where "only users with bypass permissions can create tags" ([available rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets)).
- Each required check is pinned to the app that must post it, so a writer cannot post a passing status under its name (same source).
- Settings drift is detected and proposed, never applied, extending the ruleset mirror and its drift check (itd-92) to the base permission, environments and team access, in the manner of OpenSSF Allstar's issue mode ([Allstar](https://github.com/ossf/allstar)).
- Never use the ruleset "exempt" bypass type, which skips enforcement silently ([GitHub changelog](https://github.blog/changelog/2025-09-10-github-ruleset-exemptions-and-repository-insights-updates/)).
- No two-person rule for releases while a project has one or two maintainers. Projects of that size self-approve releases, and the release approval this project removed had not been an independent check (adr-2610281718537125).

## Open Questions

- Whether the automatic release's tag push can bypass a tag rule as the workflow's own token is contested in public reports. The plan assumes a GitHub App as the bypass actor until a live test says otherwise. UNCONFIRMED.
- Publishing a security advisory is limited to admins by GitHub itself ([advisory permissions](https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/permission-levels-for-repository-security-advisories)), so abcd's embargo check can only be a courtesy for maintainers below Admin.
- The lane writing the GitHub settings audit captured iss-2610290755587946 for the code-owners side. With no code-owners file in this design, that capture takes this intent's definition of a maintainer. <!-- record-lint: forward-looking -->

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
