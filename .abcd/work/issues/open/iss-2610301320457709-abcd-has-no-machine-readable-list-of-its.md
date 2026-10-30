---
schema_version: 1
id: "iss-2610301320457709"
slug: "abcd-has-no-machine-readable-list-of-its"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "ideate assessment of a terminal runtime for coding agents, 2026-10-30"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/helpgroups.go"
remedy: "Keep abcd --help --agent as compact text by default and make --json work on it, as every other verb's does: write one JSON document derived from the same cobra tree and help placement that renderRootHelp reads (internal/surface/cli/helpgroups.go), carrying a format_version under Terraform's providers-schema rules (1.x stays backward-compatible, a reader ignores unknown fields and refuses an unknown major); a top-level instructions block for agents (the binary-resolution ladder, the --json convention and its error envelope, what exit 0, 1, 2 and 3 mean, that bare forms write nothing); and per verb and agents-block sub-verb its name, block, one-line contract (what it writes, what it refuses), the commands/*.md page to read next, and its flags with type, default and closed value set; end the text menu with one line naming the --json form; proven by a test beside TestCommandPagesMatchTheMenu that the document's verb set equals the rendered menu's and every named page exists. Grounds: a 2026-10-30 state-of-the-art pass found no source for JSON by default on a help surface and no measured gain for a model reading JSON (He et al. 2024, arXiv 2411.10541: no format wins across tasks), JSON costing about 35-50% more tokens than markdown in one small-model test, help as text with JSON opt-in in clig.dev, gh and kubectl explain, and the host's skill listing as one name-description line each; a TTY-based switch is rejected because harnesses differ in giving a pty, and a schema surface beyond this is left to the MCP front door."
---

abcd has no machine-readable list of its verbs: abcd --help --agent --json ignores --json and prints the text menu, so an agent learns the verbs, their flags and how to call them only by parsing help prose or reading every commands/*.md page. The list must be built from the same source as --help --agent and must carry machine-readable instructions for an agent to use abcd.
