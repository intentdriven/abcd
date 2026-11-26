---
schema_version: 1
id: "iss-2611251839554213"
slug: "guard-reads-git-c-with-guard-allow-as-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Judge the store write by a write verb, or a redirect, whose target resolves under the store, and read `git -C` with a reading sub-command as a read. Grounds: the refused text is an argument to a search and no path."
---

The shell guard refuses a read as a write into its allow store (guard-allow-store-write) whenever the line holds `git -C <dir>`, for a literal or a variable directory, and the text guard-allow anywhere: `git -C /tmp/w log -S guard-allow` and `git -C "$W" log --oneline; echo guard-allow` are refused, and `git log -S guard-allow` is not. Nothing in either writes, and neither names the store. Tested on 2026-11-25 on the installed 0.16.0 and on this tree; a reviewer reading the history of the rule lost a whole compound command to it.

2026-11-25: The same refusal met a delete of an uncommitted issue record whose file name holds the text, with no path into the store: the text in any argument is read as the path.
