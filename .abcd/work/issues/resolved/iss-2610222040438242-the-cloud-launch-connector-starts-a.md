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
resolution: "launch.sh refuses an environment id that is not self-hosted before it asks the host, saying why, and its documented form, the intent and the spec now say the host names an environment at the start only for a self-hosted ccpool_ id; the Anthropic-hosted start is iss-2610222106115276"
impact: fix
---

The cloud launch connector (scripts/cloud/connectors/claude/launch.sh lines 10 and 51, pinned by internal/cloudscripts/cloud_send_test.go:113, and stated as the launch form in itd-2610120738075547 and spc-2610120941441282 step 12) passes whatever environment id the person configured to `claude -p "<prompt>" --environment <id> --ref <branch> --output-format json`. Claude Code's CLI reference and cloud-environments pages, read on 2026-10-22, document that form for a self-hosted environment only: `--environment` takes a `ccpool_` id and Claude Code rejects an Anthropic-hosted `env_` id there, and `--ref` applies only with it; the test fixture itself used an `env_` id. So a person whose cloud environment is Anthropic-hosted, the usual case, gets a refusal at launch. Found while preparing the product thinker's hosted run for 2026-10-23.

Correction, 2026-10-22: this record first said `--ref` is not documented and that the documented start is `claude --cloud`; the CLI reference documents `--ref` with `--environment`, and `claude --cloud` is the Anthropic-hosted start.
