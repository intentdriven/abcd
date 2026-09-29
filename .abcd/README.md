# .abcd/

abcd's development namespace. Everything here is dev material — it stays in the
repo (transparent) and is present in every repository checkout, marketplace
installs and release source archives included, but never in the released
binaries. `docs/` (not here) holds user-facing documentation and is the only
dev-adjacent tree the release artifact carries.

Three tiers, on two axes (durability × sharing):

- **`development/`** — durable record (committed): brief, roadmap, intents,
  decisions/adrs, research. The specification for the build.
- **`work/`** — shared working (committed): `CONTEXT.md`, `DECISIONS.md`, the
  issue ledger `issues/` (adr-32), the reviews charter `reviews/`, the
  branch-ruleset mirror `rulesets/`, and `intake.md`, the external-contribution
  runbook.
- **`.work.local/`** — local ephemeral (gitignored): `NEXT.md`, `scratch/`,
  `logs/` (run output), `reviews/` (the intent-audit receipt outbox), and
  `private-names.txt` (the per-machine banlist layer).

See [`../AGENTS.md`](../AGENTS.md) for the full layout and boundaries.

## Root machine records

The tracked files at this directory's root are read by the shipped binary; each
schema is owned by the record named beside it — this list is an index, not a
second home for the schemas:

| File | Read by | Owning record |
|---|---|---|
| `config.json` | the ahoy surface (repo-scope config + `meta` setup block) | `development/brief/05-internals/03-configuration.md` |
| `rules.json` | the rules loader (per-repo domain overrides) | itd-3; `AGENTS.md` § abcd rule loader |
| `config/` | per-surface machine records (`identity.json`, `launch-payload.json`, `version-location.json`, `artefact.json`, `reading-presets.json`) | iss-62 / adr-28 / the version-location note |
| `config/pii.json` (optional, absent here) | the redaction scanner's per-repo pattern override, read by every redacting write path and the privacy lint; absent, the bundled patterns apply | [`internal/README.md`](../internal/README.md) § `adapter/scanner/` |
| `config/scripts-closure.json` (optional, absent here) | the pinned `scripts/` runtime closure the launch payload scopes that include to; absent, `scripts/` is included like any other path | `internal/core/launch/includes.go` (`defaultClosureFn`); no chapter states its schema |
| `positioning.json` | the identity surface | `development/brief/04-surfaces/19-identity.md` |
| `site.json`, `site-baseline.json` | the site renderer and its ratchet | the site surface chapter |
| `docs-lint.json`, `record-lint.json` | the docs and record gates | the lint surface chapter |
| `citations-baseline.json` | the citation-health baseline | the docs `cite` surface |
| `prose-citations-baseline.json` | record-lint's `prose_citation_resolves` baseline (the unresolvable ids the record has ruled on) | `development/brief/05-internals/06-lint.md` |
