---
id: itd-2610161710315026
slug: a-person-who-lets-abcd-s-agents-run
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Unattended runs reach only a declared list of hosts

## Press Release

> A person who lets abcd's agents run unattended can limit which websites and services those agents reach, so a mistake cannot send work to the wrong place. The limit is a declared list of hosts per project. It narrows mistakes; it does not stop code being sent elsewhere, because a run allowed to reach the forge can still push to any repository its credentials reach.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [security-isolation SOTA note](../../research/notes/2026-10-23-security-isolation-sota.md).

- **The fence already carries a host allowlist.** sandbox-runtime denies all network by default, matches `allowedDomains` by name with wildcards and ports, checks `deniedDomains` first, and on Linux removes the network namespace so every connection must cross its proxies (sandbox-runtime README).
- **A name allowlist needs a resolved-address check**: srt resolves each permitted name and drops loopback, link-local, metadata and the host's own addresses, because "whoever controls a permitted name's DNS... controls what it resolves to"; IP-literal entries skip the check, and domain fronting is a named bypass (same README).
- **DNS has been left outside an egress allowlist before, and the fence leaves it open on two of three platforms**: Docker Sandboxes "enforces an HTTP/S-only egress allowlist but does not apply it to DNS resolution" (CVE-2026-12039, a non-default configuration); srt's README says that on Windows "DNS resolution via the system resolver is not fenced" and that "This mirrors the macOS behaviour", so on macOS and Windows a lookup through the system resolver still leaves the fence and data can leave in DNS labels; only Linux, where srt removes the network namespace, closes the channel by construction (CVE-2026-12039; README).
- **Proxies filter by hostname and do not inspect TLS**; a repository's files cannot widen an admin-required sandbox, and `strictAllowlist` is honoured only from user, managed or command-line scope, after which a repository's `allowedDomains` entries are ignored (Claude Code sandboxing).
- **Report what was blocked**: Copilot's firewall writes "the blocked address and the command that tried to make the request" into the pull request, applies to the agent's shell but "does not apply directly to" MCP processes, and "should not be considered a comprehensive security solution" (Copilot agent firewall).
- **The press release's limit is the vendors' limit**: "Any approach that allows network egress can still leak data the agent can read" (Claude Code sandbox environments).

**Path 1: the declared list is the fence's `allowedDomains` for the launch**, with IP literals refused, every denied host and its command written into the run's receipt, and the receipt naming that on macOS and Windows the system resolver sits outside the fence. Who may put a host on the list is the first question below: the vendors' strict shapes take the list from the user or managed layer and ignore a repository's entries (Claude Code's `strictAllowlist` and admin-required sandbox), or merge a repository's additions with organisation entries it cannot remove (Copilot); a project file that declares hosts a user layer can only narrow would let a checked-out project widen its own reach. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Who may widen the list: Claude Code's strict allowlist takes the list from the user, managed or command-line layer and ignores a repository's entries; Copilot merges a repository's additions with organisation entries it cannot remove. Does a project file here declare, propose, or neither?
- (SOTA 2026-10-23) The system resolver answers lookups from inside the fence on macOS and Windows, so data can still leave in DNS labels there; only Linux closes the channel. Does the receipt name it per platform, and does the press release's limit?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
