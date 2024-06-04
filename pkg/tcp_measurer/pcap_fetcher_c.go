//go:build local

package tcpmeasurer

import (
	"fmt"
	"orchestrator/common/pkg/utils"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// ReadFile reads pcap file and processes it
// stratum exchange between miner is chunked into separate blocks
// 1. Miner sends request to the Stratum, ACK and PSH is true, payload is not empty
// 2. Stratum sends response to the Miner, ACK is true, PSH is true, payload is not empty
// 3. Miner sends confirmation to the Stratum, ACK is true, PSH is false, payload is empty
// so p1 and p2 is just how fast stratum software works, while 2 and 3 is show latency between miner and stratum
// sample output:
// 1. seq: 2396494688, ack: 3568706784, 2024-05-31 15:43:44.967858, len: 60, ACK: true, PSH: true,  target: 172.29.54.141:3333
//   - (mining.submit) {"id":171118,"method":"mining.submit","params":["workergroup"...]}
//   - we actually expect than miner will put params as first part of json. if not - there will be missing data in chart
//
// 2. seq: 3568706784, ack: 2396494875, 2024-05-31 15:43:44.967947, len: 41, ACK: true, PSH: true,  target: 8.46.207.95:23914
//   - {"id":171118,"result":true,"error":null}
//
// 3. seq: 2396494875, ack: 3568706825, 2024-05-31 15:43:45.005626, len: 0,  ACK: true, PSH: false, target: 172.29.54.141:3333,
// so it is chain `ack 3568706784 => seq 3568706784 ack 2396494875 => seq 2396494875 ack 3568706825`
// from all tcp dump requests we can extract 3 types of requests:
// 1. first request from miner to stratum - we use to map miner host to the miner worker group, ignore in calculations
// 2. second request from stratum to miner - source host is stratum, target is miner, ACK, PSH
// 3. third request from miner to stratum - source host is miner, target is stratum, ACK, delta between 2nd request and 3rd request is latency
func (s *Service) ReadFile(pcapFile string) error {
	handle, err := pcap.OpenOffline(pcapFile)
	if err != nil {
		return fmt.Errorf("error opening pcap file: %w", err)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetSource.NoCopy = true

	for packet := range packetSource.Packets() {
		mc := &MeasurerContainer{
			EventTime: packet.Metadata().Timestamp,
		}

		if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
			ip, _ := ipLayer.(*layers.IPv4)
			mc.RemoteHost = ip.DstIP.String()
			mc.SenderHost = ip.SrcIP.String()
		}

		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)

			mc.RemoteHost = fmt.Sprintf("%s:%d", mc.RemoteHost, tcp.DstPort)
			mc.SenderHost = fmt.Sprintf("%s:%d", mc.SenderHost, tcp.SrcPort)

			isIncoming := uint64(tcp.DstPort) == s.observePort
			hasMinerIDPayload := len(tcp.BaseLayer.Payload) >= 60
			dataTCP := tcp.ACK && tcp.PSH

			key := mc.RemoteHost
			if isIncoming {
				key = mc.SenderHost
			}

			if isIncoming && dataTCP && hasMinerIDPayload {
				// 1. first request from miner to stratum - we use to map miner host to the miner worker group, ignore in calculations
				s.mu.RLock()
				kh, _ := s.matchedMiners[key]
				s.mu.RUnlock()
				if kh != "" {
					continue
				}

				// extract miner data if it confirmation and we don't know miner yet
				if minerData, coinName := ExtractWorkerGroup(tcp.Payload); minerData != "" {
					s.mu.RLock()
					s.matchedMiners[key] = minerData
					s.matchedMinersCoin[key] = coinName
					s.mu.RUnlock()
				} else {
					payload := string(tcp.Payload)
					if strings.Contains(payload, "mining.authorize") ||
						strings.Contains(payload, "mining.subscribe") ||
						strings.Contains(payload, "mining.suggest_difficulty") ||
						strings.Contains(payload, "mining.configure") {
						continue
					}
					// s.l.Error("error extracting miner data", fmt.Errorf("miner data not found in payload"), slog.String("payload", payload))
				}
				continue
			}

			if _, ok := s.dataSeq[key]; !ok {
				s.dataSeq[key] = make(map[uint32]*MeasurerContainer, 5_000)
			}
			if !isIncoming && dataTCP {
				// 2. second request from stratum to miner - source host is stratum, target is miner, ACK, PSH
				s.dataSeq[key][tcp.Ack] = mc
				continue
			}

			confirmationTCP := tcp.ACK && !tcp.PSH
			if isIncoming && confirmationTCP {
				// 3. third request from miner to stratum - source host is miner, target is stratum, ACK, delta between 2nd request and 3rd request is latency
				req, ok := s.dataSeq[key][tcp.Seq]
				if !ok {
					continue // abandoned package
				}
				// we already have Start time, so just get latency and remove it from the map
				diff := mc.EventTime.Sub(req.EventTime)
				time5MinAggregated := utils.RoundToNearest5Minutes(mc.EventTime)
				s.mu.Lock()
				if _, ok = s.buffer[time5MinAggregated]; !ok {
					s.buffer[time5MinAggregated] = make(map[string][]float64, 5_000)
				}
				if _, ok = s.buffer[time5MinAggregated][key]; !ok {
					s.buffer[time5MinAggregated][key] = make([]float64, 0, 1_000)
				}
				s.buffer[time5MinAggregated][key] = append(s.buffer[time5MinAggregated][key], float64(diff.Milliseconds()))
				s.mu.Unlock()
				delete(s.dataSeq[key], tcp.Seq)

				// sometimes packages are lost, so we need cleanup to avoid memory leak
				dropBefore := time.Now().Add(-1 * time.Hour)
				for k := range s.dataSeq[key] {
					if s.dataSeq[key][k].EventTime.Before(dropBefore) {
						delete(s.dataSeq[key], k)
					}
				}
			}
		}
	}
	return nil
}
