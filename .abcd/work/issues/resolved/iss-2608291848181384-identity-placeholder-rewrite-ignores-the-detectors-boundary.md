---
schema_version: 1
id: "iss-2608291848181384"
slug: "identity-placeholder-rewrite-ignores-the-detectors-boundary"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/adapter/scanner/redact.go"
resolution: "Already fixed by 19348b5a (shipped in v0.9.0, iss-2609120446083912): identity placeholders are applied by the byte span the detector reported (maskIdentitySpans), not strings.ReplaceAll, so a cleared lookalike such as /rootfs/etc/hosts under HOME=/root survives beside a genuine mention; TestHomePathSelfDetectionIsAnchored and TestSpanMaskingFailsOpenOnClearedLookalikes pin it. Re-verified at 3b764ed5 in autonomous run A."
impact: fix
shipped_in: v0.9.0
resolved_by:
  commit: "19348b5a"
---

ultra-v0.6.8 follow-up observation: redactLine in internal/adapter/scanner/redact.go rewrites an identity finding's placeholder with strings.ReplaceAll over the whole line, so a local_username finding the detector reported only where the username is word-bounded also rewrites every other substring occurrence on that line — under HOME=/root a line carrying a bare 'root' word also has '/rootfs/etc/hosts' rewritten to '/[redacted-user]fs/etc/hosts'. home_path_self now sweeps by its own anchor; the other identity kinds still rewrite by substring. Pre-existing; surfaced when the home-span suppression stopped hiding it. Fix: apply identity placeholders by the reported byte span (columns stay valid after sealLine) or at least by the detector's own word boundary.

## Grounds

- pursued: redaction touches only the reported spans; a line where a cleared lookalike is rewritten because a genuine mention shares it would show it wrong
