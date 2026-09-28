---
schema_version: 1
id: "iss-2609280945018822"
slug: "the-pre-commit-name-guard-s-awk-decode-of-escaped-spellings"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: verify-fix3-drainS3"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
---

The pre-commit name guard's awk decode of escaped spellings is superlinear in LINE length on macOS awk 20200816: 19 MB of escaped JSON over 105k lines decodes in 1.9 s, but the same bytes as one line take 65 s in awk and 88 s for the commit (the verify of fix3-drainS3). A minified single-line JSON export or asset pays it; the hook still completes and reports, so it is never silent. Both hook copies (.githooks/pre-commit and internal/core/ahoy/defaults/pre-commit) carry the decode.
