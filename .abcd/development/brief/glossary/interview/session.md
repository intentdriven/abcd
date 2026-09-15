---
term: session
bounded_context: interview
definition: One complete interactive exchange between a human and the abcd grill sub-verb, spanning all rounds of Socratic questioning through PRD synthesis for a single intent or brief section.
aliases: ["grill session", "interview session"]
forbidden_synonyms: ["user session", "login session", "auth session", "HTTP session"]
status: stable
introduced_in: phase-1
starts_when: The human invokes /abcd:intent grill with a target intent or brief section.
ends_when: The PRD synthesis phase completes and the PRD file is written to disk.
not_to_be_confused_with: null
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# session (interview)

A **session** is the unit of work for the `/abcd:intent grill` sub-verb. It begins when the
human runs the command and ends when the PRD is written. Within a session, the oracle asks
Socratic questions (up to 12 total, 3 per round), the human answers, the oracle sharpens terms
and captures glossary updates inline, then silently synthesises a PRD.

This term is bounded to the **interview** context. Do not import it into HTTP/auth contexts or
use it generically — the abcd interview session is a distinct concept with its own lifecycle.

## When to use

Use "session" when describing the lifecycle of a single `/abcd:intent grill` invocation. A
session is atomic — it either completes (PRD written) or is abandoned (no PRD artifact).

## When NOT to use

**Never** use "user session", "login session", "auth session", or "HTTP session" to mean this
concept. Those terms carry authentication and web-protocol connotations that are orthogonal to
the grill interview. Within abcd documentation, unqualified "session" in the interview context
always means a grill session.

## Lifecycle

| Phase | Condition |
|-------|-----------|
| Starts when | `/abcd:intent grill <target>` is invoked by the human |
| Ends when | PRD synthesis completes and `prd.md` is written to disk |

## Examples

- "In this session, Carol worked through three rounds of Socratic questioning for `itd-27`."
- "A session that is abandoned before PRD synthesis produces no artefact."

## Related terms

- [embark](embark.md) — the opening move that initiates a session
- [intent](../core/intent.md) — the primary target of a grill session
