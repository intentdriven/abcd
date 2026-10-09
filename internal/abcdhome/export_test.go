package abcdhome

// OldName exposes the retired home folder's name to the external tests that
// stage a store under it. Those tests are external (package abcdhome_test)
// because they build their fixtures with internal/gittest, which imports
// internal/gitutil, which imports this package: an internal test importing
// gittest would be an import cycle.
const OldName = oldName
