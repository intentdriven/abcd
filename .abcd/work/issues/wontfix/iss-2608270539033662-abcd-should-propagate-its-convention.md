---
schema_version: 1
id: "iss-2608270539033662"
slug: "abcd-should-propagate-its-convention"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "attribution-defaults-propagation-2026-08-27"
found_at: "internal/core/ahoy"
wontfix_reason: "Duplicate of itd-91 (ai-attribution-preference), the drafted intent that already scopes declaring per-repo how AI is acknowledged and applying it to every commit/PR/record — the AI-attribution-footer-to-managed-repos capability. Neighbours: iss-193 (git-identity pinning in managed repos, attribution trailer explicitly out of its scope) and iss-119 (Assisted-by declared but unenforced, the enforcement-gate half). The broader 'propagate all abcd defaults' umbrella is better advanced by itd-91 plus the existing ahoy install / prepare-this-repo mandate than by a fresh open issue."
---

abcd should propagate its convention DEFAULTS to the repos it manages, not only enforce them in its own repo. The AI-attribution footer (the kernel 'Assisted-by: Claude:<model>' trailer prescribed in AGENTS.md and enforced by .github/workflows/attribution.yml + scripts/check-attribution.sh) is the motivating example: ahoy install / prepare-this-repo do not currently write the attribution convention or its CI gate into a managed repo (a grep of internal/core/ahoy and internal/core/repolint for 'Assisted-by' is empty). The same holds for other defaults abcd owns (the rules-loader domains, the commit gates). Narrower existing records: iss-193 (attribution-identity enforcement belongs in abcd for every managed repo) and itd-91 (ai-attribution-preference, draft intent) cover the attribution slice specifically; this record is the general defaults-propagation capability those are instances of — candidate for promotion to an intent.