---
schema_version: 1
id: "iss-2610160953425019"
slug: "abcd-update-s-confirmation-prints-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "v0.15.0 update"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/update.go"
remedy: "Print the line as 'replaced: sha256 <digest>' alone in the human render, dropping the ' — in no published release; <ownership proof>' clause, and keep both facts in the --json result and in the brief."
---

abcd update's confirmation prints the replaced binary as 'replaced: sha256 <digest> — in no published release; it is the binary running this command, so it is abcd by construction'. The explanation after the digest is noise to the person reading it: they need to know what was replaced, not why abcd trusted it. Printed at internal/surface/cli/update.go:238 from the ownership proof's prose (internal/core/update/update.go:110).
