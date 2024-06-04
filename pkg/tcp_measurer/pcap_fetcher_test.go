package tcpmeasurer_test

import (
	"context"
	tcpmeasurer "orchestrator/common/pkg/tcp_measurer"
	"testing"
	"time"
)

func TestService_ReadFile(t *testing.T) {
	go monitorMemoryUsage()
	appLogger := getLogger(t)
	srv := tcpmeasurer.NewService(context.Background(), appLogger, 3333, tcpmeasurer.WithCustomApp("tcpdump"))

	srv.ReadFile("samples/caapture-20240531134340.pcap")
	srv.ReadFile("samples/caapture-20240531134355.pcap")
	srv.ReadFile("samples/caapture-20240531134410.pcap")
	srv.ReadFile("samples/caapture-20240531134425.pcap")
	srv.ReadFile("samples/caapture-20240531134440.pcap")

	time.Sleep(3 * time.Second)
	srv.DumpIt()
	time.Sleep(3 * time.Second)
}