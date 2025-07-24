package service

import (
	"errors"
)

var (
	ErrInvalidParams = errors.New("invalid params")
	ErrorInternal    = errors.New("something went wrong. Please try again later")
)
