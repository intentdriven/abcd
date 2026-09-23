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
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `outbound` | gate | shipped |

The staged `chain` and `lifeboat` verbs belong to the **reserved** `/abcd:audit`
surface (itd-16's hash-chain fidelity checks, registered in
[`02-constraints/04-naming.md`](../02-constraints/04-naming.md)), not to the
conformance lint.

The outbound check is the `gate` bucket rather than `lint`, and the distinction is
the one adr-40 draws: the parent REPORTS on a repository and leaves the decision
with a human, while this one is wired into CI to make a binary pass/fail decision
about a single artefact. Its subject differs too — the parent's subject is this
repository, the sub-verb's is a piece of text the caller hands it — which is why
it takes the root for the scanner configuration explicitly rather than inheriting
the parent's.

## The outbound-policy gate

The outbound policy (`scanner.OutboundPolicy`, AGENTS.md § Attribution and
acknowledgements) bans two shapes from public text: a live agent-session URL and a
tool's own "generated with" attribution footer. Four surfaces judge that class and
all four read one definition — `scanner.HarnessLeakPatterns`. Three of them judge
text that is already committed or already stored (the store-before-commit
redactors, `abcd lint`'s `privacy-hygiene` rule, the record/docs `harness_leak`
rule). This is the fourth, and it is the only one that judges text BEFORE it is
public.

It refuses; it does not rewrite. `scanner.ScrubOutbound` is the rewrite direction
and remains without a front door by design (spc-45 scopes a forge client out): a
scrub is right for a routine sanitising text it is about to post, and wrong for
text a person already wrote. `scanner.CheckOutbound`, which backs this verb, has
no text return at all, so the door cannot become a rewriter by a later edit.

It reports only the harness-leak class, where the scrub masks everything the
scanner finds. Masking more than the policy names is free; REFUSING more than it
names is not — this runs as a required check over every commit message of every
pull request, so each extra class is a new way to go red on text that breaks no
stated rule, and a gate that reds on the innocent is a gate somebody switches off.

**Why it exists in Go rather than as a regex in the CI gate.** The footer half was
already gated by `scripts/check-attribution.sh`'s `GENERATED_RE`; the session-URL
half was gated nowhere at all, and one reached three commit messages and two
pull-request bodies of a managed public repo (iss-2609061438431625). It could not
follow the footer into the shell gate: the detector is a pattern plus an OPACITY
CLASSIFIER, and that classifier is a conjunction — a UUID, or a token carrying
both a digit and an upper-case letter, or a long lower-case hex run — which POSIX
ERE cannot express. The pattern without the classifier flags every page written
about session handling, including this repository's own research notes. The shell
gate therefore calls this verb, and there is still one definition of the class.

Exit codes are `0` clean, `1` the artefact is refused, `2` the check could not
run. That is deliberately NOT the parent's Conftest tri-state: both patterns are
hard-fail, so the middle rung has no meaning, and the distinction that matters to
a gate's caller is instead between a verdict and a check that never happened.

## What the answer looks like

The exit code is the decision, and it is Conftest's tri-state: `0` clean, `1`
warnings only, `2` any error. That is the shape a CI job can branch on without
parsing anything.

In its plain form, the verb prints a grouped, doctor-style report: a severity
glyph, the rule id, `file:line`, the message, and the fix indented under it,
closing on a count. The JSON form emits `findings` and `skipped`. Each
finding carries a stable `ruleId`, a `severity` (`error` or `warn`), a `file`, a
`message`, a `fix`, and a `policyInfo` rationale.

Two details a consumer needs. `line` is `omitempty` and tracks the individual
finding rather than the rule family: a finding that names a scanned line carries
it, and one that names a whole file carries none, so a consumer keys on the
field's presence and never on the rule id. And `skipped` names rules whose
enablement condition was not met (a docs rule in a repo with no `docs/`), so a
not-applicable rule reads as skipped rather than as passed or failed.

The verb can lint a repo other than the current working directory.


**No gate in this repository invokes it.** The verb backs onboarding and runs
when a human types it; wiring it into CI is a separate decision, tracked as
iss-2608231000561060.

## The v1 rule set

| id | severity | checks |
|---|---|---|
| `three-tier-layout` | error | `.abcd/development/` and `.abcd/work/` present as directories on disk; `.abcd/.work.local/`, when present, gitignored, which is the rule's one committedness assertion; no local-tier artefacts (`NEXT.md`, `scratch/`, `logs/`) sitting directly in one of the two shared tiers |
| `conventions-router` | error | `AGENTS.md` present at the repo root |
| `decision-durability` | warn | a committed `.abcd/work/DECISIONS.md`; decisions not living only in the gitignored layer |
| `docs-currency` | warn | reuses the docs-lint engine where `docs/` exists, and says so where it cannot: a repo with a `docs/` tree but no docs-lint configuration, and a configuration that will not load, each raise a finding against `.abcd/docs-lint.json` rather than passing quietly |
| `privacy-hygiene` | error (network-identifier findings mapped from a scanner `warn`/`info` land as `warn`) | three leak classes on any tracked text line: absolute local paths in committed files, real network identifiers, and the harness-leak pair the outbound policy bans everywhere (a live agent-session URL, and a tool's own "generated with" footer). The fix names reserved documentation values (RFC 5737/3849/2606/7042, or a persona-derived device name), and an `abcd-lint:allow` line waiver is honoured (the `abcd-audit:allow` spelling too). The network severities come from the merged scanner configuration, so a repo that raises one in `.abcd/config/pii.json` is honoured, and an override that cannot be read is itself an `error` finding saying the scan fell back to the built-in severities. Two findings report what was *not* read rather than a leak: a tracked text file over the 4 MiB scan cap, and one that could not be opened. Binary files are skipped silently |
| `identity-positioning` | warn | every registered surface still carries the canonical identity block's tagline (and pitch, where required), and every registered surface can still be found: a surface whose locator matches nothing is its own finding, because drift there would go unseen. A registry or identity block that cannot be read is reported rather than passed. Gated on `.abcd/positioning.json` being present on disk, and per-repo upgradeable to `error` (see [`19-identity.md`](19-identity.md)) |

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

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd lint`

Sub-verbs: `abcd lint outbound`.

| Flag | Type |
|---|---|
| `--root` | string |

### `abcd lint outbound`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--label` | string |
| `--root` | string |

<!-- surface-appendix:end -->
