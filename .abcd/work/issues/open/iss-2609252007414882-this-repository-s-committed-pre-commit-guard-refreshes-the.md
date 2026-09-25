---
schema_version: 1
id: "iss-2609252007414882"
slug: "this-repository-s-committed-pre-commit-guard-refreshes-the"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
---

This repository's committed pre-commit guard refreshes the sources corpus's generated banlist block with an INSTALLED abcd (the pinned PATH, then ~/.local/bin), which in this source checkout is the last release and stale by construction, against the dogfooding rule and against the commit-msg hook, which builds ./cmd/abcd from the checkout. The guard should build the checkout's own abcd the way commit-msg does, print one warning line and proceed when the build fails, and never resolve an installed binary.
