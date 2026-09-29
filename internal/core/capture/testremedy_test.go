package capture

// testRemedy is the remedy a test files with when the remedy is not what it
// tests. Every new issue carries one (ruling BX3 of 2026-09-29), so the
// package's tests default it here rather than each naming one.
const testRemedy = "a remedy this test does not read"

// testCapture is Capture with testRemedy filled in where the request names
// none. A test of the remedy itself calls Capture directly.
func testCapture(req CaptureRequest) (CaptureResult, error) {
	if req.Remedy == "" {
		req.Remedy = testRemedy
	}
	return Capture(req)
}
