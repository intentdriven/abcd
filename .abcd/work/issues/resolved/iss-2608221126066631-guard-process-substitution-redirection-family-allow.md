---
schema_version: 1
id: "iss-2608221126066631"
slug: "guard-process-substitution-redirection-family-allow"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "bughunt round 7 merge-gate dual review"
found_at: "internal/core/guard/tokenize.go"
resolution: "Process substitution now suspends and resumes the enclosing command like $( ), leaving one /dev/fd operand in its place, so a blocker flag after >(...) or <(...) is still read."
impact: fix
resolved_by:
  commit: "2733f2538576334f3053995f4f85e96e1bd8bd8b"
---

The guard tokenizer keeps process-substitution operands out of command position analysis, so a blocker-tier flag glued behind one escapes: 'git push >(cat) --force origin main' and 'git push >$(echo x) --force origin main' both return ALLOW while their plain-redirection spellings block (pre-existing on main; confirmed unchanged by the iss-2608220131352917 &> fix, same redirection family). Within the documented mistake-filter posture, but the &> precedent shows the family is worth sweeping: recognise >(...) / <(...) as redirection-shaped operands and keep the remaining argv in analysis.

## Grounds

- pursued: a process substitution is one operand of the enclosing command; shown wrong by a flag written after one escaping its entry (TestProcessSubstitutionIsAnOperand)
