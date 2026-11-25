---
id: itd-2611251831138387
slug: confirmed-finding-is-always-captured
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: additive
related_issues: [iss-2611251828531764]
origin: extracted-from-record
production_mode: hand-written
---

# A confirmed finding is always captured, and abcd holds a session to it

## Press Release

> A finding met while working is tested and captured before the work moves on, and abcd holds every session and sub-agent to it. Maya, an autonomous-development practitioner, leaves a run overnight; whatever its agents meet on the way, a defect, a drift, a page that says the wrong thing, is tested, and once confirmed it is in the record by morning, fixed or not. An agent that says it found something and captured nothing is told so before it stops, and a run does not end, land, or hand over with a finding in nobody's record. A security finding goes to the private store and never to the public one.

## Why This Matters

Graduated from iss-2611251828531764, which holds the observation: on 2026-11-25 a session kept several confirmed or confirmable findings in a local log and filed them only when told to. The product thinker stated the rule that day as a basic one with no exceptions, and that abcd must enforce it. The deferral discipline (itd-2610141956319695) holds the findings a review confirms; nothing holds a finding met on the way.

## Mechanism

We expect a list abcd keeps to hold where a rule in words did not, because an agent names a finding in the act of reporting it and the stop is a place abcd already stands: A finding declared and left in none of its states is then a fact the hook reads, not a judgement. Shown wrong if a session that keeps the list still ends with a finding in its transcript that no record or refutation answers, counted by a reader on a sample of sessions.

## Scope Conditions

- Holds for a host that runs a hook as a session and a sub-agent stop and shows it the closing text.
- Holds in a repository abcd manages whose rules are not switched off.
- Holds for a finding an agent declares; one it never mentions leaves nothing for a hook to read.

## Acceptance Criteria

- **Given** a sub-agent's brief in a managed repository, **when** it starts, **then** it is told to end with its findings in a fixed block, each with one state: captured under a named record, fixed or not; refuted with the test that did it; kept in the private store; or untested.
- **Given** a sub-agent that stops, **when** its closing text declares a finding in no state, or untested, **then** abcd adds it to the session's list and the starting session is told.
- **Given** a session with a finding on its list, **when** it stops, **then** the stop is held once, naming each finding and the act that clears it; a second stop goes through and the list stays.
- **Given** a list that is not empty, **when** the session lands a change, ends a run, or writes a handover through abcd, **then** it is refused, naming each finding, until each is captured, refuted, or in the private store.
- **Given** a confirmed finding whose fix is small or feasible, **when** it is met, **then** it is captured and the change that fixes it resolves the record, in the same change where it can be reviewed there.
- **Given** a finding that is untested, **when** the list is shown, **then** it is named as owed a test, and is never dropped by age or by a new session in the same checkout.
- **Given** a finding marked as a security one, **when** the list is shown anywhere but the person's own terminal, **then** only its count is, and its text is in no public record.
- **Given** a finding said to be already known, **when** it names a record, **then** that record must exist and be open, or the finding is listed again naming the record's state.

## Open Questions

_None open. What the rule means for the safe-fix discipline was put to the product thinker and ruled on 2026-11-25 (decision 2)._

## Decisions

1. The product thinker's, on 2026-11-25: the rule, that abcd must enforce it, that it is built, and that this comes straight after itd-2611251725190796 and itd-2611251725203320. They asked whether it is an intent; the criteria are the writing session's, for their review.
2. The product thinker's, on 2026-11-25 (dcn-2611251833399124): a confirmed finding is always captured, and built at once when small or feasible; a fix in a commit alone is no longer a state of its own.

## Decomposition

Hand-run four-piece table, 2026-11-25. The capability is this intent; the rule itself is a stance whose homes are the two disciplines (`refines` itd-2610141956319695, and it `reverses` in part itd-2610271130017437, confirmed by the product thinker); the list, the block, and the hooks are plumbing for the spec; no trust rule. Verdict: FILE-AS-IS, the routing for a person to confirm.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
