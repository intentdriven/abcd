---
id: spc-70
slug: abcd-s-presence-should-be-visible-in-the-host-harness-s-stat
intent: itd-200
origin: researcher-authored
production_mode: dictated-and-formatted
---
# abcd-s-presence-should-be-visible-in-the-host-harness-s-stat

## Summary

spc-70 delivers itd-200: in an abcd-managed repository the host's status
line becomes abcd's own row, led by one badge in three states (managed and
nobody waiting, waiting on the facilitator, waiting on the product thinker),
followed by the repository name, the branch, the model, the context
percentage, the five-hour and seven-day usage percentages, and the record's
intent and issue counts. Outside a managed repository the user's previous
status line runs untouched. The state behind the badge is a stored mode with
two writers, the agent and the human, read by the status render and by the
bare `/abcd` board alike. Where the host has no status surface, one line at
the stop names whose answer is owed. `ahoy install` offers the line, explains
it, takes the element configuration, and writes the wiring only on consent.

## Scope

- **One render in core.** `internal/core/statusline` composes the row from
  three inputs: the harness payload on stdin, the repository, and the record.
  It returns a structured row (ordered elements, each with a key, a rendered
  form and a plain form) that the CLI prints and the `/abcd` board reuses for
  its one-line state. No front door invents words.
- **The mode store.** The waiting-on state lives in the repository's local
  tier, `.abcd/.work.local/mode`, one line, one of `managed`, `facilitator`,
  `product-thinker`. Per worktree, gitignored, absent means `managed`. Only a
  managed repository has the tier, which is what keeps the state managed-only
  by construction.
- **Two writers, one verb.** `abcd mode` prints the state; `abcd mode
  <state>` sets it. The agent calls it when it stops for a verdict, naming the
  addressee; the human calls it to say which hat they wear. The plugin surface
  page documents both uses. Where the host has no status surface, the set
  form also prints the one-line notice naming whose answer is owed, once,
  because the verb call is the stop.
- **The status verb.** `abcd statusline` reads the harness's JSON payload on
  stdin and prints the row. When the current directory is not a managed
  repository it runs the user's previous status command, recorded at install
  time, with the same stdin, and prints its output verbatim; when none was
  recorded it prints nothing. This is how one harness-wide setting yields
  abcd's line in managed repositories and the user's line everywhere else.
- **Elements after the badge**, in the fixed order the intent names. Each is
  on by default and switchable in the user-level setting. Payload-sourced
  elements (model, context, five-hour, seven-day) render only when the field
  is present; an absent field drops the element with no placeholder. The
  branch is read from the repository. The counts are open issues and intents
  not yet shipped (drafts plus planned), taken as FOLDER COUNTS: the files
  named `iss-*.md` in the ledger's `open/`, and the files named `itd-*.md` in
  the intent store's `drafts/` plus `planned/`, one directory listing each,
  nothing parsed. Folder membership is the record's own status signal, so
  this is the number the board's readers give except for a record those
  readers would skip as unreadable — and the harness runs the composition on
  every refresh, where the board's readers, which load and parse every
  record, measured 0.96 s over 20,001 open records. (Written as "the same
  readers the board uses"; changed 2026-09-15 on that measurement.)
- **The user-level setting.** `~/.abcd/statusline.json`: the off switch, the
  per-element switches, the presence badge's foreground and background, and
  the recorded previous status command. The defaults ship with the binary and
  are written on first install. A presence pair below the contrast bar is
  refused at read time with the measured ratio and the default renders.
- **Install-time wiring.** `ahoy install` on the first harness detects the
  harness's status-line setting, offers the line with a one-paragraph reason,
  takes the element switches, records the previous command, writes the
  user-level setting, and points the harness at `abcd statusline`. Declining
  writes nothing. `ahoy uninstall` restores the recorded previous command.
- **The board.** The bare `/abcd` render gains the same one-line state, from
  the same render, so a host with no status surface still answers on demand.
- **Contrast.** The role badges' pairs are fixed constants held to the same
  bar; the presence pair is checked when the setting is read.

## Approach

The render is pure: payload in, row out, no network, no writes. The only
state it reads beyond its inputs is the mode file and the user-level setting.
The status verb is the harness-facing shell around it; the board is the
second consumer. Truncation safety comes from order alone: the badge is
element one and every later element is optional, so a narrow host cuts from
the right.

The previous-command fallback is what makes "managed repositories only"
true under a single harness-wide setting. It is recorded once at install
from the harness's own configuration, never guessed, and it is run with the
same payload the harness supplied, so the user's line is unchanged in every
repository abcd does not manage.

The mode verb is deliberately small: three states, a print form and a set
form. The one-line notice on set is the whole of the no-status-surface
fallback, and it fires once per set, never per prompt.

## How the Acceptance Criteria are satisfied

- **ac-1 (managed, nobody waiting).** Mode absent or `managed` renders the
  presence badge first; the render places nothing before it.
- **ac-2, ac-3 (stop parked on the product thinker or facilitator).** The
  agent runs `abcd mode product-thinker` or `abcd mode facilitator`; the next
  render reads the file; the board reads the same file.
- **ac-4 (hat set by hand).** The same verb, run by the human; the agent's
  next read of the mode returns it.
- **ac-5 (unmanaged repository).** No managed marker means the status verb
  runs the recorded previous command and prints nothing of its own; no
  setting is written.
- **ac-6 (switched off).** The off switch in the user-level setting makes the
  status verb behave as in ac-5 everywhere.
- **ac-7 (no status surface).** The set form of the mode verb prints one
  line naming the addressee; the board renders the state on demand.
- **ac-8 (install asks).** The `ahoy install` step: offer, reason, element
  switches, recorded previous command, wiring on consent, nothing on decline.
- **ac-9 (contrast bar).** The setting reader computes the ratio of the
  configured pair; below the bar it refuses with the ratio and substitutes
  the default.
- **ac-10 (truncation).** Badge first; the test renders a row and asserts
  that every prefix of it still begins with the badge.
- **ac-11 (full row).** A payload fixture carrying every field renders the
  nine elements in the named order with one separator. (The count was written
  as twelve and the list it refers to names nine: badge, repository, branch,
  model, context, five-hour, seven-day, intent count, issue count. Corrected
  2026-09-15 when the render was built against it.)
- **ac-12 (absent field).** The same fixture minus one field renders one
  element fewer and no placeholder.

## Tests

- Render tests over payload fixtures: full row, each field absent, each
  element switched off, each mode state, unmanaged directory.
- Mode verb tests: print and set, invalid state refused, the one-line notice
  on set and its absence on print.
- Setting tests: defaults written on first install, off switch, contrast
  refusal with the measured ratio, previous command recorded and restored.
- `ahoy install` tests on a fixture harness config: offer and decline write
  nothing; consent writes the setting and the wiring; uninstall restores.
- Board test: the one-line state matches the render's plain form.
- Zero-network harness: the status and mode verbs make no request.

## Out of scope

- Any second harness; guard health, peer sessions and other candidate
  elements; update-available signalling; the role-adapted communication path.
