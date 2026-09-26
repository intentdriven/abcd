---
schema_version: 1
id: "iss-324"
slug: "scanner-ispathsegmentbyte-includes-the-slash-so-a-third-part"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/adapter/scanner/identity.go"
resolution: "home_path_other now reports a /Users or /home segment that follows a path byte when the path token is absolute (begins with '/', or a file URL with an empty authority), so file:///home/<name> and file:///Users/<name> are caught and redacted; isPathSegmentByte is unchanged. FP surface measured over every tracked text file: 20 new warn findings, all but one fixture or record text quoting this very shape; relative paths and web-URL paths stay declined."
impact: fix
resolved_by:
  commit: "8636491e444b42cff5a4d4ee87c8b39dec5600d8"
---

scanner isPathSegmentByte includes the slash so a third partys home path inside a file URL escapes stage-one redaction
## Evidence
`internal/adapter/scanner/identity.go:454` — `isPathSegmentByte` includes `'/'` in its class, so `leadingBoundaryOK` suppresses a `genericHomeRe` match preceded by a slash. Empirically `open file:///home/alice/notes.md` with a different caller identity → 0 findings (`home_path_other` missed); `see file:///Users/bob/secret.txt` → 0. A third party's home path inside a `file://` URL passes Stage-1 redaction unredacted — a privacy false NEGATIVE on the redaction path.

## Adversarial verdict: CONFIRMED (minor) — RECORD-ONLY
Surfaced as a spin-off of the iss-305 repolint fix (which deliberately uses repolint's slash-EXCLUDING isPathSegmentChar to avoid importing this FN). Not fixed this round: changing the scanner's isPathSegmentByte to exclude `/` has its own false-POSITIVE surface on the redaction path that needs independent measurement before it is safe — recording it is the correct outcome. Distinct from iss-305 (repolint over-detection); this is scanner under-detection.

## Grounds

- pursued: a third-party home inside a file URL is redacted while relative paths and web URL paths are not reported; a file-URL home stored verbatim, or a relative docs path now redacted, would show it wrong
