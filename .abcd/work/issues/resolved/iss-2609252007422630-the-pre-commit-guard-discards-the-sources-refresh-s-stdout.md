---
schema_version: 1
id: "iss-2609252007422630"
slug: "the-pre-commit-guard-discards-the-sources-refresh-s-stdout"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
resolution: "both copies of the guard relay the refresh binary's output (its one-line count, or its refusal naming keys) instead of discarding it, and name a binary without the verb in one remedy line"
impact: fix
resolved_by:
  commit: "46af0661"
---

The pre-commit guard discards the sources refresh's stdout and stderr (>/dev/null 2>&1), so an older abcd that has the source verb but projects fewer patterns rewrites the generated block weaker and the commit prints nothing about it: a silent downgrade. A binary with no source verb prints 'refresh failed' on every commit, noise that teaches the committer to ignore the one line that matters. The guard should print the binary's one-line count, and one clear remedy line when the binary lacks the verb.

## Grounds

- pursued: a refresh that wrote fewer patterns is visible on the commit that ran it; a commit whose refresh output is missing, or a verb-less binary reported as a failure, would show it wrong
