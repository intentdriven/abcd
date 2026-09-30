---
name: docs
description: "Keep the citation baseline `abcd lint docs` enforces, and judge the brief against the binary: Writes nothing bare; refuses an unknown sub-verb."
argument-hint: "[cite refresh | cite confirm <url>... | fidelity [--report|--apply|--autonomous] | fidelity record --verdict-json <file|->]"
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
- `queue` — sources that refuse automated fetchers, and sources whose redirect
  chain leaves https for plaintext http, which the fetcher does not follow.
  Present these as a checklist
  with the `sites` that cite each one, and tell the user to open each link and
  confirm it.
- `dropped` — receipts removed because the docs no longer cite those addresses.

A `blocked` source is **not** recorded as broken and gets no invented entry: a
403 says the fetcher may not look, which is a different fact from the citation
being dead, and so is a redirect off https. Never suggest editing the baseline
by hand to clear one.

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

Set `abcd mode facilitator` first, then confirm on the technical facilitator's
word that they checked. An agent must never run `confirm` on its own initiative
to clear a red gate.

## `fidelity` — the brief describes every surface that ships

`spec close` and `launch ship` run this gate themselves; run it directly to see
what they will say:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" docs fidelity --json
```

Report `verdict.uncovered` (verbs, sub-verbs and agents no brief chapter names),
`verdict.review.status` (`match`, `none`, `stale`, `invalid`, `hold`,
`inconclusive`) and every line of `verdict.reasons`. Exit 1 is a refusal. Use
`--report` after a task: it states the same findings and refuses nothing.

When the review is `none` or `stale`, run the docs review for HEAD. Read the
brief chapters under `.abcd/development/brief/04-surfaces/` that describe what
the change delivered, check each sentence against the code, and compose:

```json
{"verificationResult": "PROMOTE", "judgeModel": "<pinned model id>", "tier": "full", "failing": []}
```

`judgeModel` is the pinned id of the model that judged, with its version or
date; the record verb refuses a bare family name and any id naming `latest`.

A sentence you **confirmed** false goes in `failing` with `"doc": "brief"`, its
`chapter` file name, the `sentence` verbatim from one line of the chapter (at
most 2048 bytes; quote the part on one line when it wraps), the `evidence`
(file:line), a `disposition`, and, where you can, a drafted `replacement`; the
verdict is then `HOLD`, since the record verb refuses a `PROMOTE` naming one. A
false sentence in the public docs takes `"doc": "public"`: it is reported and
never refuses. Use `INCONCLUSIVE` when you cannot judge; never
`PROMOTE` from absent evidence. Save it:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" docs fidelity record --verdict-json verdict.json
```

The binary labels the receipt with HEAD; a later commit makes it stale. When a
HOLD carries drafted replacements, `docs fidelity --apply` writes them into the
brief and flags each in `.abcd/work/brief-review-flags.json`: tell the user
which sentences changed, because the brief now carries sentences they did not
write until they read them. `--autonomous` does the same in an unattended run
and adds the reviewer's `request` to the JSON.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
