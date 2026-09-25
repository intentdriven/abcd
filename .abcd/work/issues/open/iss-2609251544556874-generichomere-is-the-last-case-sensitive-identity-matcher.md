---
schema_version: 1
id: "iss-2609251544556874"
slug: "generichomere-is-the-last-case-sensitive-identity-matcher"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
---

genericHomeRe is the last case-sensitive identity matcher, and it guards the one identity kind whose value is another person's filesystem path. internal/adapter/scanner/identity.go compiles it as (?:/Users/[A-Za-z0-9._-]+|/home/[A-Za-z0-9._-]+) with no (?i), and it is the sole detector for home_path_other. macOS and Windows filesystems fold case, so /USERS/<name>/notes.md and /Home/<name>/notes.md name the file the canonical spelling names, and a path written either way passes every redactor that routes through ScanText and the privacy lint. Every sibling matcher on this boundary already folds (email, github, homeSelf, localBare, githubRemoteRe, noreplyRe). The finding was first recorded on a parked branch whose record never reached main (0fac3697 carries its fix); it is captured afresh here so the fix can land with its record.
