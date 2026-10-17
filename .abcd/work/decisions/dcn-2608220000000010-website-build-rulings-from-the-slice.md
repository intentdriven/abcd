---
id: dcn-2608220000000010
slug: website-build-rulings-from-the-slice
date: 2026-08-22
---
2026-08-22 — Website build rulings, from the slice reviews (the interview's
nine rulings stand; these refine them): the featured-quote derivation skips
a press release that is one of the two minted placeholder templates —
mechanical template match, no other filtering, order still entered-shipped
descending then id descending. `abcd site check` holds `/references/` to
the composed rules even though the manifest selects no span there — a page
must not escape the single-source rule by being unnamed — and implements
the attribution escape as verification against the trailers git carries,
never as an exemption. The deploy stamp contract: production passes
`--version` bare (the renderer prepends `v`; `vv0.6.1` fails the gate) and
previews pass `--preview` (renders `unreleased · <commit>`); `--date` is
never injected — the CHANGELOG heading is the release date per adr-37.
`install.md`'s lead lands with the first successful deploy, reading
spc-40's "same change that first serves the script" as the deploy, so the
docs never name an endpoint that 404s. Build order is fixed: `site build`
first, `mkdocs -d site/docs` second — the purge makes the reverse loud.
