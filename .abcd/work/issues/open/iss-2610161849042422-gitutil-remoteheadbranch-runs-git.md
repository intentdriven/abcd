---
schema_version: 1
id: "iss-2610161849042422"
slug: "gitutil-remoteheadbranch-runs-git"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "review of the land origin/HEAD fix, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/remotehead.go"
remedy: "Let gitutil.RemoteHeadBranch take the caller's git runner (or add a landing seam for it), so abcd land's tests can record the ls-remote call and fake an unreachable or odd answer through Options.Git, as they do for push and fetch"
---

gitutil.RemoteHeadBranch runs git ls-remote through gitutil.Net directly rather than through abcd land's Options.Git seam, so landing tests cannot record the call or fake an answer: they need a real local bare origin for every case, and the unreachable-origin and malformed-answer cases are reached only by building such a repository. Found by the Opus reviewer of the origin/HEAD fix (iss-2610161831078335), 2026-10-16.
