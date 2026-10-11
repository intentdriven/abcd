---
schema_version: 1
id: "iss-160"
slug: "the-installed-plugin-surface-exposes-a"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "marketplace-install smoke test"
found_at: "commands/README.md"
resolution: "The loader registers every markdown file under commands/ as a command — it requires no frontmatter and exempts no name, which agents/README.md registering as an agent independently shows (iss-110), and the published plugin reference describes commands purely as flat .md files with no ignore list. So no in-place form keeps a readme unregistered, and the doc moves out of the auto-discovery root into the brief's surface registry (.abcd/development/brief/04-surfaces/README.md), which already enumerates the same surface and is already machine-checked against it. index_drift's commands entry follows the doc; one link in a 2026-07-12 plan is repointed so links_resolve stays clean."
impact: fix
---

The installed plugin surface exposes a spurious /abcd:README command: the harness registers every markdown file under commands/ as a slash command, so the folder's own README.md leaks into the command list. The doc needs a home or form that the command loader ignores.