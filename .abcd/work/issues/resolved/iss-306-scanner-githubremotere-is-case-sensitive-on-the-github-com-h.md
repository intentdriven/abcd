---
schema_version: 1
id: "iss-306"
slug: "scanner-githubremotere-is-case-sensitive-on-the-github-com-h"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/adapter/scanner/identity.go"
resolution: "githubRemoteRe is now (?i); a mixed-case GitHub.com remote no longer disables the detector or raises a spurious hard_fail."
impact: fix
---

scanner githubRemoteRe is case-sensitive on the github.com host, disabling the github_username detector on a mixed-case remote and spuriously hard-failing when name==handle
## Evidence
`internal/adapter/scanner/identity.go:94` — `githubRemoteRe` compiles the pattern `github\.com[:/]([A-Za-z0-9-]+)/`, which has no `(?i)`. git stores `remote.origin.url` byte-verbatim, so `git@GitHub.com:Alex/repo.git` or `https://GITHUB.COM/...` yields an empty capture. `ProbeIdentity` (`:56-73`) then leaves `GitRemoteUsername` empty, `m.github` (`:160`) is never compiled, and the `github_username` warn detector is silently disabled for that checkout.

The neighbouring matchers `m.homeSelf/m.email/m.name/m.github` and both noreply regexes all carry `(?i)` with comments naming this exact case-fold reason — the extractor feeding them is the missed site.

## Adversarial verdict: CONFIRMED (substantive-low)
Reproduced end-to-end through the real `ProbeIdentity`/`ScanText`: mixed-case remote → 0 findings vs lowercase → 1. Second consequence: the empty username also flips the first arm of `m.nameEqGithub` (`:168`), producing a spurious `real_name` **hard_fail** ship-blocker when `user.name` equals the handle and the email is non-noreply — partially re-opening resolved iss-283. Fix: add `(?i)` to `githubRemoteRe`. FP impact nil — the regex's only input is `remote.origin.url`, never committed text; downstream `strings.EqualFold`/`(?i)` make captured case irrelevant. Not prior art: the `(?i)` sweep provenance names m.email/m.github/m.homeSelf, never githubRemoteRe.
