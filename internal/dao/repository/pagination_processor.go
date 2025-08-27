package repository

import (
	"github.com/arwoosa/event/internal/helper"
	"github.com/arwoosa/event/internal/models"
)

// PaginationProcessor handles pagination logic for queries
type PaginationProcessor struct {
	repo *MongoEventRepository
}

// NewPaginationProcessor creates a new pagination processor
func NewPaginationProcessor(repo *MongoEventRepository) *PaginationProcessor {
	return &PaginationProcessor{
		repo: repo,
	}
}

// PaginationInput contains input parameters for pagination processing
type PaginationInput struct {
	Events     []*models.Event
	Limit      int
	Offset     int
	PageToken  *string
	TotalCount *int64
}

// ProcessPagination processes pagination for query results
func (p *PaginationProcessor) ProcessPagination(input *PaginationInput) *Pagination {
	pagination := p.buildBasePagination(input)

	if p.isCursorPagination(input.PageToken) {
		p.processCursorPagination(input, pagination)
	} else {
		p.processOffsetPagination(input, pagination)
	}

	p.addNextPageToken(input.Events, pagination)

	return pagination
}

// buildBasePagination creates base pagination structure
func (p *PaginationProcessor) buildBasePagination(input *PaginationInput) *Pagination {
	return &Pagination{
		HasNext: false,
		HasPrev: input.Offset > 0, // For offset pagination
	}
}

// isCursorPagination checks if this is cursor-based pagination
func (p *PaginationProcessor) isCursorPagination(pageToken *string) bool {
	return pageToken != nil && *pageToken != ""
}

// processCursorPagination handles cursor-based pagination logic
func (p *PaginationProcessor) processCursorPagination(input *PaginationInput, pagination *Pagination) {
	// For cursor-based pagination, check if we got limit+1 results
	pagination.HasNext = len(input.Events) > input.Limit
	if pagination.HasNext {
		// Remove the extra result used for pagination check
		input.Events = input.Events[:input.Limit]
	}

	// For cursor pagination, any valid cursor means we're not on first page
	pagination.HasPrev = true
}

// processOffsetPagination handles offset-based pagination logic
func (p *PaginationProcessor) processOffsetPagination(input *PaginationInput, pagination *Pagination) {
	// For offset-based pagination without total count, assume there might be more
	// This is a fallback when we can't calculate exact pagination info
	pagination.HasNext = len(input.Events) == input.Limit

	// If we have total count, calculate precise pagination info (overrides the assumption)
	if input.TotalCount != nil {
		p.addPageInfo(input, pagination)
	}
}

// addPageInfo adds page-based information to pagination
func (p *PaginationProcessor) addPageInfo(input *PaginationInput, pagination *Pagination) {
	pagination.TotalCount = input.TotalCount

	// Calculate current page and total pages using int64 then safe conversion
	currentPage64 := int64(input.Offset/input.Limit) + 1
	currentPage := helper.SafeInt32FromInt64(currentPage64)
	pagination.CurrentPage = &currentPage

	// Ceiling division: (totalCount + limit - 1) / limit
	totalPages64 := (*input.TotalCount + int64(input.Limit) - 1) / int64(input.Limit)
	totalPages := helper.SafeInt32FromInt64(totalPages64)
	pagination.TotalPages = &totalPages

	// With total count, we can accurately determine HasNext and HasPrev
	pagination.HasNext = currentPage64 < totalPages64
	pagination.HasPrev = currentPage64 > 1
}

// addNextPageToken adds next page token if needed
func (p *PaginationProcessor) addNextPageToken(events []*models.Event, pagination *Pagination) {
	if len(events) > 0 && pagination.HasNext {
		lastEvent := events[len(events)-1]
		nextToken := p.repo.encodeCursor(&Cursor{
			LastID:    lastEvent.ID.Hex(),
			Timestamp: lastEvent.CreatedAt,
		})
		pagination.NextPageToken = &nextToken
	}
}
