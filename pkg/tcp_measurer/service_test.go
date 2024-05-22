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
