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
	var (
		dumpData map[string][]float64
		dumpKey  time.Time
	)
	s.mu.Lock()
	for key := range s.buffer {
		s.l.Info("checking key", slog.String("key", key.String()))
		if key.Before(dumpBefore) {
			dumpData = s.buffer[key]
			delete(s.buffer, key)
			dumpKey = key
			break
		}
	}
	s.mu.Unlock()
	if len(dumpData) == 0 {
		s.l.Info("no data to dump")
		return
	}

	s.processData(dumpKey, dumpData)
}

func (s *Service) processData(dumpKey time.Time, dumpData map[string][]float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	minerCoin := make(map[string]string, len(s.matchedMiners))
	aggregated := make(map[string][]float64, len(dumpData))
	for targetHost := range dumpData {
		minerData, _ := s.matchedMiners[targetHost]
		if minerData == "" {
			continue
		}
		minerCoin[minerData] = s.matchedMinersCoin[targetHost]
		if _, ok := aggregated[minerData]; !ok {
			aggregated[minerData] = make([]float64, 0, 1000)
		}
		aggregated[minerData] = append(aggregated[minerData], dumpData[targetHost]...)
	}

	for minerData := range aggregated {
		miningCoin, _ := minerCoin[minerData]
		l := s.l.With(
			slog.String("observe_interval", dumpKey.Format(time.DateTime)),
			logger.WithWorkerGroup(minerData),
			slog.String("mining_coin", miningCoin),
			slog.Int64("total_requests", int64(len(aggregated[minerData]))),
		)
		avg, err := stats.Mean(aggregated[minerData])
		if err != nil {
			l.Error("failed to calculate average", err)
			continue
		}

		percent95, err := stats.Percentile(aggregated[minerData], 95)
		if err != nil {
			l.Error("failed to calculate 95 percentile", err)
			continue
		}
		percent99, err := stats.Percentile(aggregated[minerData], 99)
		if err != nil {
			l.Error("failed to calculate 99 percentile", err)
			continue
		}

		median, err := stats.Median(aggregated[minerData])
		if err != nil {
			l.Error("failed to calculate median", err)
			continue
		}
		maxL, err := stats.Max(aggregated[minerData])
		if err != nil {
			l.Error("failed to calculate max", err)
			continue
		}
		minL, err := stats.Min(aggregated[minerData])
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
