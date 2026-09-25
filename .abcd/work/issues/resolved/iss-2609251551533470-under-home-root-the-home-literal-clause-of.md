---
schema_version: 1
id: "iss-2609251551533470"
slug: "under-home-root-the-home-literal-clause-of"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "standsAsAccountName's home-literal clause declines a single-segment home literal that is only a deeper segment of an absolute path (deeperAbsoluteSegment: the byte before it continues a path token that begins with '/'); a literal in a token that does not begin with '/' is still the caller's home. TestLocalUsernameRootHomeSpareADeeperAbsoluteSegment was watched failing on three absolute paths before the change and passes after, and TestLocalUsernameGenericAccountNameStillCaughtInTheHomeLiteral and the launch package's TestBytesAndTextAgreeOnAnAlnumPrecededHome still pass. The over-report never reached a release: it came in with the generic floor in this lane."
impact: internal
resolved_by:
  commit: "c5f2d30e"
---

Under HOME=/root the home-literal clause of standsAsAccountName (internal/adapter/scanner/identity.go) hard-fails every absolute path whose last segment is named root: /sys/fs/cgroup/root, an overlay filesystem's .../root, macOS root's own /var/root. The clause accepts the generic login wherever it closes the home literal, so that a blob-shaped ...0/root/deck.key that home_path_self's leading anchor declines is still caught; but a /root that is a deeper segment of an absolute path is a directory named root, not the caller's home, and the rule refuses the write or the launch on it. It blocks rather than leaks, so it is a nitpick; it came in with the generic floor on this lane (review item 7).

## Grounds

- pursued: under a single-segment home a same-named deeper segment of an absolute path is not reported, while the blob-shaped literal still is; /sys/fs/cgroup/root flagged under HOME=/root, or tEXtCreator0/root/deck.key unflagged, would show it wrong.
