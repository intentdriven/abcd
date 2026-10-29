---
schema_version: 1
id: "iss-2610222047465878"
slug: "loop-samepath-is-case-sensitive-on-a"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review pair of fix/autonomous-permission-ask-denied, 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/loop.go"
remedy: "Compare with os.SameFile on os.Stat results when both paths exist, falling back to the resolved-string comparison when either does not, as ahoy's samePath does; consider making that one helper in fsutil and sweeping the loop, ahoy and embargo call sites onto it, keeping embargo's refusal on a mismatch"
---

The implement loop's samePath (internal/core/implement/loop/loop.go:1024) decides whether two paths name one directory by comparing their cleaned, symlink-resolved strings, so on a case-insensitive file system (macOS APFS by default) two spellings of one directory that differ only in case compare unequal. Its callers in the loop are unattended.go:64 and :75 (is this session a running build's lane), judgement.go:327, ratelimit.go:155, configwatch.go:273 and schedule.go:115. For the permission-prompt refusal the failure is open: an unattended lane whose host reports its directory in a different case is not recognised, and the host's prompt shows and waits, which is the behaviour before the refusal existed. Paths there normally come from git and the host's own working directory, so a mismatch is unlikely. ahoy has its own samePath that compares os.Stat results with os.SameFile, which handles case and links together; embargo's compares resolved strings and refuses on a mismatch, failing closed. Found during the review of the permission-prompt refusal, deferred from that change so as not to alter a helper six other loop call sites share.
