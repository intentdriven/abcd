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
deferred_after: "v0.8.0"
deferral_reason: "The instance is gone and the mechanism is not. On the owner's instruction the unredactable staged text was moved out of the store to their own directory with a note recording why it could never be processed, so nothing unredacted from a deleted repository now sits in abcd's store. The mechanism stands: a transcript is redacted under its own repository's configuration, so when that repository is gone no scanner can be built and the drain has nothing to run with. The three ways out are to make deletion of a repository a trigger that drains or discards first, to allow a drain under an explicitly named substitute configuration with the substitution recorded on the record, or to treat such a pile as terminal and offer only discard. Each changes what the store promises, so the choice is the maintainer's."
found_at: "internal/core/history/staging.go"
---

Staged transcripts for a repository that no longer exists can never be drained, so their unredacted text is permanent. The drain builds its scanner from the destination repository's root, because that repository's own redaction configuration must govern its own transcripts. When the repository's directory is gone, that root cannot be resolved and the drain has nothing to run with, so the raw bytes stay staged forever with no path to redaction. This is not hypothetical: one store on this machine holds 11.7 megabytes of unredacted staged text whose repository was deleted, which is the largest single pile in the store and the only one that no amount of ordinary use will clear. The retention work just landed addresses the case where nobody opens a repository again, by draining while a session is live and by reporting other repositories' backlogs at session start, but both remedies assume the repository still exists to be opened. The options are to let a deletion of the repository be a trigger that drains or discards first, to allow a drain under an explicitly named substitute configuration with the substitution recorded on the record, or to treat the pile as terminal and offer only discard. Whichever is chosen, the present behaviour is the worst of them: the text is kept, unredacted, with no way to act on it and nothing saying so.
