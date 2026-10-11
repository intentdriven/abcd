---
id: itd-2610111707085098
slug: the-status-line-should-show-which
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610111701536641]
origin: extracted-from-record
production_mode: hand-written
---

# Each session takes a persona name, shown in the status line and the session list

## Press Release

A person runs three sessions in one repository. Each opens with a name from the repository's personas (Alice, Bob, Carol), shown in its status line and in the list other sessions use to message it, so the person can tell them apart at a glance and say "ask Bob" rather than "ask abcd-a5".

> _Draft seed, written from the product thinker's rulings of 2026-10-11 below; the planning interview settles the final text._

## Why This Matters

Graduated from `iss-2610111701536641`: The status line should show which session it belongs to, the short name peer sessions use to message each other (for example abcd-60 or abcd-a5), so a person running several sessions in one repository can tell them apart and address one. Checked against the host's status-line documentation (code.claude.com/docs/en/statusline, 2026-10-11): the payload abcd's statusline verb reads on stdin carries session_id (a UUID) and session_name, which is present only when the person named the session (--name or /rename) or the host generated a title, but never that default short name, and no environment variable or file exposes it. The short name cannot be derived from session_id: this session is abcd-0b while its id begins d3616efc. internal/core/statusline/payload.go reads neither field today. Related: iss-2609291032194222, whose session-bound remedy also reads the payload's session id.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

- 2026-10-11, product thinker: abcd names a session automatically, through its own session-start hook, rather than leaving the person to type a rename. The host's hook output carries a session title for this (code.claude.com/docs/en/hooks, checked 2026-10-11); nothing under `~/.claude` is touched.
- 2026-10-11, product thinker: the names are the repository's persona names, in order: Alice, Bob, Carol and on through the list, then Alice Two, Bob Two and so on. Reason: a familiar name is easier to find than a code, and the chance of reading the name as a role is manageable.
- 2026-10-11, product thinker: after a crash that ends every session, naming starts afresh from Alice.

## Open Questions

1. When one session crashes while others keep running, how does abcd tell that its name is free again? Proposed for the technical facilitator: the hook records the host process beside the name, and a name whose process is gone is free.
2. Does a name the person gave by hand (`--name` or `/rename`) always win, and can the hook see one before it names the session? To be proved in a live session.
3. Does a resume or a compaction keep the session's name? To be proved in a live session.
4. With two repositories open, an Alice in each: does the session list need a repository prefix to tell them apart?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
