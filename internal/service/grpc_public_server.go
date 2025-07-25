package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"event/api"
	pb "event/api/event"
	"event/internal/models"
)

// PublicEventServiceServer implements the generated gRPC PublicEventService interface
type PublicEventServiceServer struct {
	pb.UnimplementedPublicEventServiceServer
	publicService *PublicService
	converter     *ProtobufConverter
}

// NewPublicEventServiceServer creates a new gRPC public event service server
func NewPublicEventServiceServer(publicService *PublicService) *PublicEventServiceServer {
	return &PublicEventServiceServer{
		publicService: publicService,
		converter:     NewProtobufConverter(),
	}
}

// SearchEvents implements the gRPC SearchEvents method
func (s *PublicEventServiceServer) SearchEvents(ctx context.Context, req *pb.SearchEventsRequest) (*api.Response, error) {
	// Convert gRPC request to service request
	serviceReq := &SearchEventsRequest{
		BrandID:     req.BrandId,
		TitleSearch: req.TitleSearch,
		LocationLat: req.LocationLat,
		LocationLng: req.LocationLng,
		SortBy:      req.SortBy,
		SortOrder:   req.SortOrder,
		PageToken:   req.PageToken,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}

	// Handle location radius
	if req.LocationRadius != nil {
		radius := int(*req.LocationRadius)
		serviceReq.LocationRadius = &radius
	}

	// Handle time filters
	if req.SessionStartTimeFrom != nil {
		serviceReq.SessionStartTimeFrom = req.SessionStartTimeFrom
	}
	if req.SessionStartTimeTo != nil {
		serviceReq.SessionStartTimeTo = req.SessionStartTimeTo
	}

	// Search events
	result, err := s.publicService.SearchEvents(ctx, serviceReq)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response
	eventsPB := make([]*pb.Event, len(result.Events))
	for i, event := range result.Events {
		eventsPB[i] = s.converter.ConvertEventToPB(event)
	}

	paginationPB := s.converter.ConvertPaginationToPB(result.Pagination)
	listResponse := &pb.EventListResponse{
		Events:     eventsPB,
		Pagination: paginationPB,
	}
	
	return s.createSuccessResponse(listResponse)
}

// GetEvent implements the gRPC GetEvent method for public access
func (s *PublicEventServiceServer) GetEvent(ctx context.Context, req *api.ID) (*api.Response, error) {
	// Get event
	event, err := s.publicService.GetEvent(ctx, req.Id)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response
	eventPB := s.converter.ConvertEventToPB(event)
	eventResponse := &pb.EventResponse{Event: eventPB}
	
	return s.createSuccessResponse(eventResponse)
}

// Helper methods

func (s *PublicEventServiceServer) handleServiceError(err error) error {
	// Handle service errors for public API
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

func (s *PublicEventServiceServer) createSuccessResponse(data interface{}) (*api.Response, error) {
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