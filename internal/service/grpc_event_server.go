package service

import (
	"context"
	"time"

	"github.com/arwoosa/event/conf"
	"github.com/arwoosa/event/gen/pb/common"
	consolepb "github.com/arwoosa/event/gen/pb/console"
	"github.com/arwoosa/event/internal/dao/repository"
	"github.com/arwoosa/event/internal/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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
	// Extract user and merchant information from context
	userID, merchantID, err := s.extractUserAndMerchantFromContext(ctx)
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
		MerchantID:    merchantID,
		UserID:        userID,
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
	// Extract merchant information from context
	_, merchantID, err := s.extractUserAndMerchantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Build filter - only set non-empty string values
	filter := &repository.EventFilter{
		Limit:  20, // Default
		Offset: 0,
	}

	// Only set filter values if they are non-empty
	if req.Status != nil && *req.Status != "" {
		filter.Status = req.Status
	}
	if req.Visibility != nil && *req.Visibility != "" {
		filter.Visibility = req.Visibility
	}
	if req.TitleSearch != nil && *req.TitleSearch != "" {
		filter.TitleSearch = req.TitleSearch
	}
	if req.SortBy != nil && *req.SortBy != "" {
		filter.SortBy = req.SortBy
	}
	if req.SortOrder != nil && *req.SortOrder != "" {
		filter.SortOrder = req.SortOrder
	}
	if req.PageToken != nil && *req.PageToken != "" {
		filter.PageToken = req.PageToken
	}

	// Handle time filters
	if req.SessionStartTimeFrom != nil && *req.SessionStartTimeFrom != "" {
		if t, err := time.Parse(time.RFC3339, *req.SessionStartTimeFrom); err == nil {
			filter.SessionStartTimeFrom = &t
		}
	}
	if req.SessionStartTimeTo != nil && *req.SessionStartTimeTo != "" {
		if t, err := time.Parse(time.RFC3339, *req.SessionStartTimeTo); err == nil {
			filter.SessionStartTimeTo = &t
		}
	}

	// Handle pagination with fallback to hardcoded defaults if config is not available
	defaultPageSize := 20 // Default fallback
	maxPageSize := 100    // Default fallback
	if s.paginationConfig != nil {
		if s.paginationConfig.DefaultPageSize > 0 {
			defaultPageSize = s.paginationConfig.DefaultPageSize
		}
		if s.paginationConfig.MaxPageSize > 0 {
			maxPageSize = s.paginationConfig.MaxPageSize
		}
	}

	filter.Limit = defaultPageSize
	if req.PageSize != nil {
		pageSize := int(*req.PageSize)
		if pageSize > 0 && pageSize <= maxPageSize {
			filter.Limit = pageSize
		}
	}
	if req.Page != nil && *req.Page > 0 {
		filter.Offset = int((*req.Page - 1) * int32(filter.Limit))
		filter.PageToken = nil // Don't use cursor pagination if page is specified
	}

	// Get events
	result, err := s.eventService.GetEventList(ctx, merchantID, filter)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response (sessions are now embedded in events)
	eventsPB := make([]*common.Event, len(result.Events))
	for i, event := range result.Events {
		eventsPB[i] = s.converter.ConvertEventToPB(event)
	}

	paginationPB := s.converter.ConvertPaginationToPB(result.Pagination)
	return &common.EventListResponse{
		Events:     eventsPB,
		Pagination: paginationPB,
	}, nil
}

// GetEvent implements the gRPC GetEvent method
func (s *EventServiceServer) GetEvent(ctx context.Context, req *common.ID) (*common.Event, error) {
	// Extract merchant information from context
	_, merchantID, err := s.extractUserAndMerchantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get event
	event, err := s.eventService.GetEvent(ctx, merchantID, req.Id)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertEventToPB(event), nil
}

// PatchEvent implements the gRPC PatchEvent method
func (s *EventServiceServer) PatchEvent(ctx context.Context, req *consolepb.PatchEventRequest) (*common.Event, error) {
	// Extract user and merchant information from context
	userID, merchantID, err := s.extractUserAndMerchantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Convert gRPC request to service request
	serviceReq := &PatchEventRequest{
		ID:     req.Id,
		UserID: userID,
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
	event, err := s.eventService.PatchEvent(ctx, merchantID, serviceReq)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertEventToPB(event), nil
}

// DeleteEvent implements the gRPC DeleteEvent method
func (s *EventServiceServer) DeleteEvent(ctx context.Context, req *common.ID) (*emptypb.Empty, error) {
	// Extract user and merchant information from context
	userID, merchantID, err := s.extractUserAndMerchantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Delete event
	err = s.eventService.DeleteEvent(ctx, merchantID, req.Id, userID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Return empty response for successful deletion
	return &emptypb.Empty{}, nil
}

// UpdateEventStatus implements the gRPC UpdateEventStatus method
func (s *EventServiceServer) UpdateEventStatus(ctx context.Context, req *consolepb.UpdateEventStatusRequest) (*common.Event, error) {
	// Extract user and merchant information from context
	userID, merchantID, err := s.extractUserAndMerchantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Update status
	event, err := s.eventService.UpdateEventStatus(ctx, merchantID, req.Id, req.Status, userID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertEventToPB(event), nil
}

// Helper methods

func (s *EventServiceServer) extractUserAndMerchantFromContext(ctx context.Context) (userID, merchantID string, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Extract user ID
	userIDValues := md.Get("user-id")
	if len(userIDValues) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing user-id header")
	}
	userID = userIDValues[0]

	// Extract merchant ID
	merchantIDValues := md.Get("merchant-id")
	if len(merchantIDValues) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing merchant-id header")
	}
	merchantID = merchantIDValues[0]

	return userID, merchantID, nil
}

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
	// Extract user and merchant information from context
	_, merchantID, err := s.extractUserAndMerchantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Call session service to delete the session
	err = s.eventService.sessionService.DeleteSessionById(ctx, req.EventId, req.SessionId, merchantID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Return empty response for successful deletion
	return &emptypb.Empty{}, nil
}
