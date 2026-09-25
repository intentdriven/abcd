---
schema_version: 1
id: "iss-337"
slug: "probe-go-s-inner-directory-descent-dirroot-openroot-name-is"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/core/lifeboat/probe.go"
resolution: "Both walk descents (probe and embark) and the probe's start-directory open go through openWalkDir, which opens rel/. so every component is opened with O_DIRECTORY|O_NOFOLLOW and a FIFO or device swapped in is refused with ENOTDIR without blocking; TestWalkDescentDoesNotBlockOnFifo and TestWalkFilesStartDoesNotBlockOnFifo hold it."
impact: fix
resolved_by:
  commit: "f8f2ce13"
---

probe.go's inner directory descent (dirRoot.OpenRoot(name)) is guarded only by the parent ReadDir's IsDir, so a directory swapped for a FIFO in that window blocks — same TOCTOU shape as readLifeboatFile but race-only and unfixable via the nonBlock const (os.Root.OpenRoot takes no flags); needs a raw openat restructure, recorded not fixed

## Evidence

- `internal/core/lifeboat/probe.go:406` -- `dirRoot.OpenRoot(name)` guarded only by the parent ReadDir's IsDir; `os.Root.OpenRoot` opens with O_NOFOLLOW|O_CLOEXEC, no O_DIRECTORY/O_NONBLOCK, so a directory swapped for a FIFO in that window blocks.

## Verifier verdict -- CONFIRMED, recorded not fixed

Real TOCTOU, same shape as iss-341, but race-only (needs an active writer in the probed tree) and unfixable with the nonBlock const because os.Root.OpenRoot takes no flags -- a fix means a raw openat(O_DIRECTORY|O_NOFOLLOW|O_NONBLOCK) or an OpenInRoot-style restructure. Filed separately from the iss-342/iss-341 same-sweep fixes by the verifier's own recommendation; larger than this round should bundle.

## Grounds

- pursued: a walk over a tree whose listed directory becomes a FIFO returns promptly instead of hanging disembark; shown wrong if a toolchain stops opening os.Root intermediate components with O_DIRECTORY, which the descent test would then catch by timing out
