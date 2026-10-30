# How to acknowledge prior art without one shared file: a state-of-the-art check

Date: 2026-10-30. Asked for by the product thinker, as the grounds of the
remedy of the issue that proposes a comprehensive system for acknowledging
prior art. Recency was judged against this date. Evidence tiers: **A** a
standard or peer-reviewed work, **B** a rigorous industry account or a
published schema, **C** a practitioner report. Every link below was opened on
that date; the FORCE11 software citation principles and APA Style's own page
on secondary sources could not be opened, so the Purdue OWL restatement is
used instead.

## The question

abcd keeps one hand-edited `ACKNOWLEDGEMENTS.md` (Development, Inspirations, a
numbered References list mirrored in
`.abcd/development/research/references.csl.json`). Many agent sessions append
to it at once, so it conflicts on merge, and nothing decides what qualifies
for an entry, so sources get cited in the record and never credited (three
such misses were found on 2026-10-30). The proposal: one file per
acknowledged source in a folder, the root file a symlink to a generated
README, regenerated only when a release is cut, and deterministic rules for
what qualifies.

## What the evidence says

1. **One file per entry, aggregated at release, ends the merge conflicts**
   (tier B). GitLab saw "merge conflicts in the changelog when multiple merge
   requests attempted to add an entry", and fixed it with one file per entry
   in an `unreleased/` folder, compiled when a release is tagged
   (<https://about.gitlab.com/blog/2018/07/03/solving-gitlabs-changelog-conflict-crisis/>).
   towncrier (<https://towncrier.readthedocs.io/en/stable/markdown.html>),
   reno (<https://docs.openstack.org/reno/latest/user/usage.html>, which gives
   each note a random 16-hex suffix and refuses a collision) and changesets
   (<https://changesets.dev/guide/why>) do the same. abcd already does it for
   its changelog, so this is the existing pattern applied to a second
   artefact. The id is non-sequential (abcd's allocator, adr-45): a numbered
   list in source is itself a conflict surface, so numbers belong to the
   render.
2. **Each entry is one CSL-JSON item, with abcd's own fields under `custom`**
   (tier B). CSL-JSON requires only `type` and `id`, refuses unknown
   top-level keys, and reserves a `custom` object for data with no field of
   its own
   (<https://raw.githubusercontent.com/citation-style-language/schema/master/schemas/input/csl-data.json>).
   So an entry carries the citation as CSL plus `custom` fields such as the
   kind of credit, what it was credited for, the records that adopted it, and
   the secondary source it came through. The committed CSL-JSON file becomes
   a concatenation of the entries. A `CITATION.cff` `references` list can be
   derived from the same entries
   (<https://raw.githubusercontent.com/citation-file-format/citation-file-format/main/schema.json>);
   CodeMeta has no "is based on" term (<https://codemeta.github.io/terms/>).
3. **What qualifies can be written as tests** (tier A for the norms, C for the
   mechanism):
   - an idea a record adopts (an ADR, an intent or a decision names it as the
     origin of a design) is credited, and the entry lists that record;
   - a source a record cites as evidence (a DOI or URL) is credited, and the
     citation resolves to an entry;
   - a tool whose artefact lands in the tree (a hook, a config, a generated
     file) is credited;
   - the entry credits the primary source; a summary or a secondary account
     is recorded as the route it came through, never as the credited work.
     Writers "should endeavor to read primary sources (original sources) and
     cite those rather than secondary sources" (Purdue OWL on APA,
     <https://owl.purdue.edu/owl/research_and_citation/apa_style/apa_formatting_and_style_guide/in_text_citations_author_authors.html>);
   - software is credited as the software, at the version used: "Do not cite
     the article instead of the software" (Katz et al., *F1000Research*,
     <https://f1000research.com/articles/9-1257>);
   - a named person is acknowledged only with their consent: contributors who
     are not authors "should be acknowledged", with "written permission to be
     acknowledged" (ICMJE,
     <https://www.icmje.org/recommendations/browse/roles-and-responsibilities/defining-the-role-of-authors-and-contributors.html>);
     CRediT's role vocabulary can be borrowed, though it "does not define
     degree" (<https://credit.niso.org/>);
   - licence notices are a different artefact: Apache-2.0 section 4(d)
     governs NOTICE files (<https://www.apache.org/licenses/LICENSE-2.0>), and
     stay out of the acknowledgements, as `go.mod` dependencies already do.
4. **The root file is a real generated file, not a symlink** (tier B against
   the symlink). Go's module zip states "Symbolic links and other irregular
   files are not allowed", and omits them
   (<https://pkg.go.dev/golang.org/x/mod/zip>). With `core.symlinks=false`
   "symbolic links are checked out as small plain files that contain the link
   text" (<https://git-scm.com/docs/git-config>). GitHub shows a symlinked
   file's target path rather than its content (two practitioner reports,
   <https://github.com/orgs/community/discussions/202021>; tier C), and
   GitLab's own issue tracker confirms the same there
   (<https://gitlab.com/gitlab-org/gitlab/-/issues/15093>).
5. **Regenerate at release, check the entries continuously** (contested on
   timing; tier C). Regenerating the root file on every merge puts it back in
   every pull request, the conflict being removed, and a bot's post-merge
   commit is refused by abcd's attribution gate. Release-time regeneration is
   the common choice (GitLab, towncrier, Rust's release notes,
   <https://forge.rust-lang.org/release/release-notes.html>). The root file
   then lags the default branch by up to one release, which the site, built
   from the entries, does not. The checks that run on every change are on the
   entries (schema, citation closure); the regenerate-and-compare check runs
   in the release cut.
6. **A commit trailer as the only record is weaker** (contested). GitLab moved
   its changelog to `Changelog:` trailers in 2021 to stop writing the same
   facts twice (<https://docs.gitlab.com/development/changelog/>), but a
   trailer cannot carry authors, DOI or version, and squash or rebase merges
   rewrite the commits it lives on.

## Not worth adopting

- The All Contributors bot: it credits people only, and its commits come from
  a bot, which abcd's attribution gate refuses.
- REUSE or Debian `copyright` files as the schema: they inventory licences per
  file, the right tool for NOTICE obligations and the wrong one for credit.
- Regenerating the root file on every merge, sequential reference numbers in
  source, and a symlinked root file (points 1, 4 and 5).
