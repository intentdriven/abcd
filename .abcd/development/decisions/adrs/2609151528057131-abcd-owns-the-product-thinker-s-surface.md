---
id: adr-2609151528057131
slug: abcd-owns-the-product-thinker-s-surface
status: accepted
date: 2026-08-29
supersedes: null
superseded_by: null
related_intents: [itd-167]
related_rfcs: [rfc-3]
related_adrs: [adr-2609151528057260]
---

# ADR-2609151528057131: abcd owns the product thinker's surface

## Context

The framework's working position has been that abcd does not render an
interface. The host harness owns the pixels, the spinner, the dialogs and the
status line it lets others draw on, and abcd owns the words. The reasoning was
sound: an opaque host banner is not something a configuration layer can fix by
drawing its own, and claiming otherwise would have been the strong form of a
metaphor the project deliberately held at arm's length.

[ADR-2609151528057260](2609151528057260-three-roles-who-each-artefact-addresses-and-when-the-loop-st.md) changes what that
position costs. The facilitator is a machine by default, the agents stop only to
obtain a verdict, and a question the product thinker must answer travels to them
rather than waiting at a terminal. The command line is the facilitator's
surface, and the product thinker does not work there.

In this repository that gap is invisible, because one person occupies both
roles and can read a question off the terminal. That is the unusual case. For
the product thinker this framework is built for, there is no facilitator in the
room to read anything aloud, and no reason for them to have a terminal at all.
Under the old position their questions would reach them through a surface owned
by a harness that knows nothing about roles, or not reach them at all.

## Decision

**abcd owns the product thinker's surface.** A stop addressed to the product
thinker is delivered through a surface abcd is responsible for, rendered in the
register that role is owed, and answered there.

The original position holds everywhere else. **abcd does not render the
facilitator's interface**: the harness owns that, the command line remains where
a facilitator works, and nothing here licenses abcd to draw a developer-facing
interface it has no business owning.

The mediated path is not replaced. Where a human facilitator and a product
thinker are together, the facilitator reading a question aloud and entering the
answer stays valid and needs nothing new.

## Consequences

The product thinker's surface becomes a thing abcd builds, deploys and keeps
working, which is a materially larger commitment than a configuration layer
carries today. It is the first artefact abcd owns that a user does not obtain by
installing a binary into a repository.

Delivery to that surface is asynchronous by construction, so the stop protocol
must serialise a question whole, carry its options, outlive the session that
raised it, and accept an answer that arrives later from somewhere else.

The narrower position is harder to hold at the edges than the original absolute
one. A refusal, a banner and a status line are all words at a terminal, and the
line between explaining a stop to a facilitator and rendering a product
thinker's interface will need judgement rather than a rule.

## Alternatives rejected

**Keep the absolute position.** Coherent, and it leaves the framework's target
user with no way to be asked anything, which makes the loop unclosable for
everyone except the person who also happens to be the facilitator.

**Treat the product thinker's surface as a separate product.** Preserves the
position by definition rather than by argument, and the dependency remains: the
framework's central claim would rest on a component the framework does not own
and cannot guarantee.
