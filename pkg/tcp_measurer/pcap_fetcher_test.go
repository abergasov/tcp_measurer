package tcpmeasurer_test

import (
	"context"
	tcpmeasurer "orchestrator/common/pkg/tcp_measurer"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestService_ReadFile(t *testing.T) {
	go monitorMemoryUsage()
	appLogger := getLogger(t)
	srv := tcpmeasurer.NewService(context.Background(), appLogger, 3333, tcpmeasurer.WithCustomApp("tcpdump"))

	//require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134340.pcap"))
	//require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134355.pcap"))
	//require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134410.pcap"))
	//require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134425.pcap"))
	require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134440.pcap"))
	//require.NoError(t, srv.ReadFilePureGO("samples/caapture-2024_12_10_11_46_39-5f230.pcap"))
	//require.NoError(t, srv.ReadFile("samples/caapture-2024_12_10_11_46_39-5f230.pcap"))

	time.Sleep(3 * time.Second)
	srv.DumpIt()
	time.Sleep(3 * time.Second)
}

func TestService_ReadFilePure(t *testing.T) {
	go monitorMemoryUsage()
	appLogger := getLogger(t)
	srv := tcpmeasurer.NewService(context.Background(), appLogger, 3333, tcpmeasurer.WithCustomApp("tcpdump"))

	require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134340.pcap"))
	require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134355.pcap"))
	require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134410.pcap"))
	require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134425.pcap"))
	require.NoError(t, srv.ReadFilePureGO("samples/caapture-20240531134440.pcap"))

	time.Sleep(3 * time.Second)
	srv.DumpIt()
	time.Sleep(3 * time.Second)
}
