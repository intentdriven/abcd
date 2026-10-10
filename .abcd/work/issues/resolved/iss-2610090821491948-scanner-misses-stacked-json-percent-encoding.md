---
schema_version: 1
id: "iss-2610090821491948"
slug: "scanner-misses-stacked-json-percent-encoding"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-74v2-f6pg-p4fq, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/percent.go"
remedy: "After each JSON layer and after the percent view, run the other decoder (bounded by the existing layer caps) and map every hit back to the raw line, and correct the jsonescape.go comment; prove it with a scanner test (watched fail first) that the three reproduction payloads hard-fail and `Redact` leaves neither the token nor the encoded form, while the plaintext, single-percent and single-\\uXXXX controls still hard-fail; sweep siblings (every pair of decoded views lineViews builds)."
resolution: "lineViews now runs each decoder over the other's output once each way (JSON layers of the percent view, the percent view of each JSON layer), every composed view mapped back to the raw line, and the comment that said the two never compose is corrected."
impact: fix
---

The secret scanner misses a token or home path written as a JSON escape stacked on a percent encoding (or the reverse), because it never runs one decoder on the other's output, so `ScanText` reports nothing, `Redact` leaves it, and the launch gate ships it.

Private security advisory GHSA-74v2-f6pg-p4fq (draft, severity high). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `lineViews` builds one percent view of the raw line and then JSON layers of that same raw line (internal/adapter/scanner/percent.go:61). The comment at internal/adapter/scanner/jsonescape.go:45 claims the two do not compose, yet `jsonUnescapeOnce` emits `%` as `%` (internal/adapter/scanner/jsonescape.go:100). `ScanBundle` counts the text file as fully scanned (internal/adapter/scanner/scanner.go:1225), and `scanRefusals` (internal/core/launch/dryrun.go:269) refuses only an unavailable scanner, a kept hard-fail or an Unscanned path. History and memory call the same `ScanText`.

Reproduction: with T a `ghp_` prefix followed by 40 `q` characters, none of these hard-fails: `token=%67` followed by T without its first character (JSON-unescape then percent-decode yields T); `token=%5Cu0067` with the same tail (the reverse order); `see %2F<home segments joined the same way> in the log` for the probed home path (no home_path_self). The plaintext token, a single `%67` and a single `g` of the first byte still match.
