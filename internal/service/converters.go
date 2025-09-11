package service

import (
	"time"

	"github.com/arwoosa/event/gen/pb/common"
	"github.com/arwoosa/event/internal/dao/repository"
	"github.com/arwoosa/event/internal/helper"
	"github.com/arwoosa/event/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/structpb"
)

// ProtobufConverter provides methods to convert between domain models and protobuf messages
type ProtobufConverter struct{}

// NewProtobufConverter creates a new protobuf converter
func NewProtobufConverter() *ProtobufConverter {
	return &ProtobufConverter{}
}

// ConvertEventToPB converts a domain Event to protobuf Event
// Sessions are now embedded in the Event model, so no separate sessions parameter needed
func (c *ProtobufConverter) ConvertEventToPB(event *models.Event) *common.Event {
	return &common.Event{
		Id:            event.ID.Hex(),
		Title:         event.Title,
		Summary:       event.Summary,
		Status:        event.Status,
		Visibility:    event.Visibility,
		CoverImageUrl: event.CoverImageURL,
		Location:      c.ConvertLocationToPB(&event.Location),
		Sessions:      c.ConvertSessionsToPB(event.Sessions),
		Detail:        c.ConvertDetailToPB(event.Detail),
		Faq:           c.ConvertFAQToPB(event.FAQ),
		CreatedAt:     event.CreatedAt.Format(time.RFC3339),
		CreatedBy:     event.CreatedBy,
		UpdatedAt:     event.UpdatedAt.Format(time.RFC3339),
		UpdatedBy:     event.UpdatedBy,
	}
}

// ConvertLocationToPB converts a domain Location to protobuf Location
func (c *ProtobufConverter) ConvertLocationToPB(location *models.Location) *common.Location {
	return &common.Location{
		Name:    location.Name,
		Address: location.Address,
		PlaceId: location.PlaceID,
		Coordinates: &common.GeoJSONPoint{
			Type:        location.Coordinates.Type,
			Coordinates: []float64{location.Coordinates.Coordinates[0], location.Coordinates.Coordinates[1]},
		},
	}
}

// ConvertSessionsToPB converts domain Sessions to protobuf Sessions
func (c *ProtobufConverter) ConvertSessionsToPB(sessions []models.Session) []*common.Session {
	if sessions == nil {
		return []*common.Session{}
	}

	sessionsPB := make([]*common.Session, len(sessions))
	for i, session := range sessions {
		sessionPB := &common.Session{
			Id:        session.ID.Hex(),
			Name:      session.Name,
			StartTime: session.StartTime.Format(time.RFC3339),
			EndTime:   session.EndTime.Format(time.RFC3339),
		}
		// Convert capacity - handle nil pointer with safe conversion
		if session.Capacity != nil {
			capacity := helper.SafeInt32FromInt(*session.Capacity)
			sessionPB.Capacity = &capacity
		}
		sessionsPB[i] = sessionPB
	}
	return sessionsPB
}

// ConvertSessionToPB converts domain Session to protobuf Session
func (c *ProtobufConverter) ConvertSessionToPB(session *models.Session) *common.Session {
	if session == nil {
		return nil
	}

	sessionPB := &common.Session{
		Id:        session.ID.Hex(),
		Name:      session.Name,
		StartTime: session.StartTime.Format(time.RFC3339),
		EndTime:   session.EndTime.Format(time.RFC3339),
	}
	// Convert capacity - handle nil pointer with safe conversion
	if session.Capacity != nil {
		capacity := helper.SafeInt32FromInt(*session.Capacity)
		sessionPB.Capacity = &capacity
	}
	return sessionPB
}

// ConvertDetailToPB converts domain DetailBlocks to protobuf DetailBlocks
func (c *ProtobufConverter) ConvertDetailToPB(detail []models.DetailBlock) []*common.DetailBlock {
	blocks := make([]*common.DetailBlock, len(detail))
	for i, block := range detail {
		pbBlock := &common.DetailBlock{
			Type: block.Type,
		}

		// Convert data based on type
		switch block.Type {
		case models.BlockTypeText:
			if textData, ok := block.Data.(models.TextData); ok {
				pbBlock.Data = &common.DetailBlock_TextData{
					TextData: &common.TextData{
						Content: textData.Content,
					},
				}
			} else if primitiveD, ok := block.Data.(primitive.D); ok {
				// Handle primitive.D from MongoDB
				if content := extractStringFromPrimitiveD(primitiveD, "content"); content != "" {
					pbBlock.Data = &common.DetailBlock_TextData{
						TextData: &common.TextData{
							Content: content,
						},
					}
				}
			}
		case models.BlockTypeImage:
			if imageData, ok := block.Data.(models.ImageData); ok {
				pbBlock.Data = &common.DetailBlock_ImageData{
					ImageData: &common.ImageData{
						Url:     imageData.URL,
						Alt:     imageData.Alt,
						Caption: imageData.Caption,
					},
				}
			} else if primitiveD, ok := block.Data.(primitive.D); ok {
				// Handle primitive.D from MongoDB
				pbBlock.Data = &common.DetailBlock_ImageData{
					ImageData: &common.ImageData{
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
func (c *ProtobufConverter) ConvertFAQToPB(faqs []models.FAQ) []*common.FAQ {
	faqsPB := make([]*common.FAQ, len(faqs))
	for i, faq := range faqs {
		faqsPB[i] = &common.FAQ{
			Question: faq.Question,
			Answer:   faq.Answer,
		}
	}
	return faqsPB
}

// ConvertPaginationToPB converts repository Pagination to protobuf Pagination
func (c *ProtobufConverter) ConvertPaginationToPB(pagination *repository.Pagination) *common.Pagination {
	paginationPB := &common.Pagination{
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
		count := helper.SafeInt32FromInt64(*pagination.TotalCount)
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
func (c *ProtobufConverter) ConvertLocationFromPB(location *common.Location) *LocationRequest {
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
func (c *ProtobufConverter) ConvertSessionsFromPB(sessions []*common.Session) []*SessionRequest {
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
func (c *ProtobufConverter) ConvertDetailFromPB(detail []*common.DetailBlock) []DetailBlockRequest {
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
func (c *ProtobufConverter) ConvertFAQFromPB(faqs []*common.FAQ) []*FAQRequest {
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

// === Form Conversion Methods ===

// EventFormToProtobuf converts a domain EventForm to protobuf EventForm
func (c *ProtobufConverter) EventFormToProtobuf(form *models.EventForm) (*common.EventForm, error) {
	// Convert interface{} to *structpb.Struct for protobuf
	var schema, uiSchema *structpb.Struct
	var err error

	if form.Schema != nil {
		schemaMap := c.convertMongoDataToMap(form.Schema)
		if schemaMap != nil {
			schema, err = structpb.NewStruct(schemaMap)
			if err != nil {
				return nil, err
			}
		}
	}

	if form.UISchema != nil {
		uiSchemaMap := c.convertMongoDataToMap(form.UISchema)
		if uiSchemaMap != nil {
			uiSchema, err = structpb.NewStruct(uiSchemaMap)
			if err != nil {
				return nil, err
			}
		}
	}

	formpb := &common.EventForm{
		Id:        form.ID.Hex(),
		EventId:   form.EventID.Hex(),
		Schema:    schema,
		Uischema:  uiSchema,
		CreatedAt: form.CreatedAt.Format(time.RFC3339),
		CreatedBy: form.CreatedBy,
		UpdatedAt: form.UpdatedAt.Format(time.RFC3339),
		UpdatedBy: form.UpdatedBy,
	}
	return formpb, nil
}

func (c *ProtobufConverter) convertMongoDataToMap(data interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case primitive.D:
		result := make(map[string]interface{})
		for _, elem := range v {
			result[elem.Key] = c.convertValue(elem.Value)
		}
		return result
	case map[string]interface{}:
		// Already correct type, but need to recursively process values
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = c.convertValue(value)
		}
		return result
	case primitive.M:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = c.convertValue(value)
		}
		return result
	default:
		// If not a map type, try to convert to single value map
		if converted := c.convertValue(v); converted != nil {
			return map[string]interface{}{"value": converted}
		}
		return nil
	}
}

// convertValue converts single MongoDB values, handling nested primitive.D, primitive.A etc
func (s *ProtobufConverter) convertValue(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case primitive.D:
		result := make(map[string]interface{})
		for _, elem := range v {
			result[elem.Key] = s.convertValue(elem.Value)
		}
		return result
	case primitive.A:
		result := make([]interface{}, len(v))
		for i, elem := range v {
			result[i] = s.convertValue(elem)
		}
		return result
	case primitive.M:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = s.convertValue(value)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = s.convertValue(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, elem := range v {
			result[i] = s.convertValue(elem)
		}
		return result
	default:
		return data
	}
}
