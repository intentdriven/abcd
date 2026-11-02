---
id: itd-2611011325430933
slug: a-person-can-set-an-old-linux-pc-or-mac
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A machine the person owns checks agents' work independently

## Press Release

> A person can set an old Linux PC or Mac on their network as an independent checker. It fetches each pushed change, runs the gates the default branch defines, and reports under an identity of its own that no agent holds, so the forge can require its check and no agent can fake it, even off grid.

## Why This Matters

> A person who wants their agents' work checked by something other than the machine the agents run on has two choices today: rent hosted CI, or trust the agents' own word. Many have an old PC or Mac in a drawer. A self-hosted GitHub Actions runner looks like the answer but is not one: for a pull request the workflow it runs comes from the pull request's own commit, so an agent that can push decides what the runner checks, and a required check can only be pinned to "GitHub Actions", not to the box. A small checker that pulls each change with a read-only key, runs the gates as the default branch defines them, and reports as a GitHub App whose private key lives only on that machine gives a check the forge can require from that one source (rulesets' integration_id) and no agent can fake. It also works off grid: it keeps checking and reports when it is back online.

## Mechanism

> We expect the check to be independent because nothing the agents hold reaches it: they cannot write to the box, its gate definition comes from the default branch rather than the change, the gates run under an account of their own that cannot read the App's key, and the forge accepts the required check only from that App, whose key a separate signer on the box holds. The gates still compile and run the pushed change's code (its tests, its init functions), so the key's isolation from the gates, not the gate definition alone, is what keeps the check unforgeable. It would be shown wrong by any path through which a pushed change alters what the box runs, or by the forge accepting the check from another source.

## Scope Conditions

- A machine the person owns on their network: an x86-64 or Arm Linux PC (the longer support runway), or a Mac for darwin-specific tests (Go 1.27 needs macOS 13 or later, which few Intel Macs run).
- Repositories on GitHub for the pinned check; fully offline use (a local forge mirror) is a later option, named, not built here.
- The box runs the person's and their agents' changes only; a change from an outside contributor is never fetched to it.
- The App is registered by the person in one browser step (dcn-2610180720389628); the box's key never leaves it, and the account that runs the gates cannot read it (a separate operating-system account, or a virtual machine, with the signer outside it).

## Acceptance Criteria

- Given a box the person has set up, when an agent pushes a change, then the box fetches it with a read-only key, runs the gates the default branch defines, and posts a check run as its own App.
- Given a ruleset requiring that check from that App, when any other source posts a check of the same name, then the forge does not count it.
- Given a pushed change that edits the gate definition, when the box runs, then it runs the default branch's definition, not the change's.
- Given a pushed change whose tests try to read the App's key or post a check themselves, when the box runs its gates, then the gates' account cannot read the key, and only the signer, after the gates' result is in, posts the check.
- Given the box loses its connection, when it is back online, then it posts every result it owes, and the landing waits for it rather than proceeding without it.
- Given the report of itd-2611011325403127, when a change was checked on the box, then the report names the box, its operating system and its time, and the hosted minutes it saved.

## Open Questions

1. Does the box replace the merge queue's run too (it can test the merged tree if it fetches the queue's ref), or only the pull-request run?
2. Is abcd's own repository the first user, with an old Mac covering the darwin tests that a Linux box cannot?
3. How does abcd tell the person the box is down or behind, and how long does a landing wait for it?

## Decomposition

Hand-run four-piece table, 2026-11-01, one of five drafts filed together from the investigation of hosted CI and local compute costs (a private ideate verdict, reframed). The product thinker ruled on 2026-11-01 that the work is filed as separate intents in this order: measure (itd-2611011325403127), fewer tests (itd-2611011325421304), self-attestation (itd-2611011325428560), an independent checker (itd-2611011325430933), and the check profiles new repositories are offered (itd-2611011325449761), the fifth at the product thinker's own request the same day.

| Part | Kind | Home |
|---|---|---|
| the pull-only checker, its App identity, the pinned check, off-grid catch-up | capability | this intent |
| a check counts only from the source the forge pins it to; the box never runs a change's own gate definition | trust rule | already standing in iss-2610290913450522's fix (required checks pinned to their app); cited, no new ADR |
| the daemon, the deploy key, the App setup | plumbing | a brief chapter for the checker |
| (none) | stance | none |

Verdict: FILE-AS-IS. Typed links: this intent answers the open question "the self-hosted runner route" of itd-2610261025516272 (`refines`, advisory) and resumes the route itd-2610120738075547 deferred ("a machine the person runs").

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
