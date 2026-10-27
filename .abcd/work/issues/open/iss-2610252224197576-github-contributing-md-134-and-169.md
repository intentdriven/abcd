---
schema_version: 1
id: "iss-2610252224197576"
slug: "github-contributing-md-134-and-169"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "state-of-the-art passes of the hand-back triage, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/CONTRIBUTING.md"
remedy: "Reword those places to say abcd's trailer is modelled on the kernel's Assisted-by convention rather than that it is the kernel format, with a dated link to the page read; ACKNOWLEDGEMENTS.md keeps its credit."
---

.github/CONTRIBUTING.md:134 and :169, AGENTS.md and the bundled COMMITTING rule (internal/core/rules/defaults/rules.json) say the Assisted-by trailer follows the Linux kernel's format, Agent:model-version. The kernel's coding-assistants page (docs.kernel.org/process/coding-assistants.html, read 2026-10-25) no longer carries a vendor or model field, so the claim that abcd's trailer is the kernel format is out of date, whatever form abcd keeps.
