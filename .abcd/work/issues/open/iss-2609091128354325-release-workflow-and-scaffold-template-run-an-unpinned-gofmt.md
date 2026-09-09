---
schema_version: 1
id: "iss-2609091128354325"
slug: "release-workflow-and-scaffold-template-run-an-unpinned-gofmt"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/release.yml"
---

AGENTS.md's definition of done now asserts that the format gate resolves gofmt from the toolchain go.mod declares and never falls back to the local gofmt, but two of the three places that actually run the gate still invoke a bare gofmt -l . The release workflow's verify job (.github/workflows/release.yml:105-113) reads whichever gofmt PATH resolves and, on failure, instructs gofmt -w ., the command the pinned target replaced; the scaffold template it is rendered from (internal/core/launch/scaffold/templates/release.yml.tmpl:126-134) carries the identical block, and abcd launch scaffold writes that block into every managed repo. The existing detector, TestFormatGateResolvesThroughTheDeclaredToolchain, checks ci.yml's step, the Makefile recipe and four prose surfaces and stops there, so neither workflow copy was ever in its roster. This is live in exactly the direction the gate was built for: a release cut on a runner whose PATH gofmt differs from the declared toolchain either passes a tree CI rejects or names a file nobody can reproduce, and a managed repo receives the unpinned form as its starting position with no Makefile target of ours to invoke instead. Two comments in the same class still name the retired command: .github/workflows/ci.yml:7 and .githooks/pre-push:18. Fix direction: the release verify job invokes make fmt-check the way ci.yml already does, and the bare template, which has no Makefile of ours to call, resolves gofmt out of the GOROOT of the toolchain the module declares. Detector: no shipped workflow and no scaffold template may invoke a gofmt that PATH resolves, and the roster must be derived from the tree rather than listed.
