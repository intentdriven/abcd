---
schema_version: 1
id: "iss-2610090841317423"
slug: "abcd-dashboard-start-never-starts-on-a"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "2026-10-09 product thinker could not open the dashboard"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/dashboard/gate.go"
remedy: "Treat a listening address whose self-check cannot connect as one to drop, not a reason to stop: keep the addresses that answered (refusing to start only when none does), name the dropped one and why in the start line, and test it with a listener whose IPv6 self-connect fails while IPv4 answers, watched fail first."
resolution: "abcd dashboard start now drops a Tailscale address its self-check cannot connect to, closing that listener and naming it and the reason in the start line and --json, and starts on the addresses that answered; it refuses only when none does. Proven live on 2026-10-09 on the reporting Mac (Tailscale 1.102.4): start dropped the unreachable IPv6 address, listened on IPv4, refused this computer, and the product thinker's iPhone opened the page and was recorded as a let-in device. A security review (SHIP) found one latent cancellation case, fixed in the same change."
impact: fix
resolved_by:
  commit: "4eecce0ad"
---

abcd dashboard start never starts on a Mac whose Tailscale cannot reach its own IPv6 address. Start listens on both of the computer's Tailscale addresses and requires a self-check fetch to answer on every one; on 2026-10-09, with Tailscale 1.102.4 on macOS, a TCP connection to the computer's own Tailscale IPv4 address succeeded while one to its own Tailscale IPv6 address timed out, so the IPv6 self-check failed and start stopped the dashboard with 'could not be reached from this computer ... check that nothing on this computer blocks the port' (exit 1). Nothing blocks the port; the dashboard is unusable from every device.

## Grounds

- pursued: dropping an unreachable address instead of stopping makes the dashboard usable on a Mac whose Tailscale cannot reach its own IPv6, without weakening the gate on the kept addresses; shown wrong if a dropped address is still served, or if this computer is admitted on any address
