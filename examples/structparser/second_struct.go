package structs

import "context"

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
