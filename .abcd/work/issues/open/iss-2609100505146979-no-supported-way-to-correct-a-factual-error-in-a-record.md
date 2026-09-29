---
schema_version: 1
id: "iss-2609100505146979"
slug: "no-supported-way-to-correct-a-factual-error-in-a-record"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (intent, decide, capture) / conventions"
related_intents: [itd-2609150819439571]
deferred_after: v0.11.1
deferral_reason: "Promoted to itd-2609150819439571 (draft), and a promoted issue keeps its folder until the intent ships. The product thinker's ruling M25 of 2026-09-23: a convention first, then possibly a verb, through the draft's planning interview. Owed: that interview, which opens on one question: per record family, where does a correction go, what does it carry, and why does the original stay?"
remedy: "Waits on the itd-2609150819439571 planning interview (M25, convention first): take into it one candidate, an appended dated Errata section on any record family that quotes the wrong sentence, states the correction and says why the original stands, never editing the original; if the interview places it per family instead, the same three fields go into each family's existing append-only section (Audit Notes for a shipped intent) and a new dated section elsewhere; either way a record-lint check that an erratum carries all three fields proves it, and a verb follows only if the interview asks for one."
---

abcd has no supported operation for correcting a factual error inside a durable record, and no documented convention saying what to do instead. The record is deliberately not rewritten, which is right, but "not rewritten" and "wrong" are different states and only the first has a mechanism.

What abcd does support. A changed decision supersedes: intents carry `superseded_by` and move to `superseded/`, ADRs carry `status`, `supersedes` and `superseded_by`. A finding about a shipped intent appends to its Audit Notes, which `intent audit ingest` writes and which are dated and additive. An issue goes to the ledger via `capture` and is routed at triage. Those cover a decision that changed, a promise that shipped narrower, and a defect.

What is missing is the fourth case: a record that states something untrue about another record, or about the code, where nothing has changed and nothing is defective. The record is simply wrong. Two instances, found in one pass over a managed repository:

- A shipped intent's Audit Note quoted a ratified ADR as requiring load and eviction events "each carrying their reason". The ADR says "a load or eviction event with its reason", a collective phrase. The divergence the note declared against the ADR does not hold on that wording, and a real divergence against the same ADR (record-kind naming) went unnoticed because the misquote looked like it had already covered the ADR.
- A second shipped intent asserted that a sibling intent's criterion "was adopted as diverged" on a date. No adoption had occurred, and the sibling's own notes ask for exactly that decision. A reader taking the two in the wrong order would conclude a gate was closed that is open.

Neither is a superseded decision, a shipped-narrower promise, or a bug. Both are errata. With no verb and no convention, the options are to rewrite the record (which the conventions forbid, and which erases the evidence that the audit reasoned from a misreading), to leave it standing and hold the correction out of band (which leaves the false sentence authoritative), or to invent a local convention per repo, which is what happened: a dated correction appended to the Audit Notes, chosen because that section is already append-only and dated.

Needed, in rough order of cost: a documented convention for errata on a durable record, saying which section takes the correction for each record family and what a correction must carry (date, what it corrects, why the original stands). Then, optionally, a verb (`intent errata`, or an `--errata` mode on the existing audit-notes writer) so the correction is a recorded operation rather than hand-appended prose, and so a reader can tell a correction from a finding. The Audit Notes route only exists for shipped intents; an ADR or a spec carrying a wrong statement has no equivalent section at all, which is the sharper half of the gap.

## Grounds

- pursued: we expect errata to be a fourth terminal disposition appended to a record rather than an edit of it, because the record families are append-only by conviction and a correction that rewrites history is indistinguishable from the error it corrects; it is shown wrong if appended errata prove unreadable in practice and readers keep acting on the uncorrected text

## Deferral 2026-09-29

Deferred past v0.11.1: Promoted to itd-2609150819439571 (draft), and a promoted issue keeps its folder until the intent ships. The product thinker's ruling M25 of 2026-09-23: a convention first, then possibly a verb, through the draft's planning interview. Owed: that interview, which opens on one question: per record family, where does a correction go, what does it carry, and why does the original stay?

## Remedy grounds (2026-09-29)

Why: ruling M25 (2026-09-23) orders a convention before any verb, and the three fields are what let a reader tell a correction from a finding. SOTA check: RFCs do not change after publication; errors are recorded as separate errata with a status of Reported, Verified, Rejected or Held for Document Update (https://www.rfc-editor.org/errata-definitions/, read 2026-09-29). Rejected: an in-place edit pointing at git history, which erases the evidence an audit reasoned from and breaks the append-only record families.
