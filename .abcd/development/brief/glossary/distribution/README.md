# Distribution context

The `distribution` bounded context holds terms for publishing and consuming abcd:
what a release is, the version it carries, and the end-user who installs it. These
distinguish the outputs of publishing from the internal development vocabulary —
an end-user is not a persona, and a version is not a sequencing unit.

Each term is a Markdown file with YAML frontmatter in the shape the glossary
README specifies. For that shape, the checks that read it, and the full term
index, see the [glossary README](../README.md).
