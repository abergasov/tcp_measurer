package tcpmeasurer_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Benchmark_HasPrefix(b *testing.B) {
	str := "2024-05-28 10:58:50.731847 02:da:71:cc:22:"
	for i := 0; i < b.N; i++ {
		require.True(b, strings.HasPrefix(str, "202"))
	}
}
func Benchmark_TimeParse(b *testing.B) {
	str := "2024-05-28 10:58:50.731847 02:da:71:cc:22:"
	for i := 0; i < b.N; i++ {
		_, err := time.Parse("2006-01-02 15:04:05.000000", str[:26])
		require.NoError(b, err)
	}
}

func Benchmark_StringsSplit(b *testing.B) {
	msg := "E.....@.+..)..._..6.dP...#.....z....,........R.y&`.b{\"params\":.[\"lp-wg8-s19jpro.cos-pb16-r8a6-92\",.\"BSV-846595-5be"
	for i := 0; i < b.N; i++ {
		data := strings.Split(msg, `":.["`)
		require.Len(b, data, 2)
	}
}

func Benchmark_StringsSplitN(b *testing.B) {
	msg := "E.....@.+..)..._..6.dP...#.....z....,........R.y&`.b{\"params\":.[\"lp-wg8-s19jpro.cos-pb16-r8a6-92\",.\"BSV-846595-5be"
	for i := 0; i < b.N; i++ {
		data := strings.SplitN(msg, `":.["`, 2)
		require.Len(b, data, 2)
	}
}

func Benchmark_StringsSplitSpace(b *testing.B) {
	msg := "a b c"
	for i := 0; i < b.N; i++ {
		require.Len(b, strings.Split(msg, " "), 3)
	}
}

func Benchmark_StringsSplitFields(b *testing.B) {
	msg := "a b c"
	for i := 0; i < b.N; i++ {
		require.Len(b, strings.Fields(msg), 3)
	}
}
