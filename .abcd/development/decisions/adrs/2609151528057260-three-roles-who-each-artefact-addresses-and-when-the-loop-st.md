---
id: adr-2609151528057260
slug: three-roles-who-each-artefact-addresses-and-when-the-loop-st
status: accepted
date: 2026-08-29
supersedes: null
superseded_by: null
related_intents: []
related_rfcs: [rfc-3]
related_adrs: []
---

# ADR-2609151528057260: Three roles, who each artefact addresses, and when the loop stops

## Context

The roles page describes a two-person team of a product thinker and a
technical facilitator, and names automating the facilitator as an aim. That
description no longer matches how the framework is meant to work, and the
mismatch is not cosmetic: it decides what the machinery is allowed to do
without asking.

Three defects follow from the two-role description.

**A decision that belongs to one role gets argued as if it belonged to the
other.** Treating both roles as "the human" makes every facilitator duty look
like something that must wait for a person. That reasoning was used, in this
repository, to argue that closing a specification after its implementation
landed had to stay manual because shipping an intent is a maintainer decision.
It is not: declaring a promise delivered is facilitator work, and the argument
only looked sound because the two roles were collapsed into one word.

**An artefact promised to the product thinker is written for an engineer.** The
roles page promises the product thinker a moment of reading the verdict on
whether the why was delivered. The verdict that exists is a per-criterion
acceptance JSON whose every claim carries a `file:line` evidence pointer. It is
the correct artefact for a facilitator and unreadable for the role it is
promised to.

**Nothing says when the agents may proceed alone.** Without a rule, every gate
is a candidate for a human wait, and the framework's premise reverses: the
human becomes the bottleneck the framework exists to remove.

## Decision

**The team is three roles, not two.** A **product thinker** (always human)
states what should exist, why, for whom, and what good looks like. A
**technical facilitator** translates that into work agents can execute, runs the
review and audit machinery, and answers when the loop stops. A **team of AI
agents** implements. The facilitator is the only role whose occupant varies.

**The facilitator is AI by default and human on activation.** Activation is a
mode, not a rewrite: it changes what waits for approval, and nothing else. Every
gate declares its behaviour in both modes, because a gate that only makes sense
with a person watching is a gate that fails silently when nobody is.

**The agents run autonomously and stop only to obtain a verdict.** A stop is an
event with an addressee and a question, never an unexplained halt. Anything that
is a step in the loop rather than a judgement is performed, not queued for
someone.

**Escalation reaches the product thinker only for what they can answer.** That
is: what should exist, whether what was delivered matches it, and trade offs
that change the design. Everything else stops at the facilitator, including
every question about evidence, technique, and whether a criterion was verifiable
at all. **A stop addressed to the product thinker that the product thinker
cannot answer is a defect in the stop, not a failure of the reader.**

**Anything addressed to the product thinker is written in a register the
product thinker controls.** The register is a configured profile with defaults
that hold until its owner changes them, and its owner is the product thinker,
not the facilitator: A reader who cannot follow the answers they are given must
be able to change how they are addressed without asking anyone. The defaults
are plain language, no filler, at least one concrete example drawn from the
product under construction rather than from the framework, trade offs stated
concretely with their options named, and **selection over composition**, so the
ordinary way to answer is to choose among options rather than to compose prose.
No `file:line` pointers and no internal identifier standing alone. An artefact
may carry a facilitator-addressed body and a product-thinker-addressed summary;
what it may not do is address one role in the register of the other.

## Consequences

Addressee becomes a property of an artefact rather than an assumption about its
reader, and a checkable one: presence of an example, absence of location
pointers, and a bounded vocabulary are all mechanical. Because the register is
configured rather than fixed, the check reads the profile rather than a constant,
and a repository whose product thinker has widened or narrowed it stays
conformant on their terms.

Every stop that reaches the product thinker owes a set of options. A question
that can only be answered in free prose is either mis-addressed or not yet
thought through, and the requirement surfaces that while it is still cheap to
fix.

A stop addressed to the product thinker is a durable artefact, not a prompt on
somebody's terminal. Its answer may arrive later, from a person who was not
present when it was raised, through a surface that is not the command line: A
stop therefore serialises whole, carries its options with it, survives the
session that raised it, and is resumable by whatever answers it. Nothing in the
loop may assume the product thinker is watching, and a stop that only exists as
a question printed to a terminal does not satisfy this. The synchronous
command-line prompt is the first implementation of that contract, not the
contract.

**Addressee is not surface.** The command line is the facilitator's surface, and
a human facilitator works there always; the product thinker does not work there
at all. The two are nonetheless often in the same room or the same call, with
the facilitator reading a question aloud and entering the answer given back. A
product-thinker-addressed artefact therefore keeps its register even when it is
rendered on a terminal, because the person it is written for is listening rather
than reading. The register follows who is being asked, never where the text
appears.

The intent-fidelity verdict gains a second face. Its per-criterion evidence
stays as it is, for the facilitator; a plain-language summary of whether the why
was delivered is what the product thinker reads.

Work that no role must judge stops waiting. Closing a specification once its
implementation has landed is the immediate case.

Responsibility does not follow the work. The product thinker remains
accountable for software they will not read, and the facilitator that reviews it
may itself be a machine. This ADR settles who acts; it deliberately does not
settle who answers for the result. That question is open in
[RFC-3](../../roadmap/rfcs/rfc-3-the-facilitators-role-and-responsibility-for-generated-code.md).

## Alternatives rejected

**Keep two roles and treat automation as an aim.** Honest about the present, but
it leaves every facilitator duty arguable as a human gate, which is the defect
that produced the specification-closing mistake.

**Make the facilitator always human.** Removes the ambiguity and the framework's
point together: the product thinker could not run the process alone, which is
the thing being built.

**Let the agents escalate freely to whoever is available.** Cheapest to
implement and the fastest way to teach a product thinker that the stops are not
worth reading, because most of them would be questions they cannot answer.
