---
schema_version: 1
id: "iss-2609120505141653"
slug: "the-origin-field-claims-a-person-invoked-the-verb-and-is-sta"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "peer exchange on capture provenance, 2026-09-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/provenance/provenance.go"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M31: route, not who: correct the KindResearcherAuthored comment in internal/core/provenance to name the route (written directly, not derived); no new kind, no who field, no backfill; a build lane owed, not holding the tag)."
resolution: "Ruled route, not who (M31): researcher-authored names the route (text written directly, not derived) and asserts nothing about who invoked the command; no new kind, no who field, no backfill. Corrected at every site that claimed who: in 97788f71 the KindResearcherAuthored comment (internal/core/provenance/provenance.go), the DraftOptions.Origin comment (internal/core/intent/create.go) and the disclosure paragraph of commands/capture.md; in f28e164a the commitCapture comment (internal/core/capture/workflow.go), the spec.Create comment (internal/core/spec/store.go) and TestSpecCreateStampsProvenance (internal/core/spec/store_test.go), the researcher-authored line of the closed spc-56, the hand-filed-draft wording in commands/capture.md link mode, internal/core/intent/lifecycle.go, internal/core/intent/intent_test.go, internal/core/capture/promote_reading_origin_test.go, internal/core/lint/schema_test.go and the closed spc-2609020626042168, and the Related note of iss-2609120511058115."
impact: fix
resolved_by:
  commit: "97788f71"
---

`origin: researcher-authored` states that a person invoked the verb. It is
stamped on the quoted-text route unconditionally, so a record an agent captured
carries a claim that a person captured it.

**This record is an instance of itself.** Its own frontmatter says
`researcher-authored`, and no person invoked the verb that wrote it.

## The claim and the write

`internal/core/provenance/provenance.go:41` states the meaning outright:

```go
// KindResearcherAuthored is the default for a verb a person invoked.
KindResearcherAuthored Kind = "researcher-authored"
```

The other two kinds are honest about their route and are derived from which
command ran: `extracted-from-record` is stamped only by `capture.Promote`, and
`contributed-by-reading <rdg-N>/<rdi-N>` only by `capture promote <rdi-N>`. Both
name something structural that the command genuinely knows.

`researcher-authored` is the default that catches everything else, and the thing
it asserts — that a person invoked it — is the one thing the command does not
know.

## Why this is not pedantry

Disclosure is the whole point of the field. The surface page says the two
provenance keys "are disclosure at field granularity, on the same footing as the
`Assisted-by:` trailer at commit granularity."

The two footings disagree. At commit granularity the convention is strict, and
deliberately so: `Assisted-by:` is required, `Assisted-by: None` is the positive
declaration for human-only work, and a free-text escape was refused on the
grounds that it would reopen the omission it closes. At field granularity the
same project stamps "a person invoked this" onto records no person invoked, by
default, silently.

Measured today, without looking far: six records captured in this session and
eleven captured by a peer session in another repository all carry
`researcher-authored`. Twenty-three including this record's own family. None was
invoked by a person.

## What the fix is not

Not a new flag. `origin` is "derived from which command ran and has no flag at
all", and that is the property worth keeping — a caller-supplied provenance field
is a field that gets guessed, which is exactly the failure recorded next door in
`iss-2609120452071388`, where two sessions guessed `--source` values that do not
exist in one day.

So the value has to come from something the process knows about itself, not from
something it is told. Candidates, none obviously right:

- A fourth kind (`agent-invoked`, or similar) selected by the same signal the
  harness already uses to know it is an agent session.
- Narrowing `researcher-authored` to mean what it can actually support — filed
  from quoted text, as opposed to derived from another record — and moving the
  who to a separate key or dropping the who entirely.
- Leaving the value and correcting the comment, if the maintainer's reading is
  that the kind names the ROUTE and the comment overstates it. That is the
  cheapest fix and it should be considered honestly rather than dismissed: the
  surface page's own gloss is "a draft filed from quoted text is
  `researcher-authored`", which says nothing about who typed it.

The third option is the one to rule on first, because if the comment is simply
wrong then there is no defect in the data and this record collapses to a
one-line correction.

## Scope note

Population is forward-only and nothing backfills a provenance stamp, so whatever
is decided applies to records written afterwards. The existing corpus keeps what
it carries, which is the right behaviour and also the reason the decision should
not sit long: every record filed meanwhile inherits the ambiguity.

## Acceptance

- **Given** the `origin` vocabulary, **when** a reader asks what
  `researcher-authored` asserts, **then** the code comment, the surface page and
  the stamped data agree.
- **Given** a record captured by an agent, **when** its `origin` is read,
  **then** it does not claim a person invoked the verb.

## Grounds

- pursued: the three statements of what researcher-authored means now agree with the stamped data, since none claims a person invoked the verb; shown wrong if any committed text still reads researcher-authored as a claim about who ran the command
