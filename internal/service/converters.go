package service

import (
	"time"

	pb "event/api/event"
	"event/internal/dao/repository"
	"event/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProtobufConverter provides methods to convert between domain models and protobuf messages
type ProtobufConverter struct{}

// NewProtobufConverter creates a new protobuf converter
func NewProtobufConverter() *ProtobufConverter {
	return &ProtobufConverter{}
}

// ConvertEventToPB converts a domain Event to protobuf Event
// Sessions are now embedded in the Event model, so no separate sessions parameter needed
func (c *ProtobufConverter) ConvertEventToPB(event *models.Event) *pb.Event {
	return &pb.Event{
		Id:            event.ID.Hex(),
		Title:         event.Title,
		MerchantId:    event.MerchantID.Hex(),
		Summary:       event.Summary,
		Status:        event.Status,
		Visibility:    event.Visibility,
		CoverImageUrl: event.CoverImageURL,
		Location:      c.ConvertLocationToPB(&event.Location),
		Sessions:      c.ConvertSessionsToPB(event.Sessions),
		Detail:        c.ConvertDetailToPB(event.Detail),
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

// ConvertSessionsToPB converts domain Sessions to protobuf Sessions
func (c *ProtobufConverter) ConvertSessionsToPB(sessions []models.Session) []*pb.Session {
	if sessions == nil {
		return []*pb.Session{}
	}

	sessionsPB := make([]*pb.Session, len(sessions))
	for i, session := range sessions {
		sessionPB := &pb.Session{
			Id:        session.ID.Hex(),
			Name:      session.Name,
			StartTime: session.StartTime.Format(time.RFC3339),
			EndTime:   session.EndTime.Format(time.RFC3339),
		}
		// Convert capacity - handle nil pointer
		if session.Capacity != nil {
			capacity := int32(*session.Capacity)
			sessionPB.Capacity = &capacity
		}
		sessionsPB[i] = sessionPB
	}
	return sessionsPB
}

// ConvertDetailToPB converts domain DetailBlocks to protobuf DetailBlocks
func (c *ProtobufConverter) ConvertDetailToPB(detail []models.DetailBlock) []*pb.DetailBlock {
	blocks := make([]*pb.DetailBlock, len(detail))
	for i, block := range detail {
		pbBlock := &pb.DetailBlock{
			Type: block.Type,
		}

		// Convert data based on type
		switch block.Type {
		case models.BlockTypeText:
			if textData, ok := block.Data.(models.TextData); ok {
				pbBlock.Data = &pb.DetailBlock_TextData{
					TextData: &pb.TextData{
						Content: textData.Content,
					},
				}
			} else if primitiveD, ok := block.Data.(primitive.D); ok {
				// Handle primitive.D from MongoDB
				if content := extractStringFromPrimitiveD(primitiveD, "content"); content != "" {
					pbBlock.Data = &pb.DetailBlock_TextData{
						TextData: &pb.TextData{
							Content: content,
						},
					}
				}
			}
		case models.BlockTypeImage:
			if imageData, ok := block.Data.(models.ImageData); ok {
				pbBlock.Data = &pb.DetailBlock_ImageData{
					ImageData: &pb.ImageData{
						Url:     imageData.URL,
						Alt:     imageData.Alt,
						Caption: imageData.Caption,
					},
				}
			} else if primitiveD, ok := block.Data.(primitive.D); ok {
				// Handle primitive.D from MongoDB
				pbBlock.Data = &pb.DetailBlock_ImageData{
					ImageData: &pb.ImageData{
						Url:     extractStringFromPrimitiveD(primitiveD, "url"),
						Alt:     extractStringFromPrimitiveD(primitiveD, "alt"),
						Caption: extractStringFromPrimitiveD(primitiveD, "caption"),
					},
				}
			}
		}
		blocks[i] = pbBlock
	}

	return blocks
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
		sessionReq := &SessionRequest{
			ID:        session.Id,
			Name:      session.Name,
			StartTime: session.StartTime,
			EndTime:   session.EndTime,
		}
		// Convert capacity - handle nil pointer
		if session.Capacity != nil {
			capacity := int(*session.Capacity)
			sessionReq.Capacity = &capacity
		}
		sessionReqs[i] = sessionReq
	}
	return sessionReqs
}

// ConvertDetailFromPB converts protobuf DetailBlocks to service DetailBlockRequest slice
func (c *ProtobufConverter) ConvertDetailFromPB(detail []*pb.DetailBlock) []DetailBlockRequest {
	if detail == nil {
		return nil
	}

	blocks := make([]DetailBlockRequest, len(detail))
	for i, block := range detail {
		blockReq := DetailBlockRequest{
			Type: block.Type,
		}

		// Convert data based on type
		switch block.Type {
		case models.BlockTypeText:
			if textData := block.GetTextData(); textData != nil {
				blockReq.Data = models.TextData{
					Content: textData.Content,
				}
			}
		case models.BlockTypeImage:
			if imageData := block.GetImageData(); imageData != nil {
				blockReq.Data = models.ImageData{
					URL:     imageData.Url,
					Alt:     imageData.Alt,
					Caption: imageData.Caption,
				}
			}
		}
		blocks[i] = blockReq
	}

	return blocks
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

// extractStringFromPrimitiveD extracts a string value from primitive.D by key
func extractStringFromPrimitiveD(d primitive.D, key string) string {
	for _, elem := range d {
		if elem.Key == key {
			if str, ok := elem.Value.(string); ok {
				return str
			}
		}
	}
	return ""
}
