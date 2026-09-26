---
name: docs
description: "Keep the citation baseline that `abcd lint docs` enforces offline: Writes nothing but that baseline; refuses an unknown sub-verb."
argument-hint: "[cite refresh | cite confirm <url>...]"
block: agents
---

# `/abcd:docs` documentation currency and citations

One job lives here: `cite` maintains the committed citation baseline that the
docs lint enforces offline. The lint itself, which grades the documentation and
writes nothing, is `/abcd:lint docs` — run `abcd lint docs`.

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
