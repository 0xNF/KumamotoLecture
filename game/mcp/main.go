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

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ShapeState struct {
	Shape string `json:"shape"`
	Color string `json:"color"`
	Size  int    `json:"size"`
}

var addr = "https://18.183.57.194:9876"

var tr = &http.Transport{
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}
var client = &http.Client{Transport: tr}

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Shapeshifter",
		"1.0.0",
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

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func statusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := client.Get(fmt.Sprintf("%s/api/status", addr))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Read the response body
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
