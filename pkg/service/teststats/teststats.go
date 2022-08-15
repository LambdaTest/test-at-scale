package teststats

import (
	"sort"
	"sync"
	"time"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/procfs"
)

//ProcStats represents the process stats for a particular pid
type ProcStats struct {
	logger                       lumber.Logger
	ExecutionResultInputChannel  chan core.ExecutionResults
	wg                           sync.WaitGroup
	ExecutionResultOutputChannel chan *core.ExecutionResults
	statsChan                    chan []*procfs.Stats
}

// New returns instance of ProcStats
func New(cfg *config.NucleusConfig, logger lumber.Logger) (*ProcStats, error) {
	return &ProcStats{
		logger:                       logger,
		ExecutionResultInputChannel:  make(chan core.ExecutionResults),
		ExecutionResultOutputChannel: make(chan *core.ExecutionResults),
		statsChan:                    make(chan []*procfs.Stats),
	}, nil

}

// CaptureTestStats combines the ps stats for each test
func (s *ProcStats) CaptureTestStats(pid int32, collectStats bool) error {

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Debugf("stats collection start 51")
		go s.getStats(pid)
		combinedExecutionResults := core.ExecutionResults{}

		s.logger.Debugf("waiting for input channel 55")
		for executionResults := range s.ExecutionResultInputChannel {

			if collectStats {
				//for ind := range executionResults.Results {
				// Refactor the impl of below 2 functions using generics when Go 1.18 arrives
				// https://www.freecodecamp.org/news/generics-in-golang/
				combinedExecutionResults.Results = append(combinedExecutionResults.Results, executionResults.Results...)
			}
		}
		processStats := <-s.statsChan
		for ind := range combinedExecutionResults.Results {
			s.appendStatsToTests(combinedExecutionResults.Results[ind].TestPayload, processStats)
			s.appendStatsToTestSuites(combinedExecutionResults.Results[ind].TestSuitePayload, processStats)
		}
		s.ExecutionResultOutputChannel <- &combinedExecutionResults
		close(s.ExecutionResultOutputChannel)
	}()

	return nil
}

func (s *ProcStats) getStats(pid int32) {
	ps, err := procfs.New(pid, global.SamplingTime, false)
	processStats := ps.GetStatsInInterval()

	if err != nil {
		s.logger.Errorf("failed to find process stats with pid %d %v", pid, err)
		return
	}
	go func() {
		s.statsChan <- processStats
	}()
}

// processStats is RecordTime sorted
func (s *ProcStats) getProcsForInterval(start, end time.Time, processStats []*procfs.Stats) []*procfs.Stats {
	n := len(processStats)
	left := sort.Search(n, func(i int) bool { return !processStats[i].RecordTime.Before(start) })
	right := sort.Search(n, func(i int) bool { return !processStats[i].RecordTime.Before(end) })

	if left <= right && 0 <= left && right <= n {
		return processStats[left:right]
	}
	// return empty slice
	return processStats[0:0]
}

func (s *ProcStats) appendStatsToTests(testResults []core.TestPayload, processStats []*procfs.Stats) {
	for r := 0; r < len(testResults); r++ {
		result := &testResults[r]
		// check if start time of test t(start) is not 0
		if !result.StartTime.IsZero() {
			// calculate end time of test t(end)
			result.EndTime = result.StartTime.Add(time.Duration(result.Duration) * time.Millisecond)
			for _, proc := range s.getProcsForInterval(result.StartTime, result.EndTime, processStats) {
				result.Stats = append(result.Stats, core.TestProcessStats{CPU: proc.CPUPercentage, Memory: proc.MemConsumed, RecordTime: proc.RecordTime})
			}
		}
	}
}

func (s *ProcStats) appendStatsToTestSuites(testSuiteResults []core.TestSuitePayload, processStats []*procfs.Stats) {
	for r := 0; r < len(testSuiteResults); r++ {
		result := &testSuiteResults[r]
		// check if start time of test suite ts(start) is not 0
		if !result.StartTime.IsZero() {
			// calculate end time of test suite ts(end)
			result.EndTime = result.StartTime.Add(time.Duration(result.Duration) * time.Millisecond)
			for _, proc := range s.getProcsForInterval(result.StartTime, result.EndTime, processStats) {
				result.Stats = append(result.Stats, core.TestProcessStats{CPU: proc.CPUPercentage, Memory: proc.MemConsumed, RecordTime: proc.RecordTime})
			}
		}
	}
}
