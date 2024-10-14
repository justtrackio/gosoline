package grpcserver

import (
	"context"
	"sync"

	"github.com/justtrackio/gosoline/pkg/log"
	"google.golang.org/grpc/codes"
	protobuf "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// HealthCheckCallback the signature of the HealthCheckCallback.
type HealthCheckCallback func(ctx context.Context) protobuf.HealthCheckResponse_ServingStatus

// ServiceHealthCallback definition.
type ServiceHealthCallback struct {
	ServiceName         string
	HealthCheckCallback HealthCheckCallback
}

// healthServer implements `service Health`.
type healthServer struct {
	protobuf.UnimplementedHealthServer

	logger     log.Logger
	cancelFunc context.CancelFunc

	mu sync.RWMutex
	// If shutdown is true, it's expected all serving status is NOT_SERVING, and
	// will stay in NOT_SERVING.
	shutdown bool
	// statusMap stores the serving status of the services this Server monitors.
	statusMap map[string]protobuf.HealthCheckResponse_ServingStatus
	updates   map[string]map[protobuf.Health_WatchServer]chan protobuf.HealthCheckResponse_ServingStatus
	callbacks []ServiceHealthCallback
}

// NewHealthServer returns a new HealthServer.
func NewHealthServer(logger log.Logger, cancelFunc context.CancelFunc) *healthServer {
	return &healthServer{
		logger:     logger,
		cancelFunc: cancelFunc,
		statusMap:  map[string]protobuf.HealthCheckResponse_ServingStatus{"": protobuf.HealthCheckResponse_SERVING},
		updates:    map[string]map[protobuf.Health_WatchServer]chan protobuf.HealthCheckResponse_ServingStatus{},
		callbacks:  []ServiceHealthCallback{},
	}
}

// Check implements `service Health`.
func (s *healthServer) Check(ctx context.Context, in *protobuf.HealthCheckRequest) (*protobuf.HealthCheckResponse, error) {
	for _, callback := range s.callbacks {
		if in.Service != "" && in.Service != callback.ServiceName {
			continue
		}

		serviceServingStatus := callback.HealthCheckCallback(ctx)
		if serviceServingStatus == protobuf.HealthCheckResponse_SERVING {
			continue
		}

		// unhealthy
		s.Shutdown()
		s.SetServingStatus(in.Service, serviceServingStatus)
		s.cancelFunc()

		s.logger.WithContext(ctx).WithFields(log.Fields{
			"health_status": serviceServingStatus.String(),
		}).Info("health-check")

		return &protobuf.HealthCheckResponse{
			Status: serviceServingStatus,
		}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if servingStatus, ok := s.statusMap[in.Service]; ok {
		s.logger.WithContext(ctx).WithFields(log.Fields{
			"health_status": servingStatus.String(),
		}).Info("health-check")

		return &protobuf.HealthCheckResponse{
			Status: servingStatus,
		}, nil
	}

	s.logger.WithContext(ctx).Info("health-check failed")

	return nil, status.Error(codes.NotFound, "unknown service")
}

// Watch implements `service Health`.
func (s *healthServer) Watch(in *protobuf.HealthCheckRequest, stream protobuf.Health_WatchServer) error {
	service := in.Service
	// update channel is used for getting service status updates.
	update := make(chan protobuf.HealthCheckResponse_ServingStatus, 1)
	s.mu.Lock()
	// Puts the initial status to the channel.
	if servingStatus, ok := s.statusMap[service]; ok {
		update <- servingStatus
	} else {
		update <- protobuf.HealthCheckResponse_SERVICE_UNKNOWN
	}

	// Registers the update channel to the correct place in the updates map.
	if _, ok := s.updates[service]; !ok {
		s.updates[service] = make(map[protobuf.Health_WatchServer]chan protobuf.HealthCheckResponse_ServingStatus)
	}
	s.updates[service][stream] = update
	defer func() {
		s.mu.Lock()
		delete(s.updates[service], stream)
		s.mu.Unlock()
	}()
	s.mu.Unlock()

	var lastSentStatus protobuf.HealthCheckResponse_ServingStatus = -1
	for {
		select {
		// Status updated. Sends the up-to-date status to the client.
		case servingStatus := <-update:
			if lastSentStatus == servingStatus {
				continue
			}
			lastSentStatus = servingStatus
			err := stream.Send(&protobuf.HealthCheckResponse{Status: servingStatus})
			if err != nil {
				return status.Error(codes.Canceled, "Stream has ended.")
			}
		// Context done. Removes the update channel from the updates map.
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Stream has ended.")
		}
	}
}

// AddCallback stores the callback reference that will later be called when the Check function is called
// this allows the service to hook into the health check.
func (s *healthServer) AddCallback(serviceName string, healthCallback HealthCheckCallback) {
	s.callbacks = append(s.callbacks, ServiceHealthCallback{
		ServiceName:         serviceName,
		HealthCheckCallback: healthCallback,
	})
}

// SetServingStatus is called when need to reset the serving status of a service
// or insert a new service entry into the statusMap.
func (s *healthServer) SetServingStatus(service string, servingStatus protobuf.HealthCheckResponse_ServingStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdown {
		s.logger.Info("health: status changing for %s to %v is ignored because health service is shutdown", service, servingStatus)
		return
	}

	s.setServingStatusLocked(service, servingStatus)
}

// Shutdown sets all serving status to NOT_SERVING, and configures the server to
// ignore all future status changes.
//
// This changes serving status for all services. To set status for a particular
// services, call SetServingStatus().
func (s *healthServer) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = true
	for service := range s.statusMap {
		s.setServingStatusLocked(service, protobuf.HealthCheckResponse_NOT_SERVING)
	}
}

// Resume sets all serving status to SERVING, and configures the server to
// accept all future status changes.
//
// This changes serving status for all services. To set status for a particular
// services, call SetServingStatus().
func (s *healthServer) Resume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = false
	for service := range s.statusMap {
		s.setServingStatusLocked(service, protobuf.HealthCheckResponse_SERVING)
	}
}

func (s *healthServer) setServingStatusLocked(service string, servingStatus protobuf.HealthCheckResponse_ServingStatus) {
	s.statusMap[service] = servingStatus
	for _, update := range s.updates[service] {
		// Clears previous updates, that are not sent to the client, from the channel.
		// This can happen if the client is not reading and the server gets flow control limited.
		select {
		case <-update:
		default:
		}
		// Puts the most recent update to the channel.
		update <- servingStatus
	}
}
