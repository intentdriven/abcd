---
id: dcn-2607120000000004
slug: abcd-auto-loop-design-recorded-plans
date: 2026-07-12
---
2026-07-12 — /abcd:auto-loop design recorded (plans/2026-07-12-abcd-auto-loop-skill.md,
pending sign-off, not built). SOTA pass (sota-researcher, primary sources) backs the
design: durable handoff + fresh-context resume over compaction/RAG (Anthropic
long-running-harnesses — compaction "isn't sufficient"); delegate reads/reviews but
keep implementation in ONE agent (Cognition "Don't Build Multi-Agents" + Anthropic
multi-agent — converging read/write boundary); reviewers must be a SEPARATE fresh-
context lens, not intrinsic self-review (Huang et al. 2310.01798; CriticGPT
2407.00215); gate irreversible actions on action-class not self-confidence (RLHF
miscalibration); attempt-journal lineage = Reflexion (NeurIPS 2023) + database WAL.
Rejected: parallel multi-agent implementation, compaction-as-primary-continuity,
RAG-over-ledger at single-milestone scale.
