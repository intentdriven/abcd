---
schema_version: 1
id: "iss-2608301908288212"
slug: "four-nits-from-the-itd-179-fix-delta-review-including-a-body"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-179-fix-delta-ruthless"
found_at: "internal/core/grounds/record.go"
---

four nits from the itd-179 fix delta review including a body line offset that is one ahead of the rendered body

Four nits from the itd-179 fix-delta ruthless review, settled at the ship commit.

1. `record.go` / `frontmatter.go` -- "body line N" counts from
   `frontmatter.Split`'s body, which keeps the blank line after the closing
   delimiter, while `capture`'s reader strips it. So the number is one ahead of
   the body a reader renders. The quoted opener text is the reliable locator;
   `Split`'s doc claim that writer and readers judge the same bytes is true of
   the SECTION and off by one line of the offset.
2. `record.go` and `intent/grounds.go` -- two doc claims say a bare body may be
   passed and comes back unchanged. No live caller passes one, and `Body` does
   not in fact take either: a text opening with a thematic break reads as a
   frontmatter opener and 67 bytes including the whole section are eaten.
   Unreachable today; the claim invites the caller that would reach it.
3. `record.go` -- the ambiguity refusal always says "`## Grounds` headings", but
   the pattern matches any depth, so a record with a `### Grounds` subsection is
   refused by a message naming a level it does not have.
4. `record.go` -- the frontmatter-only splice gains a blank line the whole-file
   form did not produce. Cosmetic; no real record has an empty body.

Three more from the fix-delta SECURITY review, which returned PASS and offered
these as nits. Item 2 above is the same claim it names independently, so the two
reviews converged on it.

5. `capture/serialize.go` -- `frontmatterBounds` accepts blank lines before the
   opening delimiter while `parseFrontmatterAndBody`, `frontmatter.Fields` and
   `frontmatter.Split` all require line 0. The reviewer found 544 divergent
   shapes and then chased the worst one end to end: it FAILS CLOSED, both verbs
   refuse and the file is byte-identical afterwards with no draft minted.
   Adjacent to the open iss-2608270908348042.
6. `grounds/record.go` -- the ambiguous-heading refusal names the count but no
   line numbers, where its sibling `readBackRefusal` names a body-relative line
   and quotes the opener. Same file, two standards of diagnosis.
7. `capture/roots.go` -- `readWithChecksum` is uncapped where intent records get
   256 KiB, so the refusal's `%q` can quote an arbitrarily long body line into an
   error string. Cosmetic.

