---
id: dcn-2611202018385878
slug: in-a-tool-list-go-names-its-version-file
date: 2026-11-20
---
2026-11-20 — In a tool list, Go names its version file as go.mod or go.work at the repository's root, and no other tool names a version file (itd-2611041815429986, step 1). Grounds: the spec's closed set is a strict reader of go.mod and go.work, and the needs-step1 Opus review accepted it, held by the test refusing tools/go.mod and ./go.mod. A monorepo whose only go.mod sits in a subdirectory cannot list Go yet; that is a known limit left for itd-2611042148424105 rather than widening the set now. Refs itd-2611041815429986, itd-2611042148424105.
