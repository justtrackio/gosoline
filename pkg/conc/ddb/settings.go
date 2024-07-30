package ddb

import "time"

type TableNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-leader-elections"`
}

type DdbLeaderElectionSettings struct {
	Naming        TableNamingSettings `cfg:"naming"`
	ClientName    string              `cfg:"client_name"    default:"default"`
	GroupId       string              `cfg:"group_id"       default:"{app_name}"`
	LeaseDuration time.Duration       `cfg:"lease_duration" default:"1m"`
}

type StaticLeaderElectionSettings struct {
	Result bool `cfg:"result"`
}
