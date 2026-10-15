---
schema_version: 1
id: "iss-2610151422138325"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Rewrite .abcd/development/brief/05-internals/03-configuration.md:1118 to match the shipped surface: internal/adapter/ also holds repoprompt/ (the RepoPrompt reviewer route, itd-6, listed in internal/README.md:290), which the tree omits."
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:1118 says: Plugin layout: internal/adapter/ holds scanner/, gitleaks/, hosting/, openaiapi/, tailscale/ ('hosting, openaiapi and tailscale are adapters outside the capability seams'). The shipped surface differs: internal/adapter/ also holds repoprompt/ (the RepoPrompt reviewer route, itd-6, listed in internal/README.md:290), which the tree omits. (v0.14.2 release crosscheck, fictional-layout, raised by opus in a34.)
