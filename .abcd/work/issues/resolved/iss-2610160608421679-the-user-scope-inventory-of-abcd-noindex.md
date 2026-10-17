---
schema_version: 1
id: "iss-2610160608421679"
slug: "the-user-scope-inventory-of-abcd-noindex"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add oracle-routing.json, statusline.json, cache-attestation, release-mark, notes/<root-sha>/ and the .gitignore and gitignore-laid marker to the inventory at 03-configuration.md:558 and to 01-ahoy's tree."
resolution: "The user-scope inventory omitted oracle-routing.json, statusline.json, cache-attestation, release-mark, the notes archive and the .gitignore with its gitignore-laid marker; 03-configuration.md now names each, with the notes archive marked staged since no code writes notes/<root-sha>/ yet, and 01-ahoy's tree, which already listed the first four, gains the notes archive (NOT BUILT) and the ignore file and marker."
impact: internal
---

The user-scope inventory of ~/.abcd.noindex/ at .abcd/development/brief/05-internals/03-configuration.md:558, which must agree with 01-ahoy's tree, omits files abcd reads and writes there: oracle-routing.json (internal/core/ahoy/oracle_routing.go), statusline.json, cache-attestation, release-mark (internal/core/ahoy/release_mark.go, enforced by abcd update), the notes archive notes/<root-sha>/ and the .gitignore with its gitignore-laid marker (internal/core/homeignore).
