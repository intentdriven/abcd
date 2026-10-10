---
schema_version: 1
id: "iss-299"
slug: "brief-readme-md-s-05-internals-directory"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/brief/README.md"
resolution: "Removed the nonexistent 'in-session dispatch' chapter from brief/README.md's 05-internals tree."
impact: fix
---

brief/README.md's 05-internals directory-tree comment enumerates ten chapters including a nonexistent 'in-session dispatch' (that is planned intent itd-2, not a chapter); the directory holds nine files and its own index has nine rows
## Evidence

- `.abcd/development/brief/README.md` — the `05-internals/` directory-tree comment lists ten
  topics ending "…, provenance, in-session dispatch"; the directory holds nine files
  (`01-agents.md` … `09-provenance-substrate.md`) and `05-internals/README.md`'s index has
  nine rows. "in-session dispatch" is planned intent `itd-2`, not a chapter — the string
  appears nowhere in `05-internals/`.

## Adversarial review

CONFIRMED (nitpick) by an independent refuter (not a content-summary: `grep -i 'in-session'`
across the chapter set is empty; uncovered by iss-38's four indexes). Fix: delete
", in-session dispatch".
