---
schema_version: 1
id: "iss-2608300318192814"
slug: "intent-writes-past-the-read-cap-on-the-draft-face"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-177 third-round security review, 2026-08-30"
found_at: "internal/core/intent/lifecycle.go (Plan draft face, Link), internal/core/intent/claims.go (byte-cap guard, maskLines)"
resolution: "One helper, writeIntentFile, now caps the final bytes at every intent write in this package — the draft face's kind and spec_id rewrites, the stamp, and Link — so the cap sits at the write rather than on one producer; the guard inside stampScopeConditions is gone, leaving one condition with one message. The informational note about the temp path is already covered: cli.Run routes every command error through scrubPaths, and WriteFileAtomic creates its temp file in the target's own directory, so the path arrives relative. The related note that a degenerate or quoted opener above the section leaves the gate reporting no section rather than naming an unclosed comment is by design: an opener nobody closes masks to end of file, which is the only safe reading, and a heading inside that span is not a heading."
impact: internal
resolved_by:
  intent: "itd-177"
  spec: "spc-55"
---

The post-stamp byte-cap guard sits in stampScopeConditions, before two further growth steps on the draft face of Plan (kind and spec_id rewrites), so a draft within a few bytes of the read cap is written past it and every intent verb then refuses the whole corpus until the file is hand-trimmed; the planned face holds the cap. Cap the final bytes at every intent write through one helper covering the kind, spec_id, stamp and Link writes. Also informational: the stamp write error wraps a PathError that re-introduces the absolute temp path into stderr (a pre-existing pattern), and a degenerate comment or an inline-code opener above the section masks the real heading so the gate reports no section rather than naming the unclosed comment.
