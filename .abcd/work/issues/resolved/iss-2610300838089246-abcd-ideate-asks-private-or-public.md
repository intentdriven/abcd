---
schema_version: 1
id: "iss-2610300838089246"
slug: "abcd-ideate-asks-private-or-public"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "product thinker ruling, 2026-10-30"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ideate.md"
remedy: "Make ideate record default to --private when neither flag is given, keep --public as the explicit opt-in and keep refusing both, and replace the private-or-public question in commands/ideate.md with a statement of the default and how to choose --public."
resolution: "ideate record now writes the private store when neither flag is given, --public stays the explicit opt-in and both together are still refused; the command page states the default instead of asking, and adr-2610300908239458 supersedes adr-2610142116480836"
impact: fix
---

/abcd:ideate asks private or public before every run, and its record verb refuses a run that names neither. The product thinker has ruled that ideate defaults to private and does not ask.
