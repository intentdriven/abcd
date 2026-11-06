---
schema_version: 1
id: "iss-2611060809080524"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/idempotency_test.go"
remedy: "Make the round-trip test compare the tree with a fresh install and read the gap count, or narrow the chapter's claim to what the test holds."
---

.abcd/development/brief/04-surfaces/01-ahoy.md:1473-1475 calls uninstall then install a tested round-trip invariant: zero actionable gaps, and state byte-identical to a fresh install save for the setup date. `TestUninstallInstallRoundTrip` (internal/core/ahoy/idempotency_test.go:179) checks only that the marker block is gone, `.abcd/config.json` survives and the re-install status is clean. It compares no bytes and asks for no gap count.
