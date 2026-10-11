---
schema_version: 1
id: "iss-2609300939590291"
slug: "once-only-breaks-under-concurrent"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/unseen_update.go"
remedy: "Claim each release with one exclusive create, os.OpenFile(cache/update-shown-<to>, O_WRONLY|O_CREATE|O_EXCL, 0o600), after the dataDirHazard and ReleaseTagShape checks; an existing path or any create error shows nothing (fail closed). O_EXCL is atomic per POSIX open(2), so exactly one concurrent caller wins; a 16-goroutine test holds it."
resolution: "The session check claims each release's unseen update with one exclusive create (O_CREATE|O_EXCL) of cache/update-shown-<tag>, or .update-shown-<tag> in the plugin root in the per-root mode; an existing path or any create error shows nothing, so exactly one of any number of concurrent session starts shows the line."
impact: fix
resolved_by:
  commit: "4fb3575b8"
---

Once-only breaks under concurrent session starts: ahoy.TakeUnseenUpdate read the cache/update-shown marker, then wrote it with a create-temp-and-rename, a read-then-write with no claim, so N sessions starting together in one plugin root (an autonomous run does this within a second) each saw no marker and each showed 'abcd updated from X to Y'; a 16-goroutine probe on one seeded cache showed the line 16 times, breaking the ruling CJ1b's once-only.

## Grounds

- pursued: 16 goroutines calling TakeUnseenUpdate on one seeded cache show the line exactly once (TestTakeUnseenUpdateShowsOnceUnderConcurrentSessions, 16 before the fix); a second show of one release from any interleaving, or a claim created through a link, would show it wrong.
