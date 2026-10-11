---
schema_version: 1
id: "iss-309"
slug: "launch-parsesemver-discards-strconv-atoi"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/launch/semver.go"
resolution: "ParseSemver returns an error on an over-int64 component instead of clamping to MaxInt64."
impact: fix
---

launch ParseSemver discards strconv.Atoi range error so an over-int64 version component clamps to MaxInt64 and can wrap negative in DeriveNext
## Evidence
`internal/core/launch/semver.go:43-45` — `major, _ := strconv.Atoi(m[1])` (×3) discards ErrRange; the regex admits an unbounded digit run (`0|[1-9]\d*`). A tag `v99999999999999999999.0.0` parses to Major=MaxInt64 silently. `ComputeRetention` then refuses a cut naming a non-existent tag; two over-int64 tags collapse to one lossy Semver. `spec/spec.go:115-130` handles this exact case explicitly — this is the un-swept sibling the principles-doc meta-rule predicts.

## Adversarial verdict: CONFIRMED (substantive-low)
Atoi clamp reproduced. Reachable via `abcd launch --dry-run` → ComputeRetention, and via `changelog.DeriveNext` where `next.Major++` on a breaking bump wraps MaxInt64→MinInt64, producing `v-9223372036854775808.0.0` — the wrap the spec.go fix warns of. Input is the repo's own git tags (self-inflicted, not attacker), hence low. Fix: mirror spec.go — return an error on ErrRange; `GitExistingTags` already drops erroring tags, so behaviour is preserved for all valid tags.
