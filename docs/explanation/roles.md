# Roles

`abcd` is built for three roles: A **product thinker**, a **technical facilitator**, and a **team of AI agents**. The product thinker is always a person. The facilitator is a person in a fresh install, and a repository can hand the merge approval to abcd with `facilitator.mode: auto`; which of the two holds the role changes what waits for approval rather than what the role is. The agents do the building.

## Product thinker

![The product thinker](../assets/img/role-product-thinker.png)

As a **product thinker**, you know who the user is. You know what *done* looks like when you see it. You know which trade-offs are acceptable and which would betray the point of the project.

You judge three things, and nothing else. What should exist, why, for whom, and what *good* means. Whether what was delivered matches what you asked for. And, once you are using the thing, whether it behaves as you expected: that last judgement is the one no review can make for you, and it arrives long after the work is done.

You do not judge how the software was built, and `abcd` does not ask you to. Anything it puts in front of you is written in plain language, carries a concrete example from the product you are building, states a trade-off as the options it is choosing between, and asks you to pick rather than to compose an answer. Making that register a setting of yours, so you can change how you are addressed when it does not work for you, is a design target, not yet shipped.

What happens in between, turning your why into engineering work agents can act on, is the facilitator's job.

## Technical facilitator

![The facilitator](../assets/img/role-facilitator.png)

The **facilitator** is a *translator*, not an engineer-on-the-team in the traditional sense. The work is to take what you wrote, shape it into plans an AI coding agent can execute well, run the audit and review machinery, and answer the questions the agents raise while they build.

A fresh install starts with a person in this role, and a person in it changes what the loop waits for, so a question that a machine facilitator would answer and move past becomes a question that waits for someone. The command line is where a facilitator works, whichever kind it is.

Every judgement about evidence, technique, and whether something was verifiable at all stops here. Only a decision the product thinker can actually answer travels further.

## The agent team

The agents plan, build, and review inside a gated loop. They run on their own and stop only to obtain a verdict, and a stop always says who it is asking and what it needs. Work that is a step in the loop rather than a judgement is simply done.

Which role each of these decisions belongs to, and what may be addressed to whom, is settled in [ADR-2609151528057260](https://github.com/intentdriven/abcd/blob/main/.abcd/development/decisions/adrs/2609151528057260-three-roles-who-each-artefact-addresses.md). Who answers for software that the accountable person will not read is not settled, and is open in [RFC-3](https://github.com/intentdriven/abcd/blob/main/.abcd/development/roadmap/rfcs/rfc-3-the-facilitators-role-and-responsibility-for-generated-code.md).
