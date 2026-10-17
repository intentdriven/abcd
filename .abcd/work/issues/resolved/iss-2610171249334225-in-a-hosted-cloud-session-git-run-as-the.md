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
resolution: "The cloud setup now hands the normal user a readable copy of the proxy's CA bundle. For each CA variable (SSL_CERT_FILE, GIT_SSL_CAINFO, CURL_CA_BUNDLE, REQUESTS_CA_BUNDLE, NODE_EXTRA_CA_CERTS) that names a file the user cannot read, root copies exactly that bundle, read-only and root-owned, into a directory only root can write (/usr/local/share/abcd/cloud-ca by default), so the user can read it but never change what root's own shells also trust; the variable then points at the copy for every command the setup runs as the user and, through the session's file of variables, for later shells. A bundle the user can already read is left as it is, and TLS verification is never turned off or swapped for the system store."
impact: fix
---

In a hosted cloud session, git run as the normal user that abcd's cloud setup script creates (runuser -u abcd) cannot reach origin over HTTPS: the session's proxy CA bundle sits under /root, which that user cannot read, so every networked git command fails with a certificate error before anything is sent. Seen live on 2026-10-17 when a claude.ai/code session ran abcd announce; it got through only after the session pointed the CA variables at the system trust store, which already holds the same CA, for that one command. The setup script made the user, so abcd added this failure.

## Grounds

- pursued: git, gh and abcd run as the normal user in a hosted session verify the proxy's certificate with no per-command workaround; a networked command run through runuser still failing its certificate check in a live session, or a copy whose bytes differ from the proxy's bundle, would show it wrong
