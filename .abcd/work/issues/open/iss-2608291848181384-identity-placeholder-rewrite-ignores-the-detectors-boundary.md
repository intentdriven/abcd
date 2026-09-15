---
schema_version: 1
id: "iss-2608291848181384"
slug: "identity-placeholder-rewrite-ignores-the-detectors-boundary"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/adapter/scanner/redact.go"
---

ultra-v0.6.8 follow-up observation: redactLine in internal/adapter/scanner/redact.go rewrites an identity finding's placeholder with strings.ReplaceAll over the whole line, so a local_username finding the detector reported only where the username is word-bounded also rewrites every other substring occurrence on that line — under HOME=/root a line carrying a bare 'root' word also has '/rootfs/etc/hosts' rewritten to '/[redacted-user]fs/etc/hosts'. home_path_self now sweeps by its own anchor; the other identity kinds still rewrite by substring. Pre-existing; surfaced when the home-span suppression stopped hiding it. Fix: apply identity placeholders by the reported byte span (columns stay valid after sealLine) or at least by the detector's own word boundary.
