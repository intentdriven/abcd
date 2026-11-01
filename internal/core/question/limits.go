// Package question holds what a question abcd puts to a person may contain:
// the field limits, written once (Limits, Default), the field view every front
// door finally shows (Fields), and the limits check (CheckLimits) that refuses
// a badly built question naming the part, the value, and the limit
// (spc-2610030944505997; itd-2610030810350727, itd-201).
//
// Nothing else in the tree states a limit. The question check in the guard
// hook, the GRILL rule text and the interview pages read Default, so changing a
// limit here changes every statement of it (itd-2610030810350727 criterion A7).
//
// The package is a library: it returns values and never prints. It measures
// text through internal/textwidth, a pure leaf, and imports nothing that does
// terminal I/O.
package question

import "fmt"

// Limits is every bound a question's fields are held to. question.Default is
// the one value the check, the GRILL domain, and the interview pages read
// (itd-2610030810350727 decision 14, criterion A7).
type Limits struct {
	HeaderColumns     int      // 12: the host's header chip
	ChipRoles         []string // "Product", "Tech", "Setup": the chip's first word
	QuestionsPerCall  [2]int   // 1..4: one question, or up to four tabs
	OptionsPerQ       [2]int   // 2..4, the decide-later option included
	LabelWords        int      // 5
	MeaningSentences  int      // 2: an option's description
	LaterLabels       []string // "Decide later", "None of these"
	Columns           int      // 80: the narrow window promised
	Rows              int      // 32: one question, or one tab, at Columns
	HostTextColumns   int      // 76: the question text's measure inside the host's frame at Columns
	HostChromeRows    int      // 8: rows the frame draws around a question, the chip included
	HostOptionColumns int      // 74: the measure of an option's label and description at Columns
	NowPrefix         string   // "Now:"
	ChangeLaterPrefix string   // "Change later:"
	NotApplicable     string   // "not applicable"
}

// ProductRole is the chip role word that names the product thinker. Where no
// mode names the addressee, a chip carrying it stands in for the mode (rule 9).
const ProductRole = "Product"

// Default is the one statement of every limit.
var Default = Limits{
	HeaderColumns:    12,
	ChipRoles:        []string{ProductRole, "Tech", "Setup"},
	QuestionsPerCall: [2]int{1, 4},
	OptionsPerQ:      [2]int{2, 4},
	LabelWords:       5,
	MeaningSentences: 2,
	LaterLabels:      []string{"Decide later", "None of these"},
	Columns:          80,
	// Rows is the bundled default, 32, and not the floor: the floor of
	// question.rows (Bounds) stays 24, the standard screen and the narrowest
	// window abcd promises, which every question the binary composes fits. The
	// default is what a question may take where nobody sets anything; a team
	// whose screens are the standard 24 sets the floor
	// (itd-2610301914350541 decisions 3, 17 and 18).
	Rows: 32,
	// The three host figures are measured, not chosen: step 5 of
	// spc-2610030944505997 read them on 2026-10-03 from Claude Code 2.1.288's
	// question view at 80 by 24 in macOS Terminal
	// (.abcd/development/research/notes/2026-10-03-host-question-layout-calibration.md).
	// The question text wrapped a line of 72 columns before an eight-column
	// word, so its measure is 76. An option's description, indented five
	// columns, wrapped after 74. Without a preview the frame drew eight rows:
	// the chip, a blank, the blank before the options, the free-text row, the
	// separator, the chat row, a blank and the footer.
	HostTextColumns:   76,
	HostChromeRows:    8,
	HostOptionColumns: 74,
	NowPrefix:         "Now:",
	ChangeLaterPrefix: "Change later:",
	NotApplicable:     "not applicable",
}

// The keys a repository sets in .abcd/config.json, under the question
// namespace, and that a person's own ~/.abcd.noindex/config.json may set lower
// (itd-2610301914350541 decisions 1 and 5). Only these three are configurable;
// every other figure in Limits stays the binary's (decision 2).
const (
	RowsKey                 = "question.rows"
	LabelWordsKey           = "question.label_words"
	DescriptionSentencesKey = "question.description_sentences"
)

// Bound is the inclusive range a configurable limit may be set to. A limit
// with no bounds could switch a protection off with a legal value (a thousand
// rows, or a label the width of a description), so each key has a floor and a
// ceiling (itd-2610301914350541 decision 3).
type Bound struct {
	Key     string
	Floor   int
	Ceiling int
}

// String renders the bound as every refusal, note and page states it: "24 to
// 60".
func (b Bound) String() string { return fmt.Sprintf("%d to %d", b.Floor, b.Ceiling) }

// Admits reports whether n lies inside the bound, both ends included.
func (b Bound) Admits(n int) bool { return n >= b.Floor && n <= b.Ceiling }

// Bounds is the one statement of the bounds, in the order Bounded takes its
// values: rows, label words, description sentences. The floor of each is drawn
// so that every question the binary composes is admitted whole at it, and the
// ceiling where the protection would stop protecting (itd-2610301914350541
// decision 3):
//
//   - question.rows, 24 to 60: the standard screen, up to a full-height
//     terminal on a large display; past it "fits one screen" protects no
//     screen a person has.
//   - question.label_words, 4 to 8: the longest label the binary composes is
//     four words ("Yes, make the change"); past eight a label wraps at the
//     option measure and reads as a description.
//   - question.description_sentences, 2 to 4: the longest description the
//     binary composes is two sentences; past four a description holds
//     material, which belongs in the question text.
//
// Nothing else in the tree writes a bound: the resolver's refusal, the
// loader's note and the generated page sentence render these.
var Bounds = []Bound{
	{Key: RowsKey, Floor: 24, Ceiling: 60},
	{Key: LabelWordsKey, Floor: 4, Ceiling: 8},
	{Key: DescriptionSentencesKey, Floor: 2, Ceiling: 4},
}

// boundFor is the bound of key, and false for a key that is not configurable.
func boundFor(key string) (Bound, bool) {
	for _, b := range Bounds {
		if b.Key == key {
			return b, true
		}
	}
	return Bound{}, false
}

// Configurable is l's three configurable figures, in Bounds' order.
func (l Limits) Configurable() [3]int {
	return [3]int{l.Rows, l.LabelWords, l.MeaningSentences}
}

// Bounded is l with its three configurable fields replaced by rows,
// labelWords and sentences, every other field unchanged. It judges nothing:
// the resolver holds the values to Bounds before it calls it.
func Bounded(l Limits, rows, labelWords, sentences int) Limits {
	l.Rows, l.LabelWords, l.MeaningSentences = rows, labelWords, sentences
	return l
}

// FloorsForTest is l at the floor of every configurable key, and CeilingsForTest at the
// ceiling of every one. They are test seams, named so: every question the binary
// composes is held to FloorsForTest(Default), and every combination of floors and
// ceilings to the fixed rules.
func FloorsForTest(l Limits) Limits {
	return Bounded(l, Bounds[0].Floor, Bounds[1].Floor, Bounds[2].Floor)
}

// CeilingsForTest is l at the ceiling of every configurable key (see FloorsForTest).
func CeilingsForTest(l Limits) Limits {
	return Bounded(l, Bounds[0].Ceiling, Bounds[1].Ceiling, Bounds[2].Ceiling)
}
