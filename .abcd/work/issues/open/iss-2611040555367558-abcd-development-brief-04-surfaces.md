---
schema_version: 1
id: "iss-2611040555367558"
slug: "abcd-development-brief-04-surfaces"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
remedy: "Skip code spans when finding raw-HTML headings, and bound the title quoted in the refusal."
---

.abcd/development/brief/04-surfaces/23-reading.md:40-46 and the refusals it lists do not include this case. The reported refusal on the changelog no longer fires on this tree, whose CHANGELOG.md:30 was reworded, but the cause stands: with a code span holding an opening `<h1>` restored at that line and committed, `abcd reading assemble` exits 2 at detection, entailment and widening, dry run included. It reads from the tag to the end of the file as one raw-HTML heading title (rawHTMLHeading, internal/core/reading/project.go:1032-1035) and quotes about 597 KB of the file in the refusal. A code span renders as text, so no heading exists. Every changelog line or included document naming that tag in backticks stops all readings until the text is reworded.
