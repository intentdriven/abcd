---
schema_version: 1
id: "iss-2609012035110990"
slug: "ghsa-j7v5-q7x6-v3rp-cwe-354-cwe-391-the-armed-gitleaks-augme"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/gitleaks/gitleaks.go"
resolution: "gitleaks.toFindings locates each reported value across the whole text (Secret, then Match when the Secret is not verbatim), splits a value that spans lines into one finding per line so the line-scoped seal masks every line, and returns gitleaks.ErrFindingNotLocated for a report that places nothing; history.Capture refuses the write on that error and, after Redact, refuses with RedactionResidualError when any augmented finding's bytes still occur in the redacted text, so verification is symmetric with detection. Proved by TestAugmentMultiLineFindingIsRedactedPerLine, TestAugmentFallsBackToMatchWhenSecretIsNotVerbatim and TestAugmentUnlocatableReportFailsClosed (gitleaks) and TestCaptureRefusesWhenAugmentedSpanIsNotMasked and TestCaptureFailsClosedOnUnlocatableGitleaksReport (history); TestAugmentReportsEveryOccurrenceOnALine and TestAugmentEmptyReportNoFindings stay green. The toFindings doc comment no longer claims a PEM body is native-covered."
impact: fix
---

GHSA-j7v5-q7x6-v3rp (CWE-354, CWE-391): the armed gitleaks augmentation silently drops multi-line and non-verbatim findings. gitleaks.toFindings skips any reported value containing a newline (its doc comment claims a PEM body is native-covered, but the native pem_private_key pattern matches the BEGIN header only) and falls back from Secret to Match only when Secret is empty, never when Secret is absent from the text, so a report that locates nothing yields no finding and no error; Augment returns zero findings on a non-empty report without erroring; and history.Capture's stage two re-scans the redacted text with the native detector only, so an augmented span can never enter BlockingResidual and the record's frontmatter counts findings the redactor never applied. The fix must establish: the needle is located on the whole text and split into one finding per line it crosses, Match is tried when Secret is not verbatim, a report that yields no finding returns gitleaks.ErrFindingNotLocated which Capture refuses the write on, and Capture verifies every augmented finding's span is sealed in the redacted text and returns RedactionResidualError on any survivor. Rejected alternative: re-running gitleaks over the redacted text doubles a 30s-timeout subprocess and is not deterministic across rule sets. Sibling out of scope: internal/core/memory/redact.go redactText is native-only on both stages with no gitleaks augmentation, so it is unaffected. Consequence: an armed repo whose rule reports something absent from the transcript now refuses the capture loudly; the remedy is the rule or enabled:false.

## Grounds

- pursued: verifying the adapter's own reported bytes against the redacted text is the symmetric check that needs no second subprocess and is exact because the adapter locates every occurrence and secrets are sealed length-preservingly; a span-exact compare was rejected because identity placeholders rewritten after the seal shift columns, and a mis-positioned finding is precisely the case a span compare cannot see. Shown wrong if a legitimately sealed transcript is refused because a reported fragment recurs in text the adapter did not locate.
