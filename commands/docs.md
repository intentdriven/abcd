---
name: docs
description: "Lint the documentation for currency and keep its citation baseline: Writes nothing but that baseline; refuses an unknown sub-verb."
argument-hint: "[lint | cite refresh | cite confirm <url>...]"
block: agents
---

# `/abcd:docs` documentation currency and citations

Two jobs live here. `lint` grades the documentation and writes nothing. `cite`
maintains the committed citation baseline that the lint then enforces offline.

## `lint` — the currency gate (zero writes, zero network)

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" docs lint --json
```

Then summarise the JSON for the user:

- `nothing_checked` and `warning` — when `nothing_checked` is `true` the lint
  checked nothing and still exited 0: relay `warning` (it says why), tell the
  user nothing was checked, never that the docs are clean, and point them at
  `.abcd/docs-lint.json`. The same warning is printed on stderr.
- `checks` — how many checks the configuration armed (banned tokens plus enabled
  rules). `0` means no rule ran.
- `documents` — how many markdown documents the configured roots hold for the
  per-document rules. `0` means those rules read nothing: the roots are empty or
  hold no markdown.
- `blockers` — how many blocker findings exist; any blocker fails the gate.
- `findings` — for each, its `File`, `Line`, `RuleID`, `Severity`, and
  `Message`; group them so the user sees what to fix.

The lint enforces present-tense docs: unambiguous change-narration (`previously`,
`formerly`, `renamed from`, `has been replaced`, `we switched`, `to be
implemented`) blocks, while phrases that also describe present state
(`deprecated`, `no longer`, `migrated from`) warn advisorily rather than block.
It also checks that relative links resolve and that no stray markdown sits at the
repo root (it belongs under `docs/`). Point the user at the offending file and
line for each finding, and note whether it is a blocker or a warning.

Where a repo arms them, the citation rules add: footnote markers and definitions
in bijection, every crosswalk table row carrying a footnote, well-formed URLs and
DOIs, refused source domains, and the committed baseline — no cited URL without a
receipt, none recorded broken, none whose recorded final address has drifted from
what the page cites, and a staleness warning past 180 days. Every one of these
reads committed files only; nothing dials out.

`--release-gate` runs the same lint with one difference: a citation past the
365-day threshold blocks instead of warning. It is for release machinery only —
an ordinary commit is never blocked by the calendar.

If `nothing_checked` is `false` and `blockers` is zero, the docs are
currency-clean.

## `cite refresh` — the one verb that reaches the network

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" docs cite refresh --json
```

This fetches every cited URL once, bounded and without retries, and rewrites the
baseline. Report from the JSON:

- `cited`, `fetched`, `preserved` — how much was checked, and how many current
  human-verified receipts were left alone.
- `outcomes` — each URL's `status` (`ok`, `broken`, `blocked`, `preserved`),
  its `final_url`, and the `detail`. List every `broken` one: those are dead
  citations the gate will block on.
- `queue` — sources that refuse automated fetchers. Present these as a checklist
  with the `sites` that cite each one, and tell the user to open each link and
  confirm it.
- `dropped` — receipts removed because the docs no longer cite those addresses.

A `blocked` source is **not** recorded as broken and gets no invented entry: a
403 says the fetcher may not look, which is a different fact from the citation
being dead. Never suggest editing the baseline by hand to clear one.

## `cite confirm` — closing the manual queue

Once the user has opened a queued link and seen the document, record it:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" docs cite confirm <url> [<url>...]
```

For a batch, or to record a redirect the user followed, pass a receipt file
instead:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" docs cite confirm --receipt receipts.json
```

```json
{
  "schema_version": 1,
  "confirmed": [
    {"url": "https://example.org/a", "final_url": "https://example.org/a-v2", "verified_on": "2026-07-20"}
  ]
}
```

Only URLs the documentation actually cites can be confirmed. The receipt records
**that** a human verified the citation and **when** — never how. Do not ask the
user for their method, and never record one: the schema has no field for it and
loading rejects unknown keys.

Confirm on the user's word that they checked. An agent must never run `confirm`
on its own initiative to clear a red gate.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
