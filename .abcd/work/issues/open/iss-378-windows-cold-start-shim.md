---
schema_version: 1
id: "iss-378"
slug: "windows-cold-start-shim"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "itd-130 planning"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed K): Keep the Windows cold-start shim parked until Windows target work begins, or close?"
remedy: "Waits on ruling K (keep parked until Windows target work begins, or close): if kept, on that trigger write hooks/bootstrap.ps1 as the PowerShell twin of hooks/bootstrap.sh, run with -NoProfile and a process-scoped -ExecutionPolicy Bypass, verifying the downloaded binary's SHA-256 with Get-FileHash against the release checksums before the swap, proven by a CI job on a Windows runner that cold-starts from no binary; if closed, resolve this record as wontfix naming the trigger so a Windows port files it afresh."
---

Windows cold-start shim for the binary: a PowerShell twin of hooks/bootstrap.sh for the no-binary cold start on Windows. Trigger: Windows target work begins (abcd ships Windows binaries one day). itd-130's abcd update core is the compiled updater that runs on Windows once a binary exists — it already uses minio/selfupdate's rename-dance for the running-exe swap — but the hook ladder in hooks.json and bootstrap.sh are POSIX shell and do not run on Windows. This seed is the cold-start shim only; the steady-state updater is itd-130.

## Remedy grounds (2026-09-29)

- Microsoft's reference calls the execution policy defence in depth rather than a security boundary, and a policy given on the command line lasts only for that process and its children (https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_execution_policies, consulted 2026-09-29), so the shim changes no machine setting and its integrity comes from the checksum, as in bootstrap.sh.
- Rejected: asking the person to change their machine's execution policy.
