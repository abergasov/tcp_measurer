package tcpmeasurer_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	tcpmeasurer "orchestrator/common/pkg/tcp_measurer"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
)

func TestService_Start(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()
	appPort := spawnTCPServer(ctx, t)
	go spawnClient(ctx, t, appPort)
	go spawnClient(ctx, t, appPort)
	go spawnClient(ctx, t, appPort)
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

type testMeasurerContainer struct {
	EventTime      string
	RemoteHost     string
	AckSec         string
	IsConfirmation bool
}

func spawnClient(ctx context.Context, t *testing.T, port int) {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
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

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", appPort))
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

func TestService_ParseString(t *testing.T) {
	// given
	appLogger := getLogger(t)
	srv := tcpmeasurer.NewService(context.Background(), appLogger, 8080, tcpmeasurer.WithCustomApp("tcpdump"))
	table := map[string]testMeasurerContainer{
		"2024-05-10 09:56:57.090040 lo    In  IP localhost.http-alt > localhost.57228: Flags [P.], seq 5248:5403, ack 1, win 260, options [nop,nop,TS val 2869961573 ecr 2869961424], length 155: HTTP": {
			EventTime:      "2024-05-10 09:56:57.090040",
			RemoteHost:     "localhost.57228",
			AckSec:         "5403",
			IsConfirmation: false,
		},
		"2024-05-10 09:56:57.090051 lo    In  IP localhost.57228 > localhost.http-alt: Flags [.], ack 5403, win 260, options [nop,nop,TS val 2869961573 ecr 2869961573], length 0": {
			EventTime:      "2024-05-10 09:56:57.090051",
			RemoteHost:     "localhost.57228",
			AckSec:         "5403",
			IsConfirmation: true,
		},
		"2024-05-13 13:54:54.641456 lo    In  IP localhost.http-alt > localhost.48226: Flags [P.], seq 2852772069:2852772222, ack 1320570704, win 260, options [nop,nop,TS val 2794107757 ecr 2794104756], length 153: HTTP": {
			EventTime:      "2024-05-13 13:54:54.641456",
			RemoteHost:     "localhost.48226",
			AckSec:         "2852772222",
			IsConfirmation: false,
		},
		"2024-05-13 13:54:54.641470 lo    In  IP localhost.48226 > localhost.http-alt: Flags [.], ack 153, win 259, options [nop,nop,TS val 2794107757 ecr 2794107757], length 0": {
			EventTime:      "2024-05-13 13:54:54.641470",
			RemoteHost:     "localhost.48226",
			AckSec:         "153",
			IsConfirmation: true,
		},
		"2024-05-13 13:54:57.641331 lo    In  IP localhost.http-alt > localhost.48226: Flags [P.], seq 153:306, ack 1, win 260, options [nop,nop,TS val 2794110756 ecr 2794107757], length 153: HTTP": {
			EventTime:      "2024-05-13 13:54:57.641331",
			RemoteHost:     "localhost.48226",
			AckSec:         "306",
			IsConfirmation: false,
		},
		"2024-05-13 13:54:57.641343 lo    In  IP localhost.48226 > localhost.http-alt: Flags [.], ack 306, win 258, options [nop,nop,TS val 2794110756 ecr 2794110756], length 0": {
			EventTime:      "2024-05-13 13:54:57.641343",
			RemoteHost:     "localhost.48226",
			AckSec:         "306",
			IsConfirmation: true,
		},
		"2024-05-13 13:54:57.644496 lo    In  IP localhost.http-alt > localhost.48224: Flags [P.], seq 153:306, ack 1, win 260, options [nop,nop,TS val 2794110760 ecr 2794107760], length 153: HTTP": {
			EventTime:      "2024-05-13 13:54:57.644496",
			RemoteHost:     "localhost.48224",
			AckSec:         "306",
			IsConfirmation: false,
		},
		"2024-05-13 13:54:57.644503 lo    In  IP localhost.48224 > localhost.http-alt: Flags [.], ack 306, win 258, options [nop,nop,TS val 2794110760 ecr 2794110760], length 0": {
			EventTime:      "2024-05-13 13:54:57.644503",
			RemoteHost:     "localhost.48224",
			AckSec:         "306",
			IsConfirmation: true,
		},
		"2024-05-13 13:54:59.811769 lo    In  IP localhost.http-alt > localhost.48222: Flags [P.], seq 153:306, ack 1, win 260, options [nop,nop,TS val 2794112927 ecr 2794109925], length 153: HTTP": {
			EventTime:      "2024-05-13 13:54:59.811769",
			RemoteHost:     "localhost.48222",
			AckSec:         "306",
			IsConfirmation: false,
		},
		"2024-05-13 13:54:59.811781 lo    In  IP localhost.48222 > localhost.http-alt: Flags [.], ack 306, win 258, options [nop,nop,TS val 2794112927 ecr 2794112927], length 0": {
			EventTime:      "2024-05-13 13:54:59.811781",
			RemoteHost:     "localhost.48222",
			AckSec:         "306",
			IsConfirmation: true,
		},
		"2024-05-13 13:55:00.641451 lo    In  IP localhost.http-alt > localhost.48226: Flags [P.], seq 306:459, ack 1, win 260, options [nop,nop,TS val 2794113757 ecr 2794110756], length 153: HTTP": {
			EventTime:      "2024-05-13 13:55:00.641451",
			RemoteHost:     "localhost.48226",
			AckSec:         "459",
			IsConfirmation: false,
		},
		"2024-05-13 13:55:00.641464 lo    In  IP localhost.48226 > localhost.http-alt: Flags [.], ack 459, win 258, options [nop,nop,TS val 2794113757 ecr 2794113757], length 0": {
			EventTime:      "2024-05-13 13:55:00.641464",
			RemoteHost:     "localhost.48226",
			AckSec:         "459",
			IsConfirmation: true,
		},
		"2024-05-13 13:55:00.644617 lo    In  IP localhost.http-alt > localhost.48224: Flags [P.], seq 306:459, ack 1, win 260, options [nop,nop,TS val 2794113760 ecr 2794110760], length 153: HTTP": {
			EventTime:      "2024-05-13 13:55:00.644617",
			RemoteHost:     "localhost.48224",
			AckSec:         "459",
			IsConfirmation: false,
		},
		"2024-05-13 13:55:00.644624 lo    In  IP localhost.48224 > localhost.http-alt: Flags [.], ack 459, win 258, options [nop,nop,TS val 2794113760 ecr 2794113760], length 0": {
			EventTime:      "2024-05-13 13:55:00.644624",
			RemoteHost:     "localhost.48224",
			AckSec:         "459",
			IsConfirmation: true,
		},
		"2024-05-13 13:55:02.811566 lo    In  IP localhost.http-alt > localhost.48222: Flags [P.], seq 306:459, ack 1, win 260, options [nop,nop,TS val 2794115927 ecr 2794112927], length 153: HTTP": {
			EventTime:      "2024-05-13 13:55:02.811566",
			RemoteHost:     "localhost.48222",
			AckSec:         "459",
			IsConfirmation: false,
		},
		"2024-05-13 13:55:02.811582 lo    In  IP localhost.48222 > localhost.http-alt: Flags [.], ack 459, win 258, options [nop,nop,TS val 2794115927 ecr 2794115927], length 0": {
			EventTime:      "2024-05-13 13:55:02.811582",
			RemoteHost:     "localhost.48222",
			AckSec:         "459",
			IsConfirmation: true,
		},
		"2024-05-13 13:55:03.641534 lo    In  IP localhost.http-alt > localhost.48226: Flags [P.], seq 459:612, ack 1, win 260, options [nop,nop,TS val 2794116757 ecr 2794113757], length 153: HTTP": {
			EventTime:      "2024-05-13 13:55:03.641534",
			RemoteHost:     "localhost.48226",
			AckSec:         "612",
			IsConfirmation: false,
		},
		"2024-05-13 13:55:03.641549 lo    In  IP localhost.48226 > localhost.http-alt: Flags [.], ack 612, win 258, options [nop,nop,TS val 2794116757 ecr 2794116757], length 0": {
			EventTime:      "2024-05-13 13:55:03.641549",
			RemoteHost:     "localhost.48226",
			AckSec:         "612",
			IsConfirmation: true,
		},
	}

	// when
	for str, expected := range table {
		resp, err := srv.ParseString(str)
		require.NoError(t, err)
		require.Equal(t, expected.EventTime, resp.EventTime.Format("2006-01-02 15:04:05.000000"))
		require.Equal(t, expected.RemoteHost, resp.RemoteHost)
		require.Equal(t, expected.AckSec, resp.AckSec)
		require.Equal(t, expected.IsConfirmation, resp.IsConfirmation)
	}
}
