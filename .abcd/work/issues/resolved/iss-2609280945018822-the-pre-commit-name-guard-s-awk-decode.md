---
schema_version: 1
id: "iss-2609280945018822"
slug: "the-pre-commit-name-guard-s-awk-decode"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: verify-fix3-drainS3"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
resolution: "The awk splits take one-character string separators instead of regular expressions, which made the one true awk's split quadratic in line length; one 19 MB line decodes in 2.5 s instead of 156 s, output byte-identical."
impact: fix
resolved_by:
  commit: "5c5a8595c"
---

The pre-commit name guard's awk decode of escaped spellings is superlinear in LINE length on macOS awk 20200816: 19 MB of escaped JSON over 105k lines decodes in 1.9 s, but the same bytes as one line take 65 s in awk and 88 s for the commit (the verify of fix3-drainS3). A minified single-line JSON export or asset pays it; the hook still completes and reports, so it is never silent. Both hook copies (.githooks/pre-commit and internal/core/ahoy/defaults/pre-commit) carry the decode.

## Grounds

- pursued: decode time grows linearly with line length (1 to 16 MB: 0.11 to 1.70 s) and the decoded bytes match the regex-split decoder on the corpus; a gawk or mawk that read a one-character string separator as a pattern, or a superlinear curve on another awk, would show it wrong
