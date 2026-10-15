---
id: dcn-2608300000000020
slug: itd-179-delta-close-body-scope-does-not
date: 2026-08-30
---
2026-08-30 — itd-179 delta close: body scope does NOT close consequence A, and the record says so. An unclosed opener in the BODY still masks the body, so an appended entry still cannot read back and every triage route still refuses — verified by a test that stayed red at the body-scope commit and only passed at the next one. What makes that survivable is the other two halves: the refusal names the opener's line and construct instead of the grounds operand, and promote dry-runs the append against the pre-flight bytes so a deterministic refusal mints nothing (iss-2608301803423101). Rejected: refusing every record carrying an unclosed opener, which would refuse writes that succeed — a `## Grounds` section ABOVE the opener takes the entry perfectly well. Rejected: a file-relative line number in the refusal, because the triage verbs append after setting their note field and the number would name a line the unwritten record on disk does not have; the number is body-relative and the opener's text is quoted beside it.
