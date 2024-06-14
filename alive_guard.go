package simple_k8s

import (
	"sync"
	"time"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type AliveGuard struct {
	mutex             sync.Mutex
	healthCheckServer *health.Server
	timeThreshold     time.Duration
	serviceName       string
	failed            bool
}

func NewAliveGuard(healtServer *health.Server, threshold time.Duration, serviceName string) *AliveGuard {
	return &AliveGuard{
		healthCheckServer: healtServer,
		timeThreshold:     threshold,
		serviceName:       serviceName,
		failed:            false,
	}
}

func (g *AliveGuard) CheckTimeDuration(startTime time.Time) {
	if g.failed {
		return
	}

	if time.Since(startTime) > g.timeThreshold {
		g.mutex.Lock()
		defer g.mutex.Unlock()
		if !g.failed {
			g.failed = true
			g.healthCheckServer.SetServingStatus(g.serviceName, healthpb.HealthCheckResponse_NOT_SERVING)
		}
	}

}

func GetAppAliveGuard(threshold time.Duration) *AliveGuard {
	return NewAliveGuard(healthCheckServer, threshold, "")
}
