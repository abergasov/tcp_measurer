package tcpmeasurer_test

import (
	"orchestrator/common/pkg/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDump(t *testing.T) {
	dumpBefore := utils.RoundToNearest5Minutes(time.Now().UTC().Add(-6 * time.Minute))

	data := map[time.Time]string{
		time.Now().Add(-10 * time.Minute): "a",
		time.Now().Add(-5 * time.Minute):  "b",
		time.Now():                        "c",
	}
	var dumpData string
	for key := range data {
		if key.Before(dumpBefore) {
			dumpData = data[key]
			delete(data, key)
			break
		}
	}
	require.Equal(t, "a", dumpData)
}
