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
---

The pre-commit guard discards the sources refresh's stdout and stderr (>/dev/null 2>&1), so an older abcd that has the source verb but projects fewer patterns rewrites the generated block weaker and the commit prints nothing about it: a silent downgrade. A binary with no source verb prints 'refresh failed' on every commit, noise that teaches the committer to ignore the one line that matters. The guard should print the binary's one-line count, and one clear remedy line when the binary lacks the verb.
