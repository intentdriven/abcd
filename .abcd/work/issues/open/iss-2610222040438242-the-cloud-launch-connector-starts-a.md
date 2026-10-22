---
schema_version: 1
id: "iss-2610222040438242"
slug: "the-cloud-launch-connector-starts-a"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "lane E (abcd-54) preparing the PT hosted run, 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/connectors/claude/launch.sh"
remedy: "re-check the launch form against code.claude.com/docs/en/claude-code-on-the-web and cli-reference with a live claude --cloud run, then change launch.sh, its test and the intent and spec's launch line to the documented form, recording how the session id and link are read back"
---

The cloud launch connector starts a hosted session with claude -p "<prompt>" --environment <env-id> --ref <branch> --output-format json (scripts/cloud/connectors/claude/launch.sh lines 10 and 51, pinned by internal/cloudscripts/cloud_send_test.go:113, and stated as the launch form in itd-2610120738075547 and spc-2610120941441282 step 12), but Claude Code's documentation read on 2026-10-22 starts a cloud session from the terminal with claude --cloud "<task>" (--remote a deprecated alias), takes the branch from the current checkout's remote branch rather than a --ref flag, and documents --environment <id> only for self-hosted environments; so the person's hand-off in step 12 may fail at launch, found while preparing the product thinker's hosted run for 2026-10-23
