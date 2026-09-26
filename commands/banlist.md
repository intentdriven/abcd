---
name: banlist
description: "Render both banned-names layers: Writes nothing; refuses an unknown word without echoing it."
argument-hint: "[list --private|--public] | add --private|--public <key> <pattern> [--severity blocker|warn] [--successor <text>] | remove --private|--public <key>"
block: agents
---

# `/abcd:banlist` — banned names, two layers

Some names must not appear in what a repo publishes: a specific agent harness (so
the surface stays host-agnostic), a partner's product, a private project whose very
name is confidential — and, most sensitively, the user's own machine identifiers:
hostnames, device names, and the addresses or prefixes of their private network.

Enforcement splits by sensitivity, because a deterministic CI gate is the right
tool for a public banned name and the **wrong** place for a private one: the rule
would have to contain the very string it forbids.

| layer | store | enforced by | visibility |
|---|---|---|---|
| public | `.abcd/docs-lint.json` (the `banned_tokens` family) | `abcd lint docs` in CI, with a per-line escape | entries render in full |
| private | `.abcd/.work.local/private-names.txt` (gitignored) | the committed `.githooks/pre-commit` and `.githooks/pre-merge-commit` guards, on this machine only | entries render **by key only** |

A public entry reads the lint's `roots` and, beyond them, its `name_roots`:
every text file under those trees, not only markdown, so a name ban reaches the
whole public surface a repository declares there.

## Render both layers (bare)

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" banlist --json
```

Summarise the JSON: for `private`, its `present` flag and each entry's `key` —
**never** ask for or repeat a private pattern value, which the binary does not
emit; for `public`, each entry's `id`, `severity`, `pattern`, and whether it is
`managed` (verb-owned, id under `names/`) or hand-curated.

State the private layer's reach **as the binary states it**: relay the `reach`
sentence the render carries rather than paraphrasing it. Paraphrase drops the
second half, and the way it drops is by omission — "protects only machines that
have opted in" alone leaves a reader believing an opted-in machine is fully
covered. It is not: a hook sees the commits git asks it about, so a fast-forward
`git pull`, a rebase, a `git am`, a `git revert` or a cherry-pick bypasses it, as
does `--no-verify`, and that list is not exhaustive. When `private.present` is
false, say that the layer is inactive on this machine — an absent store checks
nothing, and silence must never look like protection.

Two private fields need reporting apart, because they need opposite responses.
`malformed_lines` are lines the guard's engine cannot use: the guard refuses **every**
commit until they are fixed. `inert_lines` are lines it accepts but reads differently
(a Perl-style escape, an inline flag group): the guard refuses nothing, and those
names are unguarded while the store looks healthy. Report each by line number only.
`keyed` reports the store's format; when it is false and there are entries, say that
the store is in the legacy whole-line format and that `add`/`remove` refuse until its
first line declares the keyed format.

`list` is the same render with an optional scope:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" banlist list --private --json      # or --public
```

## Add an entry

```bash
printf %s '<pattern>' | "${CLAUDE_PLUGIN_ROOT}/abcd" banlist add --private <key> - --json
"${CLAUDE_PLUGIN_ROOT}/abcd" banlist add --public <key> "<pattern>" --json
```

The layer is **never** guessed: an add with no layer flag, or with both, exits 2.
`<key>` is a stable, non-sensitive handle (`[A-Za-z0-9][A-Za-z0-9._/-]*`).

**For a private add, pass the pattern as `-` and pipe it on stdin**, as above — that
is the recommended form and the only one that keeps the value out of argv. A command
argument is world-readable in `/proc/<pid>/cmdline` for the life of the process, is
captured verbatim by process auditing, and lands in the shell's history file, so a
pattern typed as an argument has already leaked to three places the layer exists to
keep it out of. A pattern beginning with `-` can *only* be entered this way; the
verb withholds the token from any flag-parse error rather than echoing it.

A private `<pattern>` is a **POSIX extended regular expression, matched
case-insensitively** by the guard's `grep -iE`. `(?i)` is therefore never needed, and
Perl escapes such as `\d`, `\w` and `\b` are not available — the verb refuses them
rather than storing an entry that would match nothing. Machine identifiers —
hostnames, IPv4/IPv6 addresses, CIDR prefixes, MAC addresses, device names — are
ordinary private entries. A private add reports the key alone; **do not echo the
pattern back to the user**, which the binary does not emit either.

Two refusals are worth relaying verbatim rather than working around. If the store's
path is not gitignored the add is refused: the whole layer rests on that file being
untracked, so add the tier line the message names and re-run. If the store predates
the keyed format — no `# abcd-banlist: keyed` first line, and at least one entry —
`add` and `remove` refuse, because a keyed line written into it would change what
every other line means; the message names the one line the user adds by hand, after
which each existing whole-line pattern needs a key.

A public add takes `--severity` (`blocker`, the default, or `warn`) and
`--successor` (the replacement the finding cites; default "a generic term"), and
writes one entry into the committed config under the `names/` id namespace. Its
pattern is a **Go (RE2) regular expression**, because `abcd lint docs` is what
enforces the public layer; the entry is stored with the `(?i)` prefix so it matches
case-insensitively like every hand-curated entry. Report the entry `id` and remind
the user to commit it: the public layer gates everyone.

## Remove an entry

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" banlist remove --private <key> --json
"${CLAUDE_PLUGIN_ROOT}/abcd" banlist remove --public  <key> --json
```

A public removal is refused for a hand-curated entry (an id outside `names/`):
those are edited in the config by a human, in a reviewable commit. An unknown key
is refused rather than treated as a no-op.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
