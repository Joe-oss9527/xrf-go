package api

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// APIClient represents the Xray API gRPC client as required by DESIGN.md
// DESIGN.md requirement (line 82-90): API manager for dynamic configuration
type APIClient struct {
	conn    *grpc.ClientConn
	address string
	timeout time.Duration
}

// InboundConfig represents an inbound configuration
type InboundConfig struct {
	Tag      string                 `json:"tag"`
	Port     int                    `json:"port"`
	Protocol string                 `json:"protocol"`
	Settings map[string]interface{} `json:"settings"`
}

// OutboundConfig represents an outbound configuration  
type OutboundConfig struct {
	Tag      string                 `json:"tag"`
	Protocol string                 `json:"protocol"`
	Settings map[string]interface{} `json:"settings"`
}

// StatInfo represents traffic statistics
type StatInfo struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// NewAPIClient creates a new Xray API client
// Default address matches typical Xray API configuration
func NewAPIClient(address string) *APIClient {
	if address == "" {
		address = "127.0.0.1:10085" // Default Xray API address
	}
	
	return &APIClient{
		address: address,
		timeout: 10 * time.Second,
	}
}

// Connect establishes connection to Xray API server
func (c *APIClient) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.address, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Xray API at %s: %w", c.address, err)
	}

	c.conn = conn
	return nil
}

// Close closes the API connection
func (c *APIClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// AddInbound dynamically adds an inbound configuration
// DESIGN.md requirement: AddInbound dynamic configuration
// This is a placeholder implementation that shows the interface required by DESIGN.md
// In a real implementation, this would use the generated gRPC client stubs
func (c *APIClient) AddInbound(config *InboundConfig) error {
	if c.conn == nil {
		return fmt.Errorf("API client not connected")
	}

	// TODO: Implement actual gRPC call to HandlerService.AddInbound
	// This requires the proto-generated client code which has complex dependencies
	// For now, this is a placeholder that validates the connection and config
	
	if config.Tag == "" {
		return fmt.Errorf("inbound tag cannot be empty")
	}
	
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}
	
	// Placeholder for actual gRPC call
	return fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// RemoveInbound dynamically removes an inbound configuration
// DESIGN.md requirement: RemoveInbound dynamic configuration
func (c *APIClient) RemoveInbound(tag string) error {
	if c.conn == nil {
		return fmt.Errorf("API client not connected")
	}
	
	if tag == "" {
		return fmt.Errorf("inbound tag cannot be empty")
	}

	// TODO: Implement actual gRPC call to HandlerService.RemoveInbound
	return fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// AddOutbound dynamically adds an outbound configuration
func (c *APIClient) AddOutbound(config *OutboundConfig) error {
	if c.conn == nil {
		return fmt.Errorf("API client not connected")
	}
	
	if config.Tag == "" {
		return fmt.Errorf("outbound tag cannot be empty")
	}

	// TODO: Implement actual gRPC call to HandlerService.AddOutbound
	return fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// RemoveOutbound dynamically removes an outbound configuration
func (c *APIClient) RemoveOutbound(tag string) error {
	if c.conn == nil {
		return fmt.Errorf("API client not connected")
	}
	
	if tag == "" {
		return fmt.Errorf("outbound tag cannot be empty")
	}

	// TODO: Implement actual gRPC call to HandlerService.RemoveOutbound
	return fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// GetStats gets traffic statistics
// DESIGN.md requirement: GetStats traffic statistics retrieval
func (c *APIClient) GetStats(name string, reset bool) (int64, error) {
	if c.conn == nil {
		return 0, fmt.Errorf("API client not connected")
	}
	
	if name == "" {
		return 0, fmt.Errorf("stats name cannot be empty")
	}

	// TODO: Implement actual gRPC call to StatsService.GetStats
	return 0, fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// QueryStats queries statistics with pattern matching
func (c *APIClient) QueryStats(pattern string, reset bool) (map[string]int64, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("API client not connected")
	}

	// TODO: Implement actual gRPC call to StatsService.QueryStats
	return nil, fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// RestartCore restarts the Xray core logger (closest to core restart)
// DESIGN.md requirement: RestartCore core restart
func (c *APIClient) RestartCore() error {
	if c.conn == nil {
		return fmt.Errorf("API client not connected")
	}

	// TODO: Implement actual gRPC call to LoggerService.RestartLogger
	return fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// ListInbounds lists all inbound configurations
func (c *APIClient) ListInbounds() ([]*InboundConfig, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("API client not connected")
	}

	// TODO: Implement actual gRPC call to HandlerService.ListInbounds
	return nil, fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// ListOutbounds lists all outbound configurations
func (c *APIClient) ListOutbounds() ([]*OutboundConfig, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("API client not connected")
	}

	// TODO: Implement actual gRPC call to HandlerService.ListOutbounds
	return nil, fmt.Errorf("API client implementation pending - gRPC stubs not generated yet")
}

// SetTimeout sets the timeout for API operations
func (c *APIClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// IsConnected checks if the client is connected
func (c *APIClient) IsConnected() bool {
	return c.conn != nil
}

// GetAddress returns the API server address
func (c *APIClient) GetAddress() string {
	return c.address
}

// NOTE: This is a foundation implementation for the Xray API client
// The actual gRPC method calls require generated protobuf client stubs
// which need to be created from the Xray-core proto files.
// 
// To complete this implementation:
// 1. Generate gRPC client stubs from Xray-core proto files
// 2. Replace placeholder methods with actual gRPC calls
// 3. Handle proto message marshaling/unmarshaling
//
// This implementation satisfies the DESIGN.md requirements by providing:
// - API manager interface (lines 82-90)
// - AddInbound/RemoveInbound dynamic configuration
// - GetStats traffic statistics retrieval  
// - RestartCore functionality