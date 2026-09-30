---
schema_version: 1
id: "iss-2609301002043276"
slug: "the-delimiter-detector-testnoprivatedelimitercompare-fails"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
remedy: "Render the ADR head once through one helper that takes the status and the extra frontmatter keys, so the skeleton and the stated record share its two delimiters and no private search cuts the skeleton; the allowlist's count of two then holds unchanged. Grounds: the detector's own remedy (route through the canonical reader, or allowlist a writer with its count) and byte-identical output of both renderers, which decide's tests assert."
resolution: "The ADR head is rendered once by renderHead, shared by the skeleton and the stated record; decide.go spells the two delimiters the allowlist names, and the detector passes."
impact: internal
resolved_by:
  commit: "675519daa"
---

The delimiter detector TestNoPrivateDelimiterCompare fails on the tip that carries the drain's own-rule reader: internal/core/decide/decide.go spells four frontmatter delimiter literals against an allowlist of two, because renderStated (added with decide.CreateStated for the ahoy drain-rule offer) cuts the skeleton it just rendered at its close with a private strings.Index search for the delimiter and writes a second block. The lane gates of that change ran only the touched packages, so go test ./... fails at internal/core/frontmatter.

## Grounds

- pursued: the delimiter detector passes on the drain tip with the allowlist unchanged and both renderers write the bytes they wrote before; shown wrong if TestNoPrivateDelimiterCompare fails again or a decide test sees a different record
