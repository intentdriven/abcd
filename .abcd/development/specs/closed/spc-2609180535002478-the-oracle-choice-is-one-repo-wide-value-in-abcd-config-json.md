---
id: spc-2609180535002478
slug: the-oracle-choice-is-one-repo-wide-value-in-abcd-config-json
intent: itd-2609170822093401
origin: researcher-authored
production_mode: hand-written
---
# abcd proposes the model each agent deserves, and facilitates it where it can

## Summary

spc-2609180535002478 delivers itd-2609170822093401: a routing table with one
row per agent — a model tier from a closed set and a fan-out bound — that
abcd ships as its own proposal, renders, and applies only once accepted. Four
layers resolve to one row per agent: an invocation `--route` flag over the
repository's committed file over the machine's accepted file over the bundled
proposal. At run time the winning row is resolved against the connections the
machine has configured: a reachable provider that can serve the tier takes the
step directly; otherwise the step goes to the harness with the tier and bound
in its request block, the fallback announced on stderr and recorded on the
receipt beside the model the host or provider reported. Provider settings
default on the connection and are overridden per row and per flag. Nothing
here enforces the payload's `model` or `agents_used` fields (that is
`itd-2609180517121254`), wires a provider (`itd-2609081951381895`), or learns
from outcomes (itd-17).

## Scope

- **One resolver in core.** `internal/core/oracle` (new) owns the routing
  types and the resolution: `Row{Tier, FanOut, Settings}`, `Table` (agent →
  Row), `Layered{Flag, Repo, Machine, Bundled}`, and
  `Resolve(agent, layered, connections) → Route{Row, Connection, Fallback,
  Source}`. The resolver never writes and never reaches a network; it takes a
  `connections` value the caller supplies, so a test can hand it a provider
  that is "reachable" without a socket. Every front door renders a `Route`;
  none invents one.
- **The bundled proposal.** A Go value in `internal/core/oracle/proposal.go`,
  one row per file in `agents/`, keyed on the agent's `name`. A test asserts
  the proposal names every agent in the roster and no other, so an agent added
  without a row, or a row for an agent removed, fails the build rather than
  surfacing at run time (the intent's second open question, answered here).
  The initial rows: `frontier` for `intent-auditor`, `lifeboat-reviewer`,
  `ruthless-reviewer`, `security-reviewer`, `release-changelog-composer`;
  `economy` for the four cold-reading positions, `docs-currency-reviewer`,
  `graveyard-interpreter`, `principle-distiller`, `press-release-composer`,
  `scribe`, `sota-researcher`; fan-out taken from each agent's contract
  ceiling. These are the best guess the press release promises, and the
  calibration for them is itd-17's.
- **Two stores, one shape.** `~/.abcd/oracle-routing.json` (machine) and
  `.abcd/config/oracle-routing.json` (repository), both
  `{"schema_version": 1, "agents": {"<name>": {"tier": …, "fan_out": …,
  "settings": {…}}}}` — the `positions`-map shape `reading-presets.json`
  already uses. Both are read through the guarded reader; a repo file is read
  only from the checkout root the session resolved. A row naming an agent not
  in the roster is reported on stderr by name and skipped; the remaining rows
  apply (AC 9).
- **The tier vocabulary.** `local`, `economy`, `frontier`, `host-decides`;
  one closed enum in core, rendered by every refusal that names it. A row's
  `fan_out` above the agent contract's ceiling is reported and clamped to the
  ceiling at read time, never raised.
- **Connections.** The resolver consumes a `Connections` interface —
  `Serves(tier) (Connection, bool)` — that the api adapter intent implements.
  Until it ships, the only implementation is the empty one, so every row
  resolves to the harness and AC 1 holds on every machine.
- **The consent step.** `ahoy install` gains a fourth consent category,
  `oracle-routing`, after `status-line`: it renders the proposal as a table
  (agent, tier, fan-out), asks once, and writes the machine file on yes; a
  second, separate prompt offers the repository file. `--yes` skips both and
  reports them under `optional_skipped`, the same shape as the status line.
  `ahoy uninstall` leaves both files, because they are the user's
  configuration.
- **The invocation override.** `--route <agent>=<tier>[@<connection>][?k=v,…]`,
  repeatable, registered on every delegating verb through one cobra helper so
  a verb cannot gain delegation without gaining the flag. A `--route` naming
  an agent the verb does not dispatch, a tier outside the enum, or a
  connection not configured on this machine is refused at exit 2 before any
  step runs.
- **The request block and the receipt.** Every delegating verb already writes
  a request block for the host; it gains a `routing:` section carrying the
  resolved tier, bound and source layer. Every ingest already writes a
  receipt; it gains `route: {tier_asked, connection_tried, connection_used,
  fallback_reason, override, settings_sent, model_reported}`, where
  `model_reported` is whatever the payload's `model` field carries today (its
  requirement is `itd-2609180517121254`'s).
- **The board.** The bare `abcd` board gains one line per agent under
  `oracle:` showing the four layers and marking the winner (AC 6), rendered
  from the same `Route` values.

## Approach

1. Land the core package with the types, the enum, the proposal and its
   roster test, and `Resolve` over an in-memory `Connections` — every layer
   and fallback case table-tested with no filesystem.
2. Add the two store readers behind the guarded reader, with the orphan-row
   report and the ceiling clamp; table-test precedence with all four layers
   populated and disagreeing.
3. Register the `--route` flag through the shared helper on the delegating
   verbs (`intent audit`, `launch ship`, `disembark review|principles|
   press-release|graveyard`, `reading ingest`, `memory ingest`); refuse the
   three malformed cases before dispatch.
4. Thread the `Route` into each verb's request block and each ingest's
   receipt; golden-test one request block and one receipt per verb.
5. Add the `oracle-routing` consent category to `ahoy install`, mirroring the
   status-line category; test the prompt order, `--yes`, and `optional_skipped`.
6. Render the board line.

Each step carries a test watched failing first; the whole is one PR or a
short stack, the spec closed in the change that lands step 6.

## How the Acceptance Criteria are satisfied

1. Empty `Connections` and no files → every `Resolve` returns the bundled
   row with `Connection: harness`, tier `host-decides` only because nothing is
   accepted: with no machine or repo file the bundled proposal is *not*
   applied, so `Resolve` reports `Source: none` and the harness leg at
   `host-decides`; the request block carries the contract's ceiling.
2. The `oracle-routing` consent category renders and writes on yes, and only
   on yes; declining writes nothing and the next install offers again.
3. A `Connections` that `Serves(tier)` returns a provider → `Route.Connection`
   is that provider and the verb dispatches to it, not to the host.
4. `Serves` returns false → `Route.Connection: harness`, `Fallback` set with
   the reason; the verb prints one stderr line from it before dispatch and the
   receipt carries all three connection fields.
5. The receipt's `route` block carries `tier_asked`, `connection_used` and
   `model_reported` verbatim from the payload.
6. Precedence is repo → machine → bundled in `Resolve`; the board renders all
   three with the winner marked.
7. `--route` is layer zero in `Layered`; a second flag for another agent
   merges; the receipt's `override` field carries the flag verbatim; the three
   refusals fire in flag parsing.
8. Settings merge connection → row → flag in `Resolve`; the receipt's
   `settings_sent` records the merged map; an unaccepted key is refused by the
   adapter's `Connection` before dispatch (the adapter intent's contract).
9. The store reader reports and skips an orphan row.

## Tests

- `internal/core/oracle`: enum refusals; proposal-roster bijection; `Resolve`
  precedence table (all 4-layer permutations for one agent); fallback reasons;
  settings merge order; ceiling clamp.
- Store readers: guarded-read refusals, schema_version, orphan row.
- CLI: `--route` parse and the three refusals; one golden request block and
  one golden receipt per delegating verb.
- Ahoy: consent order (`…, status-line, oracle-routing, user-state, …`),
  `--yes` → `optional_skipped`, decline writes nothing.
- Board: one agent with four disagreeing layers renders the winner marked.

## Out of scope

- Requiring `model` / `agents_used` on payloads — `itd-2609180517121254`.
- Any real `Connections` implementation, provider configuration, reachability
  probing, and the set of settings a provider accepts —
  `itd-2609081951381895`, `iss-2609081951416843`.
- Writing rows from measurements — itd-17.
- Enforcing a fan-out bound at ingest.
- A lint rule over the routing files.

## Progress

Part 1 (branch `feat/layered-config-resolver`, autonomous run A, 2026-09-25)
lands Approach steps 1, 2 and 6 and leaves the spec open.

**The layered resolver** has one home, `internal/core/layered`, and every
consumer reads through it. Both file families are recorded in
`.abcd/work/DECISIONS.md` (2026-09-25):

- `layered.Config`: `.abcd/config.json` and `~/.abcd/config.json`, holding
  `pace.*`, `oracle.review`, `roles.<role>.runner` and `match.threshold`;
- `layered.OracleRouting`: this spec's two routing files.

Its API:

- `Load(File, Roots{Repo, Home}) (*Stack, error)`: guarded reads, loud on any
  fault; an absent file is an absent layer.
- `(*Stack).SetFlag(key, v, origin)`: the flag layer, with the flag text kept
  verbatim as the value's origin.
- `(*Stack).Claim(namespace, keys...)`: refuses an unknown key under a
  claimed namespace; `""` claims the top level and `*` matches an open segment.
- `Get[T](s, key, bundled, check) (Value[T]{V, Layer, Origin}, error)`: strict
  decode; a bad winning value refuses and never falls through.
- `(*Stack).Lookup(key) []Found{Layer, Origin, Raw}`: every layer, highest
  first.
- `(*Stack).Members(layer, key)`, `(*Stack).Present(layer)` and `Decode[T](raw)`.
- `Layer`: `None`, `Bundled`, `Machine`, `Repo` and `Flag`.

**Delivered here, each with tests:**

- AC 1, the resolution half: `Resolve` gives `none`, the harness at
  `host-decides` under the contract ceiling, and consults no connection.
- AC 3 and AC 4, the resolution halves: a provider takes the step, or the step
  falls back to the harness with a reason.
- AC 6 in full: repo over machine over bundled, and the bare board's `oracle:`
  lines.
- AC 7, the core half: `ParseRoutes` and `Apply`, with the five refusals.
- AC 8, the merge half: connection, then row, then flag.
- AC 9 in full: an orphan row is reported and skipped.

Also delivered: the tier enum, the bundled proposal and its roster test, and the
ceiling clamp.

**Two rulings this part took** (DECISIONS, 2026-09-25):

- The bundled proposal applies only once a repository or machine table exists,
  and a `--route` accepts nothing.
- Every agent's fan-out ceiling is 1, because no contract declares one.

**What remains:**

- Step 3, `--route` on the delegating verbs through one cobra helper (AC 7 at
  the surface).
- Step 4, the request block's `routing:` section and the receipt's `route:`
  block. These give the stderr fallback line of AC 4 and the receipts of AC 4,
  5, 7 and 8. Steps 3 and 4 land together, because a flag that changes no
  request would half-work.
- Step 5, the `oracle-routing` consent category in `ahoy install` (AC 2).
- AC 10 (escalation on a failed fix round) and AC 11 (the provider allowlist).
  They follow the implement loop and the API adapter.

The spec closes in the change that lands the last of steps 3 to 5.

Part 2 (branch `feat/model-tier-routes`, autonomous run A, 2026-09-25) lands
Approach steps 3, 4 and 5 and closes this spec with a remainder.

**Delivered here, each with tests:**

- AC 2 in full: `ahoy install` asks the `oracle-routing` category after
  `status-line`. It renders the proposal as a table and writes
  `~/.abcd/oracle-routing.json` (owner-only) on consent. A separate question
  offers `.abcd/config/oracle-routing.json`, written only on its own consent.
  `--yes` writes neither and names both under `optional_skipped`. A decline is
  not recorded, and uninstall leaves both files.
- AC 4, the surface half: the stderr line before the step, and the receipt's
  three connection fields and reason.
- AC 5 in full: the receipt's `route` block carries `tier_asked`,
  `connection_used` and `model_reported` side by side. `model_reported` is the
  payload's own `model`, or a reading's `instrument.model`.
- AC 7, the surface half: `--route` on the delegating verbs through one helper.
  Its refusals exit 2 before anything is written, and the receipt carries the
  override verbatim.
- AC 8, the receipt half: `settings_sent` records the merged map.

**Where the blocks land:**

- The request block is a `routing` member on `intent audit <itd-N>` and on a
  ready `launch ship` cut. For the audit it is also a `## Routing` section in
  the request document, outside the hashed prompt.
- The receipt is a `route` member on every ingest's result.

**Shape decisions** (recorded in `.abcd/work/DECISIONS.md`, 2026-09-25):

- The verbs are seven, not eight. `memory ingest` has no roster agent: its
  distiller is the host session.
- The receipt is the ingest's result, not a new durable field.
- The repository layer is read from `layered.RootsFor`, the rules loader's root.
- Settings are bounded and refused unclean where they are read. The limits are
  16 per row, 256 bytes per value, no hidden runes in a string value, and a
  `--route` text of at most 1024 bytes.

**What remains** goes to the remainder spec this close mints:

- AC 10: one tier up after a failed fix round, recorded in the state file. It
  needs the implement loop's state file.
- AC 11: a provider route only on that provider's allowlist. It needs the API
  adapter (itd-2609081951381895) and adr-2609221009491186's allowlist.
