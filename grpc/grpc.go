package grpc

import (
	"context"
	"log"
	"strings"

	"connectrpc.com/connect"
	"github.com/Jeffail/gabs"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
)

/*api/codesurgeon.openapi.json
{
	"openapi": "3.0.0",
	"info": {
		"title": "codesurgeon",
		"version": "1.0.0"
	},
	"paths": {
		"/codesurgeon.GptService/GetOpenAPI": {
			"post": {
				"summary": "GetOpenAPI",
				"operationId": "codesurgeon.GptService.GetOpenAPI",
				"requestBody": {
					"content": {
						"application/json": {
							"schema": {
								"$ref": "#/components/schemas/codesurgeon.GetOpenAPIRequest"
							}
						}
					}
				},
				"responses": {
					"200": {
						"description": "A successful response",
						"content": {
							"application/json": {
								"schema": {
									"$ref": "#/components/schemas/codesurgeon.GetOpenAPIResponse"
								}
							}
						}
					}
				}
			}
		},
		"/codesurgeon.GptService/Introduction": {
			"post": {
				"summary": "Introduction",
				"operationId": "codesurgeon.GptService.Introduction",
				"requestBody": {
					"content": {
						"application/json": {
							"schema": {
								"$ref": "#/components/schemas/codesurgeon.IntroductionRequest"
							}
						}
					}
				},
				"responses": {
					"200": {
						"description": "A successful response",
						"content": {
							"application/json": {
								"schema": {
									"$ref": "#/components/schemas/codesurgeon.IntroductionResponse"
								}
							}
						}
					}
				}
			}
		},
		"/codesurgeon.GptService/ParseCodebase": {
			"post": {
				"summary": "ParseCodebase",
				"operationId": "codesurgeon.GptService.ParseCodebase",
				"requestBody": {
					"content": {
						"application/json": {
							"schema": {
								"$ref": "#/components/schemas/codesurgeon.ParseCodebaseRequest"
							}
						}
					}
				},
				"responses": {
					"200": {
						"description": "A successful response",
						"content": {
							"application/json": {
								"schema": {
									"$ref": "#/components/schemas/codesurgeon.ParseCodebaseResponse"
								}
							}
						}
					}
				}
			}
		},
		"/codesurgeon.GptService/SearchForFunction": {
			"post": {
				"summary": "SearchForFunction",
				"operationId": "codesurgeon.GptService.SearchForFunction",
				"requestBody": {
					"content": {
						"application/json": {
							"schema": {
								"$ref": "#/components/schemas/codesurgeon.SearchForFunctionRequest"
							}
						}
					}
				},
				"responses": {
					"200": {
						"description": "A successful response",
						"content": {
							"application/json": {
								"schema": {
									"$ref": "#/components/schemas/codesurgeon.SearchForFunctionResponse"
								}
							}
						}
					}
				}
			}
		},
		"/codesurgeon.GptService/UpsertDocumentationToFunction": {
			"post": {
				"summary": "UpsertDocumentationToFunction",
				"operationId": "codesurgeon.GptService.UpsertDocumentationToFunction",
				"requestBody": {
					"content": {
						"application/json": {
							"schema": {
								"$ref": "#/components/schemas/codesurgeon.UpsertDocumentationToFunctionRequest"
							}
						}
					}
				},
				"responses": {
					"200": {
						"description": "A successful response",
						"content": {
							"application/json": {
								"schema": {
									"$ref": "#/components/schemas/codesurgeon.UpsertDocumentationToFunctionResponse"
								}
							}
						}
					}
				}
			}
		},
		"/codesurgeon.GptService/UpsertCodeBlock": {
			"post": {
				"summary": "UpsertCodeBlock",
				"operationId": "codesurgeon.GptService.UpsertCodeBlock",
				"requestBody": {
					"content": {
						"application/json": {
							"schema": {
								"$ref": "#/components/schemas/codesurgeon.UpsertCodeBlockRequest"
							}
						}
					}
				},
				"responses": {
					"200": {
						"description": "A successful response",
						"content": {
							"application/json": {
								"schema": {
									"$ref": "#/components/schemas/codesurgeon.UpsertCodeBlockResponse"
								}
							}
						}
					}
				}
			}
		}
	},
	"components": {
		"schemas": {
			"codesurgeon.GetOpenAPIRequest": {
				"type": "object",
				"properties": {
				}
			},
			"codesurgeon.GetOpenAPIResponse": {
				"type": "object",
				"properties": {
					"openapi": {
						"type": "string"
					}
				}
			},
			"codesurgeon.IntroductionRequest": {
				"type": "object",
				"properties": {
					"short": {
						"type": "boolean"
					}
				}
			},
			"codesurgeon.IntroductionResponse": {
				"type": "object",
				"properties": {
					"introduction": {
						"type": "string"
					}
				}
			},
			"codesurgeon.SearchForFunctionRequest": {
				"type": "object",
				"properties": {
					"path": {
						"type": "string"
					},
					"function_name": {
						"type": "string"
					},
					"receiver": {
						"type": "string"
					}
				}
			},
			"codesurgeon.SearchForFunctionResponse": {
				"type": "object",
				"properties": {
					"filepath": {
						"type": "string"
					},
					"function_name": {
						"type": "string"
					},
					"documentation": {
						"type": "string"
					},
					"receiver": {
						"type": "string"
					},
					"body": {
						"type": "string"
					}
				}
			},
			"codesurgeon.UpsertDocumentationToFunctionRequest": {
				"type": "object",
				"properties": {
					"filepath": {
						"type": "string"
					},
					"function_name": {
						"type": "string"
					},
					"documentation": {
						"type": "string"
					},
					"receiver": {
						"type": "string"
					}
				}
			},
			"codesurgeon.UpsertDocumentationToFunctionResponse": {
				"type": "object",
				"properties": {
					"ok": {
						"type": "boolean"
					}
				}
			},
			"codesurgeon.UpsertCodeBlockRequest": {
				"type": "object",
				"properties": {
					"modification": {
						"$ref": "#/components/schemas/codesurgeon.UpsertCodeBlockRequest.Modification"
					}
				}
			},
			"codesurgeon.UpsertCodeBlockRequest.Modification": {
				"type": "object",
				"properties": {
					"filepath": {
						"type": "string"
					},
					"package_name": {
						"type": "string"
					},
					"code_block": {
						"type": "string"
					},
					"overwrite": {
						"type": "boolean"
					}
				}
			},
			"codesurgeon.UpsertCodeBlockResponse": {
				"type": "object",
				"properties": {
					"ok": {
						"type": "boolean"
					}
				}
			},
			"codesurgeon.ParseCodebaseRequest": {
				"type": "object",
				"properties": {
					"file_or_directory": {
						"type": "string"
					}
				}
			},
			"codesurgeon.ParseCodebaseResponse": {
				"type": "object",
				"properties": {
					"packages": {
						"$ref": "#/components/schemas/codesurgeon.Package"
					}
				}
			},
			"codesurgeon.Package": {
				"type": "object",
				"properties": {
					"package": {
						"type": "string"
					},
					"imports": {
						"type": "string"
					},
					"structs": {
						"$ref": "#/components/schemas/codesurgeon.Struct"
					},
					"functions": {
						"$ref": "#/components/schemas/codesurgeon.Function"
					},
					"variables": {
						"$ref": "#/components/schemas/codesurgeon.Variable"
					},
					"constants": {
						"$ref": "#/components/schemas/codesurgeon.Constant"
					},
					"interfaces": {
						"$ref": "#/components/schemas/codesurgeon.Interface"
					}
				}
			},
			"codesurgeon.Interface": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"methods": {
						"$ref": "#/components/schemas/codesurgeon.Method"
					},
					"docs": {
						"type": "string"
					}
				}
			},
			"codesurgeon.Struct": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"fields": {
						"$ref": "#/components/schemas/codesurgeon.Field"
					},
					"methods": {
						"$ref": "#/components/schemas/codesurgeon.Method"
					},
					"docs": {
						"type": "string"
					}
				}
			},
			"codesurgeon.Method": {
				"type": "object",
				"properties": {
					"receiver": {
						"type": "string"
					},
					"name": {
						"type": "string"
					},
					"params": {
						"$ref": "#/components/schemas/codesurgeon.Param"
					},
					"returns": {
						"$ref": "#/components/schemas/codesurgeon.Param"
					},
					"docs": {
						"type": "string"
					},
					"signature": {
						"type": "string"
					},
					"body": {
						"type": "string"
					}
				}
			},
			"codesurgeon.Function": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"params": {
						"$ref": "#/components/schemas/codesurgeon.Param"
					},
					"returns": {
						"$ref": "#/components/schemas/codesurgeon.Param"
					},
					"docs": {
						"type": "string"
					},
					"signature": {
						"type": "string"
					},
					"body": {
						"type": "string"
					}
				}
			},
			"codesurgeon.Param": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"type": {
						"type": "string"
					}
				}
			},
			"codesurgeon.Field": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"type": {
						"type": "string"
					},
					"tag": {
						"type": "string"
					},
					"private": {
						"type": "boolean"
					},
					"pointer": {
						"type": "boolean"
					},
					"slice": {
						"type": "boolean"
					},
					"docs": {
						"type": "string"
					},
					"comment": {
						"type": "string"
					}
				}
			},
			"codesurgeon.Variable": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"type": {
						"type": "string"
					},
					"docs": {
						"type": "string"
					}
				}
			},
			"codesurgeon.Constant": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"value": {
						"type": "string"
					},
					"docs": {
						"type": "string"
					}
				}
			}
		}
	}
}
*/

var _ apiconnect.GptServiceHandler = (*Handler)(nil)

type Handler struct {
	url string
}

func NewHandler(url string) *Handler {
	return &Handler{
		url: url,
	}
}

func (*Handler) SearchForFunction(ctx context.Context, req *connect.Request[api.SearchForFunctionRequest]) (*connect.Response[api.SearchForFunctionResponse], error) {
	path := req.Msg.Path
	if path == "" {
		path = "."
	}

	path, err := codesurgeon.FindFunction(path, req.Msg.Receiver, req.Msg.FunctionName)
	if err != nil {
		log.Printf("Error searching for function: %v", err)
		return &connect.Response[api.SearchForFunctionResponse]{
			Msg: &api.SearchForFunctionResponse{},
		}, nil
	}
	if path == "" {
		log.Printf("Function not found")
		return &connect.Response[api.SearchForFunctionResponse]{
			Msg: &api.SearchForFunctionResponse{},
		}, nil

	}
	return &connect.Response[api.SearchForFunctionResponse]{
		Msg: &api.SearchForFunctionResponse{
			Filepath: path,
		},
	}, nil
}

func (*Handler) UpsertDocumentationToFunction(ctx context.Context, req *connect.Request[api.UpsertDocumentationToFunctionRequest]) (*connect.Response[api.UpsertDocumentationToFunctionResponse], error) {
	msg := req.Msg
	ok, err := codesurgeon.UpsertDocumentationToFunction(msg.Filepath, msg.Receiver, msg.FunctionName, msg.Documentation)
	if err != nil {
		return nil, err
	}

	return &connect.Response[api.UpsertDocumentationToFunctionResponse]{
		Msg: &api.UpsertDocumentationToFunctionResponse{
			Ok: ok,
		},
	}, nil
}

func (*Handler) UpsertCodeBlock(ctx context.Context, req *connect.Request[api.UpsertCodeBlockRequest]) (*connect.Response[api.UpsertCodeBlockResponse], error) {
	log.Printf("UpsertCodeBlock request: %v\n", req.Msg)
	msg := req.Msg
	changes := []codesurgeon.FileChange{}

	block := msg.Modification
	// for _, block := range msg.Modification {
	change := codesurgeon.FileChange{
		PackageName: block.PackageName,
		File:        block.Filepath,
		Fragments: []codesurgeon.CodeFragment{
			{
				Content:   block.CodeBlock,
				Overwrite: block.Overwrite,
			},
		},
	}
	changes = append(changes, change)
	// }
	err := codesurgeon.ApplyFileChanges(changes)
	if err != nil {
		log.Printf("Error applying file changes: %v\n", err)
		return &connect.Response[api.UpsertCodeBlockResponse]{
			Msg: &api.UpsertCodeBlockResponse{
				Ok: false,
			},
		}, nil
	}
	log.Printf("Code block upserted successfully")

	return &connect.Response[api.UpsertCodeBlockResponse]{
		Msg: &api.UpsertCodeBlockResponse{
			Ok: true,
		},
	}, nil
}

// LoggerInterceptor is a Connect RPC middleware that logs all incoming requests.
func LoggerInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
			} else {
				log.Printf("Incoming request: %s", req.Spec().Procedure)
				log.Printf("Request headers: %v", req.Header())
				log.Printf("Request message: %v", req.Any())
			}
			res, err := next(ctx, req)
			if err != nil {
				log.Printf("Error: %v", err)
			} else {
				log.Printf("Response: %v", res)
			}
			return res, err
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)

}

// ParseCodebase handles the ParseCodebase gRPC method
func (*Handler) ParseCodebase(ctx context.Context, req *connect.Request[api.ParseCodebaseRequest]) (*connect.Response[api.ParseCodebaseResponse], error) {
	// Extract the file or directory path from the request
	fileOrDirectory := req.Msg.FileOrDirectory
	if fileOrDirectory == "" {
		fileOrDirectory = "." // Default to current directory if not provided
	}

	// Call the ParseDirectory function to parse the codebase
	parsedInfo, err := codesurgeon.ParseDirectory(fileOrDirectory)
	if err != nil {
		log.Printf("Error parsing directory: %v", err)
		return &connect.Response[api.ParseCodebaseResponse]{
			Msg: &api.ParseCodebaseResponse{},
		}, err
	}

	// Convert the parsed information to the API response format
	response := &api.ParseCodebaseResponse{
		Packages: convertParsedInfoToProto(parsedInfo),
	}

	// Return the response
	return &connect.Response[api.ParseCodebaseResponse]{Msg: response}, nil
}

// Helper function to convert parsed info to proto format
func convertParsedInfoToProto(parsedInfo *codesurgeon.ParsedInfo) []*api.Package {
	var packages []*api.Package
	for _, pkg := range parsedInfo.Packages {
		packages = append(packages, &api.Package{
			Package:    pkg.Package,
			Imports:    pkg.Imports,
			Structs:    convertStructsToProto(pkg.Structs),
			Functions:  convertFunctionsToProto(pkg.Functions),
			Variables:  convertVariablesToProto(pkg.Variables),
			Constants:  convertConstantsToProto(pkg.Constants),
			Interfaces: convertInterfacesToProto(pkg.Interfaces),
		})
	}
	return packages
}

// Helper function to convert structs from codesurgeon to proto
func convertStructsToProto(structs []codesurgeon.Struct) []*api.Struct {
	var protoStructs []*api.Struct
	for _, s := range structs {
		protoStruct := &api.Struct{
			Name:    s.Name,
			Fields:  convertFieldsToProto(s.Fields),
			Methods: convertMethodsToProto(s.Methods),
			Docs:    s.Docs,
		}
		protoStructs = append(protoStructs, protoStruct)
	}
	return protoStructs
}

// Helper function to convert fields from codesurgeon to proto
func convertFieldsToProto(fields []codesurgeon.Field) []*api.Field {
	var protoFields []*api.Field
	for _, f := range fields {
		protoField := &api.Field{
			Name:    f.Name,
			Type:    f.Type,
			Tag:     f.Tag,
			Private: f.Private,
			Pointer: f.Pointer,
			Slice:   f.Slice,
			Docs:    f.Docs,
			Comment: f.Comment,
		}
		protoFields = append(protoFields, protoField)
	}
	return protoFields
}

// Helper function to convert methods from codesurgeon to proto
func convertMethodsToProto(methods []codesurgeon.Method) []*api.Method {
	var protoMethods []*api.Method
	for _, m := range methods {
		protoMethod := &api.Method{
			Receiver:  m.Receiver,
			Name:      m.Name,
			Params:    convertParamsToProto(m.Params),
			Returns:   convertParamsToProto(m.Returns),
			Docs:      m.Docs,
			Signature: m.Signature,
			Body:      m.Body,
		}
		protoMethods = append(protoMethods, protoMethod)
	}
	return protoMethods
}

// Helper function to convert parameters from codesurgeon to proto
func convertParamsToProto(params []codesurgeon.Param) []*api.Param {
	var protoParams []*api.Param
	for _, p := range params {
		protoParam := &api.Param{
			Name: p.Name,
			Type: p.Type,
		}
		protoParams = append(protoParams, protoParam)
	}
	return protoParams
}

// Helper function to convert functions from codesurgeon to proto
func convertFunctionsToProto(functions []codesurgeon.Function) []*api.Function {
	var protoFunctions []*api.Function
	for _, f := range functions {
		protoFunction := &api.Function{
			Name:      f.Name,
			Params:    convertParamsToProto(f.Params),
			Returns:   convertParamsToProto(f.Returns),
			Docs:      f.Docs,
			Signature: f.Signature,
			Body:      f.Body,
		}
		protoFunctions = append(protoFunctions, protoFunction)
	}
	return protoFunctions
}

// Helper function to convert variables from codesurgeon to proto
func convertVariablesToProto(variables []codesurgeon.Variable) []*api.Variable {
	var protoVariables []*api.Variable
	for _, v := range variables {
		protoVariable := &api.Variable{
			Name: v.Name,
			Type: v.Type,
			Docs: v.Docs,
		}
		protoVariables = append(protoVariables, protoVariable)
	}
	return protoVariables
}

// Helper function to convert constants from codesurgeon to proto
func convertConstantsToProto(constants []codesurgeon.Constant) []*api.Constant {
	var protoConstants []*api.Constant
	for _, c := range constants {
		protoConstant := &api.Constant{
			Name:  c.Name,
			Value: c.Value,
			Docs:  c.Docs,
		}
		protoConstants = append(protoConstants, protoConstant)
	}
	return protoConstants
}

// Helper function to convert interfaces from codesurgeon to proto
func convertInterfacesToProto(interfaces []codesurgeon.Interface) []*api.Interface {
	var protoInterfaces []*api.Interface
	for _, i := range interfaces {
		protoInterface := &api.Interface{
			Name:    i.Name,
			Methods: convertMethodsToProto(i.Methods),
			Docs:    i.Docs,
		}
		protoInterfaces = append(protoInterfaces, protoInterface)
	}
	return protoInterfaces
}

func (h *Handler) Introduction(ctx context.Context, req *connect.Request[api.IntroductionRequest]) (*connect.Response[api.IntroductionResponse], error) {
	res, err := h.GetOpenAPI(ctx, connect.NewRequest(&api.GetOpenAPIRequest{}))
	if err != nil {
		return nil, err
	}

	intro, err := ai.GetGPTIntroduction(res.Msg.Openapi)
	if err != nil {
		return nil, err
	}

	return &connect.Response[api.IntroductionResponse]{
		Msg: &api.IntroductionResponse{
			Introduction: intro,
		},
	}, nil
}

func (h *Handler) GetOpenAPI(ctx context.Context, req *connect.Request[api.GetOpenAPIRequest]) (*connect.Response[api.GetOpenAPIResponse], error) {
	// Read the embedded file using the embedded FS
	data, err := codesurgeon.FS.ReadFile("api/codesurgeon.openapi.json")
	if err != nil {
		return nil, err
	}

	parsed, err := gabs.ParseJSON(data)
	if err != nil {
		return nil, err
	}
	// https://chatgpt.com/gpts/editor/g-v09HRlzOu

	// add "server" field
	url := h.url
	url = strings.TrimSuffix(url, "/")

	parsed.Array("servers")
	parsed.ArrayAppend(map[string]string{
		"url": url,
	}, "servers")

	//
	// Update "openapi" field to "3.1.0"
	parsed.Set("3.1.0", "openapi")

	// Paths to check
	paths, err := parsed.Path("paths").ChildrenMap()
	if err != nil {
		return nil, err
	}

	// Iterate over paths to update "operationId"
	for _, path := range paths {
		// Get the "post" object within each path
		post := path.Search("post")
		if post != nil {

			post.Set("false", "x-openai-isConsequential")

			// Get current "operationId"
			operationID, ok := post.Path("operationId").Data().(string)
			if ok {
				// Split the "operationId" by "."
				parts := strings.Split(operationID, ".")
				operationID := "operationId"
				// Get the last 2 parts of the "operationId" and join them with a "_"
				if len(parts) > 1 {
					operationID = strings.Join(parts[len(parts)-2:], "_")
				} else if len(parts) > 0 {
					operationID = parts[0]
				}

				// Update "operationId"
				post.Set(operationID, "operationId")
			}
		}
	}

	return &connect.Response[api.GetOpenAPIResponse]{
		Msg: &api.GetOpenAPIResponse{
			Openapi: parsed.String(),
		},
	}, nil
}
