---
schema_version: 1
id: "iss-293"
slug: "launch-parsesemver-discards-strconv-atoi"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/launch/semver.go"
resolution: "ParseSemver now returns strconv.Atoi's range error; watched-fail test added."
impact: fix
---

launch.ParseSemver discards strconv.Atoi's range error (major,_ := ...), and semverRe's numID is unbounded, so a >int64 version component clamps to MaxInt64: distinct oversized versions collide to one String() and the next Patch++ wraps negative — reachable via GitExistingTags (git tag list) and changelog.DeriveNext; the identical defect was already guarded in spec/spec.go
## Evidence

- `internal/core/launch/semver.go` — `ParseSemver` does `major, _ := strconv.Atoi(m[1])`
  (minor/patch likewise), discarding `strconv`'s out-of-range error; `semverRe`'s `numID`
  (`0|[1-9]\d*`) is unbounded and `IsStrictSemver` shares it, so no upstream cap rejects a
  20-digit component. `Atoi` clamps overflow to `MaxInt64`: two distinct oversized inputs
  collide to one `String()`, and `Patch++` wraps to `MinInt64` (a negative version).
- Reachable via `internal/core/launch/retention.go` `GitExistingTags` (parses `git tag`) into
  `ComputeRetention`, and via `internal/core/changelog` `LatestChangelogVersion` into
  `DeriveNext` where `next.Patch++` overflows.
- The identical `Atoi`-overflow defect is already guarded one package over in
  `internal/core/spec/spec.go` (`specNum`).

## Adversarial review

CONFIRMED (substantive, low-impact end) by an independent refuter: overflow reproduced;
both reach paths route through the swallowed error; the fetched-ref threat model the sibling
fix names does not exclude git tags / CHANGELOG headings. Fix: have `ParseSemver` propagate
the `strconv.Atoi` error (both arithmetic consumers already handle a non-nil error).
