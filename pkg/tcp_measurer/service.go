package tcpmeasurer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"bitbucket.org/Taal_Orchestrator/orca-std-go/logger"
)

type Service struct {
	l                  logger.AppLogger
	ctx                context.Context
	observePort        uint64
	appName            string
	data               map[string]map[string]*MeasurerContainer // targetHost -> sequence -> time.Start and time.End
	outputChain        chan string
	buffer             map[time.Time]map[string][]float64 // time5minAggregation -> targetHost -> latency
	dumpBufferInterval time.Duration
	mu                 sync.Mutex
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
		l:                  l,
		appName:            "tcpdump",
		data:               make(map[string]map[string]*MeasurerContainer),
		outputChain:        make(chan string, 10_000),
		buffer:             make(map[time.Time]map[string][]float64, 10),
		dumpBufferInterval: 5 * time.Minute,
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
	go s.getFromOutput()
	go s.dumpData()
	executor := fmt.Sprintf("sudo %s -i any -tttt 'tcp port %d and (tcp[tcpflags] & (tcp-push|tcp-ack) != 0)'", s.appName, s.observePort)
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
		s.outputChain <- scanner.Text()
	}
}
