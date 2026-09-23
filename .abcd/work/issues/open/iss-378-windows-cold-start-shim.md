---
schema_version: 1
id: "iss-378"
slug: "windows-cold-start-shim"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "itd-130 planning"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (trigger-gated: the Windows cold-start shim waits on Windows target work beginning). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Windows cold-start shim for the binary: a PowerShell twin of hooks/bootstrap.sh for the no-binary cold start on Windows. Trigger: Windows target work begins (abcd ships Windows binaries one day). itd-130's abcd update core is the compiled updater that runs on Windows once a binary exists — it already uses minio/selfupdate's rename-dance for the running-exe swap — but the hook ladder in hooks.json and bootstrap.sh are POSIX shell and do not run on Windows. This seed is the cold-start shim only; the steady-state updater is itd-130.