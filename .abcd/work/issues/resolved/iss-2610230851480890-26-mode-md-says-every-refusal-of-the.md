---
schema_version: 1
id: "iss-2610230851480890"
slug: "26-mode-md-says-every-refusal-of-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/26-mode.md"
remedy: "Complete the refusal list in 26-mode.md from the verbs' help and the autonomy package's errors."
resolution: "The mode chapter's refusal paragraph now names every refusal the away verbs make at exit 2: it adds a queued step over 500 characters, a full queue, a record the write would take past 64 KiB, and no home folder or root commit for every away verb."
impact: internal
resolved_by:
  commit: "5760fe13cfc4fe741ab900eb50387dea28a1781d"
---

26-mode.md says every refusal of the away verbs exits 2 and lists them, but omits a queued step over 500 characters, a full queue, a record that would pass 64 KiB, and no home folder or root commit.

## Grounds

- pursued: every sentinel awayRefusal maps to exit 2 has its words in the chapter's refusal paragraph; a sentinel in internal/surface/cli/mode_away.go with no matching words in 26-mode.md would show it wrong, as the detector kept beside the report (not committed, the hold could not be widened) checks.
