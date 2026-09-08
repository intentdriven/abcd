---
schema_version: 1
id: "iss-2609081953452204"
slug: "gofmt-gate-reports-a-false-positive-on-a-newer-toolchain"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/remote.go"
---

The pre-push format gate documented in AGENTS.md is a bare gofmt -l ., which reads whatever Go toolchain the developer has rather than the one CI pins. go.mod declares go 1.26.7 and every CI leg pins go-version 1.26.7, but gofmt from go 1.27 re-indents a multi-value return whose operands are composite literals, so on a 1.27 machine gofmt -l . reports internal/core/ahoy/remote.go against an unmodified checkout of main. The failure is a trap rather than a nuisance: the developer is told the tree is unformatted, reformats the file to satisfy the documented gate, and pushes a file that CI's own gofmt then reports as unformatted in the other direction. Neither direction is detectable from the gate's output, which names a file and says nothing about which toolchain judged it. Fix candidates, none chosen here: pin the gate to the go.mod toolchain (go run mvdan.cc/gofumpt or go1.26.7 gofmt via the toolchain directive), have the Makefile compare the running gofmt's version against go.mod and refuse loudly on a mismatch rather than emitting a filename, or move the format gate into make preflight where the toolchain is already asserted. Detector: on a machine whose default gofmt is newer than the go.mod toolchain, the format gate over an unmodified checkout must either pass or refuse by naming the version skew, never name a file. Surfaced while clearing the site-composer bug-fix branch, where the finding cost two subagents a detour each.
