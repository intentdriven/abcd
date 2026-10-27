---
schema_version: 1
id: "iss-2610262032120773"
slug: "itd-48-puts-role-2-in-agents"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 236 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:54"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-d79955a53018e72b
---

itd-48 puts Role 2 in `agents/intent-fidelity-reviewer.md`; the naming register says the cross-document role belongs to the intent auditor, `agents/intent-auditor.md`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:54` — "Extend `agents/intent-fidelity-reviewer.md` with a Role 2 prompt section"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:189` — "`intent_consistency` names the intent auditor's cross-document role, which reads the brief and the intents together for contradictions (`agents/intent-auditor.md`)"

itd-48 adds the Role 2 prompt section to `agents/intent-fidelity-reviewer.md`. 04-naming's `task_classes` row assigns the same cross-document role to the intent auditor, in `agents/intent-auditor.md`. The same agent and role go by two names and two file paths, so a reader cannot tell which file holds the role.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 236.
