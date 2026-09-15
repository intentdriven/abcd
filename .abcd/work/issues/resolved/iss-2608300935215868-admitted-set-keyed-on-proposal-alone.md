---
schema_version: 1
id: "iss-2608300935215868"
slug: "admitted-set-keyed-on-proposal-alone"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-189 adversarial security review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go (admittedProposals), internal/core/lint/schema.go (scanRecordStores), scripts/check-issue-resolution-cases.sh"
resolution: "The outstanding report keys its admitted set on the (run, proposal) pair, and an admission whose run field contradicts its bucket admits under neither claim; record_schema resolves the admission's proposal on the same store-declared terms as the surprise's occasioned_by, and names the bucket-vs-field contradiction. The two trailing notes are not closed here: scanRecordStores' unguarded read is pre-existing across all four original stores and is captured on its own record; the RS fixtures' cited_by is deliberate, and the comment above them says why — the gate reads bytes under a pathspec, so a fixture that carried only today's field list would pass unscoped and prove nothing."
impact: fix
resolved_by:
  intent: "itd-189"
  spec: "spc-67"
---

The outstanding report's admitted set is keyed on proposal alone — run and existence are never checked — so an admission under one run naming a proposal from another silences the other run's item, and a proposal naming nothing passes record_schema because proposal is not resolved the way occasioned_by is (key by run and proposal; resolve proposal on the same terms). Pre-existing, not this branch's: scanRecordStores reads with os.ReadDir and os.ReadFile, no symlink refusal and no size cap, so a symlinked record's frontmatter key names can appear in lint output; the new RS fixtures carry a cited_by property the closed schemas do not know.
