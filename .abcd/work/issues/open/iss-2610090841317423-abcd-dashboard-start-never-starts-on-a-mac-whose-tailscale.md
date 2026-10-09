---
schema_version: 1
id: "iss-2610090841317423"
slug: "abcd-dashboard-start-never-starts-on-a-mac-whose-tailscale"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "2026-10-09 product thinker could not open the dashboard"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/dashboard/gate.go"
remedy: "Treat a listening address whose self-check cannot connect as one to drop, not a reason to stop: keep the addresses that answered (refusing to start only when none does), name the dropped one and why in the start line, and test it with a listener whose IPv6 self-connect fails while IPv4 answers, watched fail first."
---

abcd dashboard start never starts on a Mac whose Tailscale cannot reach its own IPv6 address. Start listens on both of the computer's Tailscale addresses and requires a self-check fetch to answer on every one; on 2026-10-09, with Tailscale 1.102.4 on macOS, a TCP connection to the computer's own Tailscale IPv4 address succeeded while one to its own Tailscale IPv6 address timed out, so the IPv6 self-check failed and start stopped the dashboard with 'could not be reached from this computer ... check that nothing on this computer blocks the port' (exit 1). Nothing blocks the port; the dashboard is unusable from every device.
