---
id: spc-2610031241482088
slug: connecting-a-model-service-from-inside-claude-code-works-out
intent: itd-2610030821294016
origin: researcher-authored
production_mode: hand-written
---
# connecting-a-model-service-from-inside-claude-code-works-out

## Summary

This spec delivers
[itd-2610030821294016](../../intents/shipped/itd-2610030821294016-connecting-a-model-service-from-inside-claude-code-works-out.md):
a person connecting a model service from inside Claude Code no longer types
every value. abcd asks for the service's address, offers to look up the models
the service lists, suggests the models the person already uses through their
other connections, narrows the list as they type part of a name, and asks
where the key lives. It ends by printing one command, and every file that
command writes, for the person to paste into a terminal. Nothing is set up
from inside the session.

Three things land, in order. The adapter gains the model listing it lacks: one
GET of the service's model list, with or without a key, that follows no
redirect, gives up after a short bound, and refuses an oversize answer. The
terminal step, `abcd ahoy connect` run by the person, learns to read the key
on hidden input at a terminal, and, when no model is named, to list the
models with the key held in memory, let the person pick one in the
plain-Terminal picker, verify the pick with a real completion, and only then
write. The guided path for the session, `abcd ahoy connect --guide`, asks its
questions one turn at a time, writes nothing, and ends in the printed command.

The settled records are designed to here, never reopened: the intent's
decisions 1 to 9, its confirmed criteria G1 to G8, and its four scope
conditions.

## Scope

In:

- `internal/adapter/openaiapi`: the model listing (`ListModels`) and its
  typed failure, beside `Complete`, on the same pinned client.
- `internal/core/oracle`: the guided path (`Guide`, a new file
  `connect_guide.go`), the suggestions drawn from the person's existing
  connections, and the model-less form of `Connect` that lists with the key
  and asks a picker (`ConnectRequest.Pick`).
- `internal/core/credential`: one function naming what a home's write
  touches (`WritesFor`), which `Walk` and the guided path both read, and the
  external pointer's resolver exported (`ResolvePointer`) so the model-less
  form can list with an external key before the walk.
- `internal/core/question`: a typed part on the question type, so a question
  can take a typed answer (the address, part of a model's name, a variable's
  name); and the narrowing rule the plain-Terminal picker uses, shared, so the
  session and the Terminal narrow identically.
- `internal/term`: the hidden read of the key, inside the restore guarantee
  the picker already gives.
- `internal/surface/cli`: `--guide`, `--resume` and `--answer` on
  `abcd ahoy connect`; the hidden read replacing the terminal refusal in
  `readKey`; the picker wired as `ConnectRequest.Pick`; and the typed part
  drawn by `internal/surface/cli/ask`.
- `commands/ahoy.md` (the connect section, the guided path's page),
  the generated `docs/reference/cli/commands.md`, and the generated flag
  appendix of `.abcd/development/brief/04-surfaces/01-ahoy.md`.

It builds on, and reuses rather than restates:

- [spc-2610030911534855](spc-2610030911534855-a-person-can-run-abcd-s-interviews-in-a-plain-terminal.md),
  the plain-Terminal picker (itd-2610030810370060): its step 1 (the question
  type in `internal/core/question`, golang.org/x/term in `go.mod`) and its
  step 2 (the long list with typing to narrow, the numbered fallback, and the
  Terminal restored on every exit). The terminal step's pick is that list.
- The shipped adapter and its connect verb (itd-2609081951381895): its
  decisions 2 (no service-specific code), 6 (the walkthrough is a verb the
  person runs) and 7 (verify with the first model, then write) stand.
- The shipped credential store (itd-2609221017023290): its three homes, its
  walkthrough (`credential.Walk`) and its rule that a stored secret is never
  replaced.
- [spc-2610030944505997](spc-2610030944505997-asking-and-layout.md), the
  asking rules and the field limits: every guided question is held to them
  where they have landed.

Out:

- Judging a request against a model's published size before it is sent:
  routed to iss-2610030956156354 (decision 8). The guided path shows no size
  and stores none.
- Adopting a key the person saved by hand in the computer's password store
  (decision 7). The guided path offers the keychain home only as abcd's own
  hidden input.
- Setting the connection up from inside the session, for any home (decision
  1), and any setting that would run the command instead of printing it
  (decision 3).
- A service-specific check, sort or search parameter (the shipped intent's
  decision 2). The look-up is the one standard request every
  OpenAI-compatible service answers or does not.
- More than one model in the guided path. The command carries the one model
  picked; the person may repeat `--model` in it by hand before running it, as
  today.
- The external home's file pointer (`--file` and `--field`) in the guided
  path. The guided path offers the external home through an environment
  variable; a file pointer is written by hand with today's flags.
- A model list the service pages. The look-up reads the first answer only; a
  model not on it is found by typing its full name.
- The Terminal drawing's own limits and the asking rules' wording: the two
  question specs own them.

## Approach

### The model listing in the adapter

`openaiapi` today sends one request, `Complete`, a POST to
`{base}/chat/completions` (`client.go`, `New` at line 162). The listing is its
sibling on the same `Client`:

```go
// Models asks the service for the models it lists: one GET of {base}/models.
// keep, when given, is the caller's filter: a usable id it refuses is dropped.
func (c *Client) Models(ctx context.Context, keep func(id string) bool) (Listing, error)

type Listing struct {
	IDs     []string // in the service's own order, each sanitised and bounded
	Dropped int      // listed names that were not usable model names
}

// ListError says why there is no list, in words a question can carry.
type ListError struct {
	NeedsKey bool   // the service answered 401 or 403
	Reason   string // "answered not found", "answered with a redirect, which abcd never follows", ...
}
```

The rules are decision 4's, enforced in the adapter so that no caller can
loosen them:

- **No key unless one is given.** `New(baseURL, "")` sends no
  `Authorization` header, as `Complete` already does for a keyless server. The
  guided path always lists with the empty key; only the terminal step lists
  with a key, after the person has run the command.
- **No redirect.** The listing uses the client's pinned `CheckRedirect`
  (line 190), so a redirect is a failure, never followed, and a key header
  never reaches a second address.
- **A short bound.** The adapter's call default, `DefaultTimeout` (line 63),
  is 120 seconds, sized for a completion. The listing has its own bound,
  `ListTimeout`, 10 seconds, applied whatever the client's call timeout, so
  "the adapter's short timeout" of decision 4 is a constant of the adapter,
  not of each caller.
- **The size cap.** The body is read through `MaxResponseBytes` (line 67, 4
  MiB) plus one byte; a larger body is refused unread, as `Complete` refuses
  it.
- **The address.** `ValidateBaseURL` (line 198) runs first, unchanged: https
  anywhere, http only to this machine. A local-network address over plain
  http is refused before any request, naming the rule.

The decoder reads `data[].id`, the shape the standard list answer and its
common variants share; any other shape, or no usable id, is a `ListError`
saying the service listed no usable models. An id that shows the key in any
reading the client's scrub knows, holds a rune `termsafe` masks, or is more
than 128 bytes is dropped and counted in `Dropped`, never kept redacted; the
ids `oracle`'s `validModel` (`config.go`) and denylist refuse are dropped the
same way, through `keep`. At most
`MaxListedModels` (5,000) ids are kept, in the service's order. `oracle`'s
filter is `keep`, run inside the adapter before the kept ids are read
together for the key split across adjacent ids, so the ids that read checks
are the ids offered: an id the filter drops cannot separate two halves of the
key there and leave them adjacent in the offer.

The status mapping: 401 and 403 are `NeedsKey`; any other non-200 status, a
redirect, a timeout, an unreachable host, an oversize body and an undecodable
one each carry their reason. Every error passes through the client's `fail`,
so the key, in every form `keyForms` knows, is scrubbed from it.

### The terminal step: hidden input, and the model picked after a keyed listing

The printed command is today's verb, `abcd ahoy connect <provider>`, with two
changes the session's command needs in order to run in a terminal.

**The key on hidden input.** `readKey` (`ahoy_connect.go`) reads the key from
piped stdin only. Connect reads it through `readConnectKey`, which keeps that
piped read and, at a terminal, reads the key with echo off: one line on
stderr, "Paste the key for <provider> and press Enter. It is not shown.",
then `term.ReadHidden`, golang.org/x/term's line reader with echo off in
`internal/term`. The read runs inside the restore guarantee step 2
of spc-2610030911534855 builds (the idempotent restore reached from the
deferred call, a recovered panic, and the SIGINT, SIGTERM and SIGHUP handler),
so an interrupt during the paste never leaves the terminal with its echo off.
An empty answer is refused as an empty pipe is today. The key goes into
`ConnectRequest.Key` and nowhere else.

**No `--model`: list with the key, then pick.** `ConnectRequest` gains

```go
// Pick, when Models is empty, chooses one model from the listed ids.
// Connect calls it after listing with the key it holds; nil refuses as today.
Pick func(ctx context.Context, listed []string) (string, error)
```

When `Models` is empty and `Pick` is set, `Connect` resolves the key it holds
(the hidden or piped value for the abcd and keychain homes; the pointer's
value for the external home, through `credential.ResolvePointer`, the
existing `resolvePointer` in `external.go` exported; none for `none`), lists
through `openaiapi.Models` with that key, hands the ids to `Pick`, and sets
`Models` to the one picked. From there it is today's path unchanged:
`credential.Walk` verifies with a real completion to the picked model through
`providerService` and only then stores the key, and then the provider block is
written. A listing that fails, a pick the person cancels (the picker's decide
later), a completion that fails: each writes nothing, and the error names
which. No listing is ever taken as the test (decision 2).

The CLI sets `Pick` only when stdin, stdout and stderr are all terminals, the
letter adr-49 sets for drawing; the picker is spc-2610030911534855's long list
over the ids, chip "Setup", material "<host> lists <n> models. Type part of
a name to narrow the list." Off a terminal, a run with no `--model` is refused
as today, the message now naming both ways on: `--model`, or running the
command in a terminal to pick there.

`credential.WritesFor(home, name string) []string` names what a home's write
touches, in the tilde form: the abcd home's `StorePath`; the keychain's item
(`KeychainItem`) and the index (`IndexPath`); the external home's index.
`Walk`'s `Wrote` (`walk.go`, lines 121 to 131) is built from it, so the list
the guided path prints and the list a run reports cannot drift apart.

### The guided path in the session

`abcd ahoy connect [<provider>] --guide [--base-url <url>]` starts the guided
path; it never calls `Connect`, never writes a file, and never asks for the
key. Each run returns one turn:

```json
{
  "ask":    { "questions": [ ... ] },
  "resume": { "schema_version": 1, "provider": "...", "base_url": "...",
              "answers": [ {"id": "lookup", "value": "yes"} ],
              "listed": { "status": "listed", "host": "...", "models": [ ... ] },
              "open": "model" }
}
```

or, at the end, `"done": {"command": "...", "writes": [ ... ], "picks_in_terminal": false}`
in place of `ask`. The page asks the turn's question through the host's
question tool, then runs `abcd ahoy connect --guide --resume - --answer
'<value>'` with the turn's `resume` object on stdin; the binary appends the
answer to the question `open` names and returns the next turn. The plugin
page does no other bookkeeping.

**Replay, not state.** `oracle.Guide` is a function of the provider and
address given, the answers, and the listed models. Each turn it replays the
answers from the first question, re-deriving each question and checking each
answer against it (a value a question does not admit is refused, exit 2,
naming the question), so an edited `resume` cannot skip a question. The
listed models are the only thing carried rather than re-derived: the look-up
runs once, at the turn the replay first reaches a yes to the look-up with no
`listed` present, and every later turn narrows over what `resume` carries
(open question 2). A `resume` that was tampered with can change which names
are offered and, as built, cause one keyless request for the model list (a
`resume` whose answers say yes to the look-up and that carries no `listed` is
looked up again, as the first yes is: no key, no redirect, at the address the
replay checks again), and nothing else: each id is checked again with
`validModel`, the address with `ValidateBaseURL`, and the command is shown in
full and verified with a real completion before anything is written.

As built: `resume` carries at most 32 KiB of the listed ids as JSON
(`oracle.MaxCarriedBytes`), the first in the service's order, with `more`, the
count not carried, and a part of a name matching none carried while `more` is
above zero asks for the model's full name, which is checked as any typed name
is, so G2's one request holds (decided 2026-10-04, `.abcd/work/DECISIONS.md`).

The questions, each one turn, in order. Each is a `question.Ask`, chip
"Setup Q<n>", that the asking rules hold where spc-2610030944505997 has landed: the thing first, the
question last, decide later last, nothing marked.

1. **The address** (only when `--base-url` is absent). A typed question:
   "What is the service's address?", the typed part "Type the address,
   starting https://, or http:// for a server on this machine". An address
   `ValidateBaseURL` refuses returns the same question with the refusal's
   reason first in its material.
2. **The look-up** (G1). Material: "abcd can ask <scheme>://<host> for the
   list of models it offers. The request carries no key, follows no redirect,
   and gives up after 10 seconds." Options: "Look up the list", "Type the name
   myself", decide later. Only the first sends a request.
3. **The model.** What it carries depends on the look-up:
   - *Listed* (G2): material "<host> lists <n> models."; options are the
     suggestions, at most three; the typed part "Type part of a model's name".
     A typed fragment that is exactly one listed id chooses it. Otherwise the
     next turn is the narrowing question: "<m> of <n> match '<fragment>'", the
     first three matches as options, the typed part "Type more of the name, or
     another part", and, when more than three match, the material says the
     first three are shown. No match says so and asks again. Narrowing uses
     `question.Matches` (case-insensitive substring of the label or the
     value), the picker's rule, so a fragment narrows the same in both places.
   - *The service needs the key to list* (decision 2): no model question.
     The command is printed without `--model`, and the done turn's
     `picks_in_terminal` is true; the next question's material says "<host>
     lists its models only for a key, so the command lists them in your
     terminal after you paste the key, and you pick there."
   - *No list, or the look-up declined* (G7): a typed question, "Type the
     model's name", its material giving the reason first: "<host> publishes
     no model list: it answered not found." (the `ListError` reason), or "You
     chose to type the name rather than look it up." A typed name that
     `validModel` refuses, or the configuration's denylist refuses (`Denied`,
     `config.go` line 607), returns the question with that reason.
4. **Whether the service takes a key** (skipped when the look-up answered
   that it needs one). Options: "It needs a key", "No key: a server on this
   machine", decide later.
5. **Where the key lives** (G8). Material: `KeyHomesProse` in short (as
   built: verbatim, it takes the question past the 24 rows the asking limits
   hold it to, so the question says the choice is the person's and that abcd
   recommends the keychain, and each home's meaning sits on its option). Options: the credential store's three homes,
   external (an environment variable), abcd, and keychain, each with its
   meaning, then decide later; never marked, never reordered by preference.
   The keychain option's meaning says the key is pasted into abcd's hidden
   input in the terminal step and stored by abcd. The guided path never
   searches the keychain or the index for an item, so no item, by any name,
   is offered (decision 7).
6. **The variable** (the external home only). A typed question: "Which
   environment variable holds the key?", options the names of at most three
   variables in the guide's own environment whose names end in `_API_KEY`,
   those containing the provider's name first (open question 4); never a
   value. The material says the variable must be set in the terminal the
   command is pasted into.

The provider's name is the positional argument when given; otherwise it is
derived from the host (`local` for this machine; otherwise the host's label
before its last, `api` and `www` skipped), held to `providerNameRe`, with `-2`
and on appended while the name is already configured. The key's name is the
provider's, as `checkConnect` defaults it.

Suggestions (decision 6) come from the person's existing connections:
`LoadAPI(roots)`'s routes first (`Routes`, `config.go` line 678, each target's
model, in route order), then each configured provider's allowlist
(`Providers`, line 636, in name order), deduplicated, kept only where the id
is in the service's list exactly (open question 3), at most three. A person
with no connection sees none: the typed part alone.

**The done turn** (G3, G4). `command` is the full command line, every value
shell-quoted, for example

```text
abcd ahoy connect example --base-url https://api.example.com/v1 --model coder-large --home keychain
```

and `writes` lists, in the tilde form, every path it writes, in the order a
run reports them: `credential.WritesFor(home, name)` (none for `--home none`),
then `~/.abcd/config.json` (`layered.Config.MachineOrigin()`), so the keychain
home lists the keychain item, the index, and the settings file. The text form prints the command on a line of its own,
then "When it runs, this command writes:" and one line per path, then, for a
keyed home, "It asks for the key there, hidden as you paste it." There is no
flag that runs the command instead (decision 3); `--guide` and `--home`
together are refused. A decide-later answer to any question ends the guide
with nothing printed but "Nothing was set up; run the guide again to pick up
where this left off", and `resume` kept so the page can.

The page, `commands/ahoy.md`, adds the guided path above the manual setup:
run the guide, ask each turn's question through the question tool exactly as
returned (the typed part as the free-text row, so the question carries no
preview), pass the answer back, and at the end relay `command` and every
line of `writes` verbatim, telling the person to paste the command into a
terminal on this machine. The page's standing rule stays and is restated
there: never ask for the key, never pass it, never run the command for the
person.

### The typed part on the question type

The shared question type gains one optional field:

```go
Typed string `json:"typed,omitempty"` // the prompt for a typed answer
```

A question with a typed part takes any text as its answer, besides its
options. The host path maps it to the question tool's free-text row (so such
a question carries no side preview, which would remove that row), and the
Terminal drawing in `internal/surface/cli/ask` draws it as one line after the
options, "or type: <prompt>", the numbered reader taking a line that is not a
number as the typed answer. `question.Check` admits a question with a typed
part and no options; where spc-2610030944505997's limits have landed, its
option floor counts the typed part as one option (open question 1). `Matches`
moves into `internal/core/question` when spc-2610030911534855 step 2 keeps it
inside `internal/surface/cli/ask`, and the picker calls it there.

As built: the host's question tool takes two to four listed options, so a
question whose only listed answer is decide later cannot be put to it as it
stands. The field view keeps the typed part out of the options (it counts
toward the floor only), and says its prompt in the question text just before
the ask, since the host's free-text row carries none; the front door's mapping
onto the host's tool (the turn's `tool` member) adds one option,
`question.TypedRowLabel` ("Type my own answer"), only where the listed options
fall below the tool's floor. It points at the row for typing and answers
nothing: the guide asks the question again when it is chosen. An answer may
name an option by its label, as the host's tool returns labels.

### What the key never touches

The guided path has no key to leak: it asks no key question, lists with no
key, and its JSON has no key field. In the terminal step the key lives in
`ConnectRequest.Key` (or the pointer's resolved value) and in the adapter's
`Authorization` header; every error passes through `openaiapi.Scrub` before
it is printed, as today, and the listed ids through the client's scrub. G6's
canary test holds the whole path to that.

As built: the guide holds no key, so it knows one by its shape alone. A typed
answer to any question, or a recorded one, holding a value the secret
scanner's patterns know as a credential (a `token:` kind) is refused with
"that looks like a key; the guide takes a name, never a key" and asked again,
and a listed id or an offered variable's name holding one is dropped before
the resume object carries it. A retry never quotes the typed text back: only
the narrowing question quotes the part of a name it narrows by. A key of a
shape the scanner does not know passes as a name.

## How each acceptance criterion is met

**G1.** `TestListModelsSendsNoKeyAndFollowsNoRedirect` (in
`internal/adapter/openaiapi`) stands up a stand-in service that answers its
model list with a redirect to a second stand-in, and asserts the listing
fails naming the redirect, the second stand-in receives no request, and no
request carries an `Authorization` header. `TestListModelsBoundsTimeAndSize`
holds `ListTimeout` against a stalling stand-in and the size cap against a
body one byte over. `TestGuideAsksBeforeLookingUp` (in `internal/core/oracle`)
runs the guide to the look-up question and asserts it shows the scheme and
host and that the stand-in has received no request; answering "Type the name
myself" sends none; answering yes sends exactly one GET, keyless.

**G2.** `TestGuideSuggestsTheModelsAlreadyUsed` configures two providers
whose allowlists name two of a stand-in's 300 models, and asserts the model
question offers exactly those two plus the typed part; a person with no
connection gets the typed part alone. `TestGuideNarrowsWithoutASecondRequest`
answers "coder" and asserts the narrowing question offers the first three of
the names containing it, says how many match, and that the stand-in received
one request across every turn.

**G3.** `TestGuideWritesNothing` snapshots the fake home's `~/.abcd`, the
index, the abcd credential file and the fake keychain (through the existing
`locateKeychain` seam) before and after a guided run to its end against a
keyless stand-in on loopback, for each home, and asserts every byte
unchanged and exactly one command printed.

**G4.** `TestGuideShowsTheCommandAndEveryPath` asserts, for each home, that
the done turn's text form prints the command and then each path, and that
`writes` equals the `wrote` list of running that command against the
stand-in. `TestGuideCannotRun` asserts `--guide` with `--home` is refused.
The dated receipt `.abcd/.work.local/logs/connect-guide-<yyyy-mm-dd>.md`
records a guided run in a Claude Code session: the date, the commit, the
Claude Code version, and the command and its paths as shown on screen.

**G5.** `TestConnectPicksAfterAKeyedListing` runs the terminal step with no
`--model` against a stand-in that answers its keyless list 401, lists with
the key, and answers a completion; the hidden reader and the picker are
injected. It asserts the list request carried the key, the pick is the model
the completion asked for, and both writes landed.
`TestConnectWritesNothingWhenTheCompletionFails` uses a stand-in that lists
with the key and fails the completion, and asserts no provider block and no
stored key. `TestConnectWithoutModelRefusesOffATerminal` asserts the piped
form with no `--model` is refused, naming both ways on.
`TestHiddenKeyRestoresTerminalOnInterrupt` uses spc-2610030911534855's
pseudo-terminal helper: Ctrl-C during the paste exits 130 with `ECHO` set
again.

**G6.** `TestKeyCanaryAppearsOnlyInItsHome` runs the guide to its end and
then the printed command with a canary key, against a stand-in that echoes
the `Authorization` header into its error bodies and lists one model id
containing the key. It searches every turn's JSON and text, the terminal
step's stdout and stderr, every error, and every file under the fake home,
and asserts the canary appears only in the chosen home (the abcd credential
file, or the fake keychain's store), and that the id carrying it was dropped.

**G7.** `TestGuideFallsBackToTyping` runs the guide against stand-ins that
answer the list with 404, with an empty `data`, and with a body that is not a
list, and asserts each reaches a typed model question whose material gives
the reason ("answered not found" for the first). `TestListModelsNamesWhyThereIsNoList`
holds each reason at the adapter.

**G8.** `TestGuideNeverOffersAHandSavedItem` stores an item named for the
service in the fake keychain, outside abcd's index, and asserts the home
question offers exactly external, abcd, keychain and decide later, that no
turn names the item, and that the keychain command carries no key and reads
it in the terminal: with the item present, the terminal step's hidden read
is still called (`TestConnectReadsTheKeyHidden`).

## Open design questions

These are for the technical facilitator. Each is designed to the option
marked, and stands unless the facilitator rules otherwise.

1. **A typed answer and the asking limits' option floor.** The first-time
   user's search question has no listed option but decide later (G2), and
   the address and variable questions are typed. (a) A `Typed` part on the
   shared question type that counts as one option toward the floor, mapped
   to the host's free-text row and drawn as a line in the Terminal (designed
   to): one turn per typed answer, and the confirmed G2 shape needs no
   padding. (b) A listed "Search" or "Type it" option leading to a second
   question for the text: no change to the type, one more turn each time,
   and an option that means nothing on its own.
   - Decided: (a), on 2026-10-03 without a question (the person's ruling: obvious answers are decided): one typed answer part counted as one option keeps every question a single step.
2. **Where the listed models live between the session's turns.** G2 forbids
   a second request and G3 forbids a write. (a) Carried in the turn's
   `resume` object, out on stdout and back on stdin, each id checked again on
   the way in (designed to): no file, no second request, a few kilobytes per
   turn. (b) A cache file in the local tier or the home folder: a write, which
   G3's "written nothing" reads against. (c) One process held open for the
   whole guide: the host's shell tool cannot keep an interactive process
   across questions.
   - Decided: (a): the guided path writes nothing (G3, confirmed), so the list travels in the resume object and is checked again on the way in; (b) breaks G3.
3. **How a suggestion matches a listed id.** (a) The exact id only (designed
   to): never suggests a model the person has not used, at the cost of no
   suggestion where two services spell one model differently (a vendor
   prefix on one, none on the other). (b) Also the part after the last `/`:
   more suggestions across services, at the risk of suggesting a different
   model that shares the short name.
   - Decided: (a): an exact id never matches the wrong model; a looser match is a later change if a case shows the need.
4. **Naming environment variables for the external home.** (a) Offer at most
   three variable names from the guide's environment that end in `_API_KEY`,
   never reading a value (designed to): the person picks rather than recalls
   a name, and the names enter the conversation. (b) A typed name only:
   nothing about the environment enters the conversation, and the person
   types the name.
   - Decided: (a): offering names without reading values keeps G6 and saves typing.

## Footprint

- packages: internal/adapter/openaiapi, internal/core/oracle, internal/core/credential, internal/core/question, internal/term, internal/surface/cli, internal/surface/cli/ask, commands/ahoy.md, docs/reference, .abcd/development/brief/04-surfaces
- tests: TestListModelsSendsNoKeyAndFollowsNoRedirect, TestListModelsBoundsTimeAndSize, TestListModelsNamesWhyThereIsNoList, TestListModelsScrubsTheKey, TestConnectReadsTheKeyHidden, TestHiddenKeyRestoresTerminalOnInterrupt, TestConnectPicksAfterAKeyedListing, TestConnectWritesNothingWhenTheCompletionFails, TestConnectWithoutModelRefusesOffATerminal, TestWritesForMatchesTheWalk, TestGuideAsksBeforeLookingUp, TestGuideSuggestsTheModelsAlreadyUsed, TestGuideNarrowsWithoutASecondRequest, TestGuideWritesNothing, TestGuideShowsTheCommandAndEveryPath, TestGuideCannotRun, TestGuideFallsBackToTyping, TestGuideNeverOffersAHandSavedItem, TestGuideRefusesAnAnswerItsQuestionDoesNotAdmit, TestKeyCanaryAppearsOnlyInItsHome; the dated guided-run receipt in the local tier; the command reference and the surface appendix regenerated; record-lint and docs-lint clean

## Steps

1. The model listing in the adapter
   - criteria: the request rules of G1; the reasons of G7
   - packages: internal/adapter/openaiapi
   - tests: TestListModelsSendsNoKeyAndFollowsNoRedirect, TestListModelsBoundsTimeAndSize, TestListModelsNamesWhyThereIsNoList (401 and 403 as needing a key; 404, an empty list and an undecodable body each with its reason), TestListModelsScrubsTheKey (a key echoed in an error body and in a listed id never reaches the result); the existing `Complete` tests unchanged
   - waits on nothing
2. The terminal step: the key on hidden input, and the model picked after a keyed listing
   - criteria: G5; the terminal half of G6; the hidden-input half of G8
   - packages: internal/term, internal/core/oracle, internal/core/credential, internal/surface/cli, commands/ahoy.md, docs/reference
   - tests: TestConnectReadsTheKeyHidden, TestHiddenKeyRestoresTerminalOnInterrupt, TestConnectPicksAfterAKeyedListing, TestConnectWritesNothingWhenTheCompletionFails, TestConnectWithoutModelRefusesOffATerminal, TestWritesForMatchesTheWalk; the existing piped-key connect and credential tests unchanged; the command reference regenerated
   - lands after step 1 of this spec and after spc-2610030911534855 steps 1 and 2 (golang.org/x/term, the long list, and the restore guarantee)
3. The guided path in the session, the page, and the close
   - criteria: G1, G2, G3, G4, G6, G7, G8
   - packages: internal/core/oracle, internal/core/question, internal/surface/cli, internal/surface/cli/ask, commands/ahoy.md, docs/reference, .abcd/development/brief/04-surfaces
   - tests: TestGuideAsksBeforeLookingUp, TestGuideSuggestsTheModelsAlreadyUsed, TestGuideNarrowsWithoutASecondRequest, TestGuideWritesNothing, TestGuideShowsTheCommandAndEveryPath, TestGuideCannotRun, TestGuideFallsBackToTyping, TestGuideNeverOffersAHandSavedItem, TestGuideRefusesAnAnswerItsQuestionDoesNotAdmit, TestKeyCanaryAppearsOnlyInItsHome; every guided question passing `question.Check` and, where spc-2610030944505997 step 1 has landed, its limits; the surface appendix and command reference regenerated; the dated receipt from a Claude Code session; a docs-fidelity review recorded for HEAD, then `abcd spec close spc-2610031241482088` (the intent already declares `impact: additive`) with a `Delivers: itd-2610030821294016` trailer
   - lands after steps 1 and 2 of this spec (the printed command must run as printed) and after spc-2610030911534855 step 1 (the question type) and step 2 (the narrowing rule it shares)
