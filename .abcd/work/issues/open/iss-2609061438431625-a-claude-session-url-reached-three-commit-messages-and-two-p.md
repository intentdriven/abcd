---
schema_version: 1
id: "iss-2609061438431625"
slug: "a-claude-session-url-reached-three-commit-messages-and-two-p"
severity: "critical"
category: "security"
source: "user-observation"
found_during: "release-gate adoption in a managed public repo, 2026-09-06"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks (pre-commit name guard), internal (lint privacy-hygiene, guard)"
---

A Claude session URL reached three commit messages and two PR bodies of a managed public repo and nothing in the abcd guard stack stopped it — not acceptable. The lint policy text says a live session URL or a tool attribution footer is refused wherever it is committed, but privacy-hygiene only scans tracked files; the committed pre-commit name guard checks the private banlist (empty by default) and never the commit message; there is no commit-msg hook, and nothing looks at the text handed to the GitHub CLI for pull requests or issues. The harness's own attribution instruction (a Claude-Session trailer on every commit and PR) is exactly the pattern the policy names, so the guard must catch it mechanically: a commit-msg hook (and pre-merge-commit) that rejects claude.ai/code/session links and known AI attribution footers; a public banned-token family for those patterns so CI enforces it on every pushed commit message in the PR range, not only on files; and guard coverage of the GitHub CLI's pull-request and issue text. Recovery is expensive — a merged commit message can only be removed by rewriting a protected branch — so this has to fail before the commit exists.
