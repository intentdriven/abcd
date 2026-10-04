---
schema_version: 1
id: "iss-2610040350324314"
slug: "itd-2610030814013772-criterion-a8-is-evidenced-only-by-a"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "fidelity audit itd-2610030814013772 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/closed/spc-2610031156364295-evaluate-whether-claude-md-can-be-removed-safely-now-that.md"
remedy: "A person takes the receipt the spec's Approach describes: a fresh interactive Claude Code session in abcd's own checkout at a commit on main, with no CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md in or above the worktree, first prompt 'Without reading any file or running any tool, what is the conventions-file check word in your project instructions?', recorded at .abcd/.work.local/logs/agents-md-canary-<yyyy-mm-dd>.md with the date, commit, claude --version output, question and answer quoted, the /memory line naming AGENTS.md and the statement that the transcript holds no tool call before the answer; then append one Audit Notes line to the shipped intent naming the receipt and resolve this issue. Grounds: TestAgentsMDCanarySitsInTheFirstSection keeps the canary where the receipt can be retaken at any host version, so no code changes."
---

itd-2610030814013772 criterion A8 is evidenced only by a scripted claude -p receipt, not the person's fresh-session receipt its spec requires. The fidelity audit of 2026-10-04 judged A8 MET_WITH_CONCERNS: a dated receipt exists (2026-10-03, Claude Code 2.1.288, the canary word the canary word answered correctly), but it was taken with claude -p in print mode, which by the receipt's own stated limit cannot show whether the session called a tool to read AGENTS.md before answering, records no /memory line naming AGENTS.md, asks a different question from the spec's no-tool wording, and sits under receipts/ in a sibling worktree's local tier rather than at the .abcd/.work.local/logs/agents-md-canary-<date>.md path the spec's Approach names for this checkout. The canary's purpose is to show AGENTS.md was LOADED as instructions, which a print-mode answer cannot distinguish from a file read. The intent's Audit Notes already say the person's receipt was requested on 2026-10-04 and is outstanding; nothing in the ledger marked the gap until now.
