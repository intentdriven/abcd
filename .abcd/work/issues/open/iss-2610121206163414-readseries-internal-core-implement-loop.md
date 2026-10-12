---
schema_version: 1
id: "iss-2610121206163414"
slug: "readseries-internal-core-implement-loop"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "fixing iss-2610120925598775 on fix/build-next-followups"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/nextseries.go"
remedy: "Shape-check each pace setting's layer against the layers abcd writes and its origin against what it records (or refuse any control or bidi rune) when readSeries and ReadState decode a Pace, refusing the file as readSeries refuses an unknown outcome; tests feed U+202E and U+009B in a layer and an origin and expect the refusal."
refines: [iss-2610120925598775]
---

readSeries (internal/core/implement/loop/nextseries.go) still accepts the stored pace provenance strings in .abcd/.work.local/run/next.json unchecked: each pace setting's layer and origin (pace.*.layer, pace.*.origin) is free text that reaches 'abcd build next --json' raw, so a hand-written or repository-shipped series file can carry control or bidi runes there, the same display-spoofing class iss-2610120925598775 closed for the stored flags. The same Pace shape is read from a run's state.json and echoed by 'implement record --json'. Found by the implementer fixing iss-2610120925598775.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The series and run state files, their readers and the `--json` streams that echo them are abcd's own, so abcd added it and fixes it; a fix refuses a malformed file abcd itself reads and changes nothing about how the repository works.

A third route, named by the security reviewer of fix/build-next-followups: continueSeries given a pace flag that differs from the stored pace calls resumeWithFlags (internal/core/implement/loop/loop.go), whose refusal reason includes res.Pace.String(); describeOrigin (pace.go) concatenates each setting's origin and layer raw, and loopFail sanitizes only the text line, so the JSON refusal carries them unmasked. The fix's test should cover that refusal reason too.
