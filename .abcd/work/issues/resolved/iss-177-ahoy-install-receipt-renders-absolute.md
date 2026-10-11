---
schema_version: 1
id: "iss-177"
slug: "ahoy-install-receipt-renders-absolute"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-74-increment-2-review"
found_at: "internal/core/ahoy/apply.go"
resolution: "install receipt reports repo-relative and ~-relative paths; the scrub sits at the note seam, which a test holds to being the receipt's only writer"
impact: fix
---

ahoy install reports every written artefact as an absolute filesystem path, so an install receipt pasted into an issue or a transcript carries the developer's home directory and username

Pre-existing and repo-wide: every apply step notes its write with an absolute
path (`a.note(filepath.Join(...))`), and the CLI prints them verbatim as
`wrote: <path>`. itd-74 increment 2 adds five more such notes and deliberately
does not change the convention — a one-verb fix would leave the other steps
inconsistent, and the sibling verbs already route error text through a shared
path scrub (`scrubPaths`) that the receipt does not use. The fix is one change
across every apply step: report repo-relative paths, or scrub before rendering.

