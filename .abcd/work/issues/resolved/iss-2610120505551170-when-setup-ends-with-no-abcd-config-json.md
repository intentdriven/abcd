---
schema_version: 1
id: "iss-2610120505551170"
slug: "when-setup-ends-with-no-abcd-config-json"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "run-2610110908176735 lane-4 round-3 review (ruthless-reviewer note)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/install_summary.go"
remedy: "When the run ends with no config, have the readme.offered summary action and the text render's hint name the starter files first (answer y to creating them), then the question or --readme-parts."
resolution: "A run that ends with no .abcd/config.json now names the starter files first in the readme.offered summary item and in the text render's hint, instead of the two routes (answering at a terminal, --readme-parts) that both fail without the config; Install sets readmeAwaitsConfig from readmeStorable, read through InstallResult.ReadmeAwaitsConfig()."
impact: internal
resolved_by:
  commit: "4726c84dba6129c59c53b4b4fc7b706d6956cb37"
---

When setup ends with no .abcd/config.json (the starter files declined, at a terminal or off one), it reports the README question under optional_skipped, and both routes its hint names fail: the summary's action says to run abcd ahoy install at a terminal without --yes and answer it, which is the run just made and does not ask it, or to pass --readme-parts, which storeReadmeParts refuses because .abcd/config.json does not exist yet; the text render's extra line says the same. The true route, creating the starter files first, is named only by the separate declined-starter-files entry.

## Grounds

- pursued: a person whose run ends without the config is pointed at the one route that works; a run with no config whose summary or text render still names only the question or --readme-parts as the way to answer it would show it wrong
