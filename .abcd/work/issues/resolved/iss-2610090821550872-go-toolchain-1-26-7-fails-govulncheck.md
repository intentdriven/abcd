---
schema_version: 1
id: "iss-2610090821550872"
slug: "go-toolchain-1-26-7-fails-govulncheck"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "merge queue run 37897679940 (govulncheck job), 2026-10-09"
origin: researcher-authored
production_mode: hand-written
found_at: "go.mod"
remedy: "Bump go.mod's go directive to 1.26.9 (scripts/pinned-toolchain.sh and CI's setup-go follow it), run make preflight on the new toolchain, and confirm govulncheck passes; needs the user's dependency sign-off before it lands."
resolution: "go.mod moved to go 1.26.9; govulncheck v1.7.0 reports no reachable vulnerability on that toolchain"
impact: fix
---

main pins go 1.26.7 in go.mod, and govulncheck fails the merge queue (merge_group run 37897679940) on twelve stdlib advisories fixed in go1.26.9, among them GO-2026-6599 and GO-2026-6600 (html/template, reached through the dashboard page handler) and net/http traces.

Verified 2026-10-09 with `GOTOOLCHAIN=go1.26.7 go run golang.org/x/vuln/cmd/govulncheck@latest ./...` at main 7549ca2d5: GO-2026-6617, -6613, -6612, -6611, -6610, -6605 and -6603 (net/http), GO-2026-6608 (net/textproto), GO-2026-6607 (crypto/tls), GO-2026-6604 (os), GO-2026-6600 and GO-2026-6599 (html/template, trace internal/surface/dashboard/server.go:274 `dashboard.servePage` calls `template.Template.Execute`), each fixed in go1.26.9.
