---
schema_version: 1
id: "iss-2610121532247004"
slug: "receipt-model-and-resolution-text"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "sweeping the loop package's on-disk readers on fix/sec-build-next-series"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/receipt.go"
remedy: "Refuse a lane receipt whose model carries a control, bidi or zero-width rune, or whose resolution note or grounds carries one other than a line break or a tab, naming the field without quoting it; and refuse a run state whose stored receipt model or resolution text fails the same test, with the restore-the-run's-state-file remedy; tests forge each field with U+202E and U+009B."
resolution: "verifyReceipt refuses a receipt whose model is not one plain line, or whose resolution note or grounds carries a control, bidi or zero-width character other than a line break or a tab, naming the field without quoting it; readStateIn refuses a run state whose stored receipt or resolution text fails the same test, with the restore-the-run's-state-file remedy, so none reaches implement record --json or implement status --json."
impact: fix
resolved_by:
  commit: "426e499a35515a4e99fe69a1a7101773408f49c1"
---

A lane receipt's model and each resolution's note and grounds are stored on the lane unchecked and echoed raw on 'abcd implement record --json' and 'abcd implement status --json': verifyReceipt (internal/core/implement/loop/receipt.go) checks a receipt's run, lane, branch, commits, resolution issue, commit and impact, but takes rc.Model and each resolution's note and grounds as written, and readStateIn reads them back from state.json unchecked. A receipt an agent writes, or a hand-edited or repository-shipped state.json, can carry control or bidi runes (U+202E, U+009B) onto both streams. Found sweeping the loop package's on-disk readers after iss-2610121516559163.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The receipt and run state files, their readers and the --json streams that echo them are abcd's own, so abcd added it and fixes it, at the boundary the text enters by; the fix changes nothing about how the repository works.
