---
schema_version: 1
id: "iss-2610020700521421"
slug: "ahoy-install-writes-the-home-scope-transcript-store-without"
severity: "minor"
category: "inconsistency"
source: "managed-repo"
found_during: "peer report: ahoy install --adopt on a private consumer repo (abcd v0.9.0), reproduced at 7fb52a6b5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
remedy: "Raise a user-state gap naming ~/.abcd/transcripts/<root-sha>/records so the preview lists every home-scope path the install writes, and gate the index bootstrap and the store open on user-state alone (or name them under safe-autocreate in the preview); test: a run approving safe-autocreate and declining user-state writes nothing under ~/.abcd, and the preview names each path the install then writes."
---

ahoy install writes the home-scope transcript store without previewing it, and under a category that does not name it. The detection preview lists the user-state gaps history.bootstrap_missing and history.meta_missing (meta.json and the index entry) and nothing for the transcript store, yet the install writes ~/.abcd/transcripts/<root-sha>/records. stepHistory bootstraps ~/.abcd/history/index.json when either user-state or safe-autocreate is approved and opens the transcript store on safe-autocreate alone, so at tip a run answering y to safe-autocreate and n to user-state still wrote ~/.abcd/history/index.json and ~/.abcd/transcripts/<root-sha>/records: a person who reads the preview and declines the home-directory category still gets two writes in their home directory. If the safe-autocreate route is intended, the preview and the category question have to say so.
