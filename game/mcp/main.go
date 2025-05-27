package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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
	appName = "Shapeshifter"
	version = "1.0.0"
	portAPI = 9876
	portMCP = 9875
)

var (
	hostAPIServer string
	addrAPIServer string
	sseMode       bool

	// httpClient is defined custom because we need to work with self-signed certs on the API server
	httpClient = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
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

	// Add SSE mode flag
	rootCmd.Flags().BoolVar(&sseMode, "ssemode", false, "Enable Server-Sent Events mode")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runCommand(cmd *cobra.Command, args []string) {
	// Instantiate the server and add our Tools into it
	s := buildServer()

	// Determine SSE mode
	if cmd.Flags().Changed("ssemode") {
		sseMode, _ = cmd.Flags().GetBool("ssemode")
	}

	// Run the MCP Server
	runServer(s, sseMode)
}

// buildServer configures the MCPServer with name, version, capabilities, and tool definitions
func buildServer() *server.MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		appName,
		version,
		server.WithToolCapabilities(false),
	)

	// Add Status Tool
	addStatusTool(s)

	// Add Shapeshift Tool
	addShapeshiftTool(s)

	return s
}

// addStatusTool adds the status tool to the MCP server
func addStatusTool(s *server.MCPServer) {
	toolName := "shapeStatus"
	toolDesc := "Gets the current status of the Shape"
	tool := mcp.NewTool(toolName, mcp.WithDescription(toolDesc))
	s.AddTool(tool, statusHandler)
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

// runServer runs the MCP server in either Stdio or SSE mode
func runServer(s *server.MCPServer, sseMode bool) {
	if sseMode {
		setSSEConfig()
		runSSEServer(s)
	} else {
		setStdioConfig()
		runStdioServer(s)
	}
}

// runSSEServer starts the SSE server
func runSSEServer(s *server.MCPServer) {
	httpServer := server.NewSSEServer(s)
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

// setSSEConfig sets config values for SSE (remote) mode
func setSSEConfig() {
	hostAPIServer = "localhost"
	addrAPIServer = fmt.Sprintf("https://%s:%d", hostAPIServer, portAPI)
}

// setStdioConfig sets config values for Stdio (local) mode
func setStdioConfig() {
	hostAPIServer = "18.183.57.194"
	addrAPIServer = fmt.Sprintf("https://%s:%d", hostAPIServer, portAPI)
}
