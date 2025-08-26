package helper

import (
	"math"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
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

// SafeInt32FromInt64 safely converts int64 to int32, clamping to int32 bounds if needed.
// This prevents integer overflow in protobuf conversions where int32 is required.
func SafeInt32FromInt64(val int64) int32 {
	if val > math.MaxInt32 {
		return math.MaxInt32
	}
	if val < math.MinInt32 {
		return math.MinInt32
	}
	return int32(val)
}

// SafeInt32FromInt safely converts int to int32, clamping to int32 bounds if needed.
// This prevents integer overflow in protobuf conversions where int32 is required.
func SafeInt32FromInt(val int) int32 {
	if val > math.MaxInt32 {
		return math.MaxInt32
	}
	if val < math.MinInt32 {
		return math.MinInt32
	}
	return int32(val)
}
