---
id: dcn-2607110000000003
slug: itd-3-rules-loader-hook-is-go-not-python
date: 2026-07-11
---
2026-07-11 — itd-3 rules-loader hook is **Go**, not Python. abcd is Go-only, so
the `UserPromptSubmit` router is a Go subcommand invoked by `hooks/hooks.json` —
the intent's `hooks/prompt_router_hook.py` is a stale pre-Go-rebuild detail and
is superseded. No Python is added for the loader.
