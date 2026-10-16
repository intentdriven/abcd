---
schema_version: 1
id: "iss-2610160608376349"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md"
remedy: "Exempt implement load, a machine check with no checkout precondition, from the every-sub-verb sentence at 27-implement.md:59-61."
resolution: "27-implement.md now exempts the load check, a machine check with no checkout precondition, from the sentence that every sub-verb refuses outside a checkout"
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/27-implement.md:59-61 says that outside a checkout, or in a repository with no commit, every implement sub-verb refuses and nothing is created. `abcd implement load --site preflight` runs and exits 0 outside a checkout: internal/surface/cli/implement_load.go:148-152 resolves the checkout root only optionally, and the load help documents a machine check with no checkout precondition.
