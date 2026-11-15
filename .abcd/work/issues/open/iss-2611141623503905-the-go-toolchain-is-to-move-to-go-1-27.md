---
schema_version: 1
id: "iss-2611141623503905"
slug: "the-go-toolchain-is-to-move-to-go-1-27"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "dependency updates"
origin: researcher-authored
production_mode: hand-written
found_at: "go.mod"
remedy: "After the v0.17.0 cut, in one pull request: raise the go line in go.mod to 1.27, update the CI toolchain and any pinned build image to match, take sigstore-go 1.4.0 with go mod tidy, and run the full preflight; close the bot's pull request once it is merged. Grounds: sigstore-go 1.4.0's go.mod requires go 1.27.0; the vulnerability scan passes on the default branch with 1.3.0, so nothing forces the step earlier."
---

The Go toolchain is to move to Go 1.27 after v0.17.0 is cut, in one change of its own, with sigstore-go 1.4.0, which requires it. Until then go.mod stays on the Go 1.26 line, sigstore-go stays at 1.3.0, and the dependency bot's open pull request for it stays open. This is the owner's decision of 14 November 2026: a minor Go step changes the compiler, vet and the CI toolchain together, and was not wanted before a release that is waiting on a blocker.
