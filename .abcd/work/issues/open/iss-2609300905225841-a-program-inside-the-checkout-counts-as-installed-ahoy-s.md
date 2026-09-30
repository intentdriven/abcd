---
schema_version: 1
id: "iss-2609300905225841"
slug: "a-program-inside-the-checkout-counts-as-installed-ahoy-s"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/detect.go"
remedy: "Judge presence as the installer judges a program: onPath takes the checkout root and refuses a resolution inside it, lexically and after symlinks on both sides, through one helper extracted from tools.admit (never a copy). Grounds: install.go already refuses an in-checkout program as repository content, so presence must agree with what abcd would run."
---

A program inside the checkout counts as installed: ahoy's onPath is a bare exec.LookPath, so a gitleaks or trufflehog that PATH resolves inside the repository reads as present, and ahoy neither reports the dependency gap nor offers the install, while the installer itself refuses the same resolution as repository content (tools admit). The gh offer on feat/ahoy-offer-gh reads gh presence through the same onPath and inherits the gap.
