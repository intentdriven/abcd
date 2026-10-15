---
id: dcn-2607140000000014
slug: adopted-splitting-the-conformance
date: 2026-07-14
---
2026-07-14 (review): ADOPTED splitting the conformance reviewer into a terse binary
verdict call and a SEPARATE explanation call. Asking for verdict+explanation+fix in
one call drives GPT-4o's false-negative rate from 26.2% to 73.2% (HumanEval) and
35.9% to 87.9% (MBPP) — it does not find more bugs, it REJECTS CORRECT CODE
(arXiv 2603.00539, Springer Automated Software Engineering). All five abcd agents
currently do the forbidden thing. Largest measured effect size in the research and
free to apply. Also adopted: fix-guided verification filter (execute the proposed fix
as a counterfactual; if no test outcome changes, the rejection was hallucinated —
FNR 54.8% -> 16.3%). Full evidence:
.abcd/development/research/notes/2026-07-14-research-platform-benchmarks.md
