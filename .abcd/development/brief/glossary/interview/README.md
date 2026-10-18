# Interview context

The `interview` bounded context holds terms specific to abcd's interviews: the
knowledge floor every abcd interview's questions are held to, and the two terms
of the grill session `/abcd:intent` is designed to run — the interactive
exchange (session) and its opening move (embark). The grill is a design target
that does not ship; the interviews that ship, the planning interview among
them, are not grill sessions. These pin vocabulary that means something
narrower here than in the core context.

Each term is a Markdown file with YAML frontmatter in the shape the glossary
README specifies. For that shape, the checks that read it, and the full term
index, see the [glossary README](../README.md).
