//go:build linux
// +build linux

package procfs

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// Proc represents the process for which we want to find stats
type Proc struct {
	totalMem     uint64
	process      *process.Process
	samplingTime time.Duration
	usePss       bool
}

// Stats represents the process stats
type Stats struct {
	CPUPercentage float64
	MemPercentage float64
	MemShared     uint64
	MemSwapped    uint64
	MemConsumed   uint64
	RecordTime    time.Time
}

//New  Returns new Proc struct
func New(pid int32, samplingInterval time.Duration, usePss bool) (*Proc, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}
	machineMemory, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	return &Proc{process: p, samplingTime: samplingInterval, usePss: usePss, totalMem: machineMemory.Total}, nil
}

// GetStats returns process stats
func (ps *Proc) GetStats() (stat *Stats, err error) {

	s := Stats{}
	s.RecordTime = time.Now()
	s.CPUPercentage, err = ps.process.CPUPercent()
	if err != nil {
		return nil, err
	}

	memInfo, err := ps.process.MemoryInfo()
	if err != nil {
		return nil, err
	}

	if !ps.usePss {
		s.MemConsumed = memInfo.RSS
		s.MemSwapped = memInfo.Swap
		s.MemPercentage = (100 * float64(s.MemConsumed) / float64(ps.totalMem))
		return &s, nil
	}

	//NOTE: parsing maps is inefficient, can use smaps_rollup instead to find Pss, Ref #https://www.kernel.org/doc/Documentation/ABI/testing/procfs-smaps_rollup
	// TODO: smaps_rollup parser

	//why use pss instead of rss, Ref #https://stackoverflow.com/questions/1420426/how-to-calculate-the-cpu-usage-of-a-process-by-pid-in-linux-from-c/1424556
	maps, err := ps.process.MemoryMaps(false)
	if err != nil {
		return nil, err
	}

	var pss float64
	for _, m := range *maps {
		// add 0.5KiB as this avg error due to truncation, Ref #https://github.com/pixelb/ps_mem
		pss += float64(m.Pss) + 0.5
		s.MemSwapped += m.Swap
	}
	s.MemConsumed = uint64(pss)
	s.MemPercentage = (100 * float64(s.MemConsumed) / float64(ps.totalMem))
	return &s, nil

}

// GetStatsInInterval returns process stats after every interval
func (ps *Proc) GetStatsInInterval() []*Stats {
	return ps.GetStatsInIntervalWithContext(context.Background())
}

// GetStatsInIntervalWithContext returns process stats after every interval
func (ps *Proc) GetStatsInIntervalWithContext(ctx context.Context) []*Stats {

	var stats []*Stats
	s, err := ps.GetStats()
	if err != nil {
		return stats
	}
	//append initial values to slice, then check after an interval
	stats = append(stats, s)
	ticker := time.NewTicker(ps.samplingTime)
	for {
		select {
		case <-ticker.C:
			s, err := ps.GetStats()
			if err != nil {
				return stats
			}
			stats = append(stats, s)
		case <-ctx.Done():
			return stats
		}
	}
}
