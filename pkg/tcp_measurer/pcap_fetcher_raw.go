package tcpmeasurer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"orchestrator/common/pkg/utils"
	"os"
	"strings"
	"time"
)

// ReadFilePureGO reads pcap file and processes it
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
func (s *Service) ReadFilePureGO(pcapFile string) error {
	file, err := os.Open(pcapFile)
	if err != nil {
		return fmt.Errorf("error opening pcap file: %w", err)
	}
	defer file.Close()

	var magicNumber uint32
	if err = binary.Read(file, binary.BigEndian, &magicNumber); err != nil {
		return fmt.Errorf("error reading magic number: %w", err)
	}

	// Skip the global header
	file.Seek(24, io.SeekStart)

	packetHeader := make([]byte, 16)
	for {
		_, err = file.Read(packetHeader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading packet header: %w", err)
		}

		capLen := binary.LittleEndian.Uint32(packetHeader[8:12])
		packetData := make([]byte, capLen)
		if _, err = file.Read(packetData); err != nil {
			return fmt.Errorf("error reading packet data: %w", err)
		}

		tsSec := binary.LittleEndian.Uint32(packetHeader[:4])
		tsUSec := binary.LittleEndian.Uint32(packetHeader[4:8])

		// packetData
		// * Ethernet Header: 14 bytes
		// * IPv4 Header: 20 bytes
		//   - Source IP Address: 4 bytes
		//   - Destination IP Address: 4 bytes
		// * TCP Header: 20 bytes

		mc := &MeasurerContainer{
			EventTime:  time.Unix(int64(tsSec), int64(tsUSec)*1000),
			SenderHost: net.IP(packetData[28 : 28+4]).String(),   // Source IP Address offset
			RemoteHost: net.IP(packetData[28+4 : 28+8]).String(), // Destination IP Address offset
		}

		srcPort := binary.BigEndian.Uint16(packetData[36 : 36+2])
		dstPort := binary.BigEndian.Uint16(packetData[36+2 : 36+4])
		mc.RemoteHost = fmt.Sprintf("%s:%d", mc.RemoteHost, dstPort)
		mc.SenderHost = fmt.Sprintf("%s:%d", mc.SenderHost, srcPort)

		seq := binary.BigEndian.Uint32(packetData[40:44])
		ack := binary.BigEndian.Uint32(packetData[44:48])
		flags := packetData[47]

		flagPSN := packetData[47]&0x08 != 0
		flagACK := packetData[47]&0x10 != 0
		dataTCP := flagACK && flagPSN

		isIncoming := uint64(dstPort) == s.observePort

		hasMinerIDPayload := false
		payloadStarts := 0
		for i := 48; i < len(packetData)-10; i++ {
			if bytes.Equal(packetData[i:i+10], prefix) {
				hasMinerIDPayload = true
				payloadStarts = i
				break
			}
		}

		//mc.SenderHost, mc.RemoteHost = parseIPv4Header(packetData[14 : 14+20])
		//
		//srcPort, dstPort, seq, ack, flags, payload := parseTCPHeader(packetData[34:])
		//mc.RemoteHost = fmt.Sprintf("%s:%d", mc.RemoteHost, dstPort)
		//mc.SenderHost = fmt.Sprintf("%s:%d", mc.SenderHost, srcPort)
		//
		//hasMinerIDPayload := len(payload) >= 60
		//dataTCP = flags&(0x10|0x08) == (0x10 | 0x08) // ACK and PSH

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
			if minerData, coinName := ExtractWorkerGroup(packetData[payloadStarts:]); minerData != "" {
				s.mu.Lock()
				s.matchedMiners[key] = minerData
				s.matchedMinersCoin[key] = coinName
				s.mu.Unlock()
				if coinName == "" {
					println(string(packetData[payloadStarts:]))
				}
			} else {
				payloadStr := string(packetData[payloadStarts:])
				if strings.Contains(payloadStr, "mining.authorize") ||
					strings.Contains(payloadStr, "mining.subscribe") ||
					strings.Contains(payloadStr, "mining.suggest_difficulty") ||
					strings.Contains(payloadStr, "mining.configure") {
					continue
				}
				// s.l.Error("error extracting miner data", fmt.Errorf("miner data not found in payload"), slog.String("payload", payloadStr))
			}
			continue
		}

		s.dataMUSeq.Lock()
		if _, ok := s.dataSeq[key]; !ok {
			s.dataSeq[key] = make(map[uint32]*MeasurerContainer, 5000)
		}
		s.dataMUSeq.Unlock()

		if !isIncoming && dataTCP {
			// 2. second request from stratum to miner - source host is stratum, target is miner, ACK, PSH
			s.dataMUSeq.Lock()
			s.dataSeq[key][ack] = mc
			s.dataMUSeq.Unlock()
			continue
		}

		confirmationTCP := flags&0x10 == 0x10 && flags&0x08 == 0x00 // ACK and not PSH
		if isIncoming && confirmationTCP {
			// 3. third request from miner to stratum - source host is miner, target is stratum, ACK, delta between 2nd request and 3rd request is latency
			s.dataMUSeq.Lock()
			req, ok := s.dataSeq[key][seq]
			s.dataMUSeq.Unlock()
			if !ok {
				continue // abandoned package
			}
			// we already have Start time, so just get latency and remove it from the map
			diff := mc.EventTime.Sub(req.EventTime)
			time5MinAggregated := utils.RoundToNearest5Minutes(mc.EventTime)
			s.mu.Lock()
			if _, ok = s.buffer[time5MinAggregated]; !ok {
				s.buffer[time5MinAggregated] = make(map[string][]float64, 5000)
			}
			if _, ok = s.buffer[time5MinAggregated][key]; !ok {
				s.buffer[time5MinAggregated][key] = make([]float64, 0, 1000)
			}
			s.buffer[time5MinAggregated][key] = append(s.buffer[time5MinAggregated][key], float64(diff.Milliseconds()))
			s.mu.Unlock()
			s.dataMUSeq.Lock()
			delete(s.dataSeq[key], seq)
			s.dataMUSeq.Unlock()
		}
	}
	return nil
}
