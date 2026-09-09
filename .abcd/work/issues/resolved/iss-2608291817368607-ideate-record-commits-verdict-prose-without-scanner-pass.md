---
schema_version: 1
id: "iss-2608291817368607"
slug: "ideate-record-commits-verdict-prose-without-scanner-pass"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "v0.6.9-security-pass"
found_at: "internal/core/ideate/record.go"
related_issues: ["iss-2609020127281995"]
resolution: "Same defect, filed twice: this is the v0.6.9 security-pass sibling of iss-2609020127281995, and one change closes both. The stance question this record left open — fail-closed like intent, or redact-and-report like capture — is settled on the tier rather than on taste: ideate writes into .abcd/development/, which is intent's tier, so it takes intent's fail-closed posture. Capture's asymmetry is argued from what a refusal costs (refusing to record a finding loses the finding, and a ledger that rejects writes stops being written to), and a verdict record has no such claim on it: the three legs are host work that already happened, the payload is on disk or on stdin, and a refused run is re-runnable verbatim against a repaired scanner config. See iss-2609020127281995 for the implementation: a per-run recordRedactor in internal/core/ideate/redact.go that refuses on an unavailable or degraded scanner, stage-one redaction of every free-text field ahead of termsafe and the renderer, the literal-$HOME backstop, and a stage-two residual re-scan of the rendered record and the decision-log pointer line before either is written. Both detectors live in internal/core/ideate/record_test.go."
impact: fix
---

GitHub #486 sibling: abcd ideate record writes the verdict payload's idea text and the three legs' prose (claims, grill hits, kill attempts, rejected alternatives) to .abcd/development/research/<date>-ideate-<slug>.md through termsafe.CleanProse only — no pass through internal/adapter/scanner (internal/core/ideate/record.go validate, render.go). A secret or absolute home path in the host-produced JSON is committed verbatim, the same class capture (redactLedgerText) and intent (redactIntentText) close. Stance (fail-closed like intent, or redact-and-report like capture) is a design choice to settle before fixing.

## Grounds

- pursued: the stance is decided by which committed tier the writer lands in, not by the verb's temperament, so ideate follows intent and not capture; it would show wrong if refusing a verdict on a degraded scanner cost a gauntlet that could not be re-run, which it cannot — the payload is a file the host already produced
