---
schema_version: 1
id: "iss-2610262040343151"
slug: "itd-1-has-every-audit-run-write-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 940 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-1]
finding_ref: cfk-0233048afca4756a
---

itd-1 has every audit run write a forensic copy under .abcd/logbook/; the build sequence says .abcd/logbook/ is a retired location held retired by a detector.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:45` — "each run also writes a per-run forensic copy at `.abcd/logbook/audit/review-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:177` — "retired location, held retired by an armed detector"

itd-1's rule writes a per-run forensic copy to .abcd/logbook/audit/review-< ts>/. The build sequence says .abcd/logbook/ is retired (iss-73) and that an armed detector fails the build if non-test Go source names it. Following the discipline would break the build.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 940.
