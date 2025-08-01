package service

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"event/api"
	pb "event/api/event"
	"event/internal/dao/repository"
	"event/internal/models"
)

// EventServiceServer implements the generated gRPC EventService interface
type EventServiceServer struct {
	pb.UnimplementedEventServiceServer
	eventService *EventService
	converter    *ProtobufConverter
}

// NewEventServiceServer creates a new gRPC event service server
func NewEventServiceServer(eventService *EventService) *EventServiceServer {
	return &EventServiceServer{
		eventService: eventService,
		converter:    NewProtobufConverter(),
	}
}

// CreateEvent implements the gRPC CreateEvent method
func (s *EventServiceServer) CreateEvent(ctx context.Context, req *pb.CreateEventRequest) (*api.Response, error) {
	// Extract user and brand information from context
	userID, brandID, err := s.extractUserAndBrandFromContext(ctx)
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
		BrandID:       brandID,
		UserID:        userID,
	}

	// Create event
	event, err := s.eventService.CreateEvent(ctx, serviceReq)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Get sessions for the created event
	sessions, err := s.eventService.sessionService.GetSessionsForEvent(ctx, event.ID.Hex(), brandID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response
	eventPB := s.converter.ConvertEventToPB(event, sessions)
	eventResponse := &pb.EventResponse{Event: eventPB}

	return s.createSuccessResponse(eventResponse)
}

// GetEventList implements the gRPC GetEventList method
func (s *EventServiceServer) GetEventList(ctx context.Context, req *pb.GetEventListRequest) (*api.Response, error) {
	// Extract brand information from context
	_, brandID, err := s.extractUserAndBrandFromContext(ctx)
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

	// Handle pagination
	if req.PageSize != nil {
		pageSize := int(*req.PageSize)
		if pageSize > 0 && pageSize <= 100 {
			filter.Limit = pageSize
		}
	}
	if req.Page != nil && *req.Page > 0 {
		filter.Offset = int((*req.Page - 1) * int32(filter.Limit))
		filter.PageToken = nil // Don't use cursor pagination if page is specified
	}

	// Get events
	result, err := s.eventService.GetEventList(ctx, brandID, filter)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Get event IDs for batch session fetching
	eventIDs := make([]string, len(result.Events))
	for i, event := range result.Events {
		eventIDs[i] = event.ID.Hex()
	}

	// Get sessions for all events in batch
	sessionsByEvent, err := s.eventService.sessionService.GetSessionsForEvents(ctx, eventIDs)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response
	eventsPB := make([]*pb.Event, len(result.Events))
	for i, event := range result.Events {
		eventID := event.ID.Hex()
		sessions := sessionsByEvent[eventID]
		if sessions == nil {
			sessions = []*models.Session{}
		}
		eventsPB[i] = s.converter.ConvertEventToPB(event, sessions)
	}

	paginationPB := s.converter.ConvertPaginationToPB(result.Pagination)
	listResponse := &pb.EventListResponse{
		Events:     eventsPB,
		Pagination: paginationPB,
	}

	return s.createSuccessResponse(listResponse)
}

// GetEvent implements the gRPC GetEvent method
func (s *EventServiceServer) GetEvent(ctx context.Context, req *api.ID) (*api.Response, error) {
	// Extract brand information from context
	_, brandID, err := s.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get event
	event, err := s.eventService.GetEvent(ctx, brandID, req.Id)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Get sessions for the event
	sessions, err := s.eventService.sessionService.GetSessionsForEvent(ctx, req.Id, brandID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response
	eventPB := s.converter.ConvertEventToPB(event, sessions)
	eventResponse := &pb.EventResponse{Event: eventPB}

	return s.createSuccessResponse(eventResponse)
}

// PatchEvent implements the gRPC PatchEvent method
func (s *EventServiceServer) PatchEvent(ctx context.Context, req *pb.PatchEventRequest) (*api.Response, error) {
	// Extract user and brand information from context
	userID, brandID, err := s.extractUserAndBrandFromContext(ctx)
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
	if req.Status != nil && *req.Status != "" {
		serviceReq.Status = req.Status
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
	event, err := s.eventService.PatchEvent(ctx, brandID, serviceReq)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Get sessions for the patched event
	sessions, err := s.eventService.sessionService.GetSessionsForEvent(ctx, event.ID.Hex(), brandID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response
	eventPB := s.converter.ConvertEventToPB(event, sessions)
	eventResponse := &pb.EventResponse{Event: eventPB}

	return s.createSuccessResponse(eventResponse)
}

// DeleteEvent implements the gRPC DeleteEvent method
func (s *EventServiceServer) DeleteEvent(ctx context.Context, req *api.ID) (*api.Response, error) {
	// Extract user and brand information from context
	userID, brandID, err := s.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Delete event
	err = s.eventService.DeleteEvent(ctx, brandID, req.Id, userID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.createSuccessResponse(nil)
}

// UpdateEventStatus implements the gRPC UpdateEventStatus method
func (s *EventServiceServer) UpdateEventStatus(ctx context.Context, req *pb.UpdateEventStatusRequest) (*api.Response, error) {
	// Extract user and brand information from context
	userID, brandID, err := s.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Update status
	event, err := s.eventService.UpdateEventStatus(ctx, brandID, req.Id, req.Status, userID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Get sessions for the updated event
	sessions, err := s.eventService.sessionService.GetSessionsForEvent(ctx, event.ID.Hex(), brandID)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response
	eventPB := s.converter.ConvertEventToPB(event, sessions)
	eventResponse := &pb.EventResponse{Event: eventPB}

	return s.createSuccessResponse(eventResponse)
}

// Helper methods

func (s *EventServiceServer) extractUserAndBrandFromContext(ctx context.Context) (userID, brandID string, err error) {
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

	// Extract brand ID
	brandIDValues := md.Get("brand-id")
	if len(brandIDValues) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing brand-id header")
	}
	brandID = brandIDValues[0]

	return userID, brandID, nil
}

func (s *EventServiceServer) handleServiceError(err error) error {
	switch e := err.(type) {
	case *models.ValidationError:
		return status.Error(codes.InvalidArgument, e.Error())
	case *models.BusinessError:
		switch e.Code {
		case "PUBLISHED_IMMUTABLE":
			return status.Error(codes.FailedPrecondition, e.Error())
		case "HAS_ORDERS":
			return status.Error(codes.FailedPrecondition, e.Error())
		case "INVALID_TRANSITION":
			return status.Error(codes.FailedPrecondition, e.Error())
		default:
			return status.Error(codes.InvalidArgument, e.Error())
		}
	default:
		if err == models.ErrEventNotFound {
			return status.Error(codes.NotFound, err.Error())
		}
		return status.Error(codes.Internal, err.Error())
	}
}

func (s *EventServiceServer) createSuccessResponse(data interface{}) (*api.Response, error) {
	var anyData *anypb.Any
	if data != nil {
		msg, ok := data.(proto.Message)
		if !ok {
			return nil, status.Error(codes.Internal, "data is not a proto message")
		}

		var err error
		anyData, err = anypb.New(msg)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to create response data")
		}
	}

	return &api.Response{
		Status: "success",
		Code:   1000,
		Data:   anyData,
	}, nil
}
