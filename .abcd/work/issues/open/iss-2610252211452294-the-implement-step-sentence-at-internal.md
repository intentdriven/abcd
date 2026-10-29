---
schema_version: 1
id: "iss-2610252211452294"
slug: "the-implement-step-sentence-at-internal"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D6"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
remedy: "Reword the write clause within SentenceCap, for example 'Perform the next stage of an implement loop run's lane and exit: Writes the run's state and the lane's stages; holds a lane whose push has no preflight receipt.' (160 characters), then regenerate .abcd/development/release/surface.json and docs/reference/cli/commands.md."
---

The implement step sentence at internal/core/surface/sentences.go:221-222, rendered as the verb's one-line help and at docs/reference/cli/commands.md:2788, says the step 'refuses a push with no preflight receipt'. Since 13b7bd1aa (ruling DR6d-2) a landing whose head no preflight receipt names waits instead: a contention that holds only that lane, the call moving the others and exiting 3 only when nothing else moves (internal/core/implement/loop/land.go:439-452). The step's own Long help says so (internal/surface/cli/build.go:834-837), so the sentence and the paragraph under it disagree; an agent reading only the sentence treats a waiting lane as refused.
