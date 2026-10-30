---
id: dcn-2610301507014930
slug: how-a-judgement-route-picks-the-decision
date: 2026-10-30
---
2026-10-30 — How a judgement route picks the decision adapter's second candidate, the small text model scored by log-probability (itd-2609221009495079, spc-2609221011151661; lane implementer on the run coordinator's delegation): a provider block on the person's own machine lists, under logprob_models, the models from its own models list that are asked that way, and a route to one of them is judged by the log-probability candidate while every other route keeps the first, typed-decision candidate. No new route prefix and no new judgement mode: the route grammar (host, shadow:<provider>/<model>, <provider>/<model>) stays as it is, the provider block already holds what is per model (its allowlist and pins) and sits on the machine layer alone, so a repository cannot change how a model is asked, and the field marks only a listed model, so it admits nothing the allowlist does not. Both candidates call through the same APIConfig.Call, so the allowlist, the denylist, the key by name and the router's window check apply to both unchanged; with logprob_models absent every path is as before.
