---
schema_version: 1
id: "iss-2610252224221774"
slug: "scripts-cloud-connectors-claude-launch"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "state-of-the-art passes of the hand-back triage, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/connectors/claude/launch.sh"
remedy: "Parse the documented answer in launch.sh and pin it in cloud_send_test.go, keeping the old shape only if a recorded run shows the host still prints it; add the host id prefixes to the C12 marker list and move the ccpool_ wording out of send.sh into the connector."
---

scripts/cloud/connectors/claude/launch.sh:19-22 says the self-hosted dispatch answer's exact JSON fields are not documented, but the host's CLI reference now documents them: ok true with session_id, title, url and pool_id, or ok false with error and exit 1 (read 2026-10-24). internal/cloudscripts/cloud_send_test.go:48 still pins the generic -p result shape rather than that documented line. Separately, the C12 test's marker list names no host id forms, so the ccpool_ id form send.sh:80 names outside the connector passes the check.
