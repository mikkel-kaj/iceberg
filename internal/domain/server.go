package domain

import "time"

type Server struct {
	Name              string
	TailscaleHostname string
	HetznerID         int64
	IP                string
	AgentToken        string
	SSHKeyID          int64
	FirewallID        int64
	CreatedAt         time.Time
}
