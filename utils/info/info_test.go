package info

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ForPrint(t *testing.T) {
	// Set all to default
	Release = "0.0.0"
	BuildNumber = ""
	BuildTime = ""
	CommitHash = ""

	raw := Print("\n")
	require.Equal(t, "broadcaster 0.0.0", raw)

	CommitHash = "fakeone"
	withCommit := Print("\n")
	require.Equal(t, "Broadcaster 0.0.0\nCommit: fakeone", withCommit)

	BuildTime = "2222-02-18_08:32:16"
	withBuildTime := Print("\n")
	require.Equal(t, "Broadcaster 0.0.0\nCommit: fakeone\nBuild time: 2222-02-18_08:32:16", withBuildTime)

	Release = "5.5.6"
	withRelease := Print("\n")
	require.Equal(t, "Broadcaster 5.5.6\nCommit: fakeone\nBuild time: 2222-02-18_08:32:16", withRelease)

	diffSep := Print("; ")
	require.Equal(t, "Broadcaster 5.5.6; Commit: fakeone; Build time: 2222-02-18_08:32:16", diffSep)
}
