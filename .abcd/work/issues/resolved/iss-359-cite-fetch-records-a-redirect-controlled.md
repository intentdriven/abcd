---
schema_version: 1
id: "iss-359"
slug: "cite-fetch-records-a-redirect-controlled"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/core/cite/fetch.go"
resolution: "fetch boundary percent-encodes hidden runes in FinalURL byte-wise; baseline validate refuses them; termsafe and coverage doc claims corrected and pinned"
impact: fix
---

cite fetch records a redirect-controlled FinalURL verbatim, so C1, bidi and zero-width runes in a hostile Location query reach abcd docs lint --json and the committed citations baseline raw; the termsafe doc claim that encoding/json escapes control characters is false for those classes
## Evidence

- `internal/core/cite/fetch.go:204` assigns `out.FinalURL = resp.Request.URL.String()`. `url.Parse` rejects C0/DEL but preserves raw non-ASCII in `RawQuery`, and `String()` writes it verbatim — so the live payload set from a hostile redirect is C1 (2-byte U+009B), bidi (U+202E) and zero-width (U+200B).
- Reproduced end-to-end with `newHTTPChecker` against an httptest 302 whose Location query carried all three: `Status=="ok"`, raw bytes in `FinalURL`, `lint.SaveBaseline` accepted them (`Baseline.validate` checks only non-empty), and `abcd docs lint --json` (internal/surface/cli/cli.go:445 render) marshals them raw — `encoding/json` escapes only <0x20 plus U+2028/9.
- The false invariant is stated at `internal/termsafe/termsafe.go:22-23` and duplicated at `internal/core/lifeboat/coverage.go:149`.
- Refuter verdict: CONFIRMED substantive (minor, security). Remedy: percent-encode surviving non-printing runes at the fetch boundary (byte-wise, so invalid UTF-8 stays lossless too) + a `Baseline.validate` rung + correct both doc claims; a blanket JSON-path sanitizer was rejected (would corrupt a committed record that must round-trip). Distinct from iss-259/iss-264 (terminal text renders).
- Re-minted from the round-2 id that collided with a parallel hunt's mint.
