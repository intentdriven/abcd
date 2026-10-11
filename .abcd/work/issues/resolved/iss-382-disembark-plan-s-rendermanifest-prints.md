---
schema_version: 1
id: "iss-382"
slug: "disembark-plan-s-rendermanifest-prints"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/core/lifeboat/plan.go"
resolution: "sanitise planned file paths in RenderManifest; watched-fail TestRenderManifestSanitisesFilePaths"
impact: fix
---

disembark plan's RenderManifest prints planned file paths unsanitised, so a hostile source-repo filename injects raw C1/bidi terminal escapes into the plan render and its --json twin
## Evidence

- `internal/core/lifeboat/plan.go:363` — `fmt.Fprintf(&b, "  %8d  %s\n", f.Bytes, f.Path)` writes the planned path raw, while the same function sanitises `SourceName` (`plan.go:361`) and both omission fields (`plan.go:369`), and the sibling renderer `embark_render.go:150` sanitises `Conflict.Path` — the one unsanitised renderer of source-repo-derived paths in the package.
- `f.Path` leaves derive from source-repo directory entries via `safeLeaf` (`plan.go:516-532`), which rejects only C0/DEL — not C1 (U+0080–009F), bidi (U+202E) or zero-width. Reached at the terminal via `internal/surface/cli/cli.go:604`; `fsutil.ValidRelPath` is C0/DEL-only too, so nothing narrows it downstream.
- Reproduced on a hostile source repo: issue files named with U+202E and U+009B rendered both runes raw in `abcd disembark plan` output; the `--json` branch leaks them identically (`encoding/json` escapes neither).
- Refuter verdict: CONFIRMED (minor, security) — the untrusted-source ruling is the package's own (`coverage.go` sanitises the same boundary, pinned by `coverage_test.go:133-160`); prior art checked (iss-259, iss-264, iss-340, the sibling round's coverage-renderer fix) — none names `plan.go`/`RenderManifest`.
