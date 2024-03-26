package common

import "time"

type StartSessionReq struct {
	ClientID string
}

type StartSessionRes struct {
	ServerAddress string
	LeaseLength   time.Duration
	IsMaster      bool
}

type KeepAliveReq struct {
	ClientID string
}

type KeepAliveRes struct {
	LeaseLength time.Duration
}
