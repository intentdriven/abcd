---
schema_version: 1
id: "iss-2609261614268567"
slug: "the-pre-commit-identity-guard-githooks-pre-commit-reads-the"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd131"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
---

The pre-commit identity guard (.githooks/pre-commit) reads the author and committer from git config only and never from GIT_AUTHOR_NAME/GIT_AUTHOR_EMAIL or GIT_COMMITTER_NAME/GIT_COMMITTER_EMAIL, which git ranks above every config key. A machine identity set in the environment, the shape a harness or CI runner uses, therefore commits past the only pre-commit enforcement in a pinned repository: GIT_COMMITTER_NAME=Claude GIT_COMMITTER_EMAIL=noreply@anthropic.com git commit lands a Claude committer with the hook exiting 0. git also exports the resolved GIT_AUTHOR_* to the hook, so git commit --author and git -c author.name get past it in the same way. The comment claiming the committer is held to the pin overclaims.
