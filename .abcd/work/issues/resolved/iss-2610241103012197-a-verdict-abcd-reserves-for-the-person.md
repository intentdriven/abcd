---
schema_version: 1
id: "iss-2610241103012197"
slug: "a-verdict-abcd-reserves-for-the-person"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "security review triage session"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/embargo.md"
remedy: "For person-only verdicts, have the surface present each item's evidence and the verdict the record supports, with one line on why each other verdict does not fit, in prose beside a single confirm question; record the verdict under the person's name only on their yes. Amend asking.go's recommendation rule to allow this for verdict confirmation, and say so in commands/embargo.md."
resolution: "person-only verdicts now carry each item's evidence, the supported verdict and why the others do not fit, and are recorded only on the person's yes (asking rule and embargo triage page)"
impact: fix
---

A verdict abcd reserves for the person (embargo triage, and any verb whose record names who decided) is handed back as a bare choice, so a person without the code-level context cannot make it. commands/embargo.md says only 'The verdict is the person's decision. Ask for it; never choose it yourself.', and the asking rules (internal/core/question/asking.go:74-76) allow a recommendation only when the person asks for one. In a session on 2026-10-24 the person, handed four triage verdicts, answered 'why leave to me? I can't decide that.'; a per-item recommendation with its reasons, then a plain yes, settled all four at once.
