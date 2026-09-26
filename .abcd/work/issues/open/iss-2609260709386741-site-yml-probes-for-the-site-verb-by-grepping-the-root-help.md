---
schema_version: 1
id: "iss-2609260709386741"
slug: "site-yml-probes-for-the-site-verb-by-grepping-the-root-help"
severity: "major"
category: "bug"
source: "drift-detection"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/site.yml"
---

site.yml probes for the site verb by grepping the ROOT help listing (./abcd --help | grep -qE '^[[:space:]]+site[[:space:]]', four places: the production render step, its check step, the preview build step and the preview check step). Since the grouped root help landed (#711, merge a3cdf26e) abcd --help lists only the human-facing verbs and site appears only under abcd --help --agent, so every site run on a binary built after a3cdf26e refuses with 'this binary has no site verb': the preview job has failed on every push to main since (run 36225049347 on 22997314 and the eight before it), and the production render the release chain calls after publishing v0.11.0 refuses the same way, leaving abcdev.app unrendered for the release. The probe must read the agent listing where the binary has one and the flat listing where it does not (v0.6.2 to v0.10.0 carry site but refuse --agent as an unknown flag), and still refuse, in one line, a binary built before the site slice.
