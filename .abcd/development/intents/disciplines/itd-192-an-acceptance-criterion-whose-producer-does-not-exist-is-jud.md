---
id: itd-192
slug: an-acceptance-criterion-whose-producer-does-not-exist-is-jud
spec_id: null
kind: discipline
kind_notes: "Cross-cutting gate over fidelity audits: it fixes how one class of acceptance criterion is judged, in every audit, rather than delivering a capability with a user moment of its own. Consumed by the intent-auditor's definition. Filed as a stub at the maintainer's direction: the rule is settled, its wiring is not."
suggested_kind: discipline
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# An acceptance criterion whose producer does not exist is judged by whether this phase wires it, not by whether the narrowing was disclosed

## The rule

A fidelity audit regularly meets a criterion whose Given clause presupposes
something no code can produce: the constructor refuses the value, or the verb
that would mint it belongs to an intent that has not landed. One question
decides the verdict, and it is not whether the spec admitted the gap.

**Does this phase wire it?**

- **Wired in this phase** — the criterion is `MET_WITH_CONCERNS`, and the
  verdict names the intent that closes it.
- **Not wired in this phase** — the criterion is `NOT_MET`, and the verdict
  carries the reason a promised intent goes unwired.

Disclosure is not the test. A spec that states its own sequencing up front is
still a spec whose promise is unkept if nothing in the phase closes it, and an
auditor that accepts disclosure as satisfaction grades the writing rather than
the delivery.

The second half of the rule is the point of the first. `NOT_MET` obliges
somebody to answer why a promised intent is not being wired — and that
question, asked at the audit rather than at the retrospective, is what the
verdict exists to force.

*Dictated by the maintainer on 2026-08-30 and formatted; the ruling arose from
two verdicts that turned on it.*

## The gate

- **Given** a criterion whose producer is absent from the delivered code,
  **when** the auditor judges it, **then** the verdict turns on whether an
  intent in this phase wires it, and names that intent.
- **Given** a criterion recorded `MET_WITH_CONCERNS` on the strength of a
  scheduled intent, **when** the phase closes, **then** the verdict is
  re-checked, and flips to `NOT_MET` if that intent did not land.
- **Given** a criterion judged `NOT_MET` for want of a producer, **when** its
  verdict is read, **then** the record states why the promised intent is
  unwired.

## Fit

The discipline family is the right home: this fixes how a judgement is made
across every audit, and has no user moment of its own. It sits beside the
`itd-1` acceptance-gate discipline and is consumed by the intent-auditor's
definition rather than by a verb.

A stance sits inside this rule — a promise is judged against the phase, not
against its own disclosure — which could stand as a principle. It stays here
instead: one rule, one record, and the discipline carries the stance rather
than duplicating it. If the re-check at phase close proves to need machinery,
that is the point at which this graduates to an ADR.

## Staging

The rule is applied by hand today: two criteria in the current cycle were
judged under it, each recorded `MET_WITH_CONCERNS` against a scheduled intent,
each carrying the flip condition in the decision record.

Wiring is the unbuilt half, and it is two pieces: the judging rule stated in
the intent-auditor's definition so an auditor applies it without being told,
and a re-check at phase close so a conditional verdict cannot quietly survive
the intent it depended on. Both wait on a plan; this record is the stub that
holds the rule until then.
