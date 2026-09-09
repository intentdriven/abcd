# `/abcd:lint` — Check Repo Conformance

Find out whether a repository actually follows the working conventions, without
changing anything and without reading the conventions yourself. One command
returns a graded list of what does not conform, each finding naming the file,
the reason and the fix, so a maintainer can decide what to repair and in what
order.

It is **strictly read-only**: it performs zero writes, and remediation stays
with `/abcd:prepare-this-repo` and the maintainer. It answers a different
question from `/abcd:ahoy`: `ahoy` reports whether the *tool* is installed and
configured for a repo; `lint` reports whether the *repo* conforms. Two
questions, two verbs.

The verb applies rules about form, which adr-40's vocabulary names a lint;
`/abcd:audit` stays reserved for itd-16's hash-chain fidelity surface.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

`abcd lint` registers no sub-verbs. The staged `chain` and `lifeboat`
verbs belong to the **reserved** `/abcd:audit` surface (itd-16's hash-chain
fidelity checks, registered in [`02-constraints/04-naming.md`](../02-constraints/04-naming.md)),
not to the conformance lint.

## What the answer looks like

The exit code is the decision, and it is Conftest's tri-state: `0` clean, `1`
warnings only, `2` any error. That is the shape a CI job can branch on without
parsing anything.

Without `--json`, the verb prints a grouped, doctor-style report: a severity
glyph, the rule id, `file:line`, the message, and the fix indented under it,
closing on a count. With `--json` it emits `findings` and `skipped`. Each
finding carries a stable `ruleId`, a `severity` (`error` or `warn`), a `file`, a
`message`, a `fix`, and a `policyInfo` rationale.

Two details a consumer needs. `line` is `omitempty` and tracks the individual
finding rather than the rule family: a finding that names a scanned line carries
it, and one that names a whole file carries none, so a consumer keys on the
field's presence and never on the rule id. And `skipped` names rules whose
enablement condition was not met (a docs rule in a repo with no `docs/`), so a
not-applicable rule reads as skipped rather than as passed or failed.

`--root` lints a repo other than the current working directory.

**No gate in this repository invokes it.** The verb backs onboarding and runs
when a human types it; wiring it into CI is a separate decision, tracked as
iss-2608231000561060.

## The v1 rule set

| id | severity | checks |
|---|---|---|
| `three-tier-layout` | error | `.abcd/development/` and `.abcd/work/` present as directories on disk; `.abcd/.work.local/`, when present, gitignored, which is the rule's one committedness assertion; no local-tier artefacts (`NEXT.md`, `scratch/`, `logs/`) sitting directly in one of the two shared tiers |
| `conventions-router` | error | `AGENTS.md` present at the repo root |
| `decision-durability` | warn | a committed `.abcd/work/DECISIONS.md`; decisions not living only in the gitignored layer |
| `docs-currency` | warn | reuses the docs-lint engine where `docs/` exists |
| `privacy-hygiene` | error (network-identifier findings mapped from a scanner `warn`/`info` land as `warn`) | no absolute local paths in committed files, and no real network identifiers on any tracked text line; the fix names reserved documentation values (RFC 5737/3849/2606/7042, or a persona-derived device name), and an `abcd-lint:allow` line waiver is honoured (the `abcd-audit:allow` spelling too) |
| `identity-positioning` | warn | every registered surface still carries the canonical identity block's tagline (and pitch, where required); gated on `.abcd/positioning.json` being present on disk, and per-repo upgradeable to `error` (see [`19-identity.md`](19-identity.md)) |

## How it is built

The engine (`internal/core/repolint`) adapts its id/severity/where/fix/policy
vocabulary from repolinter's rule-object schema, with severities
`error|warn|off`. That is a different vocabulary from the record-lint engine's
(`internal/core/lint`) `blocker|warn`, and the difference is resolved at the
boundary of each rule that carries a foreign vocabulary in, so the engine itself
only ever sees one: `docs-currency` maps the record-lint findings it reuses, and
`identity-positioning` maps the positioning registry's `blocker` onto `error`,
which is why the table above can offer a per-repo upgrade.

The `abcd-lint:allow` / `abcd-audit:allow` line waiver is defined natively in
the privacy rule. Rules are declarative data behind a rule-loader seam, and
output is serialised behind a seam that makes a later SARIF export additive. No
new dependency.

## References

- Plugin command: [`commands/lint.md`](../../../../commands/lint.md)
- Design record: [`plans/2026-07-13-abcd-audit-verb.md`](../../plans/2026-07-13-abcd-audit-verb.md)
- Intent: [`itd-85`](../../intents/drafts/itd-85-audit-verb.md)
- Onboarding consumer: [`15-prepare-this-repo.md`](15-prepare-this-repo.md)
