# .abcd/

abcd's development namespace. Everything here is dev material — it stays in the
repo (transparent) and is present in every repository checkout, marketplace
installs and release source archives included, but never in the released
binaries. `docs/` (not here) holds user-facing documentation and is the only
dev-adjacent tree the release artifact carries.

Three tiers, on two axes (durability × sharing):

- **`development/`** — durable record (committed): brief, roadmap, intents,
  decisions/adrs, research. The specification for the build.
- **`work/`** — shared working (committed): `CONTEXT.md`, `decisions/` (one record per decision, `abcd decide`), the
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
| `config/` | per-surface machine records (`launch-payload.json`, `version-location.json`, `artefact.json`, `reading-presets.json`) and the optional records in the rows below | iss-62 / adr-28 / the version-location note |
| `config/oracle-routing.json` (optional) | the model-tier routing table for delegated steps, read through the layered configuration reader over the machine's `~/.abcd.noindex/oracle-routing.json`; `ahoy` offers to write it; absent, the machine's table applies, then the bundled proposal | itd-2609170822093401; `development/brief/04-surfaces/01-ahoy.md` (the `oracle-routing` offer); `internal/core/layered/layered.go` (`OracleRouting`) |
| `config/dependency-reauthor.conf` (optional) | the dependency-bump re-authoring declaration: `launch scaffold` keeps the re-author workflow current while it exists and never rewrites it; absent, the repository has not opted in | itd-2609221842494980; adr-2609292116133348; `development/brief/04-surfaces/04-launch.md` |
| `config/provenance.json` (optional, absent here) | the repository's default production mode for the records it mints, read by every minting verb that is given none; absent, the legacy `config/identity.json` answers, then hand-written | itd-178; `internal/core/provenance/declared.go` (`DeclaredMode`) |
| `config/identity.json` (legacy, absent here) | the committed identity pin `ahoy install` now removes, read only to migrate it and, while `config/provenance.json` is absent, for its production mode | itd-2610240918547728; adr-2610241022541063 |
| `config/history.json` (optional, absent here) | `history ingest`'s per-repo configuration (ingest roots, adopted harness projects, the orphan policy); absent, no roots are declared and orphans are ignored and reported | `development/brief/04-surfaces/11-history.md`; `internal/core/history/config.go` |
| `config/gitleaks.json` (optional, absent here) | the opt-in that arms the gitleaks augmenter beside the native secret scanner in every scan, the privacy lint's included; absent, the native scanner runs alone | iss-96; `internal/adapter/gitleaks/gitleaks.go` (`Config`) |
| `config/pii.json` (optional, absent here) | the redaction scanner's per-repo pattern override, read by every redacting write path and the privacy lint; absent, the bundled patterns apply | [`internal/README.md`](../internal/README.md) § `adapter/scanner/` |
| `config/scripts-closure.json` (optional, absent here) | the pinned `scripts/` runtime closure the launch payload scopes that include to; absent, `scripts/` is included like any other path | `internal/core/launch/includes.go` (`defaultClosureFn`); no chapter states its schema |
| `positioning.json` | the identity surface | `development/brief/04-surfaces/19-identity.md` |
| `site.json`, `site-baseline.json` | the site renderer and its ratchet | the site surface chapter |
| `docs-lint.json`, `record-lint.json` | the docs and record gates | the lint surface chapter |
| `citations-baseline.json` | the citation-health baseline | the docs `cite` surface |
| `prose-citations-baseline.json` | record-lint's `prose_citation_resolves` baseline (the unresolvable ids the record has ruled on) | `development/brief/05-internals/06-lint.md` |
