package teststats

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/LambdaTest/synapse/pkg/procfs"
	"github.com/LambdaTest/synapse/testutils"
)

func getDummyTimeMap() map[string]time.Time {
	tpresent, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 28 Feb 2022 16:22:01 UTC")
	if err != nil {
		fmt.Printf("Error parsing time: %v", err)
	}
	t2025, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 22 Feb 2025 16:22:01 UTC")
	tpast1, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 22 Feb 2021 16:23:01 UTC")
	tpast2, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 22 Feb 2021 16:22:05 UTC")
	tfuture1, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 22 Feb 2023 16:14:01 UTC")
	tfuture2, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", "Tue, 22 Feb 2023 16:25:01 UTC")

	return map[string]time.Time{"tpresent": tpresent, "t2025": t2025, "tpast1": tpast1, "tpast2": tpast2, "tfuture1": tfuture1, "tfuture2": tfuture2}

}

// NOTE: Tests in this package are meant to be run in a Linux environment

func TestNew(t *testing.T) {
	cfg, _ := testutils.GetConfig()
	logger, _ := testutils.GetLogger()
	type args struct {
		cfg    *config.NucleusConfig
		logger lumber.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    *ProcStats
		wantErr bool
	}{
		{"Test New",
			args{cfg, logger},
			&ProcStats{
				logger: logger,
				httpClient: http.Client{
					Timeout: global.DefaultHTTPTimeout,
				},
				ExecutionResultInputChannel:  make(chan core.ExecutionResults),
				ExecutionResultOutputChannel: make(chan core.ExecutionResults),
			}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.cfg, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.httpClient, tt.want.httpClient) || !reflect.DeepEqual(got.logger, tt.want.logger) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcStats_getProcsForInterval(t *testing.T) {
	cfg, _ := testutils.GetConfig()
	logger, _ := testutils.GetLogger()
	timeMap := getDummyTimeMap()

	type args struct {
		start        time.Time
		end          time.Time
		processStats []*procfs.Stats
	}
	tests := []struct {
		name string
		args args
		want []*procfs.Stats
	}{
		{"Test getProcsForInterval", args{timeMap["tpresent"], timeMap["tpresent"], []*procfs.Stats{}}, []*procfs.Stats{}},

		{"Test getProcsForInterval", args{timeMap["tpresent"], timeMap["t2025"], []*procfs.Stats{
			{
				CPUPercentage: 1.2,
				MemPercentage: 14.1,
				MemShared:     105.0,
				MemSwapped:    25,
				MemConsumed:   131,
				RecordTime:    timeMap["tpast1"],
			},
			{
				CPUPercentage: 1.25,
				MemPercentage: 14.15,
				MemShared:     107.0,
				MemSwapped:    25,
				MemConsumed:   131,
				RecordTime:    timeMap["tfuture1"],
			},
			{
				CPUPercentage: 1.25,
				MemPercentage: 14.15,
				MemShared:     107.0,
				MemSwapped:    25,
				MemConsumed:   131,
				RecordTime:    timeMap["tfuture2"],
			},
		}}, []*procfs.Stats{
			{
				CPUPercentage: 1.25,
				MemPercentage: 14.15,
				MemShared:     107.0,
				MemSwapped:    25,
				MemConsumed:   131,
				RecordTime:    timeMap["tfuture1"],
			},
			{
				CPUPercentage: 1.25,
				MemPercentage: 14.15,
				MemShared:     107.0,
				MemSwapped:    25,
				MemConsumed:   131,
				RecordTime:    timeMap["tfuture2"],
			},
		},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(cfg, logger)
			if err != nil {
				t.Errorf("New() error = %v", err)
			}
			got := s.getProcsForInterval(tt.args.start, tt.args.end, tt.args.processStats)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProcStats.getProcsForInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcStats_appendStatsToTests(t *testing.T) {
	cfg, _ := testutils.GetConfig()
	logger, _ := testutils.GetLogger()
	timeMap := getDummyTimeMap()

	type args struct {
		testResults  []core.TestPayload
		processStats []*procfs.Stats
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Test appendStatsToTests",
			args{[]core.TestPayload{
				{Name: "test 1", StartTime: timeMap["tpast1"], EndTime: timeMap["tfuture1"]},
			},
				[]*procfs.Stats{}},
			"[{TestID: Detail: SuiteID: Suites:[] Title: FullTitle: Name:test 1 Duration:0 FilePath: Line: Col: CurrentRetry:0 Status: CommitID: DAG:[] Filelocator: BlocklistSource: Blocklisted:false StartTime:2021-02-22 16:23:01 +0000 UTC EndTime:2021-02-22 16:23:01 +0000 UTC Stats:[]}]",
		},

		{"Test appendStatsToTests",
			args{[]core.TestPayload{
				{
					Name:      "test 1",
					StartTime: timeMap["tpast1"],
					Duration:  100,
					EndTime:   timeMap["tfuture1"],
					Stats:     []core.TestProcessStats{},
				},
				{
					Name:      "test 2",
					StartTime: timeMap["tpast2"],
					Duration:  200,
					EndTime:   timeMap["tfuture2"],
					Stats:     []core.TestProcessStats{{Memory: 100, CPU: 25.4, Storage: 250, RecordTime: timeMap["tpast2"]}},
				},
			},
				[]*procfs.Stats{
					{
						CPUPercentage: 1.2,
						MemPercentage: 14.1,
						MemShared:     105.0,
						MemSwapped:    25,
						MemConsumed:   131,
						RecordTime:    timeMap["tpast1"],
					},
					{
						CPUPercentage: 1.25,
						MemPercentage: 14.15,
						MemShared:     107.0,
						MemSwapped:    25,
						MemConsumed:   131,
						RecordTime:    timeMap["tfuture1"],
					},
					{
						CPUPercentage: 1.25,
						MemPercentage: 14.15,
						MemShared:     107.0,
						MemSwapped:    25,
						MemConsumed:   131,
						RecordTime:    timeMap["tfuture2"],
					},
				},
			},
			"[{TestID: Detail: SuiteID: Suites:[] Title: FullTitle: Name:test 1 Duration:100 FilePath: Line: Col: CurrentRetry:0 Status: CommitID: DAG:[] Filelocator: BlocklistSource: Blocklisted:false StartTime:2021-02-22 16:23:01 +0000 UTC EndTime:2021-02-22 16:23:01.1 +0000 UTC Stats:[{Memory:131 CPU:1.2 Storage:0 RecordTime:2021-02-22 16:23:01 +0000 UTC}]} {TestID: Detail: SuiteID: Suites:[] Title: FullTitle: Name:test 2 Duration:200 FilePath: Line: Col: CurrentRetry:0 Status: CommitID: DAG:[] Filelocator: BlocklistSource: Blocklisted:false StartTime:2021-02-22 16:22:05 +0000 UTC EndTime:2021-02-22 16:22:05.2 +0000 UTC Stats:[{Memory:100 CPU:25.4 Storage:250 RecordTime:2021-02-22 16:22:05 +0000 UTC}]}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(cfg, logger)
			if err != nil {
				t.Errorf("New() error = %v", err)
			}
			s.appendStatsToTests(tt.args.testResults, tt.args.processStats)
			got := fmt.Sprintf("%+v", tt.args.testResults)
			if got != tt.want {
				t.Errorf("ProcStats.appendStatsToTests() = \n%v\nwant: \n%v", got, tt.want)
			}
		})
	}
}

func TestProcStats_appendStatsToTestSuites(t *testing.T) {
	cfg, _ := testutils.GetConfig()
	logger, _ := testutils.GetLogger()
	timeMap := getDummyTimeMap()

	type args struct {
		testSuiteResults []core.TestSuitePayload
		processStats     []*procfs.Stats
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Test appendStatsToTests",
			args{[]core.TestSuitePayload{
				{SuiteID: "testSuite1", StartTime: timeMap["tpast1"], EndTime: timeMap["tfuture1"]},
			},
				[]*procfs.Stats{}},
			"[{SuiteID:testSuite1 SuiteName: ParentSuiteID: BlacklistSource: Blacklisted:false StartTime:2021-02-22 16:23:01 +0000 UTC EndTime:2021-02-22 16:23:01 +0000 UTC Duration:0 Status: Stats:[]}]",
		},

		{"Test appendStatsToTests",
			args{[]core.TestSuitePayload{
				{
					SuiteID:   "testSuite2",
					StartTime: timeMap["tpast1"],
					Duration:  100,
					EndTime:   timeMap["tfuture1"],
					Stats:     []core.TestProcessStats{},
				},
				{
					SuiteID:   "testSuite3",
					StartTime: timeMap["tpast2"],
					Duration:  200,
					EndTime:   timeMap["tfuture2"],
					Stats:     []core.TestProcessStats{{Memory: 100, CPU: 25.4, Storage: 250, RecordTime: timeMap["tpast2"]}},
				},
			},
				[]*procfs.Stats{
					{
						CPUPercentage: 1.2,
						MemPercentage: 14.1,
						MemShared:     105.0,
						MemSwapped:    25,
						MemConsumed:   131,
						RecordTime:    timeMap["tpast1"],
					},
					{
						CPUPercentage: 1.25,
						MemPercentage: 14.15,
						MemShared:     107.0,
						MemSwapped:    25,
						MemConsumed:   131,
						RecordTime:    timeMap["tfuture1"],
					},
					{
						CPUPercentage: 1.25,
						MemPercentage: 14.15,
						MemShared:     107.0,
						MemSwapped:    25,
						MemConsumed:   131,
						RecordTime:    timeMap["tfuture2"],
					},
				},
			},
			"[{SuiteID:testSuite2 SuiteName: ParentSuiteID: BlacklistSource: Blacklisted:false StartTime:2021-02-22 16:23:01 +0000 UTC EndTime:2021-02-22 16:23:01.1 +0000 UTC Duration:100 Status: Stats:[{Memory:131 CPU:1.2 Storage:0 RecordTime:2021-02-22 16:23:01 +0000 UTC}]} {SuiteID:testSuite3 SuiteName: ParentSuiteID: BlacklistSource: Blacklisted:false StartTime:2021-02-22 16:22:05 +0000 UTC EndTime:2021-02-22 16:22:05.2 +0000 UTC Duration:200 Status: Stats:[{Memory:100 CPU:25.4 Storage:250 RecordTime:2021-02-22 16:22:05 +0000 UTC}]}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(cfg, logger)
			if err != nil {
				t.Errorf("New() error = %v", err)
			}
			s.appendStatsToTestSuites(tt.args.testSuiteResults, tt.args.processStats)
			got := fmt.Sprintf("%+v", tt.args.testSuiteResults)
			if got != tt.want {
				t.Errorf("ProcStats.appendStatsToTestSuites = \n%v\nwant: \n%v", got, tt.want)
			}
		})
	}
}
