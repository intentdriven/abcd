---
schema_version: 1
id: "iss-2610240824571059"
slug: "ahoy-install-offers-a-tool-install"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "ahoy install from a terminal, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/registry.go"
remedy: "make a verified download the default tool install on macOS and Linux (dcn-2610240845548776): fetch the tool's official release, check it against the published checksums and the signed build record where one exists, and put it in ~/.local/bin; drop Homebrew as the default step, and never relay a sudo line"
---

ahoy install offers a tool install through Homebrew on an account that cannot write to Homebrew: on this machine /opt/homebrew is owned by the admin account, so brew install gitleaks fails on every run, and the failure note relays Homebrew's own sudo chown advice, although abcd never escalates privileges. The registry has one install step per platform (Homebrew), so an account without admin rights has no route that works.
