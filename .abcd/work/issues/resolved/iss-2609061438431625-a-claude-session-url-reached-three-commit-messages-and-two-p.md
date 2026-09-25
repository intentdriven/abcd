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
deferred_after: "v0.8.0"
deferral_reason: "Half of this is fixed in this cut and half of it cannot be, which is why the record stays open rather than being resolved. The CI half is closed: a check-direction primitive now judges outbound text against the same policy the scrubber holds, with its own front door, wired over every commit message in a pull request's range and over the pull request body. What is not closed is the local gate, and it cannot be without three product decisions nobody has taken. A git hook has no plugin root, so only the PATH rung of the hardened four-rung ladder survives in one, and the choice between failing closed on a missing binary, failing open, or baking an absolute path that a plugin update then invalidates is a decision whose blast radius is every managed repository. Whether such a hook installs by default or opt-in is a second decision with the same reach. Deferred so those are taken deliberately rather than inside a release. The exposure that remains is named in the record: a leaked message is already in the author's history before CI ever runs."
found_at: "hooks (pre-commit name guard), internal (lint privacy-hygiene, guard)"
resolution: "The local gate landed for this repository: the committed .githooks/commit-msg hook runs go run ./cmd/abcd lint outbound on every commit and merge message and refuses a live session URL or a tool attribution footer before the commit exists, failing closed when it cannot run. The CI half (every commit message of a pull request and its body) landed earlier. The managed-repository form of the hook and the forge-CLI text guard need three product decisions and are carried by iss-2609250834251447."
impact: fix
resolved_by:
  commit: "aebace46ea69c27bcf2db33c85dfdc57ce33d75a"
---

A Claude session URL reached three commit messages and two PR bodies of a managed public repo and nothing in the abcd guard stack stopped it — not acceptable. The lint policy text says a live session URL or a tool attribution footer is refused wherever it is committed, but privacy-hygiene only scans tracked files; the committed pre-commit name guard checks the private banlist (empty by default) and never the commit message; there is no commit-msg hook, and nothing looks at the text handed to the GitHub CLI for pull requests or issues. The harness's own attribution instruction (a Claude-Session trailer on every commit and PR) is exactly the pattern the policy names, so the guard must catch it mechanically: a commit-msg hook (and pre-merge-commit) that rejects claude.ai/code/session links and known AI attribution footers; a public banned-token family for those patterns so CI enforces it on every pushed commit message in the PR range, not only on files; and guard coverage of the GitHub CLI's pull-request and issue text. Recovery is expensive — a merged commit message can only be removed by rewriting a protected branch — so this has to fail before the commit exists.

## Grounds

- pursued: a commit or merge whose message carries a live session URL or a tool footer is refused in this repository before any commit object exists; a leaked URL reaching a commit made through git commit or git merge here with the hooks path armed would show it wrong
- pursued: review of the lane met the falsifier above before the resolution merged — an inherited grep or awk function, a PATH-prepended awk, and a scissors look-alike in a `git commit -F` message each let a leaked URL through, and a tree that did not build was reported as a finding against the message; the hook now pins its environment as the pre-commit guard does, cuts only at git's own scissors line with a verbose diff below it, and builds ./cmd/abcd before it judges, so the resolution's `go run` names the earlier mechanism; a -F message forging both the scissors and a diff header, or a substituted go that forges a pass, would still show it wrong, and CI's check over the recorded message is the backstop for both
