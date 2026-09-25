---
schema_version: 1
id: "iss-378"
slug: "windows-cold-start-shim"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "itd-130 planning"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Keep the Windows cold-start shim parked until Windows target work begins, or close?"
---

Windows cold-start shim for the binary: a PowerShell twin of hooks/bootstrap.sh for the no-binary cold start on Windows. Trigger: Windows target work begins (abcd ships Windows binaries one day). itd-130's abcd update core is the compiled updater that runs on Windows once a binary exists — it already uses minio/selfupdate's rename-dance for the running-exe swap — but the hook ladder in hooks.json and bootstrap.sh are POSIX shell and do not run on Windows. This seed is the cold-start shim only; the steady-state updater is itd-130.