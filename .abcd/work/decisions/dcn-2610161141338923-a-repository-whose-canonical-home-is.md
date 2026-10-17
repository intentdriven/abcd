---
id: dcn-2610161141338923
slug: a-repository-whose-canonical-home-is
date: 2026-10-16
---
2026-10-16 — A repository whose canonical home is itself a GitHub fork allows reservations there with reservation.fork_origin set to "allow" in .abcd/config.json (facilitator-on-delegation, drain-many-accounts): the key takes "allow" or "refuse", the bundled default, and is read through the layered configuration, the repository file winning over the machine file, only once the forge has named an origin push URL a fork, so a malformed key can stop no act on a repository that is not a fork; the fork refusal names the key (grounds: a fork that is the shared home of the work is where every account reserves, so the refusal would only stop it reserving; whether origin is the shared home is a fact about the repository, so its committed file may say so; refines dcn-2610161040378209).
