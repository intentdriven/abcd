# `/abcd:inbox` — Read and Promote Reports

`/abcd:inbox` reads the reports repositories abcd manages filed with
[`/abcd:report`](29-report.md), and files one as a capture when a person or a
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
time, sender repository named plainly, kind, severity and title. Showing one
report by its id renders it whole, waiting or promoted. Both are read-only: an
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
JSON forms carry that sentence as `notice`. The plugin page frames the whole
page the same way, list and show alike. A key name a refusal echoes is clipped
to 64 bytes. The id a caller names is
checked for its shape before anything is read and is only ever compared with
names already in the inbox: it never becomes a path.

## Promoting

Promoting a report by its id files it as a capture in the ledger of abcd's own
checkout, through capture's own core, so the promoted report is an ordinary
ledger record that the ledger's redactor has scanned before it is written.

Every report is about abcd, so a promotion runs only in a checkout whose root
commit is abcd's own (`488a0aa96ac5de805348635b27036addf15cddc2`). Run in any
other repository it is refused before anything is read or written, naming that
root commit and the one the caller's repository has, so an abcd defect is never
planted in an unrelated ledger with no undo. The root commit is a constant the
binary carries: it cannot change without rewriting every commit after it, and a
test holds the constant to the checkout the tests run in. A shallow clone, whose
first commit is not the root, is refused rather than guessed at; a fork shares
the root commit and promotes.

The capture carries:

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
left alone. A name that is a common word — three letters or fewer (`cli`,
`api`, `go`), a word such as `docs`, `site` or `tools` that names what a
repository holds, the fallback `unnamed`, or `abcd` itself — is replaced only
where it ends a forge address, never as a word: rewriting every `go` in the
account would make it nonsense and protect nothing, since the word identifies no
one, while the owner segment of an address does. The inbox names the sender so
the reader knows who is asking; anything committed carries the fingerprint.

A record id the report names is its sender's, from the sender's own ledger, not
abcd's. Copied as written, a long one would fail record-lint's
`prose_citation_resolves` and a short one would silently cite abcd's own record
of that number. So in every value the report supplies, a record family abcd
resolves (`adr`, `itd`, `iss`, `spc`) followed by a number, across whatever
separators stand between them, is written as one word: the sender's issue 12 is
written `iss12`. That form is outside the cited grammar, in the body and in the
slug capture derives from it, and the number survives for a reader to ask the
sender about. When anything was rewritten, the capture ends with a sentence
saying so.

An HTML comment in the report's prose — the template's placeholder, left in
place by a reporter who writes below it — is removed when the report is read,
so none reaches the inbox or a capture: a comment renders as nothing, and would
be text in the record no reader sees. A comment opened and never closed is
refused.

One line in `~/.abcd/inbox/promoted.jsonl` records the capture the report
became and its path, and the report is then moved to `~/.abcd/inbox/promoted/`
and kept; showing it names the capture. The line is written before the move, so a
move that fails leaves the report waiting with its capture on record: promoting
it again files nothing, finishes the move, and answers with that capture and
`resumed: true`.
Promotions hold the inbox's lock, so two sessions cannot file one report twice.
A promotion outside abcd's own checkout, an unreadable report, one already
promoted, and an id with no report are refused. A capture the ledger refuses —
a symlinked ledger, a slug that normalises to nothing — is refused too: capture
sweeps its reservation, so nothing is written and the report still waits.

## Exit codes

`0` done; `2` refused, with nothing written, and with the home and working
directories written as `~` and `.` in the message. The JSON output holds on
every path: the list is `{"notice", "tally": {"reports", "senders"}, "reports": [...]}`, show is
the entry with `notice` beside its fields, and a refusal is
the `{"abcd":"error",…}` envelope on stdout.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->
<!-- surface-appendix:end -->
