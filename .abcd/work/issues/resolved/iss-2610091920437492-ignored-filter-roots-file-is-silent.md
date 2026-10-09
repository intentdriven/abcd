---
schema_version: 1
id: "iss-2610091920437492"
slug: "ignored-filter-roots-file-is-silent"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "security-drain-2026-10-09 lane X round 3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/status.go"
remedy: "Have `abcd ahoy` (and the status reads that consult filter-roots) report a filter-roots file it ignored, naming the file and the check it failed, as trusted-roots refusals are reported; test that a group-writable or symlinked file is named."
resolution: "FiltersSwitchedOn now renders the ignored reason as a note naming ~/.abcd.noindex/filter-roots and the check it failed, gitutil.FilterRootsIgnored exposes it, and abcd ahoy reports it from any folder as the report-only machine-scope gap filter_roots.ignored; the status reads have no output channel and leave the report to ahoy."
impact: fix
---

abcd ignores a ~/.abcd.noindex/filter-roots file that fails its ownership, mode or symlink checks without saying so: the core has no output channel, so a checkout its owner listed to keep its content filters running silently gets them switched off, and the reads that depend on them (an LFS checkout's status) degrade with no notice. Found in lane X round 3 of the security-drain-2026-10-09 run.
