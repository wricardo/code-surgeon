// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: api/codesurgeon.proto

package apiconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	api "github.com/wricardo/code-surgeon/api"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// GptServiceName is the fully-qualified name of the GptService service.
	GptServiceName = "codesurgeon.GptService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// GptServiceSaveToKnowledgeBaseProcedure is the fully-qualified name of the GptService's
	// SaveToKnowledgeBase RPC.
	GptServiceSaveToKnowledgeBaseProcedure = "/codesurgeon.GptService/SaveToKnowledgeBase"
	// GptServiceAnswerQuestionProcedure is the fully-qualified name of the GptService's AnswerQuestion
	// RPC.
	GptServiceAnswerQuestionProcedure = "/codesurgeon.GptService/AnswerQuestion"
	// GptServiceGetOpenAPIProcedure is the fully-qualified name of the GptService's GetOpenAPI RPC.
	GptServiceGetOpenAPIProcedure = "/codesurgeon.GptService/GetOpenAPI"
	// GptServiceIntroductionProcedure is the fully-qualified name of the GptService's Introduction RPC.
	GptServiceIntroductionProcedure = "/codesurgeon.GptService/Introduction"
	// GptServiceParseCodebaseProcedure is the fully-qualified name of the GptService's ParseCodebase
	// RPC.
	GptServiceParseCodebaseProcedure = "/codesurgeon.GptService/ParseCodebase"
	// GptServiceSearchForGolangFunctionProcedure is the fully-qualified name of the GptService's
	// SearchForGolangFunction RPC.
	GptServiceSearchForGolangFunctionProcedure = "/codesurgeon.GptService/SearchForGolangFunction"
	// GptServiceUpsertDocumentationToFunctionProcedure is the fully-qualified name of the GptService's
	// UpsertDocumentationToFunction RPC.
	GptServiceUpsertDocumentationToFunctionProcedure = "/codesurgeon.GptService/UpsertDocumentationToFunction"
	// GptServiceUpsertCodeBlockProcedure is the fully-qualified name of the GptService's
	// UpsertCodeBlock RPC.
	GptServiceUpsertCodeBlockProcedure = "/codesurgeon.GptService/UpsertCodeBlock"
	// GptServiceExecuteBashProcedure is the fully-qualified name of the GptService's ExecuteBash RPC.
	GptServiceExecuteBashProcedure = "/codesurgeon.GptService/ExecuteBash"
	// GptServiceExecuteGoplsImplementationsProcedure is the fully-qualified name of the GptService's
	// ExecuteGoplsImplementations RPC.
	GptServiceExecuteGoplsImplementationsProcedure = "/codesurgeon.GptService/ExecuteGoplsImplementations"
	// GptServiceGitDiffProcedure is the fully-qualified name of the GptService's GitDiff RPC.
	GptServiceGitDiffProcedure = "/codesurgeon.GptService/GitDiff"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	gptServiceServiceDescriptor                             = api.File_api_codesurgeon_proto.Services().ByName("GptService")
	gptServiceSaveToKnowledgeBaseMethodDescriptor           = gptServiceServiceDescriptor.Methods().ByName("SaveToKnowledgeBase")
	gptServiceAnswerQuestionMethodDescriptor                = gptServiceServiceDescriptor.Methods().ByName("AnswerQuestion")
	gptServiceGetOpenAPIMethodDescriptor                    = gptServiceServiceDescriptor.Methods().ByName("GetOpenAPI")
	gptServiceIntroductionMethodDescriptor                  = gptServiceServiceDescriptor.Methods().ByName("Introduction")
	gptServiceParseCodebaseMethodDescriptor                 = gptServiceServiceDescriptor.Methods().ByName("ParseCodebase")
	gptServiceSearchForGolangFunctionMethodDescriptor       = gptServiceServiceDescriptor.Methods().ByName("SearchForGolangFunction")
	gptServiceUpsertDocumentationToFunctionMethodDescriptor = gptServiceServiceDescriptor.Methods().ByName("UpsertDocumentationToFunction")
	gptServiceUpsertCodeBlockMethodDescriptor               = gptServiceServiceDescriptor.Methods().ByName("UpsertCodeBlock")
	gptServiceExecuteBashMethodDescriptor                   = gptServiceServiceDescriptor.Methods().ByName("ExecuteBash")
	gptServiceExecuteGoplsImplementationsMethodDescriptor   = gptServiceServiceDescriptor.Methods().ByName("ExecuteGoplsImplementations")
	gptServiceGitDiffMethodDescriptor                       = gptServiceServiceDescriptor.Methods().ByName("GitDiff")
)

// GptServiceClient is a client for the codesurgeon.GptService service.
type GptServiceClient interface {
	SaveToKnowledgeBase(context.Context, *connect.Request[api.SaveToKnowledgeBaseRequest]) (*connect.Response[api.SaveToKnowledgeBaseResponse], error)
	AnswerQuestion(context.Context, *connect.Request[api.AnswerQuestionRequest]) (*connect.Response[api.AnswerQuestionResponse], error)
	GetOpenAPI(context.Context, *connect.Request[api.GetOpenAPIRequest]) (*connect.Response[api.GetOpenAPIResponse], error)
	Introduction(context.Context, *connect.Request[api.IntroductionRequest]) (*connect.Response[api.IntroductionResponse], error)
	ParseCodebase(context.Context, *connect.Request[api.ParseCodebaseRequest]) (*connect.Response[api.ParseCodebaseResponse], error)
	SearchForGolangFunction(context.Context, *connect.Request[api.SearchForGolangFunctionRequest]) (*connect.Response[api.SearchForGolangFunctionResponse], error)
	UpsertDocumentationToFunction(context.Context, *connect.Request[api.UpsertDocumentationToFunctionRequest]) (*connect.Response[api.UpsertDocumentationToFunctionResponse], error)
	UpsertCodeBlock(context.Context, *connect.Request[api.UpsertCodeBlockRequest]) (*connect.Response[api.UpsertCodeBlockResponse], error)
	// New RPC for executing shell commands
	ExecuteBash(context.Context, *connect.Request[api.ExecuteBashRequest]) (*connect.Response[api.ExecuteBashResponse], error)
	// New RPC for executing gopls implementations command
	ExecuteGoplsImplementations(context.Context, *connect.Request[api.ExecuteGoplsImplementationsRequest]) (*connect.Response[api.ExecuteGoplsImplementationsResponse], error)
	// RPC to list modified files and return their contents
	GitDiff(context.Context, *connect.Request[api.GitDiffRequest]) (*connect.Response[api.GitDiffResponse], error)
}

// NewGptServiceClient constructs a client for the codesurgeon.GptService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewGptServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) GptServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &gptServiceClient{
		saveToKnowledgeBase: connect.NewClient[api.SaveToKnowledgeBaseRequest, api.SaveToKnowledgeBaseResponse](
			httpClient,
			baseURL+GptServiceSaveToKnowledgeBaseProcedure,
			connect.WithSchema(gptServiceSaveToKnowledgeBaseMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		answerQuestion: connect.NewClient[api.AnswerQuestionRequest, api.AnswerQuestionResponse](
			httpClient,
			baseURL+GptServiceAnswerQuestionProcedure,
			connect.WithSchema(gptServiceAnswerQuestionMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		getOpenAPI: connect.NewClient[api.GetOpenAPIRequest, api.GetOpenAPIResponse](
			httpClient,
			baseURL+GptServiceGetOpenAPIProcedure,
			connect.WithSchema(gptServiceGetOpenAPIMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		introduction: connect.NewClient[api.IntroductionRequest, api.IntroductionResponse](
			httpClient,
			baseURL+GptServiceIntroductionProcedure,
			connect.WithSchema(gptServiceIntroductionMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		parseCodebase: connect.NewClient[api.ParseCodebaseRequest, api.ParseCodebaseResponse](
			httpClient,
			baseURL+GptServiceParseCodebaseProcedure,
			connect.WithSchema(gptServiceParseCodebaseMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		searchForGolangFunction: connect.NewClient[api.SearchForGolangFunctionRequest, api.SearchForGolangFunctionResponse](
			httpClient,
			baseURL+GptServiceSearchForGolangFunctionProcedure,
			connect.WithSchema(gptServiceSearchForGolangFunctionMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		upsertDocumentationToFunction: connect.NewClient[api.UpsertDocumentationToFunctionRequest, api.UpsertDocumentationToFunctionResponse](
			httpClient,
			baseURL+GptServiceUpsertDocumentationToFunctionProcedure,
			connect.WithSchema(gptServiceUpsertDocumentationToFunctionMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		upsertCodeBlock: connect.NewClient[api.UpsertCodeBlockRequest, api.UpsertCodeBlockResponse](
			httpClient,
			baseURL+GptServiceUpsertCodeBlockProcedure,
			connect.WithSchema(gptServiceUpsertCodeBlockMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		executeBash: connect.NewClient[api.ExecuteBashRequest, api.ExecuteBashResponse](
			httpClient,
			baseURL+GptServiceExecuteBashProcedure,
			connect.WithSchema(gptServiceExecuteBashMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		executeGoplsImplementations: connect.NewClient[api.ExecuteGoplsImplementationsRequest, api.ExecuteGoplsImplementationsResponse](
			httpClient,
			baseURL+GptServiceExecuteGoplsImplementationsProcedure,
			connect.WithSchema(gptServiceExecuteGoplsImplementationsMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		gitDiff: connect.NewClient[api.GitDiffRequest, api.GitDiffResponse](
			httpClient,
			baseURL+GptServiceGitDiffProcedure,
			connect.WithSchema(gptServiceGitDiffMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// gptServiceClient implements GptServiceClient.
type gptServiceClient struct {
	saveToKnowledgeBase           *connect.Client[api.SaveToKnowledgeBaseRequest, api.SaveToKnowledgeBaseResponse]
	answerQuestion                *connect.Client[api.AnswerQuestionRequest, api.AnswerQuestionResponse]
	getOpenAPI                    *connect.Client[api.GetOpenAPIRequest, api.GetOpenAPIResponse]
	introduction                  *connect.Client[api.IntroductionRequest, api.IntroductionResponse]
	parseCodebase                 *connect.Client[api.ParseCodebaseRequest, api.ParseCodebaseResponse]
	searchForGolangFunction       *connect.Client[api.SearchForGolangFunctionRequest, api.SearchForGolangFunctionResponse]
	upsertDocumentationToFunction *connect.Client[api.UpsertDocumentationToFunctionRequest, api.UpsertDocumentationToFunctionResponse]
	upsertCodeBlock               *connect.Client[api.UpsertCodeBlockRequest, api.UpsertCodeBlockResponse]
	executeBash                   *connect.Client[api.ExecuteBashRequest, api.ExecuteBashResponse]
	executeGoplsImplementations   *connect.Client[api.ExecuteGoplsImplementationsRequest, api.ExecuteGoplsImplementationsResponse]
	gitDiff                       *connect.Client[api.GitDiffRequest, api.GitDiffResponse]
}

// SaveToKnowledgeBase calls codesurgeon.GptService.SaveToKnowledgeBase.
func (c *gptServiceClient) SaveToKnowledgeBase(ctx context.Context, req *connect.Request[api.SaveToKnowledgeBaseRequest]) (*connect.Response[api.SaveToKnowledgeBaseResponse], error) {
	return c.saveToKnowledgeBase.CallUnary(ctx, req)
}

// AnswerQuestion calls codesurgeon.GptService.AnswerQuestion.
func (c *gptServiceClient) AnswerQuestion(ctx context.Context, req *connect.Request[api.AnswerQuestionRequest]) (*connect.Response[api.AnswerQuestionResponse], error) {
	return c.answerQuestion.CallUnary(ctx, req)
}

// GetOpenAPI calls codesurgeon.GptService.GetOpenAPI.
func (c *gptServiceClient) GetOpenAPI(ctx context.Context, req *connect.Request[api.GetOpenAPIRequest]) (*connect.Response[api.GetOpenAPIResponse], error) {
	return c.getOpenAPI.CallUnary(ctx, req)
}

// Introduction calls codesurgeon.GptService.Introduction.
func (c *gptServiceClient) Introduction(ctx context.Context, req *connect.Request[api.IntroductionRequest]) (*connect.Response[api.IntroductionResponse], error) {
	return c.introduction.CallUnary(ctx, req)
}

// ParseCodebase calls codesurgeon.GptService.ParseCodebase.
func (c *gptServiceClient) ParseCodebase(ctx context.Context, req *connect.Request[api.ParseCodebaseRequest]) (*connect.Response[api.ParseCodebaseResponse], error) {
	return c.parseCodebase.CallUnary(ctx, req)
}

// SearchForGolangFunction calls codesurgeon.GptService.SearchForGolangFunction.
func (c *gptServiceClient) SearchForGolangFunction(ctx context.Context, req *connect.Request[api.SearchForGolangFunctionRequest]) (*connect.Response[api.SearchForGolangFunctionResponse], error) {
	return c.searchForGolangFunction.CallUnary(ctx, req)
}

// UpsertDocumentationToFunction calls codesurgeon.GptService.UpsertDocumentationToFunction.
func (c *gptServiceClient) UpsertDocumentationToFunction(ctx context.Context, req *connect.Request[api.UpsertDocumentationToFunctionRequest]) (*connect.Response[api.UpsertDocumentationToFunctionResponse], error) {
	return c.upsertDocumentationToFunction.CallUnary(ctx, req)
}

// UpsertCodeBlock calls codesurgeon.GptService.UpsertCodeBlock.
func (c *gptServiceClient) UpsertCodeBlock(ctx context.Context, req *connect.Request[api.UpsertCodeBlockRequest]) (*connect.Response[api.UpsertCodeBlockResponse], error) {
	return c.upsertCodeBlock.CallUnary(ctx, req)
}

// ExecuteBash calls codesurgeon.GptService.ExecuteBash.
func (c *gptServiceClient) ExecuteBash(ctx context.Context, req *connect.Request[api.ExecuteBashRequest]) (*connect.Response[api.ExecuteBashResponse], error) {
	return c.executeBash.CallUnary(ctx, req)
}

// ExecuteGoplsImplementations calls codesurgeon.GptService.ExecuteGoplsImplementations.
func (c *gptServiceClient) ExecuteGoplsImplementations(ctx context.Context, req *connect.Request[api.ExecuteGoplsImplementationsRequest]) (*connect.Response[api.ExecuteGoplsImplementationsResponse], error) {
	return c.executeGoplsImplementations.CallUnary(ctx, req)
}

// GitDiff calls codesurgeon.GptService.GitDiff.
func (c *gptServiceClient) GitDiff(ctx context.Context, req *connect.Request[api.GitDiffRequest]) (*connect.Response[api.GitDiffResponse], error) {
	return c.gitDiff.CallUnary(ctx, req)
}

// GptServiceHandler is an implementation of the codesurgeon.GptService service.
type GptServiceHandler interface {
	SaveToKnowledgeBase(context.Context, *connect.Request[api.SaveToKnowledgeBaseRequest]) (*connect.Response[api.SaveToKnowledgeBaseResponse], error)
	AnswerQuestion(context.Context, *connect.Request[api.AnswerQuestionRequest]) (*connect.Response[api.AnswerQuestionResponse], error)
	GetOpenAPI(context.Context, *connect.Request[api.GetOpenAPIRequest]) (*connect.Response[api.GetOpenAPIResponse], error)
	Introduction(context.Context, *connect.Request[api.IntroductionRequest]) (*connect.Response[api.IntroductionResponse], error)
	ParseCodebase(context.Context, *connect.Request[api.ParseCodebaseRequest]) (*connect.Response[api.ParseCodebaseResponse], error)
	SearchForGolangFunction(context.Context, *connect.Request[api.SearchForGolangFunctionRequest]) (*connect.Response[api.SearchForGolangFunctionResponse], error)
	UpsertDocumentationToFunction(context.Context, *connect.Request[api.UpsertDocumentationToFunctionRequest]) (*connect.Response[api.UpsertDocumentationToFunctionResponse], error)
	UpsertCodeBlock(context.Context, *connect.Request[api.UpsertCodeBlockRequest]) (*connect.Response[api.UpsertCodeBlockResponse], error)
	// New RPC for executing shell commands
	ExecuteBash(context.Context, *connect.Request[api.ExecuteBashRequest]) (*connect.Response[api.ExecuteBashResponse], error)
	// New RPC for executing gopls implementations command
	ExecuteGoplsImplementations(context.Context, *connect.Request[api.ExecuteGoplsImplementationsRequest]) (*connect.Response[api.ExecuteGoplsImplementationsResponse], error)
	// RPC to list modified files and return their contents
	GitDiff(context.Context, *connect.Request[api.GitDiffRequest]) (*connect.Response[api.GitDiffResponse], error)
}

// NewGptServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewGptServiceHandler(svc GptServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	gptServiceSaveToKnowledgeBaseHandler := connect.NewUnaryHandler(
		GptServiceSaveToKnowledgeBaseProcedure,
		svc.SaveToKnowledgeBase,
		connect.WithSchema(gptServiceSaveToKnowledgeBaseMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceAnswerQuestionHandler := connect.NewUnaryHandler(
		GptServiceAnswerQuestionProcedure,
		svc.AnswerQuestion,
		connect.WithSchema(gptServiceAnswerQuestionMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceGetOpenAPIHandler := connect.NewUnaryHandler(
		GptServiceGetOpenAPIProcedure,
		svc.GetOpenAPI,
		connect.WithSchema(gptServiceGetOpenAPIMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceIntroductionHandler := connect.NewUnaryHandler(
		GptServiceIntroductionProcedure,
		svc.Introduction,
		connect.WithSchema(gptServiceIntroductionMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceParseCodebaseHandler := connect.NewUnaryHandler(
		GptServiceParseCodebaseProcedure,
		svc.ParseCodebase,
		connect.WithSchema(gptServiceParseCodebaseMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceSearchForGolangFunctionHandler := connect.NewUnaryHandler(
		GptServiceSearchForGolangFunctionProcedure,
		svc.SearchForGolangFunction,
		connect.WithSchema(gptServiceSearchForGolangFunctionMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceUpsertDocumentationToFunctionHandler := connect.NewUnaryHandler(
		GptServiceUpsertDocumentationToFunctionProcedure,
		svc.UpsertDocumentationToFunction,
		connect.WithSchema(gptServiceUpsertDocumentationToFunctionMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceUpsertCodeBlockHandler := connect.NewUnaryHandler(
		GptServiceUpsertCodeBlockProcedure,
		svc.UpsertCodeBlock,
		connect.WithSchema(gptServiceUpsertCodeBlockMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceExecuteBashHandler := connect.NewUnaryHandler(
		GptServiceExecuteBashProcedure,
		svc.ExecuteBash,
		connect.WithSchema(gptServiceExecuteBashMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceExecuteGoplsImplementationsHandler := connect.NewUnaryHandler(
		GptServiceExecuteGoplsImplementationsProcedure,
		svc.ExecuteGoplsImplementations,
		connect.WithSchema(gptServiceExecuteGoplsImplementationsMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	gptServiceGitDiffHandler := connect.NewUnaryHandler(
		GptServiceGitDiffProcedure,
		svc.GitDiff,
		connect.WithSchema(gptServiceGitDiffMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/codesurgeon.GptService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case GptServiceSaveToKnowledgeBaseProcedure:
			gptServiceSaveToKnowledgeBaseHandler.ServeHTTP(w, r)
		case GptServiceAnswerQuestionProcedure:
			gptServiceAnswerQuestionHandler.ServeHTTP(w, r)
		case GptServiceGetOpenAPIProcedure:
			gptServiceGetOpenAPIHandler.ServeHTTP(w, r)
		case GptServiceIntroductionProcedure:
			gptServiceIntroductionHandler.ServeHTTP(w, r)
		case GptServiceParseCodebaseProcedure:
			gptServiceParseCodebaseHandler.ServeHTTP(w, r)
		case GptServiceSearchForGolangFunctionProcedure:
			gptServiceSearchForGolangFunctionHandler.ServeHTTP(w, r)
		case GptServiceUpsertDocumentationToFunctionProcedure:
			gptServiceUpsertDocumentationToFunctionHandler.ServeHTTP(w, r)
		case GptServiceUpsertCodeBlockProcedure:
			gptServiceUpsertCodeBlockHandler.ServeHTTP(w, r)
		case GptServiceExecuteBashProcedure:
			gptServiceExecuteBashHandler.ServeHTTP(w, r)
		case GptServiceExecuteGoplsImplementationsProcedure:
			gptServiceExecuteGoplsImplementationsHandler.ServeHTTP(w, r)
		case GptServiceGitDiffProcedure:
			gptServiceGitDiffHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedGptServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedGptServiceHandler struct{}

func (UnimplementedGptServiceHandler) SaveToKnowledgeBase(context.Context, *connect.Request[api.SaveToKnowledgeBaseRequest]) (*connect.Response[api.SaveToKnowledgeBaseResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.SaveToKnowledgeBase is not implemented"))
}

func (UnimplementedGptServiceHandler) AnswerQuestion(context.Context, *connect.Request[api.AnswerQuestionRequest]) (*connect.Response[api.AnswerQuestionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.AnswerQuestion is not implemented"))
}

func (UnimplementedGptServiceHandler) GetOpenAPI(context.Context, *connect.Request[api.GetOpenAPIRequest]) (*connect.Response[api.GetOpenAPIResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.GetOpenAPI is not implemented"))
}

func (UnimplementedGptServiceHandler) Introduction(context.Context, *connect.Request[api.IntroductionRequest]) (*connect.Response[api.IntroductionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.Introduction is not implemented"))
}

func (UnimplementedGptServiceHandler) ParseCodebase(context.Context, *connect.Request[api.ParseCodebaseRequest]) (*connect.Response[api.ParseCodebaseResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.ParseCodebase is not implemented"))
}

func (UnimplementedGptServiceHandler) SearchForGolangFunction(context.Context, *connect.Request[api.SearchForGolangFunctionRequest]) (*connect.Response[api.SearchForGolangFunctionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.SearchForGolangFunction is not implemented"))
}

func (UnimplementedGptServiceHandler) UpsertDocumentationToFunction(context.Context, *connect.Request[api.UpsertDocumentationToFunctionRequest]) (*connect.Response[api.UpsertDocumentationToFunctionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.UpsertDocumentationToFunction is not implemented"))
}

func (UnimplementedGptServiceHandler) UpsertCodeBlock(context.Context, *connect.Request[api.UpsertCodeBlockRequest]) (*connect.Response[api.UpsertCodeBlockResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.UpsertCodeBlock is not implemented"))
}

func (UnimplementedGptServiceHandler) ExecuteBash(context.Context, *connect.Request[api.ExecuteBashRequest]) (*connect.Response[api.ExecuteBashResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.ExecuteBash is not implemented"))
}

func (UnimplementedGptServiceHandler) ExecuteGoplsImplementations(context.Context, *connect.Request[api.ExecuteGoplsImplementationsRequest]) (*connect.Response[api.ExecuteGoplsImplementationsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.ExecuteGoplsImplementations is not implemented"))
}

func (UnimplementedGptServiceHandler) GitDiff(context.Context, *connect.Request[api.GitDiffRequest]) (*connect.Response[api.GitDiffResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("codesurgeon.GptService.GitDiff is not implemented"))
}