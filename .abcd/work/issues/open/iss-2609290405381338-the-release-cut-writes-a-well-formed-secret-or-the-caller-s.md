---
schema_version: 1
id: "iss-2609290405381338"
slug: "the-release-cut-writes-a-well-formed-secret-or-the-caller-s"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release/ingest.go"
---

The release cut writes a well-formed secret or the caller's own identity into public release text. launch ship --changelog-json accepts a host-composed payload whose changelog entries[].text and press_release headline text pass through checkProse (size, structure, CleanProseLine) and the outbound policy (session URL and tool footer only), then writes them to CHANGELOG.md and RELEASE.md unredacted: a GitHub token, an AWS key or the caller's own home path in a line is written to the working tree as public release text, to be committed and published. Found by the security review of lane drainEcho2 (INFO: changelog write path). The launch dry-run scan would refuse the same hard_fail finding in the bundled CHANGELOG.md later, but only after the file is written, and a person reviewing the diff may commit it first. Fix direction: at the cut, scan the rendered changelog section and release page with the canonical scanner and refuse any hard_fail finding with a reason naming the kind and the line, never the matched text; a degraded scanner is a stop, as checkOutbound's is. Detector: a payload carrying a well-formed token in a changelog line or a headline is refused, nothing is written, and the refusal does not carry the token.
