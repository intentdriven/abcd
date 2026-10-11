---
schema_version: 1
id: "iss-2610090841586205"
slug: "abcd-s-dashboard-reaches-other-devices"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "2026-10-09 product thinker capture after the dashboard test"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/dashboard/gate.go"
remedy: "Run a primary-source check first (WireGuard's protocol and wireguard-go's licence and embedding, how a peer's public key can stand for a named device, and key distribution), then offer WireGuard as an opt-in transport the person picks at dashboard setup, admitting a connection only from a peer key the person paired, with Tailscale staying the default; record the identity rule as an amendment to the dashboard's network decision."
---

abcd's dashboard reaches other devices only through Tailscale: it listens on the computer's Tailscale addresses and admits a device only when Tailscale's own lookup names it. The product thinker wants WireGuard as a built-in alternative the person may choose instead, so the dashboard does not depend on one vendor's account and coordination service. A plain WireGuard tunnel has no lookup that names a person's device, so the alternative needs its own device identity, such as each peer's public key mapped to a named device.
