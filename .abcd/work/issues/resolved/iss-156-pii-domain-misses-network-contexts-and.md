---
schema_version: 1
id: "iss-156"
slug: "pii-domain-misses-network-contexts-and"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "managed-repo NEXT.md privacy-leak investigation 2026-07-29"
found_at: "internal/core/rules/defaults/rules.json"
resolution: "PII domain gains 15 network/infra recall keywords (the issue's six plus plural/notation/inflection and protocol vocabulary from review), a mac address alias, and a never-commit-hostnames/IPs/MACs rule line citing RFC 5737/3849/2606/7042"
impact: fix
---

PII rules domain cannot fire on the incident that most needs it: recall keywords (secret, token, credential, pii, redact, hostname, email) miss network/infra contexts (tailscale, vpn, ip, firewall, network, reachability), and the injected rule text carries no never-commit-hostnames-IPs-live-data rule at all — that rule exists only in the parent CLAUDE.md privacy section. An agent writing up a Tailscale investigation gets zero PII injection and zero relevant rule text even if it did.
