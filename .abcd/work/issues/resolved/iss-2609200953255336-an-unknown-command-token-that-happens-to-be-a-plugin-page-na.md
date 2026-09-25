---
schema_version: 1
id: "iss-2609200953255336"
slug: "an-unknown-command-token-that-happens-to-be-a-plugin-page-na"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "Gropius intent lifecycle run, session gropiusllm-97, relayed to abcd-17 on 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
resolution: "The stale-usage note no longer reads the dispatcher page abcd.md as a verb the binary lacks: abcd abcd <id> now says the token is the binary itself and names abcd <record-id>."
impact: fix
resolved_by:
  commit: "a4233ea9e87245d6f4baae659d0240499e81bd60"
---

An unknown command token that happens to be a plugin page name makes an up-to-date binary call itself stale. abcd abcd <record-id> (a misreading of the plugin page /abcd:abcd, whose invocation is the bare abcd <record-id>) prints "unknown command \"abcd\"" and then "this binary predates the abcd command the plugin surface it was provisioned from documents — this PATH copy is stale; run abcd update". Reproduced at v0.9.0 (4ae6f221) with a fresh build: the token abcd triggers the line, a made-up token does not, so the unknown-command path consults the plugin surface's page list, finds a page named abcd, and infers a verb the binary lacks. The page documents the dispatcher, not a verb; the binary has it, and the diagnostic is false on a pinned, up-to-date v0.9.0 (the reporting session's abcd version: pinned, vintage v0.9.0, up to date). A false staleness claim is the shape loud-staging forbids in the other direction: it sends the operator to update a binary that needs no update. Relayed from the Gropius session gropiusllm-97 on 2026-09-20. Wanted: the page-name check excludes the plugin's own dispatcher page (or checks the binary's command tree before claiming it predates anything), and for the token abcd specifically the refusal says did you mean abcd <record-id>.

## Grounds

- pursued: an up-to-date binary never calls itself stale for a page that documents no verb; a stale note on abcd abcd, or a verb added to pagesWithNoVerb that the binary registers, would show it wrong
