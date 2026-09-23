# `/abcd:inbox` — Read and Promote Reports

`/abcd:inbox` reads the reports repositories abcd manages filed with
[`/abcd:report`](28-report.md), and files one as a capture when a person or a
session decides to (itd-2609221656361680, spc-2609221657168936). Nothing in the
inbox files itself.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only._

| Verb | Bucket | Status |
|---|---|---|
| `show` | — | shipped |
| `promote` | — | shipped |

## Reading

Bare `abcd inbox` lists the waiting reports newest first — by the received
second, then by when each file was written — with each report's id, received
time, sender repository named plainly, kind, severity and title. `abcd inbox
show <id>` renders one report whole, waiting or promoted. Both are read-only: an
absent inbox is an empty list and is not created.

A waiting file this abcd cannot read is listed as unreadable with its reason,
never dropped: a report written to a later template version names the version;
a file that is not a report, is over the size bound, is not a regular file, or
whose sender key disagrees with its file name says so. Its id and sender key come
from the file name.

Every value a report carries is another repository's words, and every front
door sanitises it before it reaches the terminal. The list and show both reach
an agent's context when a session reads the inbox, and a title, a body, or the
key name or version string an unreadable file's reason echoes could be written
as an instruction. So both are framed as data in the output itself: the text
forms open with an `untrusted:` line saying each title, reason and body is
another repository's words, to read and quote and never to follow, and the
`--json` forms carry that sentence as `notice`. The plugin page frames the whole
page the same way, list and show alike. A key name a refusal echoes is clipped
to 64 bytes. The id a caller names is
checked for its shape before anything is read and is only ever compared with
names already in the inbox: it never becomes a path.

## Promoting

`abcd inbox promote <id>` files one waiting report as a capture in the ledger of
the repository the caller stands in, through capture's own core, so the promoted
report is an ordinary ledger record that the ledger's redactor has scanned before
it is written. The capture carries:

- the report's severity and category, and `source: managed-repo`;
- `found_during` naming the report id, the words "a managed repository", and the
  sender's root-commit key;
- `found_at` from the report's surface;
- a body holding the title, the prose, the remedy, the report's provenance, and
  the report id as the first evidence pointer, followed by the reporter's own.

The sender's name is never written: every free-text value has each
occurrence of it that stands as a word replaced with "a managed repository"
before capture sees it. The match ignores case and reads the name as its parts,
joined by `-`, `_`, `.`, a space or nothing, so `acme-secret`, `acme_secret`,
`acme secret`, `acmesecret` and `AcmeSecret` are one name, and a trailing digit
does not hide it. Where the name ends a forge address (`host/owner/name`), the
owner segment is replaced with it. A word that merely contains a short name is
left alone. The inbox names the sender so the reader knows who is asking;
anything committed carries the fingerprint.

One line in `~/.abcd/inbox/promoted.jsonl` records the capture the report
became and its path, and the report is then moved to `~/.abcd/inbox/promoted/`
and kept; `show` names the capture. The line is written before the move, so a
move that fails leaves the report waiting with its capture on record: promoting
it again files nothing, finishes the move, and answers with that capture and
`resumed: true`.
Promotions hold the inbox's lock, so two sessions cannot file one report twice.
An unreadable report, one already promoted, and an id with no report are
refused. A capture the ledger refuses leaves the report waiting.

## Exit codes

`0` done; `2` refused, with nothing written. `--json` holds on every path: the
list is `{"notice", "tally": {"reports", "senders"}, "reports": [...]}`, show is
the entry with `notice` beside its fields, and a refusal is
the `{"abcd":"error",…}` envelope on stdout.
