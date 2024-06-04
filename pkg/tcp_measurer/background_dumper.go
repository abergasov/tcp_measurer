package tcpmeasurer

import (
	"log/slog"
	"orchestrator/common/pkg/utils"
	"time"

	"bitbucket.org/Taal_Orchestrator/orca-std-go/entities"
	"bitbucket.org/Taal_Orchestrator/orca-std-go/logger"
	"github.com/montanaflynn/stats"
)

func WithDumpBufferInterval(interval time.Duration) Opt {
	return func(s *Service) {
		s.dumpBufferInterval = interval
	}
}

func (s *Service) DumpData() {
	ticker := time.NewTicker(s.dumpBufferInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.DumpIt()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) DumpIt() {
	s.l.Info("dumping data")
	dumpBefore := utils.RoundToNearest5Minutes(utils.RemoveTimezone(time.Now()).Add(-6 * time.Minute))
	var dumpData map[string][]float64
	s.mu.Lock()
	for key := range s.buffer {
		if key.Before(dumpBefore) {
			dumpData = s.buffer[key]
			delete(s.buffer, key)
			break
		}
	}
	s.mu.Unlock()
	if len(dumpData) == 0 {
		s.l.Info("no data to dump")
		return
	}

	s.processData(dumpData)
}

func (s *Service) processData(dumpData map[string][]float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for targetHost := range dumpData {
		minerData, _ := s.matchedMiners[targetHost]
		if minerData == "" {
			continue
		}
		miningCoin, _ := s.matchedMinersCoin[targetHost]
		l := s.l.With(
			logger.WithRemoteTarget(targetHost),
			logger.WithWorkerGroup(minerData),
			logger.WithCoinS(miningCoin),
			slog.Int64("total_requests", int64(len(dumpData[targetHost]))),
		)
		avg, err := stats.Mean(dumpData[targetHost])
		if err != nil {
			l.Error("failed to calculate average", err)
			continue
		}

		percent95, err := stats.Percentile(dumpData[targetHost], 95)
		if err != nil {
			l.Error("failed to calculate 95 percentile", err)
			continue
		}
		percent99, err := stats.Percentile(dumpData[targetHost], 99)
		if err != nil {
			l.Error("failed to calculate 99 percentile", err)
			continue
		}

		median, err := stats.Median(dumpData[targetHost])
		if err != nil {
			l.Error("failed to calculate median", err)
			continue
		}
		maxL, err := stats.Max(dumpData[targetHost])
		if err != nil {
			l.Error("failed to calculate max", err)
			continue
		}
		minL, err := stats.Min(dumpData[targetHost])
		if err != nil {
			l.Error("failed to calculate min", err)
			continue
		}
		l.With(
			slog.Float64("avg_latency", avg),
			slog.Float64("95_percentile", percent95),
			slog.Float64("99_percentile", percent99),
			slog.Float64("median_latency", median),
			slog.Float64("max_latency", maxL),
			slog.Float64("min_latency", minL),
			logger.WithLatencyFlag(),
			logger.WithNetworkConnectionType(entities.MinerExchangeDataWithStratum),
		).Info("miner latency")
	}
}
