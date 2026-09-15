---
schema_version: 1
id: "iss-2608291941060605"
slug: "install-change-notes-embed-unscrubbed-paths"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/ahoy/apply.go"
resolution: "both attribution opt-in notes and registerRepo's lock note render through receiptPath; a test forces a mode-0000 config.json through install --attribution and asserts the note names no absolute path"
impact: fix
---

ultra-v0.6.8 follow-up (review of the branch): registerRepo in internal/core/ahoy/apply.go appends the raw lock error to the install receipt's change-notes ('history registration for <sha> skipped (' + lockErr.Error() + ')'), and that error carries the absolute lock path under the user's home (~/.abcd/history/...), so the receipt and --json output name the home directory and username — bypassing the receiptPath scrub that note() routes every written path through. The same shape recurs in recordAttributionOptIn's new notes (an os.PathError naming <repo>/.abcd/config.json). Fix: render every change-note that embeds an error through receiptPath, whose embedded-root pass covers sentences.
