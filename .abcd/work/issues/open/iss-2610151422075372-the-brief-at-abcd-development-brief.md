---
schema_version: 1
id: "iss-2610151422075372"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/21-update.md:3 to match the shipped surface: `abcd update --help` and internal/core/update/update.go also verify the binary against SLSA build provenance (GitHub attestations API, Sigstore trusted root, pinned to intentdriven/abcd's release.yml and the tag's commit) and refuse with shape…"
---

The brief at .abcd/development/brief/04-surfaces/21-update.md:3, .abcd/development/brief/04-surfaces/21-update.md:3-6 says: The verb fetches the release, verifies the platform binary against that release's own checksums, and swaps the installed copy atomically; 'The release is verified in full before anything is written' (line 93). The shipped surface differs: `abcd update --help` and internal/core/update/update.go also verify the binary against SLSA build provenance (GitHub attestations API, Sigstore trusted root, pinned to intentdriven/abcd's release.yml and the tag's commit) and refuse with shape `unverified-provenance` (update.go:510, :591) when it cannot be verified, offline included. The chapter never mentions provenance verification, the attestations API / Sigstore TUF reach, or this refusal. (v0.14.2 release crosscheck, undocumented-surface, raised by opus+sonnet in a16.)
