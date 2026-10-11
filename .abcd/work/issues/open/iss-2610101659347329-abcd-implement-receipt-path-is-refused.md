---
schema_version: 1
id: "iss-2610101659347329"
slug: "abcd-implement-receipt-path-is-refused"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/implement.md"
remedy: "Derive the run from the receipt path when it lies under .abcd/.work.local/run/<run-id>/, refusing only a path outside any run or a --run that disagrees with it; and have every next line the drain and implement step print name --run whenever more than one run is in progress."
---

abcd implement receipt <path> is refused at the state stage when more than one run is in progress, asking for --run, although the receipt path it was given (.abcd/.work.local/run/<run-id>/lane-N/...) already names the run. On 2026-10-10 a drain lane's receipt was refused this way because 24 stale runs from an earlier drain were still in progress; the next the drain itself printed (run `abcd implement receipt <path>`) omitted --run too, so following it verbatim fails.
