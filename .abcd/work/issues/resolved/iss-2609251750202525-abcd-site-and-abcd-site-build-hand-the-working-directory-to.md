---
schema_version: 1
id: "iss-2609251750202525"
slug: "abcd-site-and-abcd-site-build-hand-the-working-directory-to"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "abcd site, site build and lint site resolve the checkout root (git toplevel, else the .git-marker walk) and default their output directory to the checkout's site/, so a subdirectory reads and builds the same site as the root; the sibling lint site had the same defect and is fixed in the same change; TestSiteVerbsReadTheCheckoutFromASubdirectory pins all three."
impact: fix
resolved_by:
  commit: "bacec65df"
---

abcd site and abcd site build hand the working directory to site.Describe/site.Build, so run from a subdirectory they report '.abcd/site.json (absent) ... nothing to build' with exit 0: a plausible wrong answer rather than a refusal, the same shape iss-2609251713073532 fixed for the release cut (internal/surface/cli/site.go:31-35, 52-58; review2-sentences).

## Grounds

- pursued: the site verbs answer the same from any directory of a checkout; the board or the build differing between the root and a subdirectory would show it wrong
