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
---

abcd site and abcd site build hand the working directory to site.Describe/site.Build, so run from a subdirectory they report '.abcd/site.json (absent) ... nothing to build' with exit 0: a plausible wrong answer rather than a refusal, the same shape iss-2609251713073532 fixed for the release cut (internal/surface/cli/site.go:31-35, 52-58; review2-sentences).
