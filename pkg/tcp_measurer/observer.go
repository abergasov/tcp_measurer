package tcpmeasurer

import (
	"fmt"
	"orchestrator/common/pkg/utils"
	"strings"
	"time"
)

func (s *Service) getFromOutput() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case line := <-s.outputChain:
			s.extractFromOutput(line)
		}
	}
}

type MeasurerContainer struct {
	EventTime      time.Time
	RemoteHost     string
	AckSec         string
	IsConfirmation bool
}

// extractFromOutput is a method that extracts data from the output of the tcpdump command
// sample result for request and for answer
// 2024-05-10 09:56:57.090040 lo    In  IP localhost.http-alt > localhost.57228: Flags [P.], seq 5248:5403, ack 1, win 260, options [nop,nop,TS val 2869961573 ecr 2869961424], length 155: HTTP
// 2024-05-10 09:56:57.090051 lo    In  IP localhost.57228 > localhost.http-alt: Flags [.], ack 5403, win 260, options [nop,nop,TS val 2869961573 ecr 2869961573], length 0
func (s *Service) extractFromOutput(str string) {
	container, err := s.ParseString(str)
	if err != nil {
		// ignore maybe it because of bad string
		return
	}
	if !container.IsConfirmation {
		if _, ok := s.data[container.RemoteHost]; !ok {
			s.data[container.RemoteHost] = make(map[string]*MeasurerContainer, 10_000)
		}
		s.data[container.RemoteHost][container.AckSec] = container
	} else if _, ok := s.data[container.RemoteHost][container.AckSec]; ok {
		// we already have Start time, so just get latency and remove it from the map
		diff := container.EventTime.Sub(s.data[container.RemoteHost][container.AckSec].EventTime)
		time5MinAggregated := utils.RoundToNearest5Minutes(container.EventTime)
		s.mu.Lock()
		if _, ok = s.buffer[time5MinAggregated]; !ok {
			s.buffer[time5MinAggregated] = make(map[string][]float64, 5_000)
		}
		if _, ok = s.buffer[time5MinAggregated][container.RemoteHost]; !ok {
			s.buffer[time5MinAggregated][container.RemoteHost] = make([]float64, 0, 1_000)
		}
		s.buffer[time5MinAggregated][container.RemoteHost] = append(s.buffer[time5MinAggregated][container.RemoteHost], float64(diff.Milliseconds()))
		s.mu.Unlock()
		delete(s.data[container.RemoteHost], container.AckSec)
	}
}

func (s *Service) ParseString(str string) (*MeasurerContainer, error) {
	t, err := time.Parse("2006-01-02 15:04:05.000000", str[:26])
	if err != nil {
		return nil, fmt.Errorf("it is not valid time: %w", err) // ignore it if it's not a valid time
	}
	parts := strings.Split(str, " > ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("bad string format")
	}

	senderParts := strings.Split(parts[0], " ")
	receiverParts := strings.Split(parts[1], " ")
	hostA := strings.ReplaceAll(senderParts[len(senderParts)-1], ":", "")
	hostB := strings.ReplaceAll(receiverParts[0], ":", "")

	if strings.Contains(str, "seq") {
		// this is an outgoing request
		// 2024-05-10 09:56:57.090040 lo    In  IP localhost.http-alt > localhost.57228: Flags [P.], seq 5248:5403, ack 1, win 260, options [nop,nop,TS val 2869961573 ecr 2869961424], length 155: HTTP
		_, remoteHost := hostA, hostB
		dataSec := strings.Split(str, "seq ") // contains `... Flags [P.], seq ` and `5248:5403, ack 1, win 260, ...`
		if len(dataSec) != 2 {
			return nil, fmt.Errorf("bad string format for extract sec")
		}
		seqSec := strings.Split(dataSec[1], ",")[0] // contains 5248:5403
		// extract `5403`
		if strings.Contains(seqSec, ":") {
			ackSec := strings.Split(seqSec, ":")[1] // contains 5403
			return &MeasurerContainer{
				EventTime:  t,
				RemoteHost: remoteHost,
				AckSec:     ackSec,
			}, nil
		}
		return nil, fmt.Errorf("bad string format for extract ack sec")
	}
	// this is a confirmation answer
	// 2024-05-10 09:56:57.090051 lo    In  IP localhost.57228 > localhost.http-alt: Flags [.], ack 5403, win 260, options [nop,nop,TS val 2869961573 ecr 2869961573], length 0
	_, remoteHost := hostB, hostA
	dataSec := strings.Split(str, "ack ") // contains `... Flags [.], ack ` and `5403, win 260, ...`
	if len(dataSec) != 2 {
		return nil, fmt.Errorf("bad string format for extract ack sec for confirmation answer")
	}
	ackSec := strings.Split(dataSec[1], ",")[0] // contains 5403
	return &MeasurerContainer{
		RemoteHost:     remoteHost,
		AckSec:         ackSec,
		EventTime:      t,
		IsConfirmation: true,
	}, nil
}
