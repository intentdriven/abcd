---
schema_version: 1
id: "iss-2610161555125041"
slug: "after-the-2026-10-16-reboot-with-machine"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "crash recovery on 2026-10-16 (abcd-22/abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/rules_prompt.go"
remedy: "Give the prompt router one deadline for the whole hook, well inside the host's timeout (for example 20 s), set when the hook starts. Each stage then takes what is left. A stage that cannot finish in time is skipped and named in the injected block (for example 'rules not loaded: the machine is loaded'), so the prompt never loses everything in silence."
---

After the 2026-10-16 reboot, with machine load near 160, abcd's UserPromptSubmit hook ran past the host's 120-second timeout. The host discarded its output, so that prompt got no rules, no status block and no memory recall, and nothing said which part was slow. The hook execs abcd hook prompt-router once the binary is present. The router's own stages are each bounded (the status read at 600 ms, the root-commit lookup at 2 s), but nothing bounds the router as a whole.
