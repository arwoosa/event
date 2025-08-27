package service

import (
	"time"

	"github.com/arwoosa/event/conf"
	"github.com/arwoosa/event/gen/pb/console"
	"github.com/arwoosa/event/internal/dao/repository"
)

// RequestParameterProcessor handles processing of gRPC request parameters
type RequestParameterProcessor struct{}

// NewRequestParameterProcessor creates a new request parameter processor
func NewRequestParameterProcessor() *RequestParameterProcessor {
	return &RequestParameterProcessor{}
}

// BuildBaseFilter creates a base EventFilter with default values
func (p *RequestParameterProcessor) BuildBaseFilter() *repository.EventFilter {
	return &repository.EventFilter{
		Limit:  20, // Default
		Offset: 0,
	}
}

// ProcessStringFilters processes string-based filter parameters
func (p *RequestParameterProcessor) ProcessStringFilters(req *console.GetEventListRequest, filter *repository.EventFilter) {
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
}

// ProcessTimeFilters processes time-based filter parameters
func (p *RequestParameterProcessor) ProcessTimeFilters(req *console.GetEventListRequest, filter *repository.EventFilter) {
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
}

// PaginationConfig holds pagination configuration values
type PaginationConfigValues struct {
	DefaultPageSize int
	MaxPageSize     int
}

// GetPaginationConfig extracts pagination configuration with fallback defaults
func (p *RequestParameterProcessor) GetPaginationConfig(config *conf.PaginationConfig) PaginationConfigValues {
	defaults := PaginationConfigValues{
		DefaultPageSize: 20,  // Default fallback
		MaxPageSize:     100, // Default fallback
	}

	if config != nil {
		if config.DefaultPageSize > 0 {
			defaults.DefaultPageSize = config.DefaultPageSize
		}
		if config.MaxPageSize > 0 {
			defaults.MaxPageSize = config.MaxPageSize
		}
	}

	return defaults
}

// ProcessPagination processes pagination parameters
func (p *RequestParameterProcessor) ProcessPagination(req *console.GetEventListRequest, filter *repository.EventFilter, config PaginationConfigValues) {
	filter.Limit = config.DefaultPageSize

	// Handle page size
	if req.PageSize != nil {
		pageSize := int(*req.PageSize)
		if pageSize > 0 && pageSize <= config.MaxPageSize {
			filter.Limit = pageSize
		}
	}

	// Handle page number (overrides cursor pagination)
	if req.Page != nil && *req.Page > 0 {
		// Safe calculation: use int64 to avoid overflow, then convert to int
		offset64 := int64(*req.Page-1) * int64(filter.Limit)
		filter.Offset = int(offset64) // Note: assumes Offset won't exceed int range
		filter.PageToken = nil        // Don't use cursor pagination if page is specified
	}
}

// ProcessAllFilters processes all request parameters into a filter
func (p *RequestParameterProcessor) ProcessAllFilters(req *console.GetEventListRequest, paginationConfig *conf.PaginationConfig) *repository.EventFilter {
	filter := p.BuildBaseFilter()

	p.ProcessStringFilters(req, filter)
	p.ProcessTimeFilters(req, filter)

	configValues := p.GetPaginationConfig(paginationConfig)
	p.ProcessPagination(req, filter, configValues)

	return filter
}
