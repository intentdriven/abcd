---
id: spc-2610142138047122
slug: an-ideate-verdict-proves-its-own-legs
intent: itd-2610141646454347
origin: researcher-authored
production_mode: hand-written
---
# an-ideate-verdict-proves-its-own-legs

## Summary

Delivers schema version 2 of the ideate verdict, validated by `abcd ideate record`
(`internal/core/ideate`), and the command page `commands/ideate.md` updated to it
in the same change.

**Payload (version 2).** `schema_version: 2`; `origin_sources` (the sources the
idea came from); `legs` as today, with three changes: a research claim's
`status` takes `quoted | verified | falsified | unverifiable`; the record-grill
leg gains `searches` (each `{terms, places, count}`); the adversarial-review leg
holds `reviewers` (each `{label, kill_attempts, verdict, sources}`) and an
optional `ruling` (`{by, verdict, reason}`); a top-level `reframing` string.

**Validation**, each a refusal with exit 2 that writes nothing:

1. Every record id in every leg resolves (`itd-`, `spc-`, `iss-`, `adr-` patterns
   in `primary_source`, grill `record`, kill-attempt text and rejected
   alternatives), through the resolver the grill uses today; the refusal names
   the leg and the id (criterion 1).
2. `primary_source` is an `https://` address, `source:<key>` (resolved against
   the local sources corpus when it exists), or a record id (criterion 2).
3. A record-grill leg with no hits and no `searches` entry is refused
   (criterion 3).
4. `verdict: reframed` without a non-empty `reframing` is refused (criterion 4).
5. Each reviewer's attempts, verdict and sources are kept under its `label`
   (criterion 5); reviewers whose verdicts differ require `ruling`
   (criterion 6).
6. A claim with `status: verified` whose `primary_source` is in
   `origin_sources` is refused; `quoted` is accepted (criterion 7).
7. `schema_version: 1` is refused with a message naming version 2
   (criterion 8).

**Render.** The research note shows the searches table, each reviewer's section,
the ruling and the reframing. Verdicts already on record are not parsed.

**Command page.** Leg 3 tells the runner to give each reviewer one idea per
context (criterion 9), and the payload example becomes version 2.

## Footprint

- packages: internal/core/ideate, internal/surface/cli, commands
- tests: internal/core/ideate (record_test.go, a new v2_test.go with one case per criterion), internal/surface/cli (the ideate record surface)

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
