---
schema_version: 1
id: "iss-2610240916382802"
slug: "ahoy-reports-an-empty-install-mode-for-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "design review of itd-2610240823055889, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/detect.go"
remedy: "classify the PATH entry from ~/.abcd.noindex/path-entry when no plugin root resolves, and report the install mode from the record rather than leaving it empty"
---

ahoy reports an empty install_mode for a working, recorded PATH entry whenever no plugin root resolves: detectInstallMode returns "" when !pluginOK, so after a successful README install (~/.local/bin/abcd on PATH and named by ~/.abcd.noindex/path-entry) a plain-terminal or session run still says nothing is installed.
