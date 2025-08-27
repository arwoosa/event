package service

import (
	"github.com/arwoosa/event/gen/pb/common"
	"github.com/arwoosa/event/internal/dao/repository"
)

// ResponseBuilder handles building gRPC responses
type ResponseBuilder struct {
	converter *ProtobufConverter
}

// NewResponseBuilder creates a new response builder
func NewResponseBuilder(converter *ProtobufConverter) *ResponseBuilder {
	return &ResponseBuilder{
		converter: converter,
	}
}

// BuildEventListResponse builds a GetEventListResponse from repository result
func (b *ResponseBuilder) BuildEventListResponse(result *repository.EventListResult) *common.EventListResponse {
	// Convert events to protobuf
	eventsPB := make([]*common.Event, len(result.Events))
	for i, event := range result.Events {
		eventsPB[i] = b.converter.ConvertEventToPB(event)
	}

	// Convert pagination to protobuf
	paginationPB := b.converter.ConvertPaginationToPB(result.Pagination)

	return &common.EventListResponse{
		Events:     eventsPB,
		Pagination: paginationPB,
	}
}
