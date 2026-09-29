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
deferred_after: v0.11.1
deferral_reason: "confirmation still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane): ruling M24 (2026-09-23) is that a project's deletion first drains or discards its staged transcripts, with the trigger brought back for confirmation because abcd cannot see a plain rm. Proposed trigger: a vanished checkout root detected at the next history run. With the repository gone its own redaction configuration is gone too, so that trigger can only name the pile and offer discard (history discard exists), narrowing the promise from drain-or-discard to discard. Owed: confirmation of that narrowing, then a lane."
---

Staged transcripts for a repository that no longer exists can never be drained, so their unredacted text is permanent. The drain builds its scanner from the destination repository's root, because that repository's own redaction configuration must govern its own transcripts. When the repository's directory is gone, that root cannot be resolved and the drain has nothing to run with, so the raw bytes stay staged forever with no path to redaction. This is not hypothetical: one store on this machine holds 11.7 megabytes of unredacted staged text whose repository was deleted, which is the largest single pile in the store and the only one that no amount of ordinary use will clear. The retention work just landed addresses the case where nobody opens a repository again, by draining while a session is live and by reporting other repositories' backlogs at session start, but both remedies assume the repository still exists to be opened. The options are to let a deletion of the repository be a trigger that drains or discards first, to allow a drain under an explicitly named substitute configuration with the substitution recorded on the record, or to treat the pile as terminal and offer only discard. Whichever is chosen, the present behaviour is the worst of them: the text is kept, unredacted, with no way to act on it and nothing saying so.

## Deferral 2026-09-29

Deferred past v0.11.1: confirmation still owed to the product thinker (re-deferred at v0.11.1 by run A's major-triage lane): ruling M24 (2026-09-23) is that a project's deletion first drains or discards its staged transcripts, with the trigger brought back for confirmation because abcd cannot see a plain rm. Proposed trigger: a vanished checkout root detected at the next history run. With the repository gone its own redaction configuration is gone too, so that trigger can only name the pile and offer discard (history discard exists), narrowing the promise from drain-or-discard to discard. Owed: confirmation of that narrowing, then a lane.
