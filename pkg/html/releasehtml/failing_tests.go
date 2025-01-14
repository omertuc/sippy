package releasehtml

import (
	"fmt"

	"github.com/openshift/sippy/pkg/html/generichtml"

	"github.com/openshift/sippy/pkg/util"

	sippyprocessingv1 "github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
)

func summaryTopFailingTestsWithBug(topFailingTestsWithBug, allTests []sippyprocessingv1.FailingTestResult, numDays int, release string) string {
	if len(topFailingTestsWithBug) == 0 {
		return ""
	}

	rows, testNames := topFailingTestsRows(topFailingTestsWithBug, allTests, release)
	s := fmt.Sprintf(`
	<table class="table">
		<tr>
			<th colspan=5 class="text-center">
				<a class="text-dark" id="TopFailingTestsWithABug" href="#TopFailingTestsWithABug">Top Failing Tests With A Bug</a>
				%s
				<i class="fa fa-info-circle" title="Most frequently failing tests with a known bug, sorted by passing rate.  The link will prepopulate a BZ template to be filled out and submitted to report a bug against the test."</i>
			</th>
		</tr>
		<tr>
			<th colspan=2/><th class="text-center">Latest %d Days</th><th/><th class="text-center">Previous 7 Days</th>
		</tr>
		<tr>
			<th>Test Name</th><th>Bugs</th><th>Pass Rate</th><th/><th>Pass Rate</th>
		</tr>
	`, generichtml.GetTestDetailsButtonHTML(release, testNames...), numDays)

	s += rows
	s += "</table>"

	return s
}

func summaryTopFailingTestsWithoutBug(topFailingTestsWithBug, allTests []sippyprocessingv1.FailingTestResult, numDays int, release string) string {
	rows, testNames := topFailingTestsRows(topFailingTestsWithBug, allTests, release)
	s := fmt.Sprintf(`
	<table class="table">
		<tr>
			<th colspan=5 class="text-center">
				<a class="text-dark"  id="TopFailingTestsWithoutABug" href="#TopFailingTestsWithoutABug">Top Failing Tests Without A Bug</a>
				%s
				<i class="fa fa-info-circle" title="Most frequently failing tests without a known bug, sorted by passing rate.  The link will prepopulate a BZ template to be filled out and submitted to report a bug against the test."</i>
			</th>
		</tr>
		<tr>
			<th colspan=2/><th class="text-center">Latest %d Days</th><th/><th class="text-center">Previous 7 Days</th>
		</tr>
		<tr>
			<th>Test Name</th><th>File a Bug</th><th>Pass Rate</th><th/><th>Pass Rate</th>
		</tr>
	`, generichtml.GetTestDetailsButtonHTML(release, testNames...), numDays)

	s += rows
	s += "</table>"

	return s
}

func summaryCuratedTests(curr, prev sippyprocessingv1.TestReport, numDays int, release string) string {
	if len(curr.CuratedTests) == 0 {
		return ""
	}
	rows, testNames := topFailingTestsRows(curr.CuratedTests, prev.ByTest, release)

	s := fmt.Sprintf(`
	<table class="table">
		<tr>
			<th colspan=5 class="text-center">
				<a class="text-dark" id="CuratedTRTTests" href="#CuratedTRTTests">Curated TRT Tests</a>
				%s
				<i class="fa fa-info-circle" title="Curated TRT tests for whatever reason they see fit, sorted by passing rate.  The link will prepopulate a BZ template to be filled out and submitted to report a bug against the test."</i>
			</th>
		</tr>
		<tr>
			<th colspan=2/><th class="text-center">Latest %d Days</th><th/><th class="text-center">Previous 7 Days</th>
		</tr>
		<tr>
			<th>Test Name</th><th>File a Bug</th><th>Pass Rate</th><th/><th>Pass Rate</th>
		</tr>
	`, generichtml.GetTestDetailsButtonHTML(release, testNames...), numDays)

	s += rows
	s += "</table>"

	return s
}

// returns the rows to display and the names of the tests being shown
func topFailingTestsRows(topFailingTests, prevTests []sippyprocessingv1.FailingTestResult, release string) (string, []string) {
	// test name | bug | pass rate | higher/lower | pass rate
	s := ""
	testNames := []string{}

	count := 0
	for _, testResult := range topFailingTests {
		// if we only have one failure, don't show it on the glass.  Keep it in the actual data so we can choose how to handle it,
		// but don't bother creating the noise in the UI for a one-off/long tail.
		if (testResult.TestResultAcrossAllJobs.Failures + testResult.TestResultAcrossAllJobs.Flakes) == 1 {
			continue
		}
		count++
		if count > 20 {
			break
		}
		testNames = append(testNames, testResult.TestName)

		testPrev := util.FindFailedTestResult(testResult.TestName, prevTests)

		s = s +
			generichtml.NewTestResultRendererForFailedTestResult("", testResult, release).
				WithPreviousFailedTestResult(testPrev).
				ToHTML()
	}

	return s, testNames
}
