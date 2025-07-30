package service

import (
	"time"

	pb "event/api/event"
	"event/internal/dao/repository"
	"event/internal/models"
)

// ProtobufConverter provides methods to convert between domain models and protobuf messages
type ProtobufConverter struct{}

// NewProtobufConverter creates a new protobuf converter
func NewProtobufConverter() *ProtobufConverter {
	return &ProtobufConverter{}
}

// ConvertEventToPB converts a domain Event to protobuf Event
func (c *ProtobufConverter) ConvertEventToPB(event *models.Event, sessions []*models.Session) *pb.Event {
	return &pb.Event{
		Id:            event.ID.Hex(),
		Title:         event.Title,
		BrandId:       event.BrandID.Hex(),
		Summary:       event.Summary,
		Status:        event.Status,
		Visibility:    event.Visibility,
		CoverImageUrl: event.CoverImageURL,
		Location:      c.ConvertLocationToPB(&event.Location),
		Sessions:      c.ConvertSessionCollectionToPB(sessions),
		Detail:        c.ConvertDetailToPB(&event.Detail),
		Faq:           c.ConvertFAQToPB(event.FAQ),
		CreatedAt:     event.CreatedAt.Format(time.RFC3339),
		CreatedBy:     event.CreatedBy.Hex(),
		UpdatedAt:     event.UpdatedAt.Format(time.RFC3339),
		UpdatedBy:     event.UpdatedBy.Hex(),
	}
}

// ConvertLocationToPB converts a domain Location to protobuf Location
func (c *ProtobufConverter) ConvertLocationToPB(location *models.Location) *pb.Location {
	return &pb.Location{
		Name:    location.Name,
		Address: location.Address,
		PlaceId: location.PlaceID,
		Coordinates: &pb.GeoJSONPoint{
			Type:        location.Coordinates.Type,
			Coordinates: []float64{location.Coordinates.Coordinates[0], location.Coordinates.Coordinates[1]},
		},
	}
}

// ConvertSessionCollectionToPB converts Session collection models to protobuf Sessions
func (c *ProtobufConverter) ConvertSessionCollectionToPB(sessions []*models.Session) []*pb.Session {
	if sessions == nil {
		return []*pb.Session{}
	}

	sessionsPB := make([]*pb.Session, len(sessions))
	for i, session := range sessions {
		sessionsPB[i] = &pb.Session{
			Id:        session.ID.Hex(),
			StartTime: session.StartTime.Format(time.RFC3339),
			EndTime:   session.EndTime.Format(time.RFC3339),
		}
	}
	return sessionsPB
}

// ConvertSessionsToPB converts domain Sessions to protobuf Sessions (deprecated - use ConvertSessionCollectionToPB)
func (c *ProtobufConverter) ConvertSessionsToPB(sessions []models.Session) []*pb.Session {
	sessionsPB := make([]*pb.Session, len(sessions))
	for i, session := range sessions {
		sessionsPB[i] = &pb.Session{
			Id:        session.ID.Hex(),
			StartTime: session.StartTime.Format(time.RFC3339),
			EndTime:   session.EndTime.Format(time.RFC3339),
		}
	}
	return sessionsPB
}

// ConvertDetailToPB converts a domain Detail to protobuf Detail
func (c *ProtobufConverter) ConvertDetailToPB(detail *models.Detail) *pb.Detail {
	return &pb.Detail{
		Content:     detail.Content,
		ContentType: detail.ContentType,
	}
}

// ConvertFAQToPB converts domain FAQ to protobuf FAQ
func (c *ProtobufConverter) ConvertFAQToPB(faqs []models.FAQ) []*pb.FAQ {
	faqsPB := make([]*pb.FAQ, len(faqs))
	for i, faq := range faqs {
		faqsPB[i] = &pb.FAQ{
			Question: faq.Question,
			Answer:   faq.Answer,
		}
	}
	return faqsPB
}

// ConvertPaginationToPB converts repository Pagination to protobuf Pagination
func (c *ProtobufConverter) ConvertPaginationToPB(pagination *repository.Pagination) *pb.Pagination {
	paginationPB := &pb.Pagination{
		HasNext: &pagination.HasNext,
		HasPrev: &pagination.HasPrev,
	}

	if pagination.NextPageToken != nil {
		paginationPB.NextPageToken = pagination.NextPageToken
	}
	if pagination.PrevPageToken != nil {
		paginationPB.PrevPageToken = pagination.PrevPageToken
	}
	if pagination.TotalCount != nil {
		count := int32(*pagination.TotalCount)
		paginationPB.TotalCount = &count
	}
	if pagination.CurrentPage != nil {
		paginationPB.CurrentPage = pagination.CurrentPage
	}
	if pagination.TotalPages != nil {
		paginationPB.TotalPages = pagination.TotalPages
	}

	return paginationPB
}

// ConvertLocationFromPB converts protobuf Location to service LocationRequest
func (c *ProtobufConverter) ConvertLocationFromPB(location *pb.Location) *LocationRequest {
	if location == nil {
		return nil
	}

	locationReq := &LocationRequest{
		Name:    location.Name,
		Address: location.Address,
		PlaceID: location.PlaceId,
	}

	if location.Coordinates != nil {
		locationReq.Coordinates = &GeoJSONPointRequest{
			Type:        location.Coordinates.Type,
			Coordinates: [2]float64{location.Coordinates.Coordinates[0], location.Coordinates.Coordinates[1]},
		}
	}

	return locationReq
}

// ConvertSessionsFromPB converts protobuf Sessions to service SessionRequest
func (c *ProtobufConverter) ConvertSessionsFromPB(sessions []*pb.Session) []*SessionRequest {
	sessionReqs := make([]*SessionRequest, len(sessions))
	for i, session := range sessions {
		sessionReqs[i] = &SessionRequest{
			StartTime: session.StartTime,
			EndTime:   session.EndTime,
		}
	}
	return sessionReqs
}

// ConvertDetailFromPB converts protobuf Detail to service DetailRequest
func (c *ProtobufConverter) ConvertDetailFromPB(detail *pb.Detail) *DetailRequest {
	if detail == nil {
		return nil
	}
	return &DetailRequest{
		Content:     detail.Content,
		ContentType: detail.ContentType,
	}
}

// ConvertFAQFromPB converts protobuf FAQ to service FAQRequest
func (c *ProtobufConverter) ConvertFAQFromPB(faqs []*pb.FAQ) []*FAQRequest {
	faqReqs := make([]*FAQRequest, len(faqs))
	for i, faq := range faqs {
		faqReqs[i] = &FAQRequest{
			Question: faq.Question,
			Answer:   faq.Answer,
		}
	}
	return faqReqs
}
