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

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

const APPNAME = "Shapeshifter"
const VERSION = "1.0.0"

type ShapeState struct {
	Shape string `json:"shape"`
	Color string `json:"color"`
	Size  int    `json:"size"`
}

var (
	addrBind string
	addrPort uint16
	addr     = "https://%s:%d"
	sseMode  bool
)
var tr = &http.Transport{
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}
var client = &http.Client{Transport: tr}

func main() {
	rootCmd := &cobra.Command{
		Use:   strings.ToLower(APPNAME),
		Short: "Shapeshifter MCP Server",
		Run:   run,
	}

	// Add the SSE mode flag
	rootCmd.Flags().BoolVar(&sseMode, "ssemode", false, "Enable Server-Sent Events mode")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(cmd *cobra.Command, args []string) {
	// Create a new MCP server
	s := server.NewMCPServer(
		APPNAME,
		VERSION,
		// Server has tools, and the tools list does not change
		server.WithToolCapabilities(false),
	)

	// Add Status Tool
	nameStatus := "shapeStatus"
	descStatus := "Gets the current status of the Shape"
	toolStatus := mcp.NewTool(nameStatus, mcp.WithDescription(descStatus))
	// Add Status Tool handler
	s.AddTool(toolStatus, statusHandler)

	// Add Shapeshift Tool
	nameShapeshift := "shapeshift"
	descShapeshift := "Changes the Shape to the given parameters. Fields will be left as default if not supplied. Color is standard web hex colors for use in HTML, convert user colors to HTML. Size is always an integer."
	toolShapeshift := mcp.NewTool(nameShapeshift, mcp.WithDescription(descShapeshift),
		mcp.WithString("shape", mcp.Enum("circle", "square", "pentagon", "hexagon", "trapezoid")),
		mcp.WithString("color"),
		mcp.WithNumber("size", mcp.Min(0), mcp.Max(200), mcp.MultipleOf(1)),
	)
	// Add Shapeshift Tool handler
	s.AddTool(toolShapeshift, mcp.NewTypedToolHandler(shapeshiftHandler))

	// Check for Server Side mode, or Local Mode
	sseMode := false
	if cmd.Flags().Changed("ssemode") {
		sseMode, _ = cmd.Flags().GetBool("ssemode")
	}
	if sseMode {
		setRemoteConfig()

		httpServer := server.NewSSEServer(s)
		log.Printf("HTTP server listening on :%d/mcp\n", addrPort)
		if err := httpServer.Start(fmt.Sprintf(":%d", addrPort)); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		setLocalConfig()

		// Start the stdio server
		if err := server.ServeStdio(s); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}
}

func setRemoteConfig() {
	addrBind = "localhost"
	addrPort = 9875
	addr = fmt.Sprintf(addr, addrBind, addrPort)
}

func setLocalConfig() {
	addrBind = "18.183.57.194"
	addrPort = 9876
	addr = fmt.Sprintf(addr, addrBind, addrPort)
}

func statusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := client.Get(fmt.Sprintf("%s/api/status", addr))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var newState ShapeState
	if err := json.Unmarshal(body, &newState); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return mcp.NewToolResultErrorFromErr("error unmarshalling response", err), nil
	}

	bytes, _ := json.Marshal(newState)

	return mcp.NewToolResultText(string(bytes)), nil
}

func shapeshiftHandler(ctx context.Context, request mcp.CallToolRequest, shape ShapeState) (*mcp.CallToolResult, error) {
	jsonData, err := json.Marshal(shape)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal shape", err), nil
	}

	resp, err := client.Post(
		fmt.Sprintf("%s/api/status", addr),
		"application/json",
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to shapeshift on server", err), nil
	}
	defer resp.Body.Close()

	return mcp.NewToolResultText("{result: \"200, OK\"}"), nil

}
