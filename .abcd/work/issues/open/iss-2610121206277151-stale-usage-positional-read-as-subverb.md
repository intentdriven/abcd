---
schema_version: 1
id: "iss-2610121206277151"
slug: "stale-usage-positional-read-as-subverb"
severity: "nitpick"
category: "bug"
source: "impl-review"
found_during: "fix/stale-usage-notes lane, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
remedy: "When the word after the verb is not a known sub-verb of the command and the verb's synopsis takes a positional there (abcd build <itd-N>), match the flag against that positional synopsis line rather than treating the word as an unknown sub-verb; test build itd-1 --<documented flag> on a stale release binary gets the note, watched fail first."
---

The stale-usage note's classifier (internal/surface/cli/staleusage.go) reads a positional argument under a verb that has sub-verbs as an unknown sub-verb: 'abcd build itd-1 --frob-top' takes itd-1 for a sub-verb of build, so a flag the build page genuinely documents for 'abcd build <itd-N>' gets no stale-binary note. It fails safe (the plain unknown-flag refusal, never wrong advice), but a stale binary is then not told why. Found by the fix for iss-2610120921425348 and iss-2610120921497084 (branch fix/stale-usage-notes), which left it outside those remedies. <!-- record-lint: forward-looking -->
