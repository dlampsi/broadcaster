// Package info is keeping data for app versions attributes and namespaces.
package info

import "strings"

const (
	AppName   = "a0feed"
	Namespace = "a0feed"
)

var (
	Release     string
	BuildNumber string
	BuildTime   string
	CommitHash  string
	EnvPrefix   = strings.ToUpper(Namespace)
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
