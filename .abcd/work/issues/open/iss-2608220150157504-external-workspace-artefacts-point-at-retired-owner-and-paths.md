---
schema_version: 1
id: "iss-2608220150157504"
slug: "external-workspace-artefacts-point-at-retired-owner-and-paths"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "abcdev-site-plan investigation 2026-08-21"
found_at: "external project workspace assets"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (technical facilitator act in the external workspace (retired owner name, landing asset links); nothing in-tree to fix or test). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Two external artefacts point at retired names and paths: the project workspace description names the repository under its pre-transfer owner, and the July landing-page asset links that owner and docs paths that no longer exist (docs/reference/commands.md, docs/reference/facilitator.md). Neither is in this tree, so no in-repo detector can arm; fixing them is a maintainer act in the external workspace