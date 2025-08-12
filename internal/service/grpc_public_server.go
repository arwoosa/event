package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
func (s *PublicEventServiceServer) SearchEvents(ctx context.Context, req *pb.SearchEventsRequest) (*pb.EventListResponse, error) {
	// Convert gRPC request to service request
	serviceReq := &SearchEventsRequest{}

	// Only set optional fields if they are provided and non-empty
	if req.BrandId != nil && *req.BrandId != "" {
		serviceReq.BrandID = req.BrandId
	}
	if req.TitleSearch != nil && *req.TitleSearch != "" {
		serviceReq.TitleSearch = req.TitleSearch
	}
	if req.LocationLat != nil {
		serviceReq.LocationLat = req.LocationLat
	}
	if req.LocationLng != nil {
		serviceReq.LocationLng = req.LocationLng
	}
	if req.SortBy != nil && *req.SortBy != "" {
		serviceReq.SortBy = req.SortBy
	}
	if req.SortOrder != nil && *req.SortOrder != "" {
		serviceReq.SortOrder = req.SortOrder
	}
	if req.PageToken != nil && *req.PageToken != "" {
		serviceReq.PageToken = req.PageToken
	}
	if req.Page != nil {
		serviceReq.Page = req.Page
	}
	if req.PageSize != nil {
		serviceReq.PageSize = req.PageSize
	}

	// Handle location radius
	if req.LocationRadius != nil {
		radius := int(*req.LocationRadius)
		serviceReq.LocationRadius = &radius
	}

	// Handle time filters
	if req.SessionStartTimeFrom != nil && *req.SessionStartTimeFrom != "" {
		serviceReq.SessionStartTimeFrom = req.SessionStartTimeFrom
	}
	if req.SessionStartTimeTo != nil && *req.SessionStartTimeTo != "" {
		serviceReq.SessionStartTimeTo = req.SessionStartTimeTo
	}

	// Search events
	result, err := s.publicService.SearchEvents(ctx, serviceReq)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	// Convert to protobuf response (sessions are now embedded in events)
	eventsPB := make([]*pb.Event, len(result.Events))
	for i, event := range result.Events {
		eventsPB[i] = s.converter.ConvertEventToPB(event)
	}

	paginationPB := s.converter.ConvertPaginationToPB(result.Pagination)

	return &pb.EventListResponse{
		Events:     eventsPB,
		Pagination: paginationPB,
	}, nil
}

// GetEvent implements the gRPC GetEvent method for public access
func (s *PublicEventServiceServer) GetEvent(ctx context.Context, req *api.ID) (*pb.Event, error) {
	// Get event
	event, err := s.publicService.GetEvent(ctx, req.Id)
	if err != nil {
		return nil, s.handleServiceError(err)
	}

	return s.converter.ConvertEventToPB(event), nil
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
