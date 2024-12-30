package neo4j2

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const PKG_TO_IGNORE_STACKTRACE = "github.com/wricardo/code-surgeon"

var GLOBAL_STACK_TO_GRAPH *StackToGraph

type StackToGraph struct {
	neo4jDriver neo4j.Driver
	sync.Mutex
	cacheReportedStacks map[string]bool
}

func NewStackToGraph(uri, username, password string) (*StackToGraph, error) {
	driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	return &StackToGraph{
		neo4jDriver: driver,
	}, nil
}

func (s *StackToGraph) SetupGlobal() {
	GLOBAL_STACK_TO_GRAPH = s
}

func (s *StackToGraph) ReportStacktrace() error {
	// Capture the stack trace
	stack := captureStackTrace()

	// Lock the cache to avoid race
	s.Lock()
	if s.cacheReportedStacks == nil {
		s.cacheReportedStacks = make(map[string]bool)
	}
	if _, ok := s.cacheReportedStacks[stack]; ok {
		s.Unlock()
		// Skip reporting the same stack trace
		return nil
	}
	s.Unlock()

	// Parse the stack trace to extract function calls
	parsedStack := parseStackTrace(stack)

	// Report the stack trace to Neo4j
	err := s.reportStackTraceToNeo4j(parsedStack)
	if err != nil {
		log.Printf("Error reporting stack trace to Neo4j: %v\n", err)
		return err
	}
	s.Lock()
	s.cacheReportedStacks[stack] = true
	s.Unlock()

	return nil
}

func (s *StackToGraph) reportStackTraceToNeo4j(stackTraceData []ParsedStackEntry) error {
	// Create a new session
	session := s.neo4jDriver.NewSession(neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close()

	stackHash := getStackHash(stackTraceData)

	stackQuery := CypherQuery{}
	stackQuery = stackQuery.Merge(MergeQuery{
		NodeType: "Stack",
		Alias:    "s",
		Properties: map[string]any{
			"id": stackHash,
		},
		SetFields: map[string]string{},
	})

	// Execute a write transaction
	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {

		res, err := stackQuery.Return("s.id as id").ExecuteSession((context.Background()), session)
		if err != nil {
			return nil, fmt.Errorf("failed to execute stack query: %w", err)
		}

		tmp, ok := res[0].Get("id")
		if !ok {
			return nil, fmt.Errorf("failed to get id from result")
		}
		stackId := tmp.(string)

		var previousNodeID string

		// Reverse the stack to represent the top-down call flow
		for i := len(stackTraceData) - 1; i >= 0; i-- {
			frame := stackTraceData[i]
			if strings.HasPrefix(frame.Package, PKG_TO_IGNORE_STACKTRACE) && frame.Function == "captureStackTrace" {
				continue
			}
			if strings.HasPrefix(frame.Package, PKG_TO_IGNORE_STACKTRACE) && frame.Function == "ReportStacktrace" && frame.Receiver == "" {
				continue
			}

			if frame.Package == "" {
				return nil, fmt.Errorf("validation error package is empty")
			} else if frame.PackageName == "" {
				return nil, fmt.Errorf("validation error packageName is empty")
			}
			if frame.Function == "" {
				frame.Function = "UNKNOWN"
			}

			// }
			ctx := context.Background()

			query := CypherQuery{}.
				Merge(MergeCall(ctx, "mc", stackId, frame))

			if previousNodeID != "" {
				query = query.
					With("mc").
					Match(MatchQuery{
						NodeType: "Call",
						Alias:    "pn",
						Properties: map[string]any{
							"id": previousNodeID,
						},
					}).
					MergeRel("mc", "HAS_PARENT", "pn", nil)
			} else {
				query = query.
					With("mc").
					Match(MatchQuery{
						NodeType: "Stack",
						Alias:    "s",
						Properties: map[string]any{
							"id": stackId,
						},
					}).
					MergeRel("mc", "HAS_PARENT", "s", nil)
			}

			if frame.Receiver != "" {
				if frame.Function == "SendMessage" {
					log.Printf("frame: %v\n", frame)
				}

				query = query.
					With("mc").
					OptionalMatch(MatchQuery{
						NodeType: "Method",
						Alias:    "m",
						Properties: map[string]any{
							"name":            frame.Function,
							"receiver":        strings.Replace(frame.Receiver, "*", "", 1),
							"packageFullName": frame.Package,
						},
					}).
					With("m", "mc").
					Raw(`
					FOREACH (_ IN CASE WHEN m IS NOT NULL THEN [1] ELSE [] END |
						MERGE (mc)-[:RELATED]->(m)
					)
					FOREACH (_ IN CASE WHEN m IS NULL THEN [1] ELSE [] END |
		MERGE (mn:MissingNode {name: "` + frame.Function + ", packageFullName: " + frame.Package + `", receiver: "` + frame.Receiver + `"})
		MERGE (mc)-[:NOT_RELATED]->(mn)
				)
				`)
			} else {
				query = query.
					With("mc").
					OptionalMatch(MatchQuery{
						NodeType: "Function",
						Alias:    "f",
						Properties: map[string]any{
							"name":            frame.Function,
							"packageFullName": frame.Package,
						},
					}).
					With("f", "mc").
					Raw(`
					FOREACH (_ IN CASE WHEN f IS NOT NULL THEN [1] ELSE [] END |
						MERGE (mc)-[:RELATED]->(f)
					)
					FOREACH (_ IN CASE WHEN f IS NULL THEN [1] ELSE [] END |
		MERGE (mn:MissingNode {name: "` + frame.Function + ", packageFullName: " + frame.Package + `"})
		MERGE (mc)-[:NOT_RELATED]->(mn)
)
				`)

			}
			query = query.Return("mc.id as id")

			res, err := query.ExecuteSession(ctx, session)
			if err != nil {
				return nil, fmt.Errorf("failed to execute merge function query: %w", err)
			}

			// query.Args["mcid"]
			tmp, ok := res[0].Get("id")
			if !ok {
				tmp, ok = query.Args["mcid"]
				if !ok {
					return nil, fmt.Errorf("failed to get id from result")
				}
			}
			previousNodeID, ok = tmp.(string)
			if !ok {
				tmp, ok = query.Args["mcid"]
				if !ok {
					return nil, fmt.Errorf("failed to get id from result")
				}
				previousNodeID, ok = tmp.(string)
				if !ok {
					return nil, fmt.Errorf("failed to get id from result, not a string")
				}
			}
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("failed to execute write transaction: %w", err)
	}

	return nil
}

// Close the Neo4j driver when the application exits
func (s *StackToGraph) Close() {
	if s.neo4jDriver != nil {
		s.neo4jDriver.Close()
	}
}

// ReportStacktrace encapsulates capturing, parsing, and reporting the stack trace to Neo4j.
func ReportStacktrace() error {
	if GLOBAL_STACK_TO_GRAPH == nil {
		return fmt.Errorf("global StackToGraph instance is not set. Call SetupGlobal before using ReportStacktrace.")
	}
	return GLOBAL_STACK_TO_GRAPH.ReportStacktrace()
}

// captureStackTrace captures the current call stack as a string.
func captureStackTrace() string {
	// Adjust buffer size if needed (1<<16 is 64KB, which is usually sufficient)
	buf := make([]byte, 1<<16)
	stackSize := runtime.Stack(buf, false)
	return string(buf[:stackSize])
}

func getStackHash(parsedStack []ParsedStackEntry) string {
	var stackHash string
	for _, frame := range parsedStack {
		stackHash += fmt.Sprintf("%s:%s:%s", frame.Package, frame.Function, frame.Line)
	}
	// return md5 of the stackHash
	return fmt.Sprintf("%x", md5.Sum([]byte(stackHash)))
}

// parseStackTrace extracts function names, file paths, and line numbers from the stack trace.
// It also cleans up function names by removing arguments.
func parseStackTrace(stackTrace string) []ParsedStackEntry {
	// Regular expression to capture function calls in the stack trace
	// Example stack trace lines:
	// main.isError({0x104486073?, 0xc?})
	//     /path/to/file/main.go:24 +0x9f
	re := regexp.MustCompile(`(?m)^(.*?)\n\s+(.*?)\:(\d+)(?: \+0x[0-9a-f]+)?$`)

	matches := re.FindAllStringSubmatch(stackTrace, -1)
	var parsedData []ParsedStackEntry

	for _, match := range matches {
		functionWithArgs := strings.TrimSpace(match[1])
		// Remove arguments from the function name using regex
		cleanFunction := cleanFunctionName(functionWithArgs)
		pkg, shortName := ParsePackageName(cleanFunction)
		cleanFunction = strings.TrimPrefix(cleanFunction, pkg+".")
		cleanFunction = strings.TrimPrefix(cleanFunction, pkg)
		cleanFunction = strings.Replace(cleanFunction, "[...]", "", -1)

		original, cleanFunction, receiver := ParseReceiver(cleanFunction)

		file := match[2]
		line := match[3]

		// Extract the folder name from the file path
		folder, folderName := ParseFolder(file)

		repository, repoOrg, repoName := ParseRepository(pkg, folder)

		parsedData = append(parsedData, ParsedStackEntry{
			Receiver:               receiver,
			Function:               cleanFunction,
			File:                   file,
			Folder:                 folder,
			FolderName:             folderName,
			Line:                   line,
			Package:                pkg,
			PackageName:            shortName,
			OriginalName:           original,
			Repository:             repository,
			RepositoryOrganization: repoOrg,
			RepositoryName:         repoName,
		})
	}

	return parsedData
}

func ParseFolder(s string) (string, string) {
	parts := strings.Split(s, "/")
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-2]
	}
	return "", ""
}

func ParseRepository(pkg string, folder string) (string, string, string) {
	if strings.HasPrefix(pkg, "github.com/") || strings.HasPrefix(pkg, "bitbucket.org/") || strings.HasPrefix(pkg, "gitlab.com/") {
		parts := strings.Split(pkg, "/")
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "/"), parts[1], parts[2]
		} else if len(parts) == 1 {
			return "", "", ""
		} else if len(parts) == 0 {
			return "", "", ""
		}
	} else if strings.HasPrefix(pkg, "gorm.io") {
		parts := strings.Split(pkg, "/")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], "/"), parts[0], parts[1]
		} else if len(parts) == 1 {
			return parts[0], parts[0], ""
		} else if len(parts) == 0 {
			return "gorm", "gorm", "gorm"
		}
	} else if strings.Index(pkg, "/") != -1 {
		parts := strings.Split(pkg, "/")
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "/"), parts[1], parts[2]
		} else if len(parts) >= 2 {
			return strings.Join(parts[:2], "/"), strings.Join(parts[:2], "/"), parts[1]
		} else if len(parts) == 1 {
			return parts[0], parts[0], parts[0]
		} else if len(parts) == 0 {
			return "", "", ""
		}
	} else if strings.Index(folder, "/") != -1 {
		parts := strings.Split(folder, "/")
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "/"), parts[len(parts)-2], parts[len(parts)-1]
		} else if len(parts) >= 2 {
			return strings.Join(parts[:2], "/"), strings.Join(parts[:2], "/"), parts[2]
		} else if len(parts) == 1 {
			return parts[0], parts[0], parts[0]
		} else if len(parts) == 0 {
			return "", "", ""
		}

	}
	return "", "", ""
}

// ParseReceiver extracts the receiver from a function name.
// It returns the original function name, cleaned function name and the receiver.
func ParseReceiver(s string) (string, string, string) {
	if s == "" {
		return "", "", ""
	}

	parts := strings.Split(s, ".")
	if len(parts) == 0 {
		return s, s, ""
	}

	// Regular expression to match anonymous functions like func1, func2, func6.1
	anonFuncPattern := regexp.MustCompile(`^func\d+(\.\d+)*$`)

	// Start from the end and find the first non-anonymous function name
	functionName := ""
	var receiver string
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if anonFuncPattern.MatchString(part) {
			continue
		} else {
			functionName = part
			if i > 0 {
				receiver = parts[i-1]
			}
			break
		}
	}

	// If functionName is still empty, it means all parts were anonymous functions
	if functionName == "" {
		functionName = parts[len(parts)-1]
		if len(parts) > 1 {
			receiver = parts[len(parts)-2]
		}
	}

	return s, functionName, ReplacePointerNotation(receiver)
}

// ReplacePointerNotation uses regex to replace patterns like "(*TypeName)" with "TypeName"
func ReplacePointerNotation(input string) string {
	// Define the regex pattern to match "(*TypeName)"
	// Explanation:
	// \\(\\*    : Matches the literal "(*"
	// ([A-Za-z0-9_]+) : Captures one or more alphanumeric characters or underscores (TypeName)
	// \\)       : Matches the literal ")"
	pattern := `\(\*([A-Za-z0-9_]+)\)`

	// Compile the regex
	re := regexp.MustCompile(pattern)

	// Replace all matches with the captured group "$1" (TypeName)
	result := re.ReplaceAllString(input, `$1`)

	return result
}

func cleanFunctionName(s string) string {
	// Remove any trailing whitespace
	s = strings.TrimRight(s, " \t\n\r")

	// Start scanning from the end of the string backwards
	parenLevel := 0
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if c == ')' {
			parenLevel++
		} else if c == '(' {
			parenLevel--
			if parenLevel == 0 {
				// We have found the outermost '('
				// Remove everything from position i onwards
				return s[:i]
			}
		}
	}
	// If no outermost '(', return the original string
	return s
}

// ParsePackageName extracts the package full path and short name from a fully qualified package path.
func ParsePackageName(s string) (fullName string, shortName string) {
	fullName = parsePackageName(s)
	parts := strings.Split(fullName, "/")
	if len(parts) > 0 {
		shortName = parts[len(parts)-1]
	} else {
		shortName = fullName
	}
	return
}

// parsePackageName extracts the package path from a fully qualified method/function string.
// It returns the package path up to the first '.' after the last '/'.
// If there is no '/', it takes the string up to the first '.'.
func parsePackageName(s string) string {
	// Find the index of the last '/'
	lastSlash := strings.LastIndex(s, "/")
	if lastSlash == -1 {
		lastSlash = 0
	} else {
		lastSlash += 1 // Move past the '/'
	}

	// Find the index of the first '.' after the last '/'
	dotIndex := strings.Index(s[lastSlash:], ".")
	if dotIndex == -1 {
		// No '.' found after the last '/', return the entire string
		return s
	}

	// The package path is up to lastSlash + dotIndex
	return s[:lastSlash+dotIndex]
}

type ParsedStackEntry struct {
	Receiver               string
	Function               string
	File                   string // /path/to/file.go
	Folder                 string // /path/to
	FolderName             string // to
	Line                   string
	Package                string // github.com/x/y/z
	PackageName            string // z
	OriginalName           string
	Repository             string // github.com/x/y
	RepositoryOrganization string // x
	RepositoryName         string // y
}

func MergeCall(ctx context.Context, alias string, stack_id string, pe ParsedStackEntry) MergeQuery {
	function := pe.Function
	receiver := pe.Receiver
	pkg := pe.Package
	packageName := pe.PackageName
	properties := map[string]any{
		"stack_id":        stack_id,
		"function":        function,
		"receiver":        receiver,
		"packageFullName": pkg,
		"packageName":     packageName,
	}
	id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%v", properties))))

	return MergeQuery{
		NodeType:   "Call",
		Alias:      alias,
		Properties: properties,
		SetFields: map[string]string{
			"id": id,
		},
	}
}
