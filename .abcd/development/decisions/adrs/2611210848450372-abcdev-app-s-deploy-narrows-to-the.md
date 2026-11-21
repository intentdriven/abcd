---
id: adr-2611210848450372
slug: abcdev-app-s-deploy-narrows-to-the
status: accepted
date: 2026-11-21
supersedes: null
superseded_by: null
related_intents: [itd-2610142145236613]
related_rfcs: []
related_adrs: [adr-48, adr-47, adr-2610142147242544]
---

# ADR-2611210848450372: abcdev.app's deploy narrows to the documentation, and both sites deploy on release from released bytes

## Context

[adr-48](0048-website-deploys-on-release-not-on-merge.md) deploys abcdev.app at each release, from the tag and the released, verified binary, with a labelled preview of `main` on every push. In that deploy the record explorer and the documentation are one site.

[adr-2610142147242544](2610142147242544-abcd-s-site-is-built-onto-its-own-github.md) moves abcd's record onto abcd's own GitHub Pages address, built by the project route every opted-in project gets, and keeps abcdev.app as abcd's documentation site. Its point 2 narrows abcdev.app's deploy and its point 3 forwards the record's old addresses. spc-2610142146250176 asks for a new decision record that amends adr-48 to match. It also asks that adr-48's rule (on release, from released bytes) holds for both sites.

Step 3 of that spec builds the narrowing and the forwarding into `abcd site build`, behind two switches in `.abcd/site.json` that start off. Step 4, the cut-over, turns them on, and each of its outward acts is the person's to confirm (dcn-2611201429119345).

## Decision

This record amends adr-48.

1. **abcdev.app's deploy narrows to the documentation.** Once `record.elsewhere.narrow` is on in `.abcd/site.json`, the deploy builds:
   - the documentation and its landing page;
   - the contributors page (`/contributors/`) and the references page (`/references/`), which the project route does not build (dcn-2611201438402486).

   It builds none of the record's pages and no `record.json`, and every link it draws into the record names the record's Pages address.
2. **The record's old addresses forward.** Once `record.elsewhere.forward` is on, `/record`, `/record/` and every page below it, and `/record.json` answer with a 301 to the same path at the record's Pages address. The rules are derived at build time from `record.elsewhere.address` and written into the built `_redirects`, so no rule is kept outside the repository. The documentation's own 301s for the paths moved under `/docs/` stay.
3. **Both switches start off, and each is turned on in a committed change of its own.** `forward` is turned on before `narrow`, each on the person's word, once the act before it is checked. With both off, the deploy builds exactly what it built before, so merging the code and the release after it change nothing public.
4. **adr-48's rule holds for both sites.** abcdev.app still deploys on release, from the tag and the released, checksum-verified binary, with the labelled preview of `main` and the emergency redeploy from the latest tag. abcd's record on its Pages address follows the same rule: the workflow `site setup` writes renders each published release with abcd's released, verified binary. The record has no preview of `main`, because a repository has one Pages site. The site-render gate still renders and checks every change.

## Alternatives Considered

- **Keep adr-48 as it is, with the record and the documentation on one deploy.** This needs no change. But the record would not run on the route abcd offers every project, which adr-2610142147242544 decided against.
- **Narrow and forward in one change, with no switches.** This is simpler, but the merge would then move the public site. Every outward act of the cut-over would be bundled into an ordinary merge and the next release, and none could be confirmed or checked on its own.
- **Write the forwarding rules by hand in `site-src/redirects`.** The rules would be visible in one file. But the address would then live in two places, the rules and the links, and they could disagree. Each switch would also stop being a single committed flip.
- **Two switches, derived rules, forward before narrow (chosen).** Each outward act is one committed change the person confirms. A failure at any step leaves the record where it was.

## Consequences

- The deploy's output depends on `.abcd/site.json` at the released tag. A switch turned on reaches abcdev.app at the next release, not at the merge, as adr-48 has it. Only the labelled preview of `main` shows it at the merge.
- Once `narrow` is on, abcdev.app's screenshots audit can no longer sample the record from the site it serves. The change that turns `narrow` on also moves the record's layout audit to a build of the project route (spc-2610142146250176, the cut-over).
- abcd's record depends on GitHub Pages being available, and on its limits. abcdev.app keeps its own response headers. The record loses framing refusal and the no-sniff header, as adr-2610142147242544 point 4 records.
- adr-48 stays accepted and is not superseded. Its decisions 1 to 5 still govern what the deploy builds.
