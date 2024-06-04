package tcpmeasurer_test

import (
	"context"
	tcpmeasurer "orchestrator/common/pkg/tcp_measurer"
	"testing"

	"bitbucket.org/Taal_Orchestrator/orca-std-go/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestService_Init(t *testing.T) {
	appLogger := getLogger(t)
	t.Run("should return nil if app is installed", func(t *testing.T) {
		srv := tcpmeasurer.NewService(context.Background(), appLogger, 1, tcpmeasurer.WithCustomApp("tcpdump"))
		require.NoError(t, srv.Init())
	})
	t.Run("should return error if app is not installed", func(t *testing.T) {
		srv := tcpmeasurer.NewService(context.Background(), appLogger, 1, tcpmeasurer.WithCustomApp(uuid.NewString()))
		require.Error(t, srv.Init())
	})
}

func TestExtractWorkerGroup(t *testing.T) {
	type tCase struct {
		input         string
		expectedGroup string
		expectedCoin  string
	}
	table := []tCase{
		{
			`{"params": ["lp-wg4-s19jpro.cos-pb12-r7b1-96", "BSV-846861-8`,
			"lp-wg4-s19jpro.cos-pb12-r7b1-96",
			"BSV",
		},
		{
			`{"params": ["lp-wg5-s19jpro.cos-pb13-r2g4-92", "BSV-846861-8`,
			"lp-wg5-s19jpro.cos-pb13-r2g4-92",
			"BSV",
		},
		{
			`{"params": ["sfm-wg3-m30s++.CA051700FE4F", "BSV-846861-89d48`,
			"sfm-wg3-m30s++.CA051700FE4F",
			"BSV",
		},
		{
			`{"params": ["sfm-wg2-m30s++.CA051700EC1B", "BSV-846861-89d48`,
			"sfm-wg2-m30s++.CA051700EC1B",
			"BSV",
		},
	}
	for _, tc := range table {
		res, coin := tcpmeasurer.ExtractWorkerGroup([]byte(tc.input))
		require.Equal(t, tc.expectedGroup, res)
		require.Equal(t, tc.expectedCoin, coin)
	}
}

func getLogger(t *testing.T) logger.AppLogger {
	appLogger, err := logger.NewAppSLogger(
		&logger.Config{
			Progname: "orca_mapicron",
		},
		"",
	)
	require.NoError(t, err)
	return appLogger
}
