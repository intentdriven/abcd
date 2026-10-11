---
schema_version: 1
id: "iss-386"
slug: "launch-scaffold-derives-a-floating-go"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/core/launch/scaffold/scaffold.go"
resolution: "admit the go.mod patch in scaffold go-version derivation; watched-fail TestDeriveRepoFactsKeepsPatchVersion"
impact: fix
---

launch scaffold derives a floating Go minor for adopter repos — goModVersionRe strips the go.mod patch, goVersionRe rejects supplying one, the toolchain directive is ignored, and a hand-pinned release.yml is refused then overwritten — the adopter-side residue of iss-289
## Evidence

- `internal/core/launch/scaffold/scaffold.go:214` — `goVersionRe = ^[0-9]+\.[0-9]+$` rejects a supplied patch; `:217` — `goModVersionRe` matches the go.mod patch group and discards it; `:226` — `defaultGoVersion = "1.25"` is a bare literal coupled to nothing. `Scaffold` (`scaffold.go:94-95`) always routes adopters through `DeriveRepoFacts` + `BareSubstitutions`, so no input yields a pin.
- Reproduced: a repo whose `go.mod` reads `go 1.25.6` derives `"1.25"`, and the rendered `release.yml` carries `go-version: '1.25'` at all three jobs — the float iss-289 recorded lagging a security patch release on real runners. A `toolchain go1.25.6` directive is ignored outright, so a `go 1.24.0` + toolchain repo scaffolds a release toolchain older than its module declares.
- An adopter cannot express the pin through the tool: a hand-pinned `release.yml` classifies `dispDiffers`, refusing later scaffolds, and `--confirm` overwrites the pin back to the float.
- `goversion_lockstep_test.go` walks `.github/workflows` only, so no test observes a bare rendering's `go-version`.
- Refuter verdict: CONFIRMED (minor, adopter-side residue of iss-289) — the strip is undocumented as a choice (the `:212-213` comment justifies only the injection-safety shape restriction, which a patch-admitting regex satisfies identically); no record or decision states a floating-minor-for-adopters policy.
