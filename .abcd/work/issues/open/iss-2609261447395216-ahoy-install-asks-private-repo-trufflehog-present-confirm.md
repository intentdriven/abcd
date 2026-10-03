---
schema_version: 1
id: "iss-2609261447395216"
slug: "ahoy-install-asks-private-repo-trufflehog-present-confirm"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (run A 2026-09-29, lane drainDrift3; rulings-owed BH): ahoy install asks to confirm deep secret scanning when a private repository has trufflehog on PATH and persists scan.deep, but nothing reads scan.deep and nothing runs trufflehog. Should abcd wire a trufflehog adapter behind the scanner seam so the answer takes effect, or stop asking and say the key is inert where it is already set?"
remedy: "Waits on ruling BH (with AF: wire a trufflehog adapter, or stop asking): if wired, add a trufflehog adapter behind the scanner seam that runs only when scan.deep is true and trufflehog is on PATH (an outside tool, needing the person's sign-off), tested with a fake trufflehog on PATH; if the question stops, remove the prompt at internal/core/ahoy/apply.go:595 and have ahoy report an existing scan.deep as inert, tested by an install run with trufflehog on PATH that asks nothing."
---

ahoy install asks 'Private repo + trufflehog present — confirm deep secret scanning' and persists scan.deep in .abcd/config.json, but no scanner reads scan.deep and nothing runs trufflehog, so the answer changes nothing: a person told they enabled deep secret scanning has not. Either wire a trufflehog adapter behind the scanner seam or stop asking (and say the value is inert where it is already set).

**Corroboration (2026-10-02), reported from a private consumer repo on v0.9.0.**
After the y/N category questions of `yes | abcd ahoy install --adopt
--docs-target agents_md --visibility private --oracle-backend host-delegated
--attribution`, the question fired as `scan_deep (true/false) [false]:` with no
`--scan-deep` passed and no hint of it in the bare detection pass, and `yes`
answered it `y`; the documentation presented `yes |` as approving categories
only. At tip origin/main 7fb52a6b5 the question still fires in that stream on a
private repository with trufflehog on PATH. Since 7fc57628a it arrives
explained, names `--scan-deep`, and the command page lists it among the value
questions, so the undocumented half is fixed. What remains is this record's
question, plus a consequence: the `y` it receives is out of set and now aborts
the whole config-value write, discarding the flagged values with no note
(captured as iss-2610020700263496). Stopping the question, the second branch
above, would remove both.

## Remedy grounds (2026-09-29)

- A setting that changes nothing while the person is told it did is the defect; either branch makes the answer true. The help text at internal/core/ahoy/prompt_help.go already says the value changes nothing, which narrows the harm but not the question.
- Rejected: leaving the prompt and relying on that help text.
