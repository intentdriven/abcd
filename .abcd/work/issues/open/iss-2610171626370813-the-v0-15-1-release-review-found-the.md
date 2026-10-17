---
schema_version: 1
id: "iss-2610171626370813"
slug: "the-v0-15-1-release-review-found-the"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md"
remedy: "Rewrite the passage at .abcd/development/brief/06-delivery/01-build-sequence.md:35-38 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
refines: [iss-2610151422144124]
---

The v0.15.1 release review found the brief at .abcd/development/brief/06-delivery/01-build-sequence.md:35-38 out of step with the shipped code. It says: Three further adapter packages stand outside the capability seams: openaiapi, hosting and tailscale In the code: internal/adapter/ holds four non-scanner, non-backend adapters: openaiapi, hosting, repoprompt (the RepoPrompt MCP reviewer route, itd-6) and tailscale (gitleaks being the scanner's external backend); 05-internals/02-adapters.md:34 and 04-universal-patterns.md:175 list repoprompt
