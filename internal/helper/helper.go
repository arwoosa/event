package helper

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

func ConvertTimeToProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	var protoTimestamp *timestamppb.Timestamp
	if !t.IsZero() {
		protoTimestamp = timestamppb.New(t)
	} else {
		protoTimestamp = nil
	}
	return protoTimestamp
}
