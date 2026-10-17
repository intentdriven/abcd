---
schema_version: 1
id: "iss-2610171249334225"
slug: "in-a-hosted-cloud-session-git-run-as-the"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "product thinker's live hosted-session announcement, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/connectors/claude/setup.sh"
remedy: "Have the cloud setup script hand the normal user a CA bundle it can read: when the proxy's CA file is not readable by that user, point git's http.sslCAInfo for the user (or GIT_SSL_CAINFO in the environment runuser passes) at the system trust store when it holds the same CA, and otherwise copy the bundle to a path the user can read; never disable TLS verification. Pin it with a cloudscripts test where the CA file is root-only."
---

In a hosted cloud session, git run as the normal user that abcd's cloud setup script creates (runuser -u abcd) cannot reach origin over HTTPS: the session's proxy CA bundle sits under /root, which that user cannot read, so every networked git command fails with a certificate error before anything is sent. Seen live on 2026-10-17 when a claude.ai/code session ran abcd announce; it got through only after the session pointed the CA variables at the system trust store, which already holds the same CA, for that one command. The setup script made the user, so abcd added this failure.
