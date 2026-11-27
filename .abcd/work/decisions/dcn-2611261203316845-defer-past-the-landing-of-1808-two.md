---
id: dcn-2611261203316845
slug: defer-past-the-landing-of-1808-two
date: 2026-11-26
---
2026-11-26 — Defer, past the landing of #1808, two review notes on it, to fix in a change of their own on main once it lands: (1) minor: the SubagentStart hook (internal/surface/cli/hook_subagent_start.go, the fact write before the deadline select) can keep a told fact for a sub-agent that was never told, when the deadline and the work's end race or the stdout write fails, so its stop records rules-not-run beside start-failed; (2) nitpick: the word-reader audit's reason for assignmentStart (internal/core/guard/unknownsites_test.go) says it steps only fixed text, while it steps repeat's count whatever it is. Grounds: neither lets a command through or loses work, #1808 holds a review pair for its head, and changing the head again costs a full review and preflight cycle. Reverse: fix both in #1808 before it merges.
