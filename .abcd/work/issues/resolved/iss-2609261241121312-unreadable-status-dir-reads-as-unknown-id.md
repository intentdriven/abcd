---
schema_version: 1
id: "iss-2609261241121312"
slug: "unreadable-status-dir-reads-as-unknown-id"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drain1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/alloc.go"
resolution: "every status-directory read in capture goes through one readStatusDir: an absent directory is no entries, any other read error is a fault naming the directory, so an unreadable folder never reads as an unknown id, an empty folder or a free id"
impact: fix
resolved_by:
  commit: "0dd36e7e"
---

An unreadable ledger status directory reads as an unknown issue id: internal/core/capture/alloc.go findIssue continues past an os.ReadDir error, so with open/ unreadable (chmod 000) resolve, wontfix and defer print 'unknown issue id ... not found in any status directory' and exit 2, a refusal of the caller's input, when the ledger could not be read and the exit should be 1. The same swallow sits in issPresent (the mint's occupancy check reads an unreadable directory as holding nothing), checkOneStatusPerID (list and status read it as empty) and the orphan-placeholder sweep. An absent status directory stays tolerated; any other read error should be a fault.

## Grounds

- pursued: resolve, wontfix and defer against an unreadable open/ exit 1 naming the folder, and list, status and a forced mint fault on an unreadable resolved/; any of them answering unknown id or rendering the folder as empty would show it wrong
