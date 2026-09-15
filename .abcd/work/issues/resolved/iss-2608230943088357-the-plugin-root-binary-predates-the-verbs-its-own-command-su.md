---
schema_version: 1
id: "iss-2608230943088357"
slug: "the-plugin-root-binary-predates-the-verbs-its-own-command-su"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "abcd-update-invocation-2026-08-23"
found_at: "commands/update.md"
resolution: "An unknown command or flag now carries a second line when the binary can prove from disk that it is stale: the command surface at the resolved plugin root documents the refused verb or flag, or the disk-only vintage says the binary is behind its checkout tip or differs from the pinned release. The remedy follows where the binary sits (make build, a plugin update, or abcd update); cobra's own line, the exit code and the JSON envelope are unchanged, and a typo with no evidence behind it stays verbatim."
impact: fix
resolved_by:
  commit: "d48b08e5cbe2fc0bf6fb49f80729907ae7173199"
---

The plugin-root binary predates the verbs its own command surface documents, and the failure is an unknown-command error rather than a named refusal. Verified 2026-08-23: the commands/update.md page instructs the reader to run the plugin-root binary first, and that binary on this machine is v0.6.1, whose help lists no update verb at all. Running it as documented gives 'abcd: unknown flag: --yes', and running the bare verb gives 'abcd: unknown command "update" for "abcd"'. The page's own resolution ladder recovers the situation, since a source checkout falls through to go run ./cmd/abcd, but only for a reader who knows to keep going: the first rung produces an error that looks like a malformed invocation rather than a stale install. This refines iss-228, which records the same root cause with a different consequence -- a plugin-root binary sitting a month stale so ahoy's first resolution rung reported pre-iss-171 gaps with no staleness signal. That entry is about wrong gap reporting; this is about a documented command surface pointing at a binary that cannot serve it. The general shape is that the plugin surface and the plugin binary ship from one release but a repository checkout can be arbitrarily far ahead of the installed plugin, so every commands/*.md page documenting a verb newer than the reader's plugin has this failure mode, and update is merely the first to be noticed. Two directions, neither adopted: have the command pages degrade legibly when the resolved binary lacks the verb, by checking for it before invoking; or have the binary answer an unknown verb by naming its own version and vintage, so the error says stale rather than wrong. Worth deciding whether this is inherent to plugin-versus-checkout skew and therefore only a documentation problem, or whether the unknown-command path should carry install context.

## Grounds

- pursued: the command page beside the plugin root is evidence the binary itself can read, so an unknown verb whose page exists is expected to read as a stale install with its way out rather than a malformed invocation; a reader who still has to fall through the resolution ladder blind after a plugin update would show this wrong.
