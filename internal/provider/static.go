package provider

import (
	"context"
	"os/exec"
	"time"
)

type Static struct {
	ProviderName string
	ProviderType string
	Features     []Capability
	Message      string
	Healthy      bool
	Binary       string
}

func (p Static) Name() string               { return p.ProviderName }
func (p Static) Type() string               { return p.ProviderType }
func (p Static) Capabilities() []Capability { return append([]Capability(nil), p.Features...) }
func (p Static) HealthCheck(context.Context) ProviderHealth {
	healthy, message := p.Healthy, p.Message
	if p.Binary != "" {
		if _, err := exec.LookPath(p.Binary); err != nil {
			healthy = false
			message = p.Binary + " binary is unavailable"
		} else {
			healthy = true
			message = p.Binary + " binary is available"
		}
	}
	return ProviderHealth{Healthy: healthy, Message: message, CheckedAt: time.Now().UTC()}
}
