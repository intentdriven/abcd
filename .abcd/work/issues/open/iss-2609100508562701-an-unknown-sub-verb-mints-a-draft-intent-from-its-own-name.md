---
schema_version: 1
id: "iss-2609100508562701"
slug: "an-unknown-sub-verb-mints-a-draft-intent-from-its-own-name"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli (intent)"
---

An unknown sub-verb is treated as press-release text and mints a draft intent from its own name. Two independent sessions hit this in the same run, which makes it a likelihood rather than a possibility.

`abcd intent status <itd-id>` is not a verb. The bare `intent` verb treats any positional text as a press-release draft, so it filed a new draft intent whose title is the words "status <itd-id>". One session caught it and deleted the stray file by hand in its worktree; the other did the same. A less careful run — which is the ordinary case in an unattended session — leaves a nonsense record in `drafts/`, minted through the shared id allocator, indistinguishable in the store from a real one.

The help text does say that quoted text files a draft. That is not the problem. The problem is that a bare record id, or a single common sub-verb name like `status` or `show`, is far more likely a query than a press release, and abcd already applies exactly this reasoning elsewhere: the lone-token rule refuses a single word rather than filing it.

Wanted: refuse a first positional argument that looks like a sub-verb name or contains a record id, with a did-you-mean, instead of filing it. And, as a cheap second guard, print the path the verb is about to create before writing it, so an operator who is about to mint something unintended sees it happen.
