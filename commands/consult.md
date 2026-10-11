---
name: consult
description: Consult the local sources corpus (the user-level home's sources store, default ~/.abcd.noindex/sources) and record source→decision provenance in its append-only ledger. Use when the user says "consult sources", "check the corpus", "what do my sources say", or when a design/research decision arises where prior literature or private working material plausibly matters. Confidential sources are NEVER cited or named in public artifacts.
block: people
---

# Consult sources

A local-only corpus at `~/.abcd.noindex/sources/` holds source documents (working
papers, private-repo notes, PDFs, books) the agent may **consult** but must
never **cite** publicly. Metadata lives in `sources.json` (CSL-JSON; the
`custom` block carries `confidential`, `permission_status`, `keywords`,
`aliases`, `ban_authors`). Every write goes through the `abcd source` verbs
(`/abcd:source`); reading is plain search. First run
`"${CLAUDE_PLUGIN_ROOT}/abcd" source --json`: exit 3 means there is no corpus —
say so and stop, because this command never creates it.

## Hard rule (overrides convenience, always)

For any entry with `custom.confidential: true`: never write its title, author
names, aliases, or any identifying string into anything tracked by git or sent
anywhere external — commits, commit messages, PR/issue text, docs, code
comments, published artifacts, pasted output. This covers **identifying
paraphrase** too: do not describe a confidential source so specifically that a
reader could identify it ("a forthcoming paper showing X beats Y on Z", a
private repo's distinctive architecture) — the mechanical guards catch literal
strings only; this rule is the paraphrase layer. Refer generically ("a working
paper on X", "prior private work"). The ledger holds the real reference;
citation is the user's manual decision, made when permission exists (public
citation requires BOTH the source's permission_status AND the ledger line's
cited_publicly flag).

Conversation with the user is fine — discuss confidential sources freely there.

## Consult

1. Search the corpus: `grep -ril "<term>" ~/.abcd.noindex/sources/confidential/
   ~/.abcd.noindex/sources/public/` — try keywords, author surnames, and CSL keys.
   The path IS the classification: any hit under `confidential/` falls under
   the hard rule above. Each source is a folder (`<class>/<key>/`) holding
   `original.<ext>`, `text.md`, and any summaries/notes.
2. Read matched files freely for conversation; the folder's class governs
   what may leave it.
3. If nothing relevant surfaces, say so — do not pad.

To add a source, use `/abcd:ingest`.

## Record influence (the ledger)

Whenever a source **meaningfully influences a decision** (supports it,
contradicts it, supplies a method, or shapes background understanding — not
mere incidental reading), record ONE line:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" source ledger --decision "<DECISIONS.md date | ADR id | intent id | free text>" \
  --claim "<what was decided/claimed>" --source <CSL key> \
  --influence supports|contradicts|method|background \
  [--locator "<pp./§>"] [--used-in <repo-relative path>]... --json
```

The verb appends the line to this repository's ledger with
`cited_publicly: false` and commits it in the corpus. `--used-in` makes
acknowledgment machine-readable in both directions: an idea is traced to its
source even when the consuming document only paraphrases (public sources) or
must stay silent (confidential sources). Fill it whenever the influence landed
in an identifiable document, not just a conversation.

The ledger is append-only: a correction is a new line (`--corrects <N>`).
**Never run `ledger --flip`** — flipping `cited_publicly` is the user's act,
and the verb refuses it anyway unless the source is public and citable.

Always tell the user in conversation which key was recorded against which
decision, and the line number, so they can decide about citing.

## Guard wiring

- The repository's committed pre-commit guard runs
  `abcd source sync-banlist --refresh` on every commit (in a managed
  repository, once the clone opts in with
  `git config --local abcd.sourcesBinary /absolute/path/to/abcd`), which
  regenerates a fenced block of confidential titles and aliases in the untracked
  `.abcd/.work.local/private-names.txt` when that store already exists;
  leakage is then blocked mechanically, not just by this command's rule. The
  first sync in a repository, and the one after adding or declassifying a
  source, is `"${CLAUDE_PLUGIN_ROOT}/abcd" source sync-banlist`, run by hand so
  the block is current before the next commit.
- Before any document that drew on confidential material is committed, posted,
  or otherwise shared, run `"${CLAUDE_PLUGIN_ROOT}/abcd" source cite-check <file>`
  (exit 1 = a confidential identifier is present; its report names only the
  CSL key, the field and the position, so the report itself is safe to relay).
  It runs the same matcher as the guard, and it covers literal strings only.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.
