package tcpmeasurer_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	tcpmeasurer "orchestrator/common/pkg/tcp_measurer"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"bitbucket.org/Taal_Orchestrator/orca-std-go/logger"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
)

type cLogger struct {
	t           *testing.T
	matchedLogs map[string]int
}

func (l cLogger) Write(p []byte) (n int, err error) {
	data := strings.SplitN(string(p), `"msg":`, 2)
	preparedData := strings.ReplaceAll(data[1], "\n", "")
	_, ok := l.matchedLogs[preparedData]
	require.True(l.t, ok)
	l.matchedLogs[preparedData]++
	print(data[1])
	return len(p), nil
}

func (l cLogger) Close() {
	for i, count := range l.matchedLogs {
		require.Equalf(l.t, 1, count, i, "log message should be logged once: %s", i)
	}
}

func TestService(t *testing.T) {
	// given
	currDir, err := os.Getwd()
	require.NoError(t, err)

	dataEL, errEL := os.ReadFile("expected_logs.dat")
	require.NoError(t, errEL)
	expectedLogRows := strings.Split(string(dataEL), "\n")
	matchedLogs := make(map[string]int, len(expectedLogRows))
	for _, row := range expectedLogRows {
		if row == "" {
			continue
		}
		matchedLogs[row] = 0
	}

	tLogger := cLogger{t: t, matchedLogs: matchedLogs}
	appLogger, err := logger.NewAppSLogger(
		&logger.Config{
			Progname: "orca_mapicron",
			Writers:  []io.Writer{tLogger},
		},
		"",
	)
	require.NoError(t, err)

	srv := tcpmeasurer.NewService(
		context.Background(),
		appLogger,
		3333,
		tcpmeasurer.WithCustomApp("tcpdump"),
		tcpmeasurer.WithParseFilesInterval(1*time.Second),
		tcpmeasurer.WithFilesPath(fmt.Sprintf("%s/samples", currDir)),
	)

	// when
	require.NoError(t, srv.ReadFile("samples/caapture-20240531134340.pcap"))
	require.NoError(t, srv.ReadFile("samples/caapture-20240531134355.pcap"))
	require.NoError(t, srv.ReadFile("samples/caapture-20240531134410.pcap"))
	require.NoError(t, srv.ReadFile("samples/caapture-20240531134425.pcap"))
	require.NoError(t, srv.ReadFile("samples/caapture-20240531134440.pcap"))

	// then
	srv.DumpIt()
	tLogger.Close()
}

func TestLineByLineFetcher_MEMORY(t *testing.T) {
	// given
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	// Give the pprof server a moment to start
	time.Sleep(1 * time.Second)
	appLogger := getLogger(t)
	srv := tcpmeasurer.NewService(context.Background(), appLogger, 8080, tcpmeasurer.WithCustomApp("tcpdump"))

	runtime.GC()

	// when
	srv.DumpIt()

	// then
	f, err := os.Create("heap.prof")
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, pprof.WriteHeapProfile(f))
	println("go tool pprof heap.prof")
}

func TestLineByLineFetcher_CPU(t *testing.T) {
	// given
	appLogger := getLogger(t)
	srv := tcpmeasurer.NewService(context.Background(), appLogger, 8080, tcpmeasurer.WithCustomApp("tcpdump"))

	// Start CPU profiling
	cpuProfileFile, err := os.Create("cpu.prof")
	require.NoError(t, err)
	defer cpuProfileFile.Close()
	require.NoError(t, pprof.StartCPUProfile(cpuProfileFile))
	defer pprof.StopCPUProfile()

	// when
	srv.DumpIt()
	println("go tool pprof cpu.prof")
}

func TestLineByLineFetcher_Observe(t *testing.T) {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	go monitorMemoryUsage()

	appLogger := getLogger(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv := tcpmeasurer.NewService(ctx, appLogger, 8080, tcpmeasurer.WithCustomApp("tcpdump"))
	go srv.DumpData()

	counter := 0
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		default:
			counter++
		}
	}
	// then
	f, err := os.Create("heap.prof")
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, pprof.WriteHeapProfile(f))
	println("go tool pprof heap.prof")
}

func monitorMemoryUsage() {
	counter := 1
	for {
		printMemUsage(counter)
		time.Sleep(1 * time.Second)
		counter++
	}
}

func printMemUsage(i int) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For a human-friendly display
	fmt.Printf("%d Alloc = %v MiB", i, bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func TestService_Start(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()
	appPort := spawnTCPServer(ctx, t)
	println("Server is listening on port", appPort)
	go spawnClient(ctx, t, appPort)
	go spawnClient(ctx, t, appPort)
	go spawnClient(ctx, t, appPort)
	time.Sleep(5 * time.Hour)
	appLogger := getLogger(t)
	srv := tcpmeasurer.NewService(ctx,
		appLogger,
		uint64(appPort),
		tcpmeasurer.WithCustomApp("tcpdump"),
		//tcpmeasurer.WithDumpBufferInterval(5*time.Second),
	)

	// when
	require.NoError(t, srv.Init())

	// then
	require.Equal(t, "signal: killed", srv.Start().Error())
}

func spawnClient(ctx context.Context, t *testing.T, port int) {
	conn, err := net.Dial("tcp", fmt.Sprintf("192.168.1.63:%d", port))
	require.NoError(t, err)
	println("Connected to server.")
	defer conn.Close()

	c := bufio.NewReader(conn)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			userInput, errR := c.ReadString('\n')
			require.NoError(t, errR)
			if userInput == "" {

			}
			//println(userInput)
		}
	}
}

func spawnTCPServer(ctx context.Context, t *testing.T) int {
	appPort, err := freeport.GetFreePort()
	require.NoError(t, err)
	appPort = 3333

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("192.168.1.63:%d", appPort))
	require.NoError(t, err)

	listener, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	print(fmt.Sprintf("Server is listening on port %d...", appPort))

	go func() {
		defer listener.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, errA := listener.AcceptTCP()
				if errA != nil {
					t.Log("Failed to accept connection.", errA)
				}
				go handleRequest(conn)
			}
		}
	}()
	return appPort
}

func handleRequest(conn net.Conn) {
	println("Handling request...")
	defer conn.Close()

	t := time.NewTicker(3 * time.Second)
	counter := 0
	enc := json.NewEncoder(conn)
	defer t.Stop()
	for range t.C {
		//println(fmt.Sprintf("Sending message %d...", counter))
		counter++
		if err := enc.Encode(fmt.Sprintf(
			"ping %d%s%s%s%s",
			counter,
			uuid.NewString(),
			uuid.NewString(),
			uuid.NewString(),
			uuid.NewString(),
		)); err != nil {
			fmt.Println("Error sending message:", err.Error())
			return
		}
	}
}
