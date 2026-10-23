package worker

import "testing"

func TestVerificationReportRequiresExecutedCasesAndCompleteFraming(t *testing.T) {
	cases := []struct {
		format, raw             string
		valid                   bool
		passed, failed, skipped int64
	}{
		{"junit", `<testsuites><testsuite><testcase name="real"/><testcase name="failure"><failure message="x"/></testcase><testcase name="skip"><skipped/></testcase></testsuite></testsuites>`, true, 1, 1, 1},
		{"junit", `<testsuite tests="1"/>`, false, 0, 0, 0},
		{"junit", `<testsuite><testcase/></testsuite><testsuite><testcase/></testsuite>`, false, 0, 0, 0},
		{"junit", `<testsuite failures="1"><testcase/></testsuite>`, false, 0, 0, 0},
		{"junit", `<testsuite><testcase><skipped/></testcase></testsuite>`, false, 0, 0, 0},
		{"junit", `<testsuite><testcase/><error/></testsuite>`, false, 0, 0, 0},
		{"junit", `<!DOCTYPE fake><testsuite><testcase/></testsuite>`, false, 0, 0, 0},
		{"junit", `<testsuite><testcase>`, false, 0, 0, 0},
		{"tap", "TAP version 13\n1..3\nok 1 - checks\nnot ok 2 - fail\nok 3 - optional # SKIP platform\n", true, 1, 1, 1},
		{"tap", "1..0\n", false, 0, 0, 0},
		{"tap", "1..2\nok 1\n", false, 0, 0, 0},
		{"tap", "1..1\nok 1 # SKIP all\n", false, 0, 0, 0},
		{"tap", "1..1\nnot ok 1 # TODO later\n", false, 0, 0, 0},
		{"tap", "1..1\nok 1\nok 1\n", false, 0, 0, 0},
		{"tap", "1..1\nBail out!\n", false, 0, 0, 0},
		{"exit", "0", false, 0, 0, 0},
	}
	for _, tc := range cases {
		counts, err := parseTestReport(tc.format, []byte(tc.raw))
		if (err == nil) != tc.valid {
			t.Errorf("%s %q validity=%v: %v", tc.format, tc.raw, tc.valid, err)
		}
		if tc.valid && (counts.Passed != tc.passed || counts.Failed != tc.failed || counts.Skipped != tc.skipped) {
			t.Errorf("wrong observed counts: %+v", counts)
		}
	}
}
