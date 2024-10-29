package structs

import (
	"context"
	"errors"

	"connectrpc.com/connect"
)

type (
	SecondStruct struct {
		String string
	}

	ThirdStruct struct {
		String string
	}
)

type privateStruct struct {
	String string
}

func (s *privateStruct) MyPrivateStructMethod(ctx context.Context, x string) (string, error) {
	return "", nil
}

func (s *FirstStruct) MyOtherTestMethod(ctx context.Context, x string) (string, error) {
	return "", nil
}

func (s *FirstStruct) TakesThirdStruct(ctx context.Context, x *ThirdStruct) (*string, error) {
	return nil, nil
}

// UnimplementedZivoAPIHandler returns CodeUnimplemented from all methods.
type UnimplementedZivoAPIHandler struct{}

func (UnimplementedZivoAPIHandler) SendSms(ctx context.Context, r *connect.Request[FirstStruct]) (*connect.Response[FirstStruct], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("trinity.zivo.ZivoAPI.SendSms is not implemented"))
}
