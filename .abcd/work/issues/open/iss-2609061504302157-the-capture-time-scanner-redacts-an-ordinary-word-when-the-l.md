---
schema_version: 1
id: "iss-2609061504302157"
slug: "the-capture-time-scanner-redacts-an-ordinary-word-when-the-l"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "2026-09-06 use in a managed repo"
origin: researcher-authored
production_mode: hand-written
found_at: "internal"
---

The capture-time scanner redacts an ordinary word when the local account name happens to be that word. On a machine whose account name is a common three-letter abbreviation for development, a capture containing the phrase 'the source checkout's <that word> build' came back with the word replaced by [redacted-user] and redacted: 1 in the JSON. The private-names layer is doing what it was told, but a banned name that is also a dictionary word or a conventional abbreviation needs a word-boundary and context rule, or at least a diagnostic naming which layer and which entry fired, so the author can tell a real leak from a false positive without reopening the file.

**Corroboration (2026-09-18, Gropius managed-repo session gropiusllm-56, relayed
to abcd-17).** Second independent hit, on a different verb and a different
machine: `abcd capture resolve` at v0.9.0 rewrote the same three-letter word in
a resolution note to `[redacted-user]` because it is that machine's account
name; the note was about the binary's default version string, which is that
word. The author found the rewrite only by reading the record back. The
session's ask matches this record's: a whole-word or context rule for a banned
name that is also a dictionary word, or at least a diagnostic naming the span
so the author can reword before the record is written. The launch-payload half
of the same collision is iss-2608291444328326.
