---
schema_version: 1
id: "iss-2610222106115276"
slug: "the-cloud-hand-off-cannot-start-a"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "lane E (abcd-54) fixing iss-2610222040438242, 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/connectors/claude/launch.sh"
remedy: "probe claude --cloud \"<task>\" live once (from a checkout on the ref, stdout not a terminal): record whether it exits after provisioning, what it prints, and whether --output-format json applies on creation; then add the Anthropic-hosted branch to launch.sh with that output's parser and a test of it, or record that the hosted environment cannot be started unattended"
---

The cloud hand-off cannot start a session on the Anthropic-hosted cloud environment: Claude Code names an environment at the start only for a self-hosted one (--environment takes a ccpool_ id and rejects Anthropic-hosted env_ ids, CLI reference read 2026-10-22), and the documented way to start an Anthropic-hosted session from a terminal, claude --cloud "<task>", takes its environment from the /remote-env default and its branch from the current checkout, and documents neither its output on creation (no machine-readable form is documented for creating, only for sending to an existing session) nor whether it exits or attaches; so scripts/cloud/connectors/claude/launch.sh refuses an env_ id rather than guess, and the planned hand-off (itd-2610120738075547) works only on a self-hosted environment until the create form is probed live
