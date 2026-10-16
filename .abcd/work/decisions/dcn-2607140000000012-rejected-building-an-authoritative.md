---
id: dcn-2607140000000012
slug: rejected-building-an-authoritative
date: 2026-07-14
---
2026-07-14 (research): REJECTED building an authoritative benchmark from abcd's own
data. Three independently fatal defects: (1) the corpus is n=2 labelled triples, both
graded by the model that wrote the code, with exactly one NOT_MET; (2) circularity —
the system would generate the spec, write the code, judge conformance AND supply the
labels, and self-preference appears perplexity-driven so it SURVIVES swapping model
family; (3) N=1 has no methodological remedy (Epoch attacks SWE-bench Verified for
concentration at 12 repos; we have one). Arithmetic also fails: ~500 oracle-backed
tasks needed to resolve model differences, in-situ A/B needs 2,200-15,500 paired
observations. CHOSEN instead: a task-quality instrument + methodology, validated
against an EXTERNAL reference — OpenAI's 731-task SWE-bench Pro audit. Ground truth
must originate outside the tool (executable test or retrospective event oracle, never
an LLM judge). Template is Aider polyglot: the tool supplies the harness and
distribution; ground truth comes from outside (225 external Exercism exercises).
