---
schema_version: 1
id: "iss-2609261604492132"
slug: "the-gitleaks-install-explanation-the-person-consents-on-says"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd63"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/registry.go"
---

The gitleaks install explanation the person consents on says Homebrew 'sends nothing anywhere', which is untrue: brew install may first update its taps over the network, downloads the package, and by default sends Homebrew's install analytics. The Effects text is the sentence the install question is answered on, so it must state what the install does to the machine and the network.
