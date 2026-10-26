---
schema_version: 1
id: "iss-2610262229350942"
slug: "gh-pr-create-fill-fill-first-and"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review of fix/catch-ci-failures-before-push"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/forge.go"
remedy: "Treat a --fill flag on a posting gh pr command as a body the line does not fix: mark the post opaque so the check warns (forge-text-unread), with a case per spelling."
wontfix_reason: "gh composes a --fill body from the commits, which the guard does not read and commands/guard.md already names as unseen; the mention check stands down when it sees no body, so --fill causes no misjudgement, and abcd land composes a checked body instead"
---

gh pr create --fill, --fill-first and --fill-verbose have gh compose the title and body from the branch's commits, which the shell guard's forge-text check neither judges nor reports as unread, so a body the attribution gate refuses after the push is posted with no warning.

## Grounds

- declined: gh composes a --fill body from the commits, which the guard does not read and commands/guard.md already names as unseen; the mention check stands down when it sees no body, so --fill causes no misjudgement, and abcd land composes a checked body instead
