# releases/

The archive of release pages. Each file is the `RELEASE.md` of one earlier
feature release, named after the version its own heading names
(`<version>.md`), and kept byte for byte as the release left it.

The pages are written by the release cut, never by hand. When `abcd launch ship`
writes a new `RELEASE.md` at the repository root, it moves the page it replaces
here first; a cut that ships fixes alone writes no page, so nothing moves. The
root page is always the latest feature release, and the history is this folder.

A page is composed from the press releases of the intents its release shipped,
and quotes them word for word, which is why `forbidden_synonyms` exempts this
folder as it exempts `intents/shipped/`. Every other record rule still reads it.
The procedure is `commands/launch.md` (step 3 of the ship).
