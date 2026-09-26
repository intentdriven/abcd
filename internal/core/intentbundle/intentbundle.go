// Package intentbundle is the bundle's record vocabulary as DATA: the words a
// bundle's survivor records its bundle of one in (itd-34, decision 3).
//
// It is a leaf on the core/condition precedent. The WRITER that states a
// bundle of one (`abcd intent reclassify`, in core/intent) and the GATE that
// accepts it as the one legitimate bundle of one (record_schema, in core/lint)
// must read the same words, and core/intent's tests import core/lint, so a lint
// importing intent back is an import cycle. Two hand-kept copies of the phrase
// are the drift this package exists to rule out: a writer that rewords its line
// would leave every survivor it writes refused by the gate.
package intentbundle

// OneMember is how a survivor's `reclassification_history` states that bundle
// name now has one member: the writer's history line begins with it, and the
// gate looks for it in the survivor's history.
func OneMember(name string) string {
	return "bundle " + name + " now has one member"
}
