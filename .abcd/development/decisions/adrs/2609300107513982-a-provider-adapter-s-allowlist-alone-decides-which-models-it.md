---
id: adr-2609300107513982
slug: a-provider-adapter-s-allowlist-alone-decides-which-models-it
status: accepted
date: 2026-09-30
supersedes: adr-2609221009491186
superseded_by: null
related_intents: [itd-2609081951381895, itd-2609221009495079, itd-2609170822093401, itd-6]
related_rfcs: []
related_adrs: [adr-2609221009491186, adr-25]
---

# ADR-2609300107513982: A provider adapter's allowlist alone decides which models it serves, and abcd bundles no vendor denylist

## Context

adr-2609221009491186 (2026-09-22) made every provider adapter default-deny by
model. Its decision 2 put a bundled vendor denylist above the per-provider
allowlist: `anthropic/*` at minimum, refusing a listed model whatever the
allowlist said, extendable by the repository and the machine and never
shortened below the bundled set. The shipped resolver in
`internal/core/oracle` carried that list, refused a listed model it matched
at read, at the setup and at the call, and discarded an answer whose reported
model it matched.

On 2026-09-29 the product thinker answered ruling AA(a), on which spellings
the `anthropic/*` prefix should chase, with "I think we need an allow list
rather than trying to catch a deny list": the allowlist is the control. The
technical facilitator then ruled H9 the same day: the bundled denylist is
retired, the allowlist alone decides, and ruling Y (whether a person may
remove a bundled entry on their machine) closes as moot, since no bundled
entry remains to remove. Whether `oracle.denylist` survives as an optional
extension the configuration writes was left to the implementing lane: keep
it only if it costs nothing.

Ruling H10 of 2026-09-29 settles how a change like this is recorded: an ADR
whose decision text is now false is superseded, not edited in place. This
record is that successor. It changes decision 2 only; decisions 1, 3, 4 and 5
and the consequence added on 2026-09-29 stand as adr-2609221009491186 stated
them, and are carried here so the record in force states the whole decision,
as adr-2609292012006845 did for adr-2609212115255771.

## Decision

We will keep every provider adapter default-deny by model, with the
provider's allowlist alone deciding which models it serves. Decisions 1, 3, 4
and 5 are adr-2609221009491186's, carried forward word for word; decision 2
is revised.

1. **An allowlist per provider.** Each provider block in the configuration
   lists the model identifiers it may serve. A role or a judgement
   configured for a model not on its provider's list is refused when the
   configuration is read, before any call is made, naming the list.
2. **No bundled vendor denylist; the allowlist alone decides.** abcd ships no
   denylist of vendor prefixes, so a model a provider lists is served
   whichever vendor made it, and a model it does not list is refused by
   decision 1. `oracle.denylist` stays as an optional extension the
   configuration writes, in the repository's `.abcd/config.json` or the
   machine's `~/.abcd/config.json`: an entry refuses a model it matches even
   when a provider lists it, at the read, at the setup and when a provider
   reports answering with it, and the layers are a union, so neither removes
   the other's entries. With no entry written the list is empty.
3. **Everything else runs on the host.** A model that is neither listed nor
   the host's own is not a route; the host and the person's subscription are
   where frontier models run.
4. **The key is named, never stored.** The adapter reads its credential from
   the machine's configuration or the environment by name; nothing about a
   key enters the repository.
5. **The run record names the route.** Every call through an adapter records
   the provider, the model identifier asked for and the model the provider
   reports, so a substitution by the aggregator is visible.

## Alternatives Considered

- **Keep the bundled `anthropic/*` denylist as a backstop.** Rejected by
  ruling H9: the product thinker's answer to AA(a) makes the allowlist the
  control, and a prefix list is the thing that answer declined to chase. The
  prefix never caught `openrouter/anthropic/...`, `anthropic.claude-...` or a
  bare `claude-...` spelling, so it read as a guarantee it did not give.
- **Drop `oracle.denylist` altogether, refusing any configuration that still
  names it.** Rejected: the setting already existed, and with the bundled list
  gone it keeps working with no added code, so ruling H9's condition (keep it
  only if it costs nothing) is met. Dropping it would add a refusal and take
  away the person's way to rule a vendor out across every provider at once.
- **Amend adr-2609221009491186's decision 2 in place.** Rejected by ruling
  H10: an accepted ADR whose decision text is false is superseded.
- **Supersede decision 2 alone, leaving the other four in force in the old
  record.** Rejected: the record has no partial supersession, and a record
  marked superseded whose other decisions still bind reads as retired to a
  reader following `superseded_by`. Carrying decisions 1, 3, 4 and 5 forward
  word for word keeps one record in force.

## Consequences

- itd-2609081951381895's criterion 3 and its spec read the allowlist alone:
  a listed model of any vendor is admitted, an unlisted one is refused naming
  the list, and an `oracle.denylist` entry the configuration writes refuses
  a listed model naming the entry. The intent's Audit Notes name ruling H9
  and the criterion's old wording (iss-2609300110451242).
- A person who wants a frontier model through an aggregator lists it in the
  provider block on their machine, deliberately, and the run record shows
  it. A person who wants a vendor kept off every provider writes its prefix
  in `oracle.denylist`.
- An aggregator that answers a listed model's request with another model is
  visible in the run record (decision 5); it is refused only when an
  `oracle.denylist` entry names the reported model.
- `abcd ahoy --providers` reports the `oracle.denylist` entries written, or
  that none is written; it names no bundled entry.
- The brief's adapters and configuration chapters state the allowlist-alone
  invariant.
- **Consequence carried from adr-2609221009491186 (added there 2026-09-29): a
  route to a provider that holds a key is the machine's alone.** Decision 4
  keeps the key's value out of the repository; the product thinker's ruling
  AA(b) of 2026-09-29 keeps the repository from spending it too. Only a route
  the person set up on their own machine may use their paid key, so a role or
  a judgement type the repository's configuration points at a provider whose
  block names a key is refused when the configuration is read, naming
  `~/.abcd/config.json` as where to set it. A repository's route to a
  provider that holds no key (a local server) and a `--route` the person
  types are unaffected. The allowlist, and any `oracle.denylist` entry, still
  apply to every route that is admitted.

## Amendment — 2026-10-02: a repository's route to an unlisted model is skipped, not refused

Decision 1 refuses a role or a judgement type configured for a model not on its
provider's list when the configuration is read. The technical facilitator's
ruling CD3 of 2026-10-02 narrows the refusal for one layer: a route in a
repository's `.abcd/config.json` naming a model that a configured provider
holding no key does not list is skipped with a diagnostic naming the file, the
route and the list, and the machine's route to the name, if any, applies in its
place, so a checkout's configuration never takes the commands that read it
down. The model is still never asked for: the route is not loaded. The
machine's own route to an unlisted model is refused as decision 1 says, a model
an `oracle.denylist` entry matches is refused from any layer, and a
repository's route to a provider that holds a key is skipped before that
provider's list is consulted, so the list is never read on a repository's
behalf. adr-25's amendment of the same date records the ruling beside CD4.
