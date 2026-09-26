---
schema_version: 1
id: "iss-194"
slug: "release-retention-is-documented-in-the-present-tense-but-has"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "The code half: the preview's retention plan reads the release-tag listing's error: a failed listing refuses the plan with its reason and reaches would_refuse_on (TestPreviewRetentionRefusesWhenTagsAreUnreadable). The doc-currency half was already fixed at tip: the brief states the prune is computed and previewed, not executed (itd-70)."
impact: fix
resolved_by:
  commit: "a48f83238131"
---

Release retention is documented in the present tense but has never run: the newest-per-line policy in brief/04-surfaces/04-launch.md:134 and the release glossary entry describe pruning as operating behaviour, and internal/core/launch/retention.go:30 implements it exactly, but nothing executes it. internal/core/launch/ship.go:41 stops before the network ('the real GitHub Release + SLSA + tag push + retention prune are a later phase'), and .github/workflows/release.yml — which publishes every release this project has actually cut — carries no retention step at all. The brief's own phase note schedules the prune as itd-70, unshipped. Same implemented-but-never-run class as iss-183, and found the same way: by reading the record against the live surface. Live evidence, 2026-08-07: six releases exist; the real preview keeps v0.1.0, v0.2.0, v0.3.0, v0.4.2 and would prune v0.4.0 and v0.4.1, so the repository's own release list contradicts the prose. Scope here is the doc-currency half only — state that the policy is computed and previewable but not yet enforced, on the channel-truthful pattern iss-183 established; wiring the prune is itd-70 and is not assumed by this item. Maintainer disposition (2026-08-07): v0.4.0 and v0.4.1 are deliberately left in place rather than pruned by hand, so itd-70 has a live fixture to exercise on arrival — the first real prune should be the shipped mechanism's, not a manual one. Related observation, possibly separable: internal/core/launch/dryrun.go:126 discards the error from GitExistingTags, so a shallow or tagless checkout yields an empty existing set and the retention preview reports nothing to prune — indistinguishable from a genuine nothing-to-prune. Reproduced in a shallow clone during this investigation before tags were fetched.

## Grounds

- pursued: a checkout whose tags cannot be listed now previews a refused retention plan naming the listing failure instead of an empty prune; a failed tag listing that still previews nothing-to-prune would show it wrong. A shallow checkout whose listing SUCCEEDS but is incomplete is not detected by this fix.
