package agent

import "errors"

var errUnauthorized = errors.New("unauthorized")

type StatusResponse struct {
	Metrics  *ServerMetrics  `json:"metrics"`
	Services []ServiceHealth `json:"services"`
}
