# `/abcd:dashboard` — The Product Thinker's Dashboard, Through Tailscale

`/abcd:dashboard` runs the one network listener abcd has: a server on this
computer that the product thinker opens from a phone, an iPad or another
computer on their own Tailscale network (itd-2610032150577708,
spc-2610040741034208). Who may reach it, and on what terms, is the standing
rule [adr-2610032150581128](../../decisions/adrs/2610032150581128-abcd-is-reachable-on-a-network-only-through-the-dashboard.md)
and brief invariant 7's inbound clause
([`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md)).

This chapter describes what ships with the spec's first step: the verb's
start, stop and status, the listener, the identity check at the connection,
and the server's hardening. The server answers one fixed page naming the
person it let in; the summary, the item pages and the brief come with the
later steps.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `start` | — | shipped |
| `stop` | — | shipped |
| `status` | — | shipped |

The server process is a hidden sub-verb that only start runs, so it has no row.

## Where it listens, and who gets an answer

The server listens on this computer's own Tailscale addresses, the ones
Tailscale names for it inside 100.64.0.0/10 and fd7a:115c:a1e0::/48, and on
nothing else: never the unspecified address, never loopback, never an address
of the local network. It serves plain HTTP with no certificate, because a
certificate would publish the computer's Tailscale name in the public
certificate-transparency logs for good, while Tailscale already encrypts and
authenticates every connection (decision 23). Nothing is published through
Tailscale Serve or Funnel: Serve hides which device connects (decision 21), and
start refuses a port that Serve's or Funnel's own configuration names.

Each connection is judged before a byte is sent. A peer address outside the
Tailscale prefixes is closed at once, with no lookup. Otherwise Tailscale's own
lookup is asked who is at that address, and the connection is let in only when
it names an untagged device of a person: a tag names a machine, not a person.
Any untagged device on the person's Tailscale network is let in, whoever's it
is, which is the cost decision 4 accepted. A lookup that fails, a tagged
device, and a connection past the cap are closed having been sent nothing, so a
device off the network gets no answer at all. No request header ever decides
who is connecting: a forged identity header with a failing lookup is refused,
and with a passing one the page names the lookup's person, not the header's.
Lookups are cached for each address for thirty seconds and at most four run at
once; a connection waiting on one counts against the cap of thirty-two open
connections.

## The server's hardening

A request whose host is not the computer's Tailscale name, with or without the
port, is answered 421 before any route runs, which defeats a page that points a
name of its own at this address. Every response, refusals included, carries a
content-security policy that loads nothing from anywhere but this server and
forbids framing, together with no-sniff, no-referrer, no-store and a
same-origin opener policy. Headers must arrive within five seconds and a
request within fifteen; a response may take thirty, an idle connection lives
sixty; headers are capped at 16 KiB and bodies at 64 KiB. Routes are a fixed
table and no route joins a request path onto a folder. No route writes yet:
any method but reading is refused, and Go's cross-origin protection already
judges a write by its origin against its host, which is how the later notes
are guarded over plain HTTP.

## Starting, stopping, reading the state

Start refuses outside a checkout abcd manages, when a dashboard already runs
on this computer, when Tailscale is not running or reports a release older
than the one this abcd was shown to read, when the `tailscale` command found
sits in the working tree or in a folder anyone can write, and when the port is
Serve's or Funnel's. It finds the command on the path, then in the macOS app's
bundle. It launches the server as a detached process in its own session, so
closing the Terminal does not stop it, hands it its configuration and a
one-time value over a pipe rather than its command line, passes it nothing of
its own environment but the path and the home folder, so no token in the
Terminal lives on in a process that faces the network, and waits for it to
report that it is listening. It then fetches its own address through
Tailscale, so the gate judges this computer as it judges any device, and only
when that answers does it print one line: where to open it, that anyone on the
Tailscale network can, and every address it listens on. A failed fetch stops
the server and names the address that could not be reached. The default port is
8080.

Stop reads the run file and signals only the process start launched, checked
by its process id, its start time and its executable read again just before
the signal, then waits for it to exit and for its addresses to answer nothing,
and removes the run file. A run file naming a process that is gone, or one
that is another program, is removed with nothing signalled. Nothing stops a
process by name or pattern, and abcd installs no login item, launch agent or
service unit, so the dashboard never starts by itself.

Status reads whether the server runs, where and since when, and the devices
that opened a page in this run, each by its Tailscale device name and person.
The run file and the devices list live in `~/.abcd.noindex/dashboard/` and name
no project content. A device is shut out by removing it from the Tailscale
network; abcd keeps no device list of its own on the tailnet.

## Exit codes

`0` done; `1` a failure after something started (the server could not be
reached and was stopped, or it did not stop in time and its run file is kept);
`2` refused, with nothing started or stopped.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd dashboard`

Sub-verbs: `abcd dashboard serve`, `abcd dashboard start`, `abcd dashboard status`, `abcd dashboard stop`.

Flags: none.

### `abcd dashboard serve`

Sub-verbs: none.

Flags: none.

### `abcd dashboard start`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--port` | int |

### `abcd dashboard status`

Sub-verbs: none.

Flags: none.

### `abcd dashboard stop`

Sub-verbs: none.

Flags: none.

<!-- surface-appendix:end -->
