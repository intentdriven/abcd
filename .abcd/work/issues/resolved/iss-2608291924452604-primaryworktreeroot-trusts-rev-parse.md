---
schema_version: 1
id: "iss-2608291924452604"
slug: "primaryworktreeroot-trusts-rev-parse"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "v0.6.9-security-review"
found_at: "internal/core/banlist/worktree.go"
resolution: "PrimaryWorktreeRoot holds every rev-parse answer to one absolute line through gitutil.RevParseAbsPath, so a git that echoes the unknown --path-format option fails the resolution closed by refusal, matching the pre-commit guard's outcome on the same git; the flag is kept for guard parity. Swept: report/inbox.go repoName reads the same answer but cannot mis-name the sender on an old git (a two-line answer's base is .git only when the path ends there). TestPrimaryWorktreeRootRefusesAnAnswerThatIsNotOnePath pins it."
impact: internal
resolved_by:
  commit: "f439faf84"
---

PrimaryWorktreeRoot trusts rev-parse stdout: git rev-parse echoes an unrecognised option to stdout and exits 0, and --path-format only exists from git 2.31, so on an older git the --git-dir, --git-common-dir and --show-toplevel answers read as the flag text plus the path and every comparison silently fails closed to no primary store; validate each answer as a single absolute path or drop the flag

## Grounds

- pursued: no inherited private store is ever resolved from a rev-parse answer that is not one absolute path; a primary resolved under a git that prefixes its answers would show it wrong
