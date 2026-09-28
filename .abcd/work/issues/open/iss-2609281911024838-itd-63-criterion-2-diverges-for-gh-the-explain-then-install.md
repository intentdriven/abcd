---
schema_version: 1
id: "iss-2609281911024838"
slug: "itd-63-criterion-2-diverges-for-gh-the-explain-then-install"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-63"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/detect.go"
---

itd-63 criterion 2 diverges for gh: the explain-then-install mode's install half is offered for gitleaks alone. ahoy.DependencyTools is {gitleaks} (internal/core/ahoy/detect.go), so ahoy install never puts gh to the question and --install-tool gh is refused by installToolNames (internal/surface/cli/cli.go), while ahoy remote and site setup refuse a missing gh with tools.Missing (internal/core/ahoy/remote.go) and show the Homebrew step the person must then run by hand. A required tool the registry knows how to install is explained but never installed on a yes, so for gh the criterion's 'the install runs only on an explicit yes' never arises. Either a gh dependency gap joins DependencyTools (offered only when a verb that needs gh is in play), or the record states that the mode installs gitleaks only.
