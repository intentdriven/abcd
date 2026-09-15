---
schema_version: 1
id: "iss-2609090951287096"
slug: "pinned-gofmt-merges-toolchain-download-output-into-goroot"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
---

The format gate resolves the pinned toolchain by capturing the go environment query for the toolchain root with stderr merged into stdout, so on the first run on a machine that does not yet have the declared toolchain cached, the download progress line is prepended to the captured value. The executable test on the resulting path then fails and the target refuses, saying the toolchain could not be resolved and the fetch needs network, while the fetch has in fact just succeeded. Reproduced with an uncached toolchain in an isolated module cache: the captured value comes back as two lines, the first the download notice and the second the real root, and the executable test on that value is false. A second run succeeds because the toolchain is now cached, so the gate self-heals, but the failure lands on the run that matters most, on a new machine or a fresh CI image, and it diagnoses the opposite of what happened. It matters because this loud refusal was chosen deliberately over a silent fallback, and a loud refusal naming the wrong cause spends the trust that choice was buying. Fix direction: capture stdout alone and leave stderr for the diagnostic, or take the last line of the captured value, and make the refusal distinguish a fetch that failed from one that merely printed. Detector: with the pinned toolchain absent from the module cache, the format gate must resolve it and run, not refuse.
