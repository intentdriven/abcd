---
schema_version: 1
id: "iss-2610101819067941"
slug: "the-cold-reading-exclusion-floor-disagrees-with-itself-about"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
remedy: "Make the redactor match excluded headings the way the verifier does (namesExcludedHeading, case-insensitive), so a heading differing only in case is redacted rather than refused; a test assembles a record whose excluded heading is lower-case."
---

The cold-reading exclusion floor disagrees with itself about heading case. TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset (smoke lane) failed when a record carried a '## Open questions' heading (lower-case q): the redactor, redactExcluded in internal/core/reading/project.go, looks the heading up exactly, case-sensitive, while the verifier compares with strings.EqualFold. So a record whose excluded heading differs only in case is not redacted but is refused, and reading assemble fails for the whole repository. It fails closed, so nothing leaks, but any record author can break the assembler with a lower-case heading.
