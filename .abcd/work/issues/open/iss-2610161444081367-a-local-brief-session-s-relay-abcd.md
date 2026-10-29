---
schema_version: 1
id: "iss-2610161444081367"
slug: "a-local-brief-session-s-relay-abcd"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/session.go"
remedy: "On a machine with Tailscale running, start abcd session start, run tailscale serve --bg <port>, open the printed link from a second tailnet device, and record the Host the relay receives; widen the Hosts list in newSessionStartCommand to exactly that form if it differs, with a test pinning it."
---

A local brief session's relay (abcd session start) refuses any request whose Host header is not the computer's tailnet name or its loopback address, with 421. Which Host header Tailscale Serve passes to the loopback backend was not verified on a live tailnet (the build sandbox could not run the tailscale command), so if Serve sends a form the allowlist does not name, every request through Serve is refused and the session cannot be reached.
