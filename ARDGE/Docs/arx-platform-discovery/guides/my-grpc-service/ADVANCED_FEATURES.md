# gRPC 進階功能指南

基於 arx-platform-core 的架構，本指南展示如何添加進階功能。

## 1. Request ID 追蹤

### 1.1 Request ID 攔截器

**檔案：`internal/server/interceptors/request_id.go`**

```go
package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// requestIDKey 用於在 context 中存儲 request ID
type requestIDKey struct{}

// RequestIDUnaryInterceptor 為每個請求生成唯一的 request ID
// 這有助於追蹤跨系統的請求流程
func RequestIDUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 生成新的 request ID
		requestID := uuid.New().String()

		// 將 request ID 存儲在 context 中
		ctx = context.WithValue(ctx, requestIDKey{}, requestID)

		return handler(ctx, req)
	}
}

// RequestIDStreamInterceptor 為每個串流請求生成唯一的 request ID
func RequestIDStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// 生成新的 request ID
		requestID := uuid.New().String()

		// 將 request ID 存儲在 context 中
		ctx := context.WithValue(ss.Context(), requestIDKey{}, requestID)

		// 建立包裝的串流，使用新的 context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// GetRequestID 從 context 中取得 request ID
func GetRequestID(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDKey{}).(string)
	if !ok {
		return "unknown"
	}
	return requestID
}

// wrappedServerStream 包裝 grpc.ServerStream 以支援自定義 context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
```

### 1.2 在 Logging 中使用 Request ID

修改 `internal/server/interceptors/logging.go`：

```go
// LoggingUnaryInterceptor 修改版本，包含 request ID
func LoggingUnaryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		requestID := GetRequestID(ctx)  // 取得 request ID

		// 執行 RPC 方法
		resp, err := handler(ctx, req)

		// 計算執行時間
		duration := time.Since(start)

		// 記錄結果（包含 request ID）
		if err != nil {
			log.Error("RPC failed",
				"request_id", requestID,
				"method", info.FullMethod,
				"error", err.Error(),
				"duration_ms", duration.Milliseconds(),
			)
		} else {
			log.Info("RPC completed",
				"request_id", requestID,
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
			)
		}

		return resp, err
	}
}
```

### 1.3 更新伺服器設置

修改 `internal/server/grpc.go`：

```go
grpcServer := grpc.NewServer(
	grpc.ChainUnaryInterceptor(
		interceptors.RequestIDUnaryInterceptor(),  // 首先生成 request ID
		interceptors.RecoveryUnaryInterceptor(log),
		interceptors.LoggingUnaryInterceptor(log),
	),
	grpc.ChainStreamInterceptor(
		interceptors.RequestIDStreamInterceptor(),
		interceptors.RecoveryStreamInterceptor(log),
		interceptors.LoggingStreamInterceptor(log),
	),
)
```

## 2. 性能監控攔截器

**檔案：`internal/server/interceptors/performance.go`**

```go
package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"github.com/chslin/my-grpc-service/internal/logger"
)

// PerformanceUnaryInterceptor 監控慢請求（超過閾值）
// 預設閾值為 1000ms
func PerformanceUnaryInterceptor(log logger.Logger, slowThresholdMs int) grpc.UnaryServerInterceptor {
	thresholdDuration := time.Duration(slowThresholdMs) * time.Millisecond

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// 執行 RPC 方法
		resp, err := handler(ctx, req)

		duration := time.Since(start)

		// 如果執行時間超過閾值，記錄警告
		if duration > thresholdDuration {
			requestID := GetRequestID(ctx)
			log.Warn("RPC call exceeded slow threshold",
				"request_id", requestID,
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
				"threshold_ms", slowThresholdMs,
			)
		}

		return resp, err
	}
}

// PerformanceStreamInterceptor 監控串流 RPC 的性能
func PerformanceStreamInterceptor(log logger.Logger, slowThresholdMs int) grpc.StreamServerInterceptor {
	thresholdDuration := time.Duration(slowThresholdMs) * time.Millisecond

	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// 執行串流 RPC 方法
		err := handler(srv, ss)

		duration := time.Since(start)

		// 如果執行時間超過閾值，記錄警告
		if duration > thresholdDuration {
			ctx := ss.Context()
			requestID := GetRequestID(ctx)
			log.Warn("stream RPC call exceeded slow threshold",
				"request_id", requestID,
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
				"threshold_ms", slowThresholdMs,
			)
		}

		return err
	}
}
```

## 3. 配置管理

**檔案：`internal/config/config.go`**

```go
package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

// Config 包含整個應用程式的配置
type Config struct {
	GRPC GRPCConfig `toml:"grpc"`
	Log  LogConfig  `toml:"log"`
}

// GRPCConfig gRPC 伺服器配置
type GRPCConfig struct {
	Address           string `toml:"address"`
	SlowThresholdMs   int    `toml:"slow_threshold_ms"`
	MaxConnectionIdle int    `toml:"max_connection_idle"`
	MaxConnectionAge  int    `toml:"max_connection_age"`
}

// LogConfig 日誌配置
type LogConfig struct {
	Level  string `toml:"level"`  // debug, info, warn, error
	Format string `toml:"format"` // plain, json
}

// LoadConfig 從檔案載入配置
func LoadConfig(filePath string) (*Config, error) {
	var cfg Config

	// 設置預設值
	cfg.GRPC.Address = ":50051"
	cfg.GRPC.SlowThresholdMs = 1000
	cfg.Log.Level = "info"
	cfg.Log.Format = "plain"

	// 從檔案載入配置（會覆蓋預設值）
	if _, err := toml.DecodeFile(filePath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", filePath, err)
	}

	return &cfg, nil
}
```

修改 `configs/config.toml`：

```toml
[grpc]
address = ":50051"
slow_threshold_ms = 1000
max_connection_idle = 600    # 秒
max_connection_age = 3600    # 秒

[log]
level = "info"
format = "plain"
```

### 使用配置啟動伺服器

修改 `cmd/server/main.go`：

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chslin/my-grpc-service/internal/config"
	"github.com/chslin/my-grpc-service/internal/logger"
	"github.com/chslin/my-grpc-service/internal/server"
)

func main() {
	// 1. 載入配置
	cfg, err := config.LoadConfig("configs/config.toml")
	if err != nil {
		panic(err)
	}

	// 2. 初始化 Logger
	levelMap := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}

	logLevel := levelMap[cfg.Log.Level]
	log := logger.NewSlogAdapter(
		slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})),
	)

	// 3. 建立 gRPC 伺服器
	grpcServer, err := server.NewGRPCServer(cfg.GRPC.Address, log, cfg.GRPC.SlowThresholdMs)
	if err != nil {
		log.Error("failed to create gRPC server", "error", err.Error())
		os.Exit(1)
	}

	// 4. 非阻塞式啟動伺服器
	if err := grpcServer.StartAsync(); err != nil {
		log.Error("failed to start gRPC server", "error", err.Error())
		os.Exit(1)
	}

	// 5. 等待伺服器啟動完成
	readyCtx, readyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := grpcServer.WaitForReady(readyCtx); err != nil {
		readyCancel()
		log.Error("gRPC server failed to become ready", "error", err.Error())
		os.Exit(1)
	}
	readyCancel()

	log.Info("gRPC server started successfully", "address", cfg.GRPC.Address)

	// 6. 設置優雅關閉信號處理
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	<-ctx.Done()

	// 7. 優雅關閉伺服器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := grpcServer.Stop(shutdownCtx); err != nil {
		log.Error("error during shutdown", "error", err.Error())
		os.Exit(1)
	}

	log.Info("gRPC server stopped gracefully")
}
```

修改 `internal/server/grpc.go` 以支援配置：

```go
// NewGRPCServer 建立新的 gRPC 伺服器，接受配置參數
func NewGRPCServer(address string, log logger.Logger, slowThresholdMs int) (*GRPCServer, error) {
	// ... 現有代碼 ...

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RequestIDUnaryInterceptor(),
			interceptors.RecoveryUnaryInterceptor(log),
			interceptors.PerformanceUnaryInterceptor(log, slowThresholdMs),  // 添加性能監控
			interceptors.LoggingUnaryInterceptor(log),
		),
		grpc.ChainStreamInterceptor(
			interceptors.RequestIDStreamInterceptor(),
			interceptors.RecoveryStreamInterceptor(log),
			interceptors.PerformanceStreamInterceptor(log, slowThresholdMs),
			interceptors.LoggingStreamInterceptor(log),
		),
	)

	// ... 其餘代碼 ...
}
```

## 4. 健康檢查

**檔案：`internal/server/health.go`**

```go
package server

import (
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc"
)

// HealthChecker 封裝 gRPC 健康檢查服務
type HealthChecker struct {
	server *health.Server
}

// NewHealthChecker 建立新的健康檢查服務
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		server: health.NewServer(),
	}
}

// Register 在 gRPC 伺服器上註冊健康檢查
func (hc *HealthChecker) Register(grpcServer *grpc.Server) {
	grpc_health_v1.RegisterHealthServer(grpcServer, hc.server)
}

// SetServingStatus 設置服務的健康狀態
func (hc *HealthChecker) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	hc.server.SetServingStatus(service, status)
}

// SetNotServing 設置服務為不可用
func (hc *HealthChecker) SetNotServing(service string) {
	hc.server.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
}
```

修改 `internal/server/grpc.go` 以包含健康檢查：

```go
type GRPCServer struct {
	grpcServer    *grpc.Server
	listener      net.Listener
	address       string
	log           logger.Logger
	healthChecker *HealthChecker  // 添加
}

func NewGRPCServer(address string, log logger.Logger, slowThresholdMs int) (*GRPCServer, error) {
	// ... 現有代碼 ...

	// 建立健康檢查服務
	healthChecker := NewHealthChecker()

	grpcServer := grpc.NewServer(
		// ... 攔截器設置 ...
	)

	// 註冊健康檢查
	healthChecker.Register(grpcServer)

	// 建立並註冊服務
	greetingService := services.NewGreetingService()
	v1.RegisterGreetingServiceServer(grpcServer, greetingService)

	// 設置 Greeting 服務的健康狀態
	healthChecker.SetServingStatus("hello.v1.GreetingService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &GRPCServer{
		grpcServer:    grpcServer,
		listener:      listener,
		address:       address,
		log:           log,
		healthChecker: healthChecker,
	}, nil
}

// GetHealthChecker 取得健康檢查器
func (s *GRPCServer) GetHealthChecker() *HealthChecker {
	return s.healthChecker
}
```

## 5. 連接池（用於後端服務）

**檔案：`internal/client/connector.go`**

```go
package client

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Connector 定義連接建立的介面
type Connector interface {
	Dial(ctx context.Context) (*grpc.ClientConn, error)
	Close() error
}

// SimpleConnector 簡單連接器
type SimpleConnector struct {
	target string
}

// NewSimpleConnector 建立簡單連接器
func NewSimpleConnector(target string) *SimpleConnector {
	return &SimpleConnector{target: target}
}

// Dial 建立 gRPC 連接
func (c *SimpleConnector) Dial(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		ctx,
		c.target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// Close 無需做任何事（單個連接不需要關閉）
func (c *SimpleConnector) Close() error {
	return nil
}

// Pool 連接池
type Pool struct {
	connector   Connector
	poolSize    int
	connections chan *grpc.ClientConn
	mu          sync.Mutex
	closed      bool
}

// NewPool 建立新的連接池
func NewPool(connector Connector, poolSize int) (*Pool, error) {
	if poolSize <= 0 {
		return nil, fmt.Errorf("pool size must be greater than 0")
	}

	pool := &Pool{
		connector:   connector,
		poolSize:    poolSize,
		connections: make(chan *grpc.ClientConn, poolSize),
	}

	// 預建立連接
	ctx := context.Background()
	for i := 0; i < poolSize; i++ {
		conn, err := connector.Dial(ctx)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection: %w", err)
		}
		pool.connections <- conn
	}

	return pool, nil
}

// Acquire 取得池中的連接
func (p *Pool) Acquire(ctx context.Context) (*grpc.ClientConn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.Unlock()

	select {
	case conn := <-p.connections:
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release 將連接歸還到池中
func (p *Pool) Release(conn *grpc.ClientConn) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return conn.Close()
	}
	p.mu.Unlock()

	select {
	case p.connections <- conn:
		return nil
	default:
		// 池已滿，關閉連接
		return conn.Close()
	}
}

// Close 關閉池及所有連接
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.connections)

	// 關閉所有連接
	for conn := range p.connections {
		conn.Close()
	}

	return nil
}

// PooledConnector 池化連接器的包裝
type PooledConnector struct {
	pool *Pool
}

// NewPooledConnector 建立池化連接器
func NewPooledConnector(connector Connector, poolSize int) (*PooledConnector, error) {
	pool, err := NewPool(connector, poolSize)
	if err != nil {
		return nil, err
	}

	return &PooledConnector{pool: pool}, nil
}

// WithConnection 在池中取得連接並執行函式
func (pc *PooledConnector) WithConnection(ctx context.Context, fn func(*grpc.ClientConn) error) error {
	conn, err := pc.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer pc.pool.Release(conn)

	return fn(conn)
}

// Close 關閉連接池
func (pc *PooledConnector) Close() error {
	return pc.pool.Close()
}
```

### 使用連接池的服務示例

**檔案：`internal/services/backend.go`**

```go
package services

import (
	"context"

	"github.com/chslin/my-grpc-service/internal/client"
	"github.com/chslin/my-grpc-service/internal/logger"
)

// BackendService 使用連接池與後端服務通訊
type BackendService struct {
	pooledConnector *client.PooledConnector
	log             logger.Logger
}

// NewBackendService 建立新的後端服務
func NewBackendService(pooledConnector *client.PooledConnector, log logger.Logger) *BackendService {
	return &BackendService{
		pooledConnector: pooledConnector,
		log:             log,
	}
}

// CallRemoteService 透過連接池呼叫遠端服務
func (bs *BackendService) CallRemoteService(ctx context.Context) error {
	return bs.pooledConnector.WithConnection(ctx, func(conn interface{}) error {
		// 在此處使用連接
		bs.log.Info("calling remote service via pooled connection")
		return nil
	})
}
```

## 6. 使用 cmux 進行多路復用

**檔案：`internal/server/cmux_setup.go`**

```go
package server

import (
	"net"

	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"github.com/chslin/my-grpc-service/internal/logger"
)

// SetupCMux 設置 cmux 在同一端口處理 gRPC 和 HTTP 流量
// gRPC 使用 HTTP/2，HTTP 使用 HTTP/1.x
func SetupCMux(address string, log logger.Logger) (net.Listener, cmux.CMux) {
	// 建立 TCP 監聽器
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Error("failed to listen on port", "address", address, "error", err.Error())
		panic(err)
	}

	// 建立 cmux
	m := cmux.New(listener)

	return listener, m
}

// GetGRPCListener 取得 gRPC 專用監聽器（HTTP/2）
func GetGRPCListener(m cmux.CMux) net.Listener {
	return m.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type"),
	)
}

// GetHTTPListener 取得 HTTP 專用監聽器（HTTP/1.x）
func GetHTTPListener(m cmux.CMux) net.Listener {
	return m.Match(cmux.Any())
}
```

修改 `cmd/server/main.go` 以支援 cmux：

```go
// ... 現有導入 ...

import (
	// ... 現有導入 ...
	"net/http"
	"net/http/pprof"

	"github.com/chslin/my-grpc-service/internal/server"
)

func main() {
	// ... 現有代碼 ...

	// 使用 cmux 設置多路復用
	listener, m := server.SetupCMux(cfg.GRPC.Address, log)

	// 建立 gRPC 伺服器
	grpcServer, err := server.NewGRPCServer(cfg.GRPC.Address, log, cfg.GRPC.SlowThresholdMs)
	if err != nil {
		log.Error("failed to create gRPC server", "error", err.Error())
		os.Exit(1)
	}

	// 取得 gRPC 監聽器
	grpcListener := server.GetGRPCListener(m)

	// 在背景啟動 gRPC 伺服器
	go func() {
		if err := grpcServer.grpcServer.Serve(grpcListener); err != nil {
			log.Error("gRPC server error", "error", err.Error())
		}
	}()

	// 設置 HTTP pprof 伺服器（用於性能分析）
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 註冊 pprof 端點
	httpMux.HandleFunc("/debug/pprof/", pprof.Index)
	httpMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	httpMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	httpMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	httpMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	httpListener := server.GetHTTPListener(m)
	httpServer := &http.Server{
		Handler: httpMux,
	}

	// 在背景啟動 HTTP 伺服器
	go func() {
		if err := httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err.Error())
		}
	}()

	// 啟動 cmux
	go func() {
		if err := m.Serve(); err != nil {
			log.Error("cmux error", "error", err.Error())
		}
	}()

	log.Info("gRPC and HTTP servers started",
		"address", cfg.GRPC.Address,
		"pprof_endpoint", "http://"+cfg.GRPC.Address+"/debug/pprof/",
	)

	// ... 優雅關閉邏輯 ...
}
```

## 7. 單元測試示例

**檔案：`internal/services/greeting_test.go`**

```go
package services_test

import (
	"context"
	"testing"

	"github.com/chslin/my-grpc-service/internal/services"
	v1 "github.com/chslin/my-grpc-service/api/hello/v1"
)

// TestGreet_Success 測試成功情況
func TestGreet_Success(t *testing.T) {
	service := services.NewGreetingService()

	resp, err := service.Greet(context.Background(), &v1.GreetRequest{Name: "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got '%s'", resp.Message)
	}
}

// TestGreet_EmptyName 測試空名字
func TestGreet_EmptyName(t *testing.T) {
	service := services.NewGreetingService()

	_, err := service.Greet(context.Background(), &v1.GreetRequest{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

// TestGreet_Multiple 表格驅動測試
func TestGreet_Multiple(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"alice", "Alice", "Hello, Alice!", false},
		{"bob", "Bob", "Hello, Bob!", false},
		{"empty", "", "", true},
	}

	service := services.NewGreetingService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.Greet(context.Background(), &v1.GreetRequest{Name: tt.input})

			if (err != nil) != tt.wantErr {
				t.Errorf("Greet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resp.Message != tt.want {
				t.Errorf("Greet() = %v, want %v", resp.Message, tt.want)
			}
		})
	}
}
```

## 8. 更新 Makefile

```makefile
# ... 現有內容 ...

# 執行單元測試
test:
	go test -v ./...

# 執行整合測試
test-integration:
	go test -v -tags=integration ./...

# 執行測試並生成覆蓋率
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 性能測試
test-bench:
	go test -v -bench=. -benchmem ./...

# 執行 linter
lint:
	golangci-lint run ./...

# 代碼格式化
fmt:
	go fmt ./...
	goimports -w .

# 檢查代碼質量
check: lint test
	@echo "All checks passed!"
```

## 9. 更新 go.mod

```
require (
	github.com/BurntSushi/toml v1.5.0
	github.com/google/uuid v1.6.0
	github.com/soheilhy/cmux v0.1.5
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.10
)
```

## 10. 完整伺服器設置檢查清單

- [ ] 初始化 Go 模組
- [ ] 建立 proto 定義
- [ ] 生成 gRPC 代碼
- [ ] 實現 Logger 介面
- [ ] 實現基本攔截器（Recovery、Logging）
- [ ] 實現服務業務邏輯
- [ ] 建立 gRPC 伺服器
- [ ] 建立主程式入口點
- [ ] 建立測試客戶端
- [ ] 添加 Request ID 攔截器
- [ ] 添加性能監控攔截器
- [ ] 實現配置管理
- [ ] 添加健康檢查
- [ ] 實現連接池（如需要）
- [ ] 設置 cmux 多路復用（如需要）
- [ ] 編寫單元測試
- [ ] 編寫集成測試
- [ ] 設置 CI/CD 管線
