package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

const (
	appName      = "Shapeshifter"
	version      = "1.0.0"
	portAPI      = 9876
	portMCP      = 9875
	flagHTTPMode = "httpmode"
)

var (
	// hostAPIServer is either 'localhost' when running in HTTP Mode because it is co-located with the API server, or the IP of the API server if stdio
	hostAPIServer string
	// addrAPIServer is the fully formatted scheme:host:port of the API server where the Shapeshift methods actually reside
	addrAPIServer string
	// httpMode enables this program to run in remote (Server Side Events) mode
	httpMode bool

	// httpClient is defined custom because we need to work with self-signed certs on the API server
	httpClient = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	shapeList = []string{"circle", "square", "trapezoid", "pentagon", "hexagon"}
)

// ShapeState represents the current state of a shape
type ShapeState struct {
	Shape string `json:"shape"`
	Color string `json:"color"`
	Size  int    `json:"size"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   strings.ToLower(appName),
		Short: "Shapeshifter MCP Server",
		Run:   runCommand,
	}

	// Add HTTP mode flag
	rootCmd.Flags().BoolVar(&httpMode, flagHTTPMode, false, "Enable Streamable HTTP Mode")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runCommand(cmd *cobra.Command, args []string) {
	// Instantiate the server and add our Tools into it
	s := buildServer()

	// Determine HHTP mode
	if cmd.Flags().Changed(flagHTTPMode) {
		httpMode, _ = cmd.Flags().GetBool(flagHTTPMode)
	}

	// Run the MCP Server
	runServer(s, httpMode)
}

// buildServer configures the MCPServer with name, version, capabilities, and tool definitions
func buildServer() *server.MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		appName,
		version,
		server.WithToolCapabilities(false),
		server.WithPromptCapabilities(false),
		server.WithResourceCapabilities(false, false),
	)

	// Add Status Tool
	addStatusTool(s)

	// Add Shapeshift Tool
	addShapeshiftTool(s)

	// Add ShapesList Resource
	addShapesListResource(s)

	// Add Prompt Tool
	addDoubleShiftPrompt(s)

	return s
}

func addShapesListResource(s *server.MCPServer) {
	resourceName := "shapeList"
	resourceDesc := "Lists all the shapes that are available for shifting"
	resourceUri := "shapeshift://shapelist"
	resourceMimeType := "text/plain"
	resource := mcp.NewResource(resourceUri,
		resourceName,
		mcp.WithMIMEType(resourceMimeType),
		mcp.WithResourceDescription(resourceDesc),
	)
	s.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      resourceUri,
				MIMEType: resourceMimeType,
				Text:     strings.Join(shapeList, ", "),
			},
		}, nil

	})
}

func addDoubleShiftPrompt(s *server.MCPServer) {
	promptName := "doubleShift"
	promptDesc := "Asks the LLM to shift shapes twice between two user specified shapes"

	promptText := "Shapeshift into a {{shape1}}, and if that succeeds, then shapeshift into a {{shape2}}"
	prompt := mcp.NewPrompt(
		promptName,
		mcp.WithPromptDescription(promptDesc),
		mcp.WithArgument("shape1", mcp.RequiredArgument(), mcp.ArgumentDescription("first shape to shift into")),
		mcp.WithArgument("shape2", mcp.RequiredArgument(), mcp.ArgumentDescription("second shape to shift into")),
	)

	s.AddPrompt(prompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		s1, ok := request.Params.Arguments["shape1"]
		if !ok {
			return nil, errors.New("could not call prompt, missing shape1")
		}
		s2, ok := request.Params.Arguments["shape2"]
		if !ok {
			return nil, errors.New("could not call prompt, missing shape2")
		}
		txt := promptText
		txt = strings.ReplaceAll(txt, "{{shape1}}", s1)
		txt = strings.ReplaceAll(txt, "{{shape2}}", s2)
		return &mcp.GetPromptResult{
			Description: prompt.Description,
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: txt,
					},
				},
			},
		}, nil
	})
}

// addStatusTool adds the status tool to the MCP server
func addStatusTool(s *server.MCPServer) {
	toolName := "shapeStatus"
	toolDesc := "Gets the current status of the Shape"
	tool := mcp.NewTool(toolName, mcp.WithDescription(toolDesc))
	s.AddTool(tool, statusHandler)
}

// statusHandler retrieves the current shape status
func statusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/api/status", addrAPIServer))
	if err != nil {
		return mcp.NewToolResultErrorFromErr("mcp server failed to query shape status", err), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("mcp server failed to read request body", err), nil
	}

	var newState ShapeState
	if err := json.Unmarshal(body, &newState); err != nil {
		return mcp.NewToolResultErrorFromErr("mcp server failed to serialize response", err), nil
	}

	bytes, _ := json.Marshal(newState)
	return mcp.NewToolResultText(string(bytes)), nil
}

// addShapeshiftTool adds the shapeshift tool to the MCP server
func addShapeshiftTool(s *server.MCPServer) {
	toolName := "shapeshift"
	toolDesc := "Changes the Shape to the given parameters. Fields will be left as default if not supplied."
	tool := mcp.NewTool(
		toolName,
		mcp.WithDescription(toolDesc),
		mcp.WithString("shape", mcp.Enum("circle", "square", "pentagon", "hexagon", "trapezoid")),
		mcp.WithString("color"),
		mcp.WithNumber("size", mcp.Min(0), mcp.Max(200), mcp.MultipleOf(1)),
	)
	s.AddTool(tool, mcp.NewTypedToolHandler(shapeshiftHandler))
}

// shapeshiftHandler changes the shape according to the input
func shapeshiftHandler(ctx context.Context, request mcp.CallToolRequest, shape ShapeState) (*mcp.CallToolResult, error) {
	jsonData, err := json.Marshal(shape)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("mcp server failed to deserialize request", err), nil
	}

	resp, err := httpClient.Post(
		fmt.Sprintf("%s/api/status", addrAPIServer),
		"application/json",
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to shapeshift on API server", err), nil
	}
	defer resp.Body.Close()

	return mcp.NewToolResultText("{result: \"200, OK\"}"), nil
}

// runServer runs the MCP server in either Stdio or HTTP mode
func runServer(s *server.MCPServer, httpMode bool) {
	if httpMode {
		setHTTPConfig()
		runHTTPServer(s)
	} else {
		setStdioConfig()
		runStdioServer(s)
	}
}

// runHTTPServer starts the HTTP server
func runHTTPServer(s *server.MCPServer) {
	httpServer := server.NewStreamableHTTPServer(s)
	log.Printf("HTTP server listening on :%d/mcp\n", portMCP)
	if err := httpServer.Start(fmt.Sprintf(":%d", portMCP)); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// runStdioServer starts the Stdio server
func runStdioServer(s *server.MCPServer) {
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// setHTTPConfig sets config values for HTTP (remote) mode
func setHTTPConfig() {
	hostAPIServer = "localhost"
	addrAPIServer = fmt.Sprintf("https://%s:%d", hostAPIServer, portAPI)
}

// setStdioConfig sets config values for Stdio (local) mode
func setStdioConfig() {
	hostAPIServer = "18.183.57.194"
	addrAPIServer = fmt.Sprintf("https://%s:%d", hostAPIServer, portAPI)
}
