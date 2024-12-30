package neo4j2

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
	codesurgeon "github.com/wricardo/code-surgeon"
)

func ToNeo4j(ctx context.Context, path string, deep bool, myEnv map[string]string, recursive bool) error {
	neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
	neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
	neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]
	driver, closeFn, err := Connect(ctx, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
	if err != nil {
		log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
		return err
	} else {
		defer closeFn()
	}
	sess := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer sess.Close(ctx)

	// Execute the command to list Go modules in JSON format
	cmd := exec.Command("go", "list", "-json", path)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute go list command: %w", err)
	}

	// fmt.Println(string(output))

	// Parse the JSON output and pretty-print
	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for decoder.More() {
		fmt.Println("decoder.More()")
		var module codesurgeon.GoList
		if err := decoder.Decode(&module); err != nil {
			log.Printf("failed to decode module: %v", err)
			continue
		}

		if recursive {
			log.Info().Msgf("Parsed %s", module.Dir)
			infos, err := codesurgeon.ParseDirectoryRecursive(module.Dir)
			if err != nil {
				log.Info().Err(err).Msgf("Error parsing file %s", module.Dir)
				return err
			}
			for _, info := range infos {
				shouldContinue, err1 := toNeo4j(ctx, info, module.ImportPath, module.Dir, sess, deep)
				if err1 != nil {
					if shouldContinue {
						continue
					}

					return err1
				}
			}
		} else {
			log.Info().Msgf("Parsed %s", module.Dir)
			info, err := codesurgeon.ParseDirectory(module.Dir)
			if err != nil {
				log.Info().Err(err).Msgf("Error parsing file %s", module.Dir)
			}

			shouldContinue, err1 := toNeo4j(ctx, info, module.Dir, module.ImportPath, sess, deep)
			if err1 != nil {
				if shouldContinue {
					continue
				}
			}

		}

	}

	return nil
}

// toNeo4j processes parsed information and upserts it into Neo4j using the provided session.
func toNeo4j(ctx context.Context, info *codesurgeon.ParsedInfo, moduleDir, moduleImportPath string, session neo4j.SessionWithContext, deep bool) (bool, error) {
	var err error
	for _, mod := range info.Modules {
		for _, pkg := range mod.Packages {
			err = UpsertPackage(ctx, session, mod, pkg)
			if err != nil {
				log.Info().Err(err).Msgf("Error upserting package %s", pkg.Package)
				return true, err
			}

			for k, struct_ := range pkg.Structs {
				log.Info().Msgf("struct %d: %s", k, struct_.Name)
				if err = UpsertStruct(ctx, session, mod, info.Packages[0], struct_); err != nil {
					log.Info().Err(err).Msgf("Error upserting struct %s", struct_.Name)
					return true, err
				}
				// methods
				for k2, method := range struct_.Methods {
					funcFilePath := ""
					if deep {
						funcFilePath, err = codesurgeon.FindFunction(moduleDir, struct_.Name, method.Name)
						if err != nil {
							log.Info().Err(err).Msgf("Error finding function file %s", method.Name)
						} else {
							log.Info().Msgf("funcFilePath: %s", funcFilePath)
						}
					}
					err = UpsertMethod(ctx, session, mod, pkg, method, struct_)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting function %s", method.Name)
						return true, err
					}
					log.Info().Msgf("method %d %d: %s", k, k2, method.Name)
					for _, param := range method.Params {
						err = UpsertMethodParam(ctx, session, mod, pkg, struct_, method, param)
						if err != nil {
							log.Info().Err(err).Msgf("Error upserting function param %s", param.Name)
							return true, err
						}
					}
					for _, result := range method.Returns {
						err = UpsertMethodReturn(ctx, session, mod, pkg, method, result)
						if err != nil {
							log.Info().Err(err).Msgf("Error upserting function return %s", result.Name)
							return true, err
						}
					}

				}
				// fields
				for k2, field := range struct_.Fields {
					// Upsert each field of the struct into Neo4j
					err = UpsertStructField(ctx, session, mod, pkg, struct_, field)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting struct field %s", field.Name)
						return true, err
					}
					log.Info().Msgf("field %d %d: %s", k, k2, field.Name)
				}

			}
			fmt.Println("len functions", len(info.Packages[0].Functions))
			for k, function := range info.Packages[0].Functions {
				funcFilePath := ""
				if deep {
					funcFilePath, err = codesurgeon.FindFunction(moduleDir, "", function.Name)
					if err != nil {
						log.Info().Err(err).Msgf("Error finding function file %s", function.Name)
					} else {
						log.Info().Msgf("funcFilePath: %s", funcFilePath)
					}
				}
				err = UpsertFunction(ctx, session, mod, pkg, function)
				if err != nil {
					log.Info().Err(err).Msgf("Error upserting function %s", function.Name)
					return true, err
				}
				log.Info().Msgf("function %d: %s", k, function.Name)
				for _, param := range function.Params {
					err = UpsertFunctionParam(ctx, session, mod, pkg, function, param)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting function param %s", param.Name)
						return true, err
					}
				}
				for _, ret := range function.Returns {
					err = UpsertFunctionReturn(ctx, session, mod, pkg, function, ret)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting function return %s", ret.Name)
						return true, err
					}
				}
			}
			for _, interface_ := range info.Packages[0].Interfaces {
				log.Info().Msgf("interface: %s", interface_.Name)
				if err = UpsertInterface(ctx, session, mod, pkg, interface_); err != nil {
					log.Info().Err(err).Msgf("Error upserting interface %s", interface_.Name)
					return true, err
				}
				for _, method := range interface_.Methods {
					err = UpsertInterfaceMethod(ctx, session, mod, pkg, interface_, method)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting function %s", method.Name)
						return true, err
					}
					for _, param := range method.Params {
						err = UpsertInterfaceMethodParam(ctx, session, mod, pkg, interface_, method, param)
						if err != nil {
							log.Info().Err(err).Msgf("Error upserting function param %s", param.Name)
							return true, err
						}
					}
					for _, result := range method.Returns {
						err = UpsertInterfaceMethodReturn(ctx, session, mod, pkg, interface_, method, result)
						if err != nil {
							log.Info().Err(err).Msgf("Error upserting function return %s", result.Name)
							return true, err
						}
					}
				}
			}
		}
	}
	return false, nil
}
