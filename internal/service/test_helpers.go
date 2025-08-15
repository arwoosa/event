package service

import (
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"event/internal/dao/repository"
	"event/internal/models"
	"event/internal/testutils"
)

// Conversion functions to avoid circular imports

// ConvertTestCreateEventRequest converts testutils request to service request
func ConvertTestCreateEventRequest(req *testutils.TestCreateEventRequest) *CreateEventRequest {
	serviceReq := &CreateEventRequest{
		Title:         req.Title,
		Summary:       req.Summary,
		Visibility:    req.Visibility,
		CoverImageURL: req.CoverImageURL,
		MerchantID:    req.MerchantID,
		UserID:        req.UserID,
	}

	if req.Location != nil {
		serviceReq.Location = &LocationRequest{
			Name:    req.Location.Name,
			Address: req.Location.Address,
			PlaceID: req.Location.PlaceID,
		}
		if req.Location.Coordinates != nil {
			serviceReq.Location.Coordinates = &GeoJSONPointRequest{
				Type:        req.Location.Coordinates.Type,
				Coordinates: req.Location.Coordinates.Coordinates,
			}
		}
	}

	if len(req.Detail) > 0 {
		serviceReq.Detail = make([]DetailBlockRequest, len(req.Detail))
		for i, block := range req.Detail {
			serviceReq.Detail[i] = DetailBlockRequest{
				Type: block.Type,
				Data: block.Data,
			}
		}
	}

	if req.Sessions != nil {
		serviceReq.Sessions = make([]*SessionRequest, len(req.Sessions))
		for i, session := range req.Sessions {
			serviceReq.Sessions[i] = &SessionRequest{
				ID:        session.ID,
				StartTime: session.StartTime,
				EndTime:   session.EndTime,
			}
		}
	}

	if req.FAQ != nil {
		serviceReq.FAQ = make([]*FAQRequest, len(req.FAQ))
		for i, faq := range req.FAQ {
			serviceReq.FAQ[i] = &FAQRequest{
				Question: faq.Question,
				Answer:   faq.Answer,
			}
		}
	}

	return serviceReq
}

// ConvertTestPatchEventRequest converts testutils patch request to service request
func ConvertTestPatchEventRequest(req *testutils.TestPatchEventRequest) *PatchEventRequest {
	serviceReq := &PatchEventRequest{
		ID:      req.ID,
		Title:   req.Title,
		Summary: req.Summary,
		UserID:  req.UserID,
	}

	if req.Sessions != nil {
		serviceReq.Sessions = make([]*SessionRequest, len(req.Sessions))
		for i, session := range req.Sessions {
			serviceReq.Sessions[i] = &SessionRequest{
				ID:        session.ID,
				StartTime: session.StartTime,
				EndTime:   session.EndTime,
			}
		}
	}

	return serviceReq
}

// ConvertTestSearchEventsRequest converts testutils search request to service request
func ConvertTestSearchEventsRequest(req *testutils.TestSearchEventsRequest) *SearchEventsRequest {
	return &SearchEventsRequest{
		MerchantID:  req.MerchantID,
		TitleSearch: req.TitleSearch,
		PageSize:    req.PageSize,
	}
}

// CreateTestEventForMerchant creates a test event with specific merchant ID
func CreateTestEventForMerchant(merchantID string) *models.Event {
	event := testutils.TestEvent()
	merchant, _ := primitive.ObjectIDFromHex(merchantID)
	event.MerchantID = merchant
	return event
}

// Mock matchers to avoid circular imports

// MatchAnyEventFilter matches any EventFilter
func MatchAnyEventFilter() interface{} {
	return mock.MatchedBy(func(f *repository.EventFilter) bool {
		return f != nil
	})
}

// MatchAnyPublicEventFilter matches any PublicEventFilter
func MatchAnyPublicEventFilter() interface{} {
	return mock.MatchedBy(func(f *repository.PublicEventFilter) bool {
		return f != nil
	})
}
