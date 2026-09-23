---
schema_version: 1
id: "iss-62"
slug: "managed-repo-identity-gate"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "git-identity-incident"
related_intents: [itd-131]
---

abcd-managed repos must guarantee the user's chosen git identity (name + email) is honoured on every commit. ahoy doctor should detect when the effective git author/committer identity diverges from the user's GitHub identity — a stray repo-local user.name/user.email override (e.g. a sandbox 'Test User <test@example.com>' shadowing the global identity), or committer != author — and, rather than silently setting it, PROPOSE a default (the global git identity, or the gh-authenticated account) and ASK the user to confirm before writing repo-local config. Motivation: a repo-local 'Test User <test@example.com>' override silently authored/committed 54 commits (pushed to origin) before detection this run, forcing a history rewrite + force-push to correct. A managed-repo identity gate at ahoy/doctor time catches it before the first commit.