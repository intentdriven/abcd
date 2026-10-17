---
schema_version: 1
id: "iss-2610160006073622"
slug: "the-cloud-connector-s-launch-step-relies"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "review of spc-2610120941441282, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/connectors/claude/launch.sh"
remedy: "Prove in the spec's live step 12 that the launch starts a hosted session and that an unknown flag is an error, and have the launch step refuse when the host's answer does not name a hosted session"
---

The cloud connector's launch step relies on host command-line flags (an environment id and a starting ref) that the host's local help does not list as documented for hosted sessions; if the host ignored them, the brief would start on the person's own computer, where only the brief's first-step place check stops it
