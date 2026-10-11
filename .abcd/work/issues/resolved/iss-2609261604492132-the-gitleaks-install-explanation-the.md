---
schema_version: 1
id: "iss-2609261604492132"
slug: "the-gitleaks-install-explanation-the"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd63"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/registry.go"
resolution: "Both Homebrew entries' Effects text opens with one shared sentence naming the network fetch (package-list update, download) and Homebrew's own install analytics, sent unless the person's Homebrew settings turn them off; no entry claims the install sends nothing. The child environment is unchanged (an analytics opt-out is a product-thinker ruling)."
impact: fix
resolved_by:
  commit: "09938d67"
---

The gitleaks install explanation the person consents on says Homebrew 'sends nothing anywhere', which is untrue: brew install may first update its taps over the network, downloads the package, and by default sends Homebrew's install analytics. The Effects text is the sentence the install question is answered on, so it must state what the install does to the machine and the network.

## Grounds

- pursued: the sentence the person consents on is true of what brew install does; an Effects text that omits the network or the analytics, or claims nothing is sent, would show it wrong
