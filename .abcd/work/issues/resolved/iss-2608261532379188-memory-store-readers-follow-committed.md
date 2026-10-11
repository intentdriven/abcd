---
schema_version: 1
id: "iss-2608261532379188"
slug: "memory-store-readers-follow-committed"
severity: "major"
category: "security"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "internal/core/memory/bare.go"
resolution: "Every memory-store and ingest-source read routes through fsutil.ReadGuarded with in-package caps; WalkDir crawls skip non-regular entries; watched-fail symlink tests on bare, lint, budget, writer and licence paths"
impact: fix
resolved_by:
  commit: "9e8f3235"
---

Every memory-store reader outside ingest follows a committed symlink unbounded. internal/core/memory/ingest.go adjudicates the store as a trust boundary (maxMemoryPageBytes: a committed page symlink to /dev/zero would hang or OOM the CLI) and routes its own reads through fsutil.ReadGuarded, but the sibling readers kept raw os.ReadFile: readOrEmpty (bare.go, reached with no type check for index.md/contradictions.md), bareHeadroomLines and the lint WalkDir sweeps (lint.go — WalkDir yields symlinks as non-dir entries and the read follows them), loadQuotationBudget (coverage.go reading config.json), the .coverage_index.json reads (bare.go, coverage.go — its literal sibling .sources_index.json is guarded), triStateRead (writer.go), and the ingest licence probes (provenance.go manifestLicence/licenceFileLicence under an arbitrary sourceRoot). Reproduced: five distinct hangs via committed mode-120000 fixtures against abcd memory and abcd memory lint. The ReadDir-filtered sites (ask.go QueryPages, barePageInfos) carry the ReadDir-to-open swap window and no size cap — the exact TOCTOU ingest.go documents closing. Same class as the resolved lint.LoadConfig unguarded read (major/security). Detector: watched-fail tests planting a symlinked page/index; acceptance: every memory-store and source read routes through the guarded primitive.