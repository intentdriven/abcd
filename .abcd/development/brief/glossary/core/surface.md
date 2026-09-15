---
term: surface
bounded_context: core
definition: A verb's front door — the markdown command file under commands/ plus the transport package under internal/surface/ that reaches the core. "A surface chapter" is the brief's design record for one such front door, and "a rendered surface" is a public text held to the repository's identity block; both are qualified.
aliases: ["front door", "command surface"]
forbidden_synonyms: ["UI", "interface", "endpoint", "API"]
status: stable
introduced_in: phase-1
starts_when: null
ends_when: null
not_to_be_confused_with: core/transport
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# surface

A **surface** is where a user reaches a verb. Every user-facing command has two halves: the
markdown file under [`commands/`](../../../../../commands) that the harness registers as
`/abcd:<verb>`, and the transport package under `internal/surface/` that formats what the core
returns. The core never writes to stdout and never knows a transport; a surface is the only
thing that does. `internal/surface/cli` is the shipped door; the MCP door is a design target
that the transport-agnostic core exists to admit.

## Senses

| Sense | The one spelling | Where it lives |
|---|---|---|
| A verb's front door: the command file and the transport package behind it | **a surface** | [`commands/`](../../../../../commands), `internal/surface/` |
| The brief's design record for one front door — purpose, flow, acceptance criteria | **a surface chapter** | [`04-surfaces/`](../../04-surfaces/README.md), numbered `NN-<verb>.md` |
| A public text held to the repository's canonical identity block | **a rendered surface** | [`04-surfaces/19-identity.md`](../../04-surfaces/19-identity.md) |

**The registry is what binds the first two.** [`04-surfaces/README.md`](../../04-surfaces/README.md)
carries one row per user-facing command, and the `surface_coverage` record-lint rule asserts the
row and the front door agree in both directions: a `shipped` row must have a
`commands/<name>.md`, a `staged` row must not, and a real front door with no row fails the
gate. So "the surface set" means the registry's rows, and an operator-internal verb is
deliberately outside it.

**A fourth use is this glossary's own.** The glossary README calls the committed terms "a
deliberate frame surface" — the framing's machine-visible fingerprint. That is a metaphor on
the front-door sense, and it is written in full wherever it appears.

**Do not confuse the noun with the verb.** The record also uses "surface" as a verb — a review
or an interview *surfaces* a finding. The verb takes an object and needs no qualifier; the noun
without one always means the front door.

## When to use

Use "a surface" for the front door a user types at. Use "a surface chapter" for the brief page
that specifies one. Use "a rendered surface" only in the identity sense — a README, a site page,
a plugin description measured against the identity block.

## When NOT to use

Do not use "surface" for the [transport](transport.md), which carries context to an
[oracle](oracle.md) rather than to a user. Do not use it for an operator-internal verb: those
carry no command file and no registry row by design.

## Examples

- "Wired or it isn't done: every verb is reachable from both the CLI and the plugin markdown surface."
- "The surface chapter for `/abcd:reading` is `04-surfaces/23-reading.md`."
- "`abcd identity` reports each rendered surface's verdict and writes nothing."

## Related terms

- [transport](transport.md) — how context reaches an oracle, not how a user reaches a verb
- [record](record.md) — what the surfaces mint
- [reading-position](reading-position.md) — a position is not a surface; the surface is `/abcd:reading`
