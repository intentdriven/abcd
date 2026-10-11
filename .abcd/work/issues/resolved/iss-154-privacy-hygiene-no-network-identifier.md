---
schema_version: 1
id: "iss-154"
slug: "privacy-hygiene-no-network-identifier"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "managed-repo NEXT.md privacy-leak investigation 2026-07-29"
found_at: "internal/core/audit/rule_privacy.go"
resolution: "privacy-hygiene now flags network identifiers outside the reserved documentation ranges, consuming the scanner's canonical pattern set"
impact: fix
---

privacy-hygiene detects only absolute home paths — no network-identifier patterns (IPv4 incl. CGNAT/tailnet 100.64/10, hostnames, device names, ports-in-context). Field incident: an agent committed a Tailscale investigation (tailnet IP, two device hostnames, firewall posture, service port, presence/away patterns) into a public repo's committed tier and audit passed it silently. The v1 deferral note names only emails and private-repo names; network identifiers are not even recorded as deferred.

Maintainer resolution (2026-07-29): the exempt set is "values that name no individual host", not "reserved documentation ranges" alone. Alongside loopback and unspecified it covers the IANA special-use ranges an address in which identifies a link, a group, a test harness or a protocol mechanism rather than a machine: IPv4 link-local 169.254.0.0/16, multicast 224.0.0.0/4, benchmarking 198.18.0.0/15 and IETF protocol assignments 192.0.0.0/24; IPv6 link-local fe80::/10, multicast ff00::/8, the NAT64 well-known prefix 64:ff9b::/96 and benchmarking 2001:2::/48. Netmasks and CIDR prefix declarations join them on the same rationale — a range is not a host. What stays flagged is what identifies private topology, which is the incident class: RFC 1918, CGNAT/tailnet 100.64.0.0/10, IPv6 unique-local fc00::/7, and 6to4 2002::/16 (it embeds a routable IPv4 address, so it names a host). The consistency of the rationale governs, not the convenience of the residue. The plan's STOP threshold counts findings, not distinct identifiers.

Design note (2026-07-29): the detector should be an allowlist inversion — flag any network identifier NOT inside the reserved documentation ranges (RFC 5737 IPv4 blocks, RFC 3849 2001:db8::/32, RFC 2606 example domains, RFC 7042 doc MACs; loopback/unspecified exempt). The principle `examples-use-reserved-identifiers` (principles/) is what makes this feasible at near-zero false positives: every legitimate committed example uses a reserved value, so anything outside the ranges is a finding. Shipping this lint promotes that principle to a discipline per the itd-79 path.