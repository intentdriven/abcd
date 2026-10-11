---
schema_version: 1
id: "iss-384"
slug: "baseline-validate-sanitises-final-url"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/core/lint/baseline.go"
resolution: "sanitise the key and echoed fields in Baseline.validate; watched-fail TestBaselineErrorSanitisesEchoedFields"
impact: fix
---

Baseline.validate sanitises final_url alone while echoing the entry key, url and quoted fields raw, so a committed citations-baseline entry writes ESC/bidi/zero-width to the terminal through the blocking docs-lint gate
## Evidence

- `internal/core/lint/baseline.go:187` sanitises `e.FinalURL` (the iss-359 rung) while the same loop emits the map key (`who := "citation baseline entry " + k`, `baseline.go:175`), `e.URL` (`:178`), and `quote(e.LastChecked)`/`quote(e.Outcome)`/`quote(e.Verification)`/`quote(e.VerifiedOn)` (`:191-207`) raw; the resulting `*BaselineError` reaches the terminal via `cli.Run`'s error surface (`internal/surface/cli/cli.go:2859-2868`), which applies `scrubPaths` (a path redactor), never `termsafe`.
- The file's own threat model names a hostile branch (`baseline.go:71-74`), and the honest producer (`SaveBaseline` via `cite refresh`) validates on the way out — so the raw-emitting branches are reachable only with hand-edited or hostile committed input.
- Reproduced end-to-end: a poisoned committed baseline entry (key carrying U+202E + ESC + U+200B, `outcome` carrying ESC) made the blocking `docs lint` gate exit 1 with raw ESC, bidi and zero-width bytes on stderr — including through the `:187` refusal message itself, whose `who` prefix carries the unsanitised key.
- Refuter verdict: CONFIRMED (minor, security) — the incomplete-fix residue of iss-359's `Baseline.validate` rung; distinct from iss-259/iss-264 (different surfaces, per iss-359's own precedent). The wider gap — `cli.Run` never passing core errors through `termsafe` — is noted as the generalisation.
