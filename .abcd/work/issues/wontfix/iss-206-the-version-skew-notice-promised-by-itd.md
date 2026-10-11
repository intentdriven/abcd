---
schema_version: 1
id: "iss-206"
slug: "the-version-skew-notice-promised-by-itd"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "first manual plugin install test (2026-08-10)"
found_at: "internal/surface/cli/skew.go"
related_intents: [itd-108]
related_issues: [iss-205]
wontfix_reason: "the issue's own instruction defers to itd-108, which is still in drafts; premise may be superseded before it ever applies"
---

The version-skew notice promised by itd-105 ('Given an installed surface newer than the newest published binary, when a session starts, then the skew is surfaced') can never fire, and fails silently. itd-105's warrant says the harness names each plugin cache directory for the commit it was cloned from; that holds only as a 12-character SHORT sha. Observed .binary-meta from the first manual install (2026-08-10): plugin_root_basename=872f0a1e6aab, which is exactly 'git rev-parse --short=12 HEAD', and plugin_sha=unknown. hooks/bootstrap.sh gates plugin_sha on exactly 40 hex chars, and internal/surface/cli/skew.go resolvedSHA() requires 40 again, so plugin_sha is permanently 'unknown' and binarySkewNotice() returns "" forever on every machine. This is precisely the failure the comment at hooks/bootstrap.sh anticipated and armed a diagnostic for — recording the raw ungated plugin_root_basename beside the gated field so that 'why has the skew notice never fired' has an answer in the file rather than only in a comment. The diagnostic worked exactly as designed; the gate is what is wrong. Fix direction: accept 7-40 hex in both places and compare on the shorter value's prefix length, since one side is a short sha and the other a full one.

**Do not fix this ahead of itd-108 — the premise may be about to disappear (2026-08-10).** Two ways itd-108 bears on this record. First, it adds a SECOND, independent cause: an archive-sourced install's cache directory is not named for a commit at all, so widening the hex gate would still leave plugin_sha unresolvable. Any fix must stop assuming the basename is a commit rather than merely accept a shorter one. Second and more consequentially, itd-108 aims to make the surface and the binary resolve to the same cut by construction — the catalog names the latest release, and a plugin update creates the fresh plugin root that makes bootstrap.sh fetch the matching binary from that same release. If that lands, surface-ahead-of-binary skew is not a condition that goes unreported; it is a condition that cannot arise in steady state, and the notice this issue exists to repair has little left to say. Sequencing follows from that: settle itd-108's direction first, then decide whether to repair this notice, narrow it to the residual window (a release cut between a user's plugin update and their binary fetch), or retire it and the .binary-meta plugin_sha field with it. Fixing the gate now risks paying for a mechanism the next intent removes.