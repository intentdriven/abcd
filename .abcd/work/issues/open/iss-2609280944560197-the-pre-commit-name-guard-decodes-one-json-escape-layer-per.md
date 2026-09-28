---
schema_version: 1
id: "iss-2609280944560197"
slug: "the-pre-commit-name-guard-decodes-one-json-escape-layer-per"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: verify-fix3-drainS3"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
---

The pre-commit name guard decodes one JSON escape layer per run of backslashes and never re-reads its own output, so a backslash spelled as the unicode escape of U+005C (or as %5C) decodes to a literal backslash that is not read again as an escape: a banned name hidden one layer behind such a backslash (the escaped backslash followed by u0075 and the rest of the name) commits, while scanner.DecodedViews reads it through its second layer (maxJSONDecodeLayers is 3). The carrier is a hand-crafted file, since no mainstream encoder writes a backslash that way; the hook comments claim the guard reads what the scanner's redactors read, which overclaims by exactly this case. Both hook copies (.githooks/pre-commit and internal/core/ahoy/defaults/pre-commit) carry the decode.
