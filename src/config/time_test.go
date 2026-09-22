package config_test

import (
	"testing"
	"time"

	"example.com/brave-revival/src/config"
	"github.com/stretchr/testify/require"
)

func TestValidTime(t *testing.T) {
	actual, err := config.ParseTime("6a982b6a")
	require.NoError(t, err)

	expected := time.Date(2026, 9, 2, 22, 58, 2, 0, config.StandardTimeZone)
	require.Equal(t, expected, actual)

	serialized := config.FormatTime(actual)
	require.Equal(t, "6a982b6a", serialized)
}
