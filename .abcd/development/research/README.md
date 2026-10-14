# Research

Investigations that inform the design without being decisions themselves —
surveys of prior art, prompt R&D, throwaway prototypes, and the reference material
they draw on. A note here may seed an intent or inform an ADR, but the intent and
the ADR are where the forward-looking record and the ratified choice live.

| Path | What it holds |
|---|---|
| [`notes/`](notes) | Dated and spec-scoped research notes — SOTA surveys, audit syntheses, investigation write-ups |
| [`ideas/`](ideas) | Public ideate verdicts, one folder per repository root commit (`ideas/<root-sha>/<date>-ideate-<slug>.md`); a private verdict stays in `~/.abcd.noindex/ideas/<root-sha>/` (iss-2610090740139804) |
| [`prompting/`](prompting) | Prompt R&D — the general best-practices baseline and per-agent prompt drafts |
| [`data/`](data) | Measured corpora kept as data — detector output that grounds a note or an intent, and would otherwise survive only in ephemeral storage |
| [`abcdev-site/`](abcdev-site) | The abcdev.app website work cluster — the investigation plan, the clickable prototype, the build prompt (adr-47/adr-48) |
| `_references.md` | Registry of external sources cited across the research |
| `legacy-harvest.md` | What carried over from the v0 scaffolding into the plugin |
| `related-work.md` | Prior art and comparable tools abcd is positioned against |

For the wider record this sits in, see the [development map](../README.md).
