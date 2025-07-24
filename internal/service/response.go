package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"grpc_gateway_framework/api"
)

func ResponseError(c codes.Code, err error) (*api.Response, error) {
	return nil, status.Error(c, err.Error())
}

func ResponseSuccess(d any) (*api.Response, error) {
	var anyData *anypb.Any
	if d != nil {
		msg, ok := d.(proto.Message)
		if !ok {
			return nil, status.Error(codes.Internal, "output data is wrong")
		}

		var err error
		anyData, err = anypb.New(msg)
		if err != nil {
			return nil, status.Error(codes.Internal, "output data is wrong")
		}
	}

	return &api.Response{
		Status: "success",
		Code:   int32(CodeSuccess),
		//Message: CodeSuccess.Msg(),
		Data: anyData,
	}, nil
}
