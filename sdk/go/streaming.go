package featureflags

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// StreamingClient handles real-time configuration updates via Server-Sent Events
type StreamingClient struct {
	config *StreamingConfig
	logger zerolog.Logger

	// Connection state
	status StreamingStatus
	conn   *http.Response
	reader *bufio.Scanner

	// Synchronization
	mutex     sync.RWMutex
	stopChan  chan struct{}
	doneChan  chan struct{}
	readyChan chan struct{}

	// Reconnection
	reconnectDelay time.Duration
	maxRetries     int
	retryCount     int

	// Statistics
	connectTime    time.Time
	lastEventTime  time.Time
	eventsReceived int64
	reconnects     int64
	errors         int64
}

// StreamingConfig holds configuration for the streaming client
type StreamingConfig struct {
	EvaluatorEndpoint string
	APIKey            string
	Environment       string
	Reconnect         bool
	HeartbeatInterval time.Duration
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	Cache             *Cache
	Offline           *OfflineHandler
}

// NewStreamingClient creates a new streaming client
func NewStreamingClient(config *StreamingConfig, logger zerolog.Logger) (*StreamingClient, error) {
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = -1 // Infinite retries by default
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = time.Second
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}

	client := &StreamingClient{
		config:         config,
		logger:         logger.With().Str("component", "streaming").Logger(),
		status:         StatusDisconnected,
		stopChan:       make(chan struct{}),
		doneChan:       make(chan struct{}),
		readyChan:      make(chan struct{}),
		reconnectDelay: config.InitialDelay,
		maxRetries:     config.MaxRetries,
	}

	client.logger.Info().
		Str("endpoint", config.EvaluatorEndpoint).
		Str("environment", config.Environment).
		Bool("reconnect", config.Reconnect).
		Dur("heartbeat_interval", config.HeartbeatInterval).
		Msg("Streaming client created")

	return client, nil
}

// Start starts the streaming client
func (sc *StreamingClient) Start(ctx context.Context) error {
	sc.logger.Info().Msg("Starting streaming client")

	// Start connection goroutine
	go sc.run()

	sc.logger.Info().Msg("Streaming client started")
	return nil
}

// run is the main streaming loop
func (sc *StreamingClient) run() {
	defer close(sc.doneChan)

	for {
		select {
		case <-sc.stopChan:
			sc.logger.Info().Msg("Streaming client stopping")
			sc.disconnect()
			return
		default:
			if sc.shouldConnect() {
				sc.connect()
			}
			time.Sleep(time.Second) // Prevent tight loop
		}
	}
}

// shouldConnect determines if we should attempt to connect
func (sc *StreamingClient) shouldConnect() bool {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	// Don't connect if we're already connected or connecting
	if sc.status == StatusConnected || sc.status == StatusConnecting {
		return false
	}

	// Don't reconnect if disabled
	if !sc.config.Reconnect && sc.retryCount > 0 {
		return false
	}

	// Check retry limits
	if sc.maxRetries > 0 && sc.retryCount >= sc.maxRetries {
		sc.logger.Warn().
			Int("retry_count", sc.retryCount).
			Int("max_retries", sc.maxRetries).
			Msg("Maximum retries reached, stopping reconnection attempts")
		return false
	}

	return true
}

// connect establishes a connection to the streaming endpoint
func (sc *StreamingClient) connect() {
	sc.setStatus(StatusConnecting)

	sc.logger.Info().
		Int("retry_count", sc.retryCount).
		Dur("delay", sc.reconnectDelay).
		Msg("Connecting to streaming endpoint")

	// Apply reconnection delay
	if sc.retryCount > 0 {
		time.Sleep(sc.reconnectDelay)
	}

	// Create request
	url := fmt.Sprintf("%s/api/v1/stream?environment=%s", sc.config.EvaluatorEndpoint, sc.config.Environment)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		sc.handleConnectionError(fmt.Errorf("failed to create request: %w", err))
		return
	}

	// Set headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Authorization", "Bearer "+sc.config.APIKey)
	req.Header.Set("User-Agent", "feature-flags-go-sdk/1.0.0")

	// Make request
	client := &http.Client{
		Timeout: 0, // No timeout for streaming connections
	}

	resp, err := client.Do(req)
	if err != nil {
		sc.handleConnectionError(fmt.Errorf("connection failed: %w", err))
		return
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		sc.handleConnectionError(fmt.Errorf("server returned status %d", resp.StatusCode))
		return
	}

	// Connection successful
	sc.mutex.Lock()
	sc.conn = resp
	sc.reader = bufio.NewScanner(resp.Body)
	sc.connectTime = time.Now()
	sc.retryCount = 0
	sc.reconnectDelay = sc.config.InitialDelay
	sc.mutex.Unlock()

	sc.setStatus(StatusConnected)

	// Signal ready if this is the first connection
	select {
	case sc.readyChan <- struct{}{}:
	default:
	}

	sc.logger.Info().Msg("Connected to streaming endpoint")

	// Start reading events
	sc.readEvents()
}

// readEvents reads and processes events from the stream
func (sc *StreamingClient) readEvents() {
	defer sc.disconnect()

	sc.logger.Debug().Msg("Starting to read events")

	for sc.reader.Scan() {
		line := strings.TrimSpace(sc.reader.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Process event line
		if strings.HasPrefix(line, "data: ") {
			data := line[6:] // Remove "data: " prefix
			sc.processEvent(data)
		}
	}

	// Check for scanner error
	if err := sc.reader.Err(); err != nil && err != io.EOF {
		sc.logger.Error().
			Err(err).
			Msg("Error reading from stream")
		sc.errors++
	}
}

// processEvent processes a single event from the stream
func (sc *StreamingClient) processEvent(data string) {
	sc.mutex.Lock()
	sc.lastEventTime = time.Now()
	sc.eventsReceived++
	sc.mutex.Unlock()

	// Parse event
	var update ConfigUpdate
	if err := json.Unmarshal([]byte(data), &update); err != nil {
		sc.logger.Warn().
			Err(err).
			Str("data", data).
			Msg("Failed to parse event")
		return
	}

	sc.logger.Debug().
		Str("type", string(update.Type)).
		Str("flag_key", update.FlagKey).
		Int64("version", update.Version).
		Msg("Received config update")

	// Handle different event types
	switch update.Type {
	case UpdateTypeFlag:
		sc.handleFlagUpdate(update)
	case UpdateTypeSegment:
		sc.handleSegmentUpdate(update)
	case UpdateTypeEnvironment:
		sc.handleEnvironmentUpdate(update)
	case UpdateTypeHeartbeat:
		sc.handleHeartbeat(update)
	case UpdateTypeError:
		sc.handleErrorEvent(update)
	default:
		sc.logger.Warn().
			Str("type", string(update.Type)).
			Msg("Unknown event type")
	}
}

// handleFlagUpdate handles flag update events
func (sc *StreamingClient) handleFlagUpdate(update ConfigUpdate) {
	// Update cache
	if sc.config.Cache != nil && update.Flag != nil {
		// Invalidate existing cache entries for this flag
		cacheKeys := sc.config.Cache.Keys()
		for _, key := range cacheKeys {
			if strings.Contains(key, "flag:"+update.FlagKey+":") {
				sc.config.Cache.Delete(key)
			}
		}

		sc.logger.Debug().
			Str("flag_key", update.FlagKey).
			Msg("Invalidated cache entries for updated flag")
	}

	// Update offline configuration
	if sc.config.Offline != nil && update.Flag != nil {
		if err := sc.config.Offline.UpdateFlag(update.Flag); err != nil {
			sc.logger.Warn().
				Err(err).
				Str("flag_key", update.FlagKey).
				Msg("Failed to update offline configuration")
		}
	}
}

// handleSegmentUpdate handles segment update events
func (sc *StreamingClient) handleSegmentUpdate(update ConfigUpdate) {
	// Segments affect flag evaluation, so invalidate cache
	if sc.config.Cache != nil {
		sc.config.Cache.Clear()
		sc.logger.Debug().Msg("Cleared cache due to segment update")
	}
}

// handleEnvironmentUpdate handles environment update events
func (sc *StreamingClient) handleEnvironmentUpdate(update ConfigUpdate) {
	// Full environment update, clear all caches
	if sc.config.Cache != nil {
		sc.config.Cache.Clear()
		sc.logger.Debug().Msg("Cleared cache due to environment update")
	}
}

// handleHeartbeat handles heartbeat events
func (sc *StreamingClient) handleHeartbeat(update ConfigUpdate) {
	sc.logger.Debug().
		Time("timestamp", update.Timestamp).
		Msg("Received heartbeat")
}

// handleErrorEvent handles error events from the server
func (sc *StreamingClient) handleErrorEvent(update ConfigUpdate) {
	sc.logger.Warn().
		Time("timestamp", update.Timestamp).
		Msg("Received error event from server")
	sc.errors++
}

// handleConnectionError handles connection errors
func (sc *StreamingClient) handleConnectionError(err error) {
	sc.setStatus(StatusError)
	sc.errors++
	sc.retryCount++

	// Exponential backoff
	sc.reconnectDelay = sc.reconnectDelay * 2
	if sc.reconnectDelay > sc.config.MaxDelay {
		sc.reconnectDelay = sc.config.MaxDelay
	}

	sc.logger.Warn().
		Err(err).
		Int("retry_count", sc.retryCount).
		Dur("next_delay", sc.reconnectDelay).
		Msg("Connection error, will retry")
}

// disconnect closes the current connection
func (sc *StreamingClient) disconnect() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if sc.conn != nil {
		sc.conn.Body.Close()
		sc.conn = nil
		sc.reader = nil
	}

	if sc.status == StatusConnected {
		sc.setStatusLocked(StatusDisconnected)
		sc.reconnects++
		sc.logger.Info().Msg("Disconnected from streaming endpoint")
	}
}

// setStatus sets the connection status
func (sc *StreamingClient) setStatus(status StreamingStatus) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.setStatusLocked(status)
}

// setStatusLocked sets the connection status (assumes lock is held)
func (sc *StreamingClient) setStatusLocked(status StreamingStatus) {
	if sc.status != status {
		oldStatus := sc.status
		sc.status = status

		sc.logger.Debug().
			Str("old_status", string(oldStatus)).
			Str("new_status", string(status)).
			Msg("Streaming status changed")
	}
}

// GetStatus returns the current connection status
func (sc *StreamingClient) GetStatus() StreamingStatus {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	return sc.status
}

// IsConnected returns true if connected
func (sc *StreamingClient) IsConnected() bool {
	return sc.GetStatus() == StatusConnected
}

// WaitForReady waits for the streaming client to be ready
func (sc *StreamingClient) WaitForReady(ctx context.Context) error {
	// If already connected, return immediately
	if sc.IsConnected() {
		return nil
	}

	// Wait for ready signal or context cancellation
	select {
	case <-sc.readyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-sc.doneChan:
		return fmt.Errorf("streaming client closed")
	}
}

// GetStats returns streaming client statistics
func (sc *StreamingClient) GetStats() map[string]interface{} {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	stats := map[string]interface{}{
		"status":          string(sc.status),
		"events_received": sc.eventsReceived,
		"reconnects":      sc.reconnects,
		"errors":          sc.errors,
		"retry_count":     sc.retryCount,
		"reconnect_delay": sc.reconnectDelay,
		"last_event_time": sc.lastEventTime,
	}

	if !sc.connectTime.IsZero() {
		stats["connect_time"] = sc.connectTime
		stats["uptime"] = time.Since(sc.connectTime)
	}

	return stats
}

// IsHealthy returns true if the streaming client is healthy
func (sc *StreamingClient) IsHealthy() bool {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	// Consider healthy if:
	// 1. Connected, or
	// 2. Recently connected and attempting to reconnect
	if sc.status == StatusConnected {
		return true
	}

	if sc.status == StatusConnecting || sc.status == StatusReconnecting {
		return true
	}

	// Not healthy if too many errors or no recent events
	if sc.errors > 10 {
		return false
	}

	if !sc.lastEventTime.IsZero() && time.Since(sc.lastEventTime) > sc.config.HeartbeatInterval*3 {
		return false
	}

	return true
}

// Reconnect forces a reconnection
func (sc *StreamingClient) Reconnect() {
	sc.logger.Info().Msg("Forcing reconnection")

	sc.disconnect()
	sc.setStatus(StatusReconnecting)

	// Reset retry state
	sc.mutex.Lock()
	sc.retryCount = 0
	sc.reconnectDelay = sc.config.InitialDelay
	sc.mutex.Unlock()
}

// Close closes the streaming client
func (sc *StreamingClient) Close() {
	sc.logger.Info().Msg("Closing streaming client")

	// Signal stop
	close(sc.stopChan)

	// Wait for completion
	<-sc.doneChan

	stats := sc.GetStats()
	sc.logger.Info().
		Interface("stats", stats).
		Msg("Streaming client closed")
}

// UpdateConfig updates the streaming configuration
func (sc *StreamingClient) UpdateConfig(config *StreamingConfig) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// Update configuration
	oldEndpoint := sc.config.EvaluatorEndpoint
	oldEnvironment := sc.config.Environment

	sc.config = config

	// If endpoint or environment changed, force reconnection
	if config.EvaluatorEndpoint != oldEndpoint || config.Environment != oldEnvironment {
		sc.logger.Info().
			Str("old_endpoint", oldEndpoint).
			Str("new_endpoint", config.EvaluatorEndpoint).
			Str("old_environment", oldEnvironment).
			Str("new_environment", config.Environment).
			Msg("Streaming config changed, forcing reconnection")

		go sc.Reconnect()
	}
}

// GetLastEventTime returns the time of the last received event
func (sc *StreamingClient) GetLastEventTime() time.Time {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	return sc.lastEventTime
}

// GetUptime returns the connection uptime
func (sc *StreamingClient) GetUptime() time.Duration {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	if sc.connectTime.IsZero() || sc.status != StatusConnected {
		return 0
	}

	return time.Since(sc.connectTime)
}

// SetReconnectEnabled enables or disables automatic reconnection
func (sc *StreamingClient) SetReconnectEnabled(enabled bool) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.config.Reconnect = enabled

	sc.logger.Info().
		Bool("reconnect_enabled", enabled).
		Msg("Reconnection setting updated")
}
