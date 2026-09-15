---
schema_version: 1
id: "iss-2608291814568971"
slug: "home-path-backstop-unanchored-and-fail-closed-check-dead"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/redact.go"
resolution: "scanner.SweepCallerHome replaces the home only where it stands as a path — the name may continue only with an alphanumeric byte, so /rootfs and /Users/alexandra are another name and left intact, while a punctuation suffix such as /root-cause or /root.old is swept to the safe side — the home_path_self detector and Redact's placeholder rewrite apply the same anchor, and the memory store's dead Contains check is now scanner.SurvivingCallerHome — the boundary-aware survivor gate history already holds"
impact: fix
---

ultra-v0.6.8 C3: the literal HOME backstop in internal/core/memory/redact.go (and its twin in internal/core/history/history.go) is strings.ReplaceAll(text, home, "~") with no path-boundary anchor, so with HOME=/root a page body naming /rootfs/etc/hosts or 'the /root-cause' is silently rewritten to ~fs/etc/hosts and 'the ~-cause' before it is committed; any short home ([redacted-path] inside [redacted-path]/...) corrupts the same way. The strings.Contains(redacted, home) fail-closed check that follows ReplaceAll can never be true, so the 'home path survived redaction' refusal is dead code. Fix once in the shared home (C7): anchor the replacement on a path boundary and make the survivor check boundary-aware so it can actually fire.
