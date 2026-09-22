---
id: adr-2609221017021499
slug: every-external-credential-abcd-holds-is-named-in
status: accepted
date: 2026-09-22
supersedes: null
superseded_by: null
related_intents: [itd-2609221017023290, itd-2609081951381895, itd-2609061543533170, itd-63]
related_rfcs: []
related_adrs: [adr-25, adr-2609221009491186]
---

# ADR-2609221017021499: Every external credential abcd holds is named in configuration and kept in one store the person chose; never the harness, never the repository

## Context

The first adapter to need a secret (the OpenAI-compatible API adapter, for
OpenRouter) was given a one-time walkthrough at `ahoy` with three homes for
the key: a setup outside abcd, abcd-only on the machine, or the platform
keychain. The site setup needs a hosting credential; a transcript cloud hook
and a forge token are foreseeable. The product thinker observed on
2026-09-22 that the walkthrough is a pattern for every credential, and the
record's rule is one canonical primitive: a generic-smelling thing is built
once in its home and extended, never copied.

## Decision

We will hold every external credential through one credential store.

1. **A name in configuration, never a value.** Configuration names a
   credential; the value lives in the store. Nothing under the repository
   and nothing in the harness's settings ever carries a value.
2. **Three homes, the person's choice**, made once per credential at an
   `ahoy` walkthrough that first explains what the credential unlocks and
   what works without it (itd-63's mode): a setup accessible outside abcd
   (an existing tool's configuration or a named environment variable; the
   store holds the pointer), abcd-only on the machine (the user-level
   `~/.abcd/` store, owner-only permissions), or the platform keychain
   (macOS Keychain; the secret service on Linux). The keychain is named as
   the recommendation in the walkthrough's prose and never as a marked
   option.
3. **One reader.** Every adapter resolves a credential through the store's
   one function by name; an adapter that reads a secret any other way is a
   defect. A name that resolves to nothing is a refusal naming the
   walkthrough, never a silent unauthenticated call.
4. **Scanned before it can be written.** The store's write path runs the
   secret scanner on any file it touches, and a value that would land in a
   tracked path is refused.

## Alternatives Considered

- **Each adapter its own walkthrough and file.** Rejected: three copies of one
  thing within a month, each a place to audit and each a place to leak.
- **The harness's own secret handling.** Rejected: abcd is host-agnostic, the
  harness's settings are the one place the product thinker asked to keep
  secrets out of, and a harness file is often synced or committed.
- **Keychain only.** Rejected: some people will not want it, and Linux has no
  single equivalent everywhere; the choice is theirs.

## Consequences

- itd-2609221017023290 builds the store and the walkthrough; the API adapter
  (itd-2609081951381895) and the site setup (itd-2609061543533170) read
  through it, and their own walkthrough criteria are met by calling it.
- A future hook to an external service adds a name and an explanation, not a
  store.
- The brief's adapters chapter gains the invariant beside adr-2609221009491186.
