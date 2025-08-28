package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/arwoosa/event/gen/pb/common"
	consolepb "github.com/arwoosa/event/gen/pb/console"
	"github.com/arwoosa/event/internal/dao/repository"
	"github.com/arwoosa/event/internal/errors"
)

// InternalServiceServer implements the generated gRPC InternalService interface
type InternalServiceServer struct {
	consolepb.UnimplementedInternalServiceServer
	eventRepo   repository.EventRepository
	sessionRepo repository.SessionRepository
	converter   *ProtobufConverter
}

// NewInternalServiceServer creates a new gRPC internal service server
func NewInternalServiceServer(eventRepo repository.EventRepository, sessionRepo repository.SessionRepository) *InternalServiceServer {
	return &InternalServiceServer{
		eventRepo:   eventRepo,
		sessionRepo: sessionRepo,
		converter:   NewProtobufConverter(),
	}
}

// GetEventById implements the gRPC GetEventById method for internal services
func (s *InternalServiceServer) GetEventById(ctx context.Context, req *common.ID) (*common.Event, error) {
	// Get event without merchant validation (for internal service use)
	event, err := s.eventRepo.FindByID(ctx, req.Id)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertEventToPB(event), nil
}

// GetSessionById implements the gRPC GetSessionById method for internal services
func (s *InternalServiceServer) GetSessionById(ctx context.Context, req *common.ID) (*common.Session, error) {
	// Get session without merchant validation (for internal service use)
	session, err := s.sessionRepo.FindByID(ctx, req.Id)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertSessionToPB(session), nil
}

// Helper methods

func (s *InternalServiceServer) handleServiceError(err error) error {
	// Handle service errors for internal API
	switch e := err.(type) {
	case *errors.ValidationError:
		return status.Error(codes.InvalidArgument, e.Error())
	case *errors.BusinessError:
		return status.Error(codes.InvalidArgument, e.Error())
	default:
		if err == errors.ErrEventNotFound {
			return status.Error(codes.NotFound, err.Error())
		}
		return status.Error(codes.Internal, err.Error())
	}
}
