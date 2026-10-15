---
schema_version: 1
id: "iss-2610151422144124"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md"
remedy: "Rewrite .abcd/development/brief/06-delivery/01-build-sequence.md:35 to match the shipped surface: internal/adapter/ holds four packages beyond the scanner seam and its gitleaks backend: hosting, openaiapi, tailscale and repoprompt (the RepoPrompt reviewer-route MCP client, itd-6, imported by internal/core/oracle/review.go and listed in…"
---

The brief at .abcd/development/brief/06-delivery/01-build-sequence.md:35, .abcd/development/brief/06-delivery/01-build-sequence.md:26-29 says: Three further adapter packages stand outside the capability seams: `openaiapi`, `hosting`, and `tailscale`. The shipped surface differs: internal/adapter/ holds four packages beyond the scanner seam and its gitleaks backend: hosting, openaiapi, tailscale and repoprompt (the RepoPrompt reviewer-route MCP client, itd-6, imported by internal/core/oracle/review.go and listed in internal/README.md:290). The count is three where the tree has four, and the chapter never names repoprompt. (v0.14.2 release crosscheck, stale-count, raised by opus+sonnet in a37.)
