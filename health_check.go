package simple_k8s

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const HealthPort = 9666
const CheckName = "alive"

var healthCheckServer *health.Server = &health.Server{}

func EnableLivelinessCheck() {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", HealthPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	*healthCheckServer = *health.NewServer()
	healthCheckServer.SetServingStatus(CheckName, healthpb.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(s, healthCheckServer)
	s.Serve(lis)
}

func DisableLivelinessCheck() {

	healthCheckServer.SetServingStatus(CheckName, healthpb.HealthCheckResponse_NOT_SERVING)
}
