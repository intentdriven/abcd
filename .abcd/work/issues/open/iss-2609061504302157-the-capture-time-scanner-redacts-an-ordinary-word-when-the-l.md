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
