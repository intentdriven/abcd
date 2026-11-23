package intent

// IngestVerdict exposes the file-path form of the verdict ingest to the
// external tests (package intent_test) that stage a verdict on disk; every
// front door reads the payload itself and calls IngestVerdictBytes.
var IngestVerdict = ingestVerdict
