package service

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/arwoosa/event/conf"
	"github.com/arwoosa/event/gen/pb/common"
	consolepb "github.com/arwoosa/event/gen/pb/console"
	"github.com/arwoosa/event/internal/errors"

	"github.com/arwoosa/vulpes/ezgrpc"
)

// EventServiceServer implements the generated gRPC EventService interface
type EventServiceServer struct {
	consolepb.UnimplementedEventServiceServer
	eventService     *EventService
	converter        *ProtobufConverter
	paginationConfig *conf.PaginationConfig
}

// NewEventServiceServer creates a new gRPC event service server
func NewEventServiceServer(eventService *EventService, paginationConfig *conf.PaginationConfig) *EventServiceServer {
	return &EventServiceServer{
		eventService:     eventService,
		converter:        NewProtobufConverter(),
		paginationConfig: paginationConfig,
	}
}

// CreateEvent implements the gRPC CreateEvent method
func (s *EventServiceServer) CreateEvent(ctx context.Context, req *consolepb.CreateEventRequest) (*consolepb.CreateEventResponse, error) {
	// Extract user information from context
	user, err := ezgrpc.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	// Convert gRPC request to service request
	serviceReq := &CreateEventRequest{
		Title:   req.Title,
		Summary: req.Summary,
		// Status field removed - events are always created as draft
		Visibility:    req.Visibility,
		CoverImageURL: req.CoverImageUrl,
		Location:      s.converter.ConvertLocationFromPB(req.Location),
		Sessions:      s.converter.ConvertSessionsFromPB(req.Sessions),
		Detail:        s.converter.ConvertDetailFromPB(req.Detail),
		FAQ:           s.converter.ConvertFAQFromPB(req.Faq),
		UserID:        user.ID,
	}

	// Create event
	event, err := s.eventService.CreateEvent(ctx, serviceReq)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return &consolepb.CreateEventResponse{
		Id:        event.ID.Hex(),
		CreatedAt: event.CreatedAt.Format(time.RFC3339),
	}, nil
}

// GetEventList implements the gRPC GetEventList method
func (s *EventServiceServer) GetEventList(ctx context.Context, req *consolepb.GetEventListRequest) (*common.EventListResponse, error) {
	// Process request parameters into filter
	processor := NewRequestParameterProcessor()
	filter := processor.ProcessAllFilters(req, s.paginationConfig)

	// Get events from service
	result, err := s.eventService.GetEventList(ctx, filter)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Build response
	responseBuilder := NewResponseBuilder(s.converter)
	return responseBuilder.BuildEventListResponse(result), nil
}

// GetEvent implements the gRPC GetEvent method
func (s *EventServiceServer) GetEvent(ctx context.Context, req *common.ID) (*common.Event, error) {
	// Get event
	event, err := s.eventService.GetEvent(ctx, req.Id)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertEventToPB(event), nil
}

// PatchEvent implements the gRPC PatchEvent method
func (s *EventServiceServer) PatchEvent(ctx context.Context, req *consolepb.PatchEventRequest) (*common.Event, error) {
	// Extract user information from context
	user, err := ezgrpc.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	// Convert gRPC request to service request
	serviceReq := &PatchEventRequest{
		ID:     req.Id,
		UserID: user.ID,
	}

	// Only set optional fields if they are provided and non-empty
	if req.Title != nil && *req.Title != "" {
		serviceReq.Title = req.Title
	}
	if req.Summary != nil && *req.Summary != "" {
		serviceReq.Summary = req.Summary
	}
	if req.Visibility != nil && *req.Visibility != "" {
		serviceReq.Visibility = req.Visibility
	}
	if req.CoverImageUrl != nil && *req.CoverImageUrl != "" {
		serviceReq.CoverImageURL = req.CoverImageUrl
	}

	if req.Location != nil {
		serviceReq.Location = s.converter.ConvertLocationFromPB(req.Location)
	}
	if len(req.Sessions) > 0 {
		serviceReq.Sessions = s.converter.ConvertSessionsFromPB(req.Sessions)
	}
	if req.Detail != nil {
		serviceReq.Detail = s.converter.ConvertDetailFromPB(req.Detail)
	}
	if len(req.Faq) > 0 {
		serviceReq.FAQ = s.converter.ConvertFAQFromPB(req.Faq)
	}

	// Patch event
	event, err := s.eventService.PatchEvent(ctx, serviceReq)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertEventToPB(event), nil
}

// DeleteEvent implements the gRPC DeleteEvent method
func (s *EventServiceServer) DeleteEvent(ctx context.Context, req *common.ID) (*emptypb.Empty, error) {
	// Extract user information from context
	user, err := ezgrpc.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// Delete event
	err = s.eventService.DeleteEvent(ctx, req.Id, user.ID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Return empty response for successful deletion
	return &emptypb.Empty{}, nil
}

// UpdateEventStatus implements the gRPC UpdateEventStatus method
func (s *EventServiceServer) UpdateEventStatus(ctx context.Context, req *consolepb.UpdateEventStatusRequest) (*common.Event, error) {
	// Extract user information from context
	user, err := ezgrpc.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// Update status
	event, err := s.eventService.UpdateEventStatus(ctx, req.Id, req.Status, user.ID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertEventToPB(event), nil
}

// Helper methods

func (s *EventServiceServer) handleServiceError(err error) error {
	switch e := err.(type) {
	case *errors.ValidationError:
		return status.Error(codes.InvalidArgument, e.Error())
	case *errors.BusinessError:
		switch e.Code {
		case errors.ErrorCodePublishedImmutable:
			return status.Error(codes.FailedPrecondition, e.Error())
		case errors.ErrorCodeHasOrders:
			return status.Error(codes.FailedPrecondition, e.Error())
		case errors.ErrorCodeSessionHasOrders:
			return status.Error(codes.FailedPrecondition, e.Error())
		case errors.ErrorCodeLastSession:
			return status.Error(codes.FailedPrecondition, e.Error())
		case errors.ErrorCodeInvalidTransition:
			return status.Error(codes.FailedPrecondition, e.Error())
		default:
			return status.Error(codes.InvalidArgument, e.Error())
		}
	default:
		if err == errors.ErrEventNotFound {
			return status.Error(codes.NotFound, err.Error())
		}
		if err == errors.ErrSessionNotFound {
			return status.Error(codes.NotFound, err.Error())
		}
		return status.Error(codes.Internal, err.Error())
	}
}

// DeleteSession implements the gRPC DeleteSession method
func (s *EventServiceServer) DeleteSession(ctx context.Context, req *consolepb.DeleteSessionRequest) (*emptypb.Empty, error) {
	// Call session service to delete the session
	err := s.eventService.sessionService.DeleteSessionById(ctx, req.Id, req.SessionId)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Return empty response for successful deletion
	return &emptypb.Empty{}, nil
}
