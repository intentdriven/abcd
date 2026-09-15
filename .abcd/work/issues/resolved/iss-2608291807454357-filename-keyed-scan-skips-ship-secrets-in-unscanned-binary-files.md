---
schema_version: 1
id: "iss-2608291807454357"
slug: "filename-keyed-scan-skips-ship-secrets-in-unscanned-binary-files"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "v0.6.9-security-pass"
found_at: "internal/adapter/scanner/scanner.go"
resolution: "Skip-listed bundle files are now read through the guarded, capped primitive and their bytes scanned with the secret, harness-leak and long-literal identity rules — the verdict is extension-independent for home path, email, session URL, and a real name that is multi-word or at least 8 bytes — reported as ScannedBinary (a plaintext allow-listed name) or ContentUnverified (the default for the byte branch) instead of an unverified Skipped allow; an unreadable or oversized one is an Unscanned refusal that says why."
impact: fix
---

GHSA-9wv7-88w3-f77m: filename-keyed scan skips let a hostile release payload ship a secret in an unscanned binary file. skipByName/skipByFragment in the launch payload scanner skip files before reading them, Skipped entries count as an accepted allow, and the zero-coverage sentinel stays quiet while anything else scans, so a secret placed in notes.png (no valid image data needed) is Included in the payload and passes every gate with zero findings. CWE-693.
