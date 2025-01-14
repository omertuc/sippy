package generichtml

import (
	"fmt"
	"net/url"
	"regexp"

	bugsv1 "github.com/openshift/sippy/pkg/apis/bugs/v1"
	"github.com/openshift/sippy/pkg/util"
)

func bugLink(bug bugsv1.Bug) string {
	if !util.IsActiveBug(bug) {
		return fmt.Sprintf(`<a target="_blank" href="%s"><strike>%d</strike></a> `, bug.Url, bug.ID)
	}
	return fmt.Sprintf(`<a target="_blank" href="%s">%d</a> `, bug.Url, bug.ID)
}

// bugHTMLForTest release and testName are required.  variant is optional, if specified it excludes test that have a
// different variant specified, but includes bugs without any variant
func bugHTMLForTest(bugList []bugsv1.Bug, release, variant, testName string) string {
	if len(bugList) == 0 {
		return openABugHTML(testName, release)
	}

	bugHTML := "Associated Bugs: "
	for _, bug := range bugList {
		bugHTML += bugLink(bug)
	}

	return bugHTML
}

func openABugHTML(testName, release string) string {
	short_desc := testName
	if len(short_desc) > 255 {
		short_desc = short_desc[:255]
	}
	searchURL := testToSearchURL(testName)

	exampleJob :=
		`
FIXME: Replace this paragraph with a particular job URI from the search results to ground discussion.  A given test may fail for several reasons, and this bug should be scoped to one of those reasons.  Ideally you'd pick a job showing the most-common reason, but since that's hard to determine, you may also chose to pick a job at random.  Release-gating jobs (release-openshift-...) should be preferred over presubmits (pull-ci-...) because they are closer to the released product and less likely to have in-flight code changes that complicate analysis.

FIXME: Provide a snippet of the test failure or error from the job log
`

	bug := fmt.Sprintf(
		"<a target=\"_blank\" href=https://bugzilla.redhat.com/enter_bug.cgi?classification=Red%%20Hat&product=OpenShift%%20Container%%20Platform&cf_internal_whiteboard=buildcop&short_desc=%[1]s&cf_environment=%[2]s&comment=test:%%0A%[2]s%%20%%0A%%0Ais%%20failing%%20frequently%%20in%%20CI,%%20see%%20search%%20results:%%0A%[3]s%%0A%%0A%[4]s&version=%[5]s>Open a bug</a>",
		url.QueryEscape(short_desc),
		url.QueryEscape(testName),
		url.QueryEscape(searchURL),
		url.QueryEscape(exampleJob),
		release)

	return bug
}

// testName is the non-encoded test.Name
func testToSearchURL(testName string) string {
	encodedTestName := url.QueryEscape(regexp.QuoteMeta(testName))
	return fmt.Sprintf("https://search.ci.openshift.org/?maxAge=168h&context=1&type=bug%%2Bjunit&name=&maxMatches=5&maxBytes=20971520&groupBy=job&search=%s", encodedTestName)
}
