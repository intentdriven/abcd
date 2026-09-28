---
schema_version: 1
id: "iss-62"
slug: "managed-repo-identity-gate"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "git-identity-incident"
related_intents: [itd-131]
resolution: "Delivered by itd-131 (spc-34): ahoy and doctor check the effective author and committer against the pin and flag a machine identity, and the install proposes the pinned (else global) identity and writes repo-local config only on a confirm at a terminal. The gh-authenticated fallback the capture floated is excluded by itd-131's decision (adr-38: no network lookup in an implicit path)."
impact: additive
resolved_by:
  intent: "itd-131"
  spec: "spc-34"
  commit: "f3a937d7"
---

abcd-managed repos must guarantee the user's chosen git identity (name + email) is honoured on every commit. ahoy doctor should detect when the effective git author/committer identity diverges from the user's GitHub identity — a stray repo-local user.name/user.email override (e.g. a sandbox 'Test User <test@example.com>' shadowing the global identity), or committer != author — and, rather than silently setting it, PROPOSE a default (the global git identity, or the gh-authenticated account) and ASK the user to confirm before writing repo-local config. Motivation: a repo-local 'Test User <test@example.com>' override silently authored/committed 54 commits (pushed to origin) before detection this run, forcing a history rewrite + force-push to correct. A managed-repo identity gate at ahoy/doctor time catches it before the first commit.

## Grounds

- pursued: a sandbox override or a divergent committer is reported before the first commit and mended only on a person's confirm; a repo-local override that commits without being reported, or a write made without a confirm, would show it wrong
