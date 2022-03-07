package sippyserver

import (
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/klog"

	"github.com/openshift/sippy/pkg/buganalysis"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/db/models"
	"github.com/openshift/sippy/pkg/testgridanalysis/testgridconversion"
	"github.com/openshift/sippy/pkg/testgridanalysis/testidentification"
	"github.com/openshift/sippy/pkg/testgridanalysis/testreportconversion"
)

func (a TestReportGeneratorConfig) PrepareDatabase(
	dbc *db.DB,
	dashboard TestGridDashboardCoordinates,
	variantManager testidentification.VariantManager,
	syntheticTestManager testgridconversion.SyntheticTestManager,
	bugCache buganalysis.BugCache, // required to associate tests with bug
) error {
	testGridJobDetails, _ := a.TestGridLoadingConfig.load(dashboard.TestGridDashboardNames)
	rawJobResultOptions := testgridconversion.ProcessingOptions{
		SyntheticTestManager: syntheticTestManager,
		// Load the last 30 days of data.  Note that we do not prune data today, so we'll be accumulating
		// data over time for now.
		StartDay: 0,
		NumDays:  30,
	}
	rawJobResults, _ := rawJobResultOptions.ProcessTestGridDataIntoRawJobResults(testGridJobDetails)

	// allJobResults holds all the job results with all the test results.  It contains complete frequency information and
	allJobResults := testreportconversion.ConvertRawJobResultsToProcessedJobResults(
		dashboard.ReportName, rawJobResults, bugCache, dashboard.BugzillaRelease, variantManager)

	// Load all job and test results into database:
	klog.Info("loading ProwJobs into db")

	// Build up a cache of all prow jobs we know about to speedup data entry.
	// Maps job name to db ID.
	prowJobCache := map[string]uint{}
	var idNames []models.IDName
	dbc.DB.Model(&models.ProwJob{}).Find(&idNames)
	for _, idn := range idNames {
		if _, ok := prowJobCache[idn.Name]; !ok {
			prowJobCache[idn.Name] = idn.ID
		}
	}

	testCache := map[string]uint{}
	var tests []models.Test
	dbc.DB.Find(&tests)
	for _, idn := range tests {
		if _, ok := testCache[idn.Name]; !ok {
			testCache[idn.Name] = idn.ID
		}
	}

	jobRuns := []models.ProwJobRun{}
	for i := range allJobResults {
		jr := allJobResults[i]
		// Create ProwJob if we don't have one already:
		// TODO: we do not presently update a ProwJob once created, so any change in our variant detection code for ex
		// would not make it to the db.
		if _, ok := prowJobCache[allJobResults[i].Name]; !ok {
			dbProwJob := models.ProwJob{
				Name:        jr.Name,
				Release:     jr.Release,
				Variants:    jr.Variants,
				TestGridURL: jr.TestGridURL,
			}
			err := dbc.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&dbProwJob).Error
			if err != nil {
				return errors.Wrapf(err, "error loading prow job into db: %s", allJobResults[i].Name)
			}
			prowJobCache[jr.Name] = dbProwJob.ID
		}

		// CreateJobRuns if we don't have them already:
		for _, jobRun := range jr.AllRuns {
			pjr := models.ProwJobRun{
				Model: gorm.Model{
					ID: jobRun.ProwID,
				},
				ProwJobID:             prowJobCache[jr.Name],
				URL:                   jobRun.URL,
				TestFailures:          jobRun.TestFailures,
				Failed:                jobRun.Failed,
				InfrastructureFailure: jobRun.InfrastructureFailure,
				KnownFailure:          jobRun.KnownFailure,
				Succeeded:             jobRun.Succeeded,
				Timestamp:             time.Unix(int64(jobRun.Timestamp)/1000, 0), // Timestamp is in millis since epoch
				OverallResult:         jobRun.OverallResult,
			}

			failedTests := make([]models.Test, len(jobRun.FailedTestNames))
			for i, ftn := range jobRun.FailedTestNames {
				ft := models.Test{}
				r := dbc.DB.Where("name = ?", ftn).First(&ft)
				if errors.Is(r.Error, gorm.ErrRecordNotFound) {
					ft = models.Test{
						Name: ftn,
					}
					err := dbc.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&ft).Error
					if err != nil {
						return errors.Wrapf(err, "error loading test into db: %s", ft.Name)
					}
				}
				failedTests[i] = ft
			}
			pjr.FailedTests = failedTests
			jobRuns = append(jobRuns, pjr)
		}
	}
	err := dbc.DB.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(jobRuns, 100).Error
	if err != nil {
		return errors.Wrap(err, "error loading prow jobs into db")
	}

	klog.Info("done loading ProwJobs")

	return nil
}