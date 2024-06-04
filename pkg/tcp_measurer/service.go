package tcpmeasurer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	"bitbucket.org/Taal_Orchestrator/orca-std-go/logger"
)

type MeasurerContainer struct {
	EventTime  time.Time
	RemoteHost string
	SenderHost string
}

type Service struct {
	l                  logger.AppLogger
	ctx                context.Context
	observePort        uint64
	observePortStr     string
	observeInterface   string
	appName            string
	data               map[string]map[uint32]*MeasurerContainer // targetHost -> sequence -> time.Start and time.End
	dataSeq            map[string]map[uint32]*MeasurerContainer // targetHost -> sequence -> time.Start and time.End
	buffer             map[time.Time]map[string][]float64       // time5minAggregation -> targetHost -> latency
	dumpBufferInterval time.Duration
	parseFilesInterval time.Duration
	filesPath          string

	mu                sync.RWMutex
	matchedMiners     map[string]string
	matchedMinersCoin map[string]string
}

type Opt func(*Service)

func WithCustomApp(appName string) Opt {
	return func(s *Service) {
		s.appName = appName
	}
}

func NewService(ctx context.Context, l logger.AppLogger, observePort uint64, opts ...Opt) *Service {
	srv := &Service{
		ctx:                ctx,
		observePort:        observePort,
		observePortStr:     fmt.Sprintf("%d", observePort),
		observeInterface:   "any",
		l:                  l.With(slog.String("service", "tcpmeasurer")),
		appName:            "tcpdump",
		data:               make(map[string]map[uint32]*MeasurerContainer),
		dataSeq:            make(map[string]map[uint32]*MeasurerContainer),
		buffer:             make(map[time.Time]map[string][]float64, 10),
		dumpBufferInterval: 5 * time.Minute,
		parseFilesInterval: 5 * time.Second,
		filesPath:          "/tmp/",
		matchedMiners:      make(map[string]string),
		matchedMinersCoin:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(srv)
	}
	return srv
}

func (s *Service) Init() error {
	_, err := exec.LookPath(s.appName)
	if err != nil {
		return fmt.Errorf("app %s is not installed: %w", s.appName, err)
	}
	return nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.buffer {
		s.processData(s.buffer[key])
	}
}

func (s *Service) Start() error {
	go s.parsePCAPFiles()
	go s.DumpData()

	executor := fmt.Sprintf(
		"sudo %s -i %s -ttttt -X -s 128 -e -w %s/caapture-%s.pcap -G %d 'tcp port %d and (tcp[tcpflags] & (tcp-syn|tcp-ack) != 0)'",
		s.appName,
		s.observeInterface,
		strings.TrimSuffix(s.filesPath, "/"),
		`%Y_%m_%d_%H_%M_%S`, // file format, we will sort it
		15,                  // how often rotate files
		s.observePort,
	)
	s.l.Info("executor", slog.String("executor", executor))
	cmd := exec.CommandContext(s.ctx, "/bin/sh", "-c", executor)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to Start command: %w", err)
	}

	go s.copyOutput(stdout)
	go s.copyOutput(stderr)
	return cmd.Wait()
}

func (s *Service) copyOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s.l.Info(scanner.Text())
	}
}

var (
	prefix    = []byte(`{"params":`)
	separator = []byte(`",`)
	bsv       = []byte(`"BSV-`)
	bch       = []byte(`"BCH-`)
)

func ExtractWorkerGroup(payload []byte) (workerGroup, coin string) {
	if !bytes.HasPrefix(payload, prefix) {
		return "", ""
	}
	position := 0
	for i := range payload {
		if i+1 > len(payload) {
			break
		}
		if separator[0] == payload[i] && separator[1] == payload[i+1] {
			position = i
			break
		}
	}
	if position == 0 {
		return "", ""
	}

	if bytes.Contains(payload[position:], bsv) {
		coin = "BSV"
	} else if bytes.Contains(payload[position:], bch) {
		coin = "BCH"
	}
	return string(payload[len(prefix)+3 : position]), coin
}
