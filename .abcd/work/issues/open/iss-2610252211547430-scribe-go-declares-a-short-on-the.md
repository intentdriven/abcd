---
schema_version: 1
id: "iss-2610252211547430"
slug: "scribe-go-declares-a-short-on-the"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/scribe.go"
remedy: "Delete the Short fields from the visible commands, and have the sentence gate fail when a visible constructor declares one."
---

scribe.go declares a Short on the visible commands scribe, scribe assemble and scribe ingest (internal/surface/cli/scribe.go:37, 52, 115). applySentences overwrites each one with the manifest sentence (internal/surface/cli/sentences.go:51-63; manifest at internal/core/surface/sentences.go:397-402), so these strings are dead and already differ from the sentences users see. The comment at sentences.go:24-25 says 'No constructor declares a Short for a visible command, so there is no second copy to drift.' Several capture sub-verbs in cli.go (5691, 5731, 5768) do the same.
