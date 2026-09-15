---
schema_version: 1
id: "iss-2609020219198779"
slug: "the-rules-root-bound-from-ghsa-vvqc-3mv2-5p49-does-not-close"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/root.go"
---

The rules-root bound from GHSA-vvqc-3mv2-5p49 does not close the user-scope ~/.abcd when the home directory is itself a git working tree (dotfiles-in-home). ResolveRoot bounds the walk at the toplevel, and for a session in a non-repo directory beneath a version-controlled home that toplevel IS the home, so the home-scope rules.json and guard.json still govern the session: injected rules, the kill switch and the guard registry all come from a file outside any project. Cross-UID /tmp is NOT closed either, contrary to what this record first said: git refuses on ownership and the .git-marker fallback then resolves the planted tree's OWN root, which is precisely what a foreign-uid `git init` in a shared directory wants — that residual is recorded separately as iss-2609020259564193 and needs an ownership policy. What is closed is the one-command plant: a bare `.git` file or an empty `.git` directory no longer bounds the fallback, because the marker must now look like a repository (a `.git` directory carrying HEAD, or a `.git` file beginning "gitdir: "). A version-controlled home and a foreign-uid repository both remain open. Evidence: ResolveRoot in internal/core/rules/root.go. The fix needs a decision first — whether a home-directory toplevel is a legitimate configuration scope (spc-23 plans a user layer that would make it one) or must be excluded outright — so nothing is changed here; the doc comment now states the exact scope and names this record.
