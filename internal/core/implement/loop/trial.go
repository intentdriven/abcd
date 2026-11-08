package loop

// trial.go is the build loop's trial mode (itd-2610161733316518 decision 4,
// spc-2610161949559050 step 3). A model-matrix copy runs the loop's stages in
// it: the run takes no reservation on the shared remote and no claim in the
// shared run, and its landing lands nothing. Copies of one intent are many
// runs of one record, so a reservation would make the second copy back off the
// first's footprint, and a copy that landed would contaminate every re-run of
// the feature (decision 3: no copy lands). The feature itself is built
// afterwards through the ordinary loop, and a trial run is never resumed as a
// build of its record nor a build as a trial.

// StageTrial is the run record's stage for the line a trial run's start writes.
const StageTrial = "trial"

// trialNote is the record's line at a trial run's start.
const trialNote = "a model-matrix trial: the run takes no reservation and lands nothing"

// trialLandNote is the landing's line for a trial lane.
const trialLandNote = "a model-matrix trial lands nothing: no push, no merge, and no record moved"
