---
schema_version: 1
id: "iss-2610151422086989"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/21-update.md:48 to match the shipped surface: A bare update whose latest release is older than ~/.abcd.noindex/release-mark is refused with shape `older-than-mark` (internal/surface/cli/update.go:107); the update writes release-mark and naming an older tag lowers it (update --help). The chapter documents…"
---

The brief at .abcd/development/brief/04-surfaces/21-update.md:48 says: The verb takes an optional release tag; with none, it resolves the latest (no further condition on the bare form). The shipped surface differs: A bare update whose latest release is older than ~/.abcd.noindex/release-mark is refused with shape `older-than-mark` (internal/surface/cli/update.go:107); the update writes release-mark and naming an older tag lowers it (update --help). The chapter documents neither the release-mark record nor this refusal. (v0.14.2 release crosscheck, undocumented-surface, raised by opus in a16.)
