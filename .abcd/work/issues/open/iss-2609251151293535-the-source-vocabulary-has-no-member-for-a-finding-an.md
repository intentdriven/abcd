---
schema_version: 1
id: "iss-2609251151293535"
slug: "the-source-vocabulary-has-no-member-for-a-finding-an"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/issueschema/issueschema.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker, jointly with iss-2609020716571275: does autonomous-hunt join the closed --source vocabulary, or does such a loop file under agent-finding?"
remedy: "Waits on ruling N, jointly with iss-2609020716571275: if autonomous-hunt joins the vocabulary, add it to issueschema.Sources, the one list record-lint and the help text read, with a test that a record carrying it parses and lints clean; if agent-finding stands, name that mapping in commands/capture.md's source guidance and have the hunt loop's filing step write agent-finding. Either way a record carrying an unknown source stays refused and named, never coerced."
---

The source vocabulary has no member for a finding an autonomous bug-hunt loop files, and a downstream loop wrote source autonomous-hunt, which the reader refuses and skips. Decomposed out of iss-2609120452071388 when its code halves were fixed, because this half is a closed-vocabulary ruling for the product thinker, not an implementer's fix: either such a loop uses an existing member (agent-finding is the nearest) or autonomous-hunt joins issueschema.Sources, with record-lint and the help text following from the one list. Until it is ruled, a record carrying it is skipped with the schema layer named on the board.

## Remedy grounds (2026-09-29)

- Why: the record's two answers, each with the one place it changes; the vocabulary is a single list at internal/core/issueschema/issueschema.go, so either answer is small. The ruling is unanswered and none is picked.
- Rejected: accepting unknown sources leniently, which would undo the closed vocabulary the reader enforces.
