---
schema_version: 1
id: "iss-2610262259078996"
slug: "the-preflight-s-govulncheck-scan-warns"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review of fix/forge-check-gaps"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/preflight-scans.sh"
remedy: "On the finding path only, run the same govulncheck over the merge base's tree and warn only for the vulnerability ids present there too; stop for any id the branch's head adds."
resolution: "The scan of a finding on a branch that moves no dependency now scans the fork point's tree too, and warns only when that scan reports every id the head's does; an id only the head reports, or a fork point that cannot be scanned, stops the preflight."
impact: fix
---

The preflight's govulncheck scan warns, rather than stops, on a finding when the branch changes neither go.mod nor go.sum, taking that to mean the finding stands on the default branch; govulncheck judges reachability, so a branch that adds a call newly reaching a vulnerable symbol in a module already required is only warned about, and with govulncheck not a required check its pull request can land with the finding.
