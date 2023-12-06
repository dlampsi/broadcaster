// Package info is keeping data for app versions attributes and namespaces.
package info

const (
	AppName   = "Broadcaster"
	Namespace = "broadcaster"
	EnvPrefix = "bctr"
)

var (
	Release     string
	BuildNumber string
	BuildTime   string
	CommitHash  string
)

func Print(sep string) string {
	s := AppName + " " + Release
	if CommitHash != "" {
		s = s + sep + "Commit: " + CommitHash
	}
	if BuildTime != "" {
		s = s + sep + "Build time: " + BuildTime
	}
	return s
}
