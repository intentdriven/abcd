---
schema_version: 1
id: "iss-2609091717146700"
slug: "staged-transcripts-for-a-repository-that-no-longer-exists-ca"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "draining the real staging backlog after the retention fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/staging.go"
remedy: "Apply ruling M24 through the proposed trigger: SurveyBacklog marks a store key whose checkout can no longer be resolved as orphaned, the session-start backlog report names that pile, its size and the exact discard command, and history discard accepts the root sha so it runs with the checkout gone, the discard staying a person's act; if the confirmation asks for a drain instead, add one under an explicitly named substitute configuration recorded on every record it writes. Prove it with a test that stages a transcript under a temporary root, removes the root, and asserts the next run reports the pile and discard empties it."
deferred_after: v0.11.1
deferral_reason: "confirmation still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane): ruling M24 (2026-09-23) is that a project's deletion first drains or discards its staged transcripts, with the trigger brought back for confirmation because abcd cannot see a plain rm. Proposed trigger: a vanished checkout root detected at the next history run. With the repository gone its own redaction configuration is gone too, so that trigger can only name the pile and offer discard (history discard exists), narrowing the promise from drain-or-discard to discard. Owed: confirmation of that narrowing, then a lane."
---

Staged transcripts for a repository that no longer exists can never be drained, so their unredacted text is permanent. The drain builds its scanner from the destination repository's root, because that repository's own redaction configuration must govern its own transcripts. When the repository's directory is gone, that root cannot be resolved and the drain has nothing to run with, so the raw bytes stay staged forever with no path to redaction. This is not hypothetical: one store on this machine holds 11.7 megabytes of unredacted staged text whose repository was deleted, which is the largest single pile in the store and the only one that no amount of ordinary use will clear. The retention work just landed addresses the case where nobody opens a repository again, by draining while a session is live and by reporting other repositories' backlogs at session start, but both remedies assume the repository still exists to be opened. The options are to let a deletion of the repository be a trigger that drains or discards first, to allow a drain under an explicitly named substitute configuration with the substitution recorded on the record, or to treat the pile as terminal and offer only discard. Whichever is chosen, the present behaviour is the worst of them: the text is kept, unredacted, with no way to act on it and nothing saying so.

## Deferral 2026-09-29

Deferred past v0.11.1: confirmation still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane): ruling M24 (2026-09-23) is that a project's deletion first drains or discards its staged transcripts, with the trigger brought back for confirmation because abcd cannot see a plain rm. Proposed trigger: a vanished checkout root detected at the next history run. With the repository gone its own redaction configuration is gone too, so that trigger can only name the pile and offer discard (history discard exists), narrowing the promise from drain-or-discard to discard. Owed: confirmation of that narrowing, then a lane.

## Remedy grounds (2026-09-29)

- Why: M24 is answered (a deletion drains or discards first) and only the trigger's confirmation is owed; a vanished root is the one signal abcd can observe after a plain rm, and with the repository's own redaction configuration gone, discard is the only act that needs no substitute. Discard stays a front-door act because internal/core/history/staging_lifetime.go keeps it out of every hook and drain by design.
- Rejected: an automatic discard at detection, which would delete text on a guess that the checkout is gone for good rather than moved.
