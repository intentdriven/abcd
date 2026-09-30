---
schema_version: 1
id: "iss-2609300015353414"
slug: "itd-130-criterion-9-reads-given-stdout-is-not-a-tty-when-the"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/update.go"
remedy: "Record the reading and leave the criterion text as shipped (ruling H10): an Audit Notes line on itd-130 saying criterion 9 is met as 'stdout carries only the receipt; progress goes to stderr, and only when stderr is a terminal', the convention git documents for fetch ('Progress status is reported on the standard error stream by default when it is attached to a terminal', git-fetch(1) --progress), pinned by a piped-run test. Shown wrong if the product thinker reads criterion 9 as keyed on stdout itself."
resolution: "Criterion 9 is recorded as met the way the verb delivers it: stdout carries only the receipt, and progress goes to stderr only when stderr is a terminal. An Audit Notes line on itd-130 says so (ruling H10), with the criterion text untouched. TestUpdatePipedPrintsNoProgress and TestUpdateTerminalStderrGetsProgress pin the gate, which now reads the writer progress goes to."
impact: internal
resolved_by:
  commit: "f5780c9ddfa1978340a728b71922c3eefef8881e"
---

itd-130 criterion 9 reads 'Given stdout is not a TTY, when the update downloads, then no progress output is emitted', but the delivered gate keys progress on stderr, the stream the progress is written to: stdout carries only the receipt in every mode, and progress appears on stderr only when stderr is a terminal. The code is the right reading and the criterion's wording is loose: keyed literally on stdout, it would hide progress from a person watching 'abcd update | tee log' on a terminal while still writing it into a redirected '2>log'. spc-32 line 93 already words the test as 'stdout carries only the receipt'. Found by the itd-130 fidelity audit (receipt rcp-264f7b144576, ac-9 concern), gap 2 of iss-2609012111162089.

## Grounds

- pursued: a piped or hooked abcd update shows no progress and stdout carries only the receipt, while a person at a terminal still sees progress on stderr; shown wrong if the product thinker reads criterion 9 as keyed on stdout itself, or if a run with stderr redirected to a file shows progress in it
