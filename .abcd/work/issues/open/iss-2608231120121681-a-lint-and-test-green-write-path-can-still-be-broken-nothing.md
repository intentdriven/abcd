---
schema_version: 1
id: "iss-2608231120121681"
slug: "a-lint-and-test-green-write-path-can-still-be-broken-nothing"
severity: "major"
category: "process"
source: "user-observation"
found_during: "record-review"
found_at: "CLAUDE.md"
details: "The definition of done requires make preflight clean, gofmt clean, and a test watched fail then pass. On 2026-08-23 a change to the issue-ledger write path satisfied all three and still broke the verb outright: abcd capture refused every issue whose text contained a home path. It was found by running the built binary against a real capture, which nothing in the definition of done asks for."
suggested_fix: "Add a functional check against a built binary to the definition of done for changes to a user-invocable write path: build, run the verb, assert on what lands on disk. Script-first — a documented step before any harness. The narrower lesson is that a unit test constructing a request by hand does not exercise the surface that constructs it in production."
related_issues: ["iss-2608231025198888", "iss-2608230847432286", "iss-2608230957104179"]
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M11: automate it: extend the smoke lane to exercise write-path verbs against a scratch repository and assert what lands on disk, with no hand step added to the definition of done; a build lane owed, not holding the tag)."
---

a lint-and-test-green write path can still be broken; nothing requires running the built binary

The definition of done asks for `make preflight`, `gofmt`, and a test watched
fail then pass. A change to the issue-ledger write path on 2026-08-23 satisfied
every one of them and still broke the verb.

## What happened

Redaction was added to `abcd capture` so free text could not carry a home path
into a committed record (iss-2608231025198888). It redacted the RENDERED record,
which treats the structural frontmatter as free text. The CLI derives the slug
from the issue text, so a home path in the body reached the slug, redaction
rewrote it to a bracketed placeholder, and the kebab-case validator then refused
the whole capture:

```
abcd: redaction produced an invalid record: malformed frontmatter:
slug "the-path-entry-is-users-[redacted-user]-local-bin-abcd-and-it-moved"
is not kebab-case
```

The verb was unusable for exactly the inputs the change existed to handle. It
reached main and the installed binary.

## Why every gate passed

The unit tests constructed a `CaptureRequest` by hand and passed a written-out
`Slug`. Production does not: the CLI derives the slug from the issue text. So
the tests exercised a caller that does not exist, and the caller that does was
never covered. `make preflight` was green, `gofmt` was clean, and both tests had
been watched fail before the change and pass after — the discipline was followed
exactly and caught nothing.

It surfaced only by building the binary and running one real capture.

## The shape, again

A check measuring the right property over a subject set that excludes the real
case. That is the same shape as iss-2608230847432286 and iss-2608230957104179,
and this instance sits inside the fix for one of them, which is the strongest
evidence in the ledger that the shape is not obvious to the person committing it.

## What this is not

Not an argument for more process on every change. The floor in
adversarial-review-scales-with-blast-radius is right, and this must not become a
mandatory review stage on ordinary work. The claim is narrower: for a change to
a user-invocable path that WRITES, the built binary is the only artefact that
answers the question the gates are being asked to answer, and running it once
costs seconds.

## Honest limit

n=1, and the author of the record is the author of the defect. A single
instance argues for a documented step, not for tooling. Per
recurrence-is-signal, a second occurrence is what would justify more.
