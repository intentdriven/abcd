---
schema_version: 1
id: "iss-2610151422138805"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Rewrite .abcd/development/brief/05-internals/03-configuration.md:667 to match the shipped surface: The RepoPrompt adapter is not staged: internal/adapter/repoprompt ships and the chapter's own 'reviewer route's keys' section (oracle.review = rp) describes it as read by `implement step`. No .abcd/rp/ namespace is written, so only the 'staged with its…"
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:667 says: Two namespaces are not part of [repo scope]: `logbook/`, a retired name, and `rp/`, staged with its adapter. The shipped surface differs: The RepoPrompt adapter is not staged: internal/adapter/repoprompt ships and the chapter's own 'reviewer route's keys' section (oracle.review = rp) describes it as read by `implement step`. No .abcd/rp/ namespace is written, so only the 'staged with its adapter' reason is stale. (v0.14.2 release crosscheck, false-claim, raised by opus in a34.)
