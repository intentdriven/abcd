---
schema_version: 1
id: "iss-2608231008315498"
slug: "site-contributors-page-rethink-and-assisted-total-mismatch"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "user-observation"
found_at: "internal/core/site/explorer.go"
resolution: "The contributors page carries only its two folded full-width panels, Authors of record and Assisted-by trailers, with no stat tiles, and the None declaration is counted beneath the chart, not in it, so the panel total equals the bar sum (bed0026c; pinned by TestAuthorshipBarSumEqualsAssistedTotal). The last piece, the disclosure rate moving to the health page, lands in 52329d9f as the last tile of the health page's row of counts."
impact: fix
resolved_by:
  commit: "52329d9f"
---

Contributors page rethink: keep only the two things that matter — Authors of record, and Assisted-by trailers — stacked one under the other as expandable panels; drop the three stat tiles, with 'commits disclose AI assistance' moving to the Health page (depends on that page landing with the IA intent seed iss-2608230752354909). Also a real data inconsistency: the Assisted-by panel's note reads 1035 (a.Assisted) while its bars sum to 1036, because the tally includes the 'None' row — the positive human-only declaration — which a.Assisted deliberately excludes (contributors.go counts DeclaredNone and continues before incrementing Assisted). A chart whose total is 'assisted' must not carry a row that is by definition not assistance: either split None out as its own labelled figure or relabel the panel to the declarations it actually counts, and make the note equal what the bars sum to (report D of the 2026-08-23 second pass).

## Grounds

- pursued: the health page carries the share of authored commits disclosing assistance with its fraction and the merges excluded, and the contributors page carries two folded panels and no tile (TestHealthCarriesTheDisclosureRate); a tile on the contributors page, or no rate on the health page, would show it wrong
