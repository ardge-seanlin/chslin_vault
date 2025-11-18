# gRPC 伺服器完整設置指南

基於 arx-platform-core 的架構，從零開始建立 gRPC 伺服器。

## 目錄結構

```
my-grpc-service/
├── api/
│   └── hello/
│       └── v1/
│           ├── hello.pb.go
│           ├── hello_grpc.pb.go
│           └── hello.proto
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── client/
│       └── main.go
├── internal/
│   ├── logger/
│   │   └── logger.go
│   ├── server/
│   │   ├── grpc.go
│   │   └── interceptors/
│   │       ├── logging.go
│   │       └── recovery.go
│   └── services/
│       └── greeting.go
├── configs/
│   └── config.toml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 步驟 1：初始化專案

```bash
# 建立專案目錄
mkdir my-grpc-service
cd my-grpc-service

# 初始化 Git 和 Go 模組
git init
go mod init github.com/chslin/my-grpc-service

# 下載核心依賴
go get google.golang.org/grpc@v1.77.0
go get google.golang.org/protobuf@v1.36.10
```

## 步驟 2：建立 Proto 定義

**檔案：`api/hello/v1/hello.proto`**

```protobuf
syntax = "proto3";

package hello.v1;

option go_package = "github.com/chslin/my-grpc-service/api/hello/v1";

service GreetingService {
  rpc Greet(GreetRequest) returns (GreetResponse);
  rpc GreetStream(GreetRequest) returns (stream GreetResponse);
}

message GreetRequest {
  string name = 1;
}

message GreetResponse {
  string message = 1;
}
```

## 步驟 3：生成 gRPC 程式碼

```bash
# 方案 A：使用 protoc 直接生成（簡單）
protoc \
  --go_out=api \
  --go-grpc_out=api \
  api/hello/v1/hello.proto

# 方案 B：使用 Buf（推薦，與 arx-platform-core 相同）
# 先安裝 buf：brew install buf
# 建立 buf.yaml 和 buf.gen.yaml（見下方）
# 執行：buf generate
```

**檔案：`buf.yaml`**

```yaml
version: v2

modules:
  - path: api
```

**檔案：`buf.gen.yaml`**

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go:v1.36.10
    out: api
    opt: paths=source_relative

  - remote: buf.build/grpc/go:v1.5.1
    out: api
    opt: paths=source_relative
```

## 步驟 4：Logger 介面實現

**檔案：`internal/logger/logger.go`**

```go
package logger

import (
	"log/slog"
)

// Logger 定義日誌記錄介面（與 arx-platform-core 相同）
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// SlogAdapter 實現 Logger 介面，使用標準 log/slog
type SlogAdapter struct {
	log *slog.Logger
}

// NewSlogAdapter 建立新的 slog 適配器
func NewSlogAdapter(log *slog.Logger) *SlogAdapter {
	return &SlogAdapter{log: log}
}

// Info 記錄資訊等級日誌
func (s *SlogAdapter) Info(msg string, args ...interface{}) {
	s.log.Info(msg, args...)
}

// Warn 記錄警告等級日誌
func (s *SlogAdapter) Warn(msg string, args ...interface{}) {
	s.log.Warn(msg, args...)
}

// Error 記錄錯誤等級日誌
func (s *SlogAdapter) Error(msg string, args ...interface{}) {
	s.log.Error(msg, args...)
}

// Debug 記錄除錯等級日誌
func (s *SlogAdapter) Debug(msg string, args ...interface{}) {
	s.log.Debug(msg, args...)
}
```

## 步驟 5：攔截器實現

### 5.1 Recovery 攔截器

**檔案：`internal/server/interceptors/recovery.go`**

```go
package interceptors

import (
	"context"
	"fmt"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/chslin/my-grpc-service/internal/logger"
)

// RecoveryUnaryInterceptor 捕獲 panic 並轉換為 gRPC 錯誤
// 確保伺服器不會因為單個請求中的 panic 而當機
func RecoveryUnaryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (_ interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"method", info.FullMethod,
					"stack", string(debug.Stack()),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// RecoveryStreamInterceptor 為串流 RPC 捕獲 panic
func RecoveryStreamInterceptor(log logger.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered in stream",
					"panic", fmt.Sprintf("%v", r),
					"method", info.FullMethod,
					"stack", string(debug.Stack()),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(srv, ss)
	}
}
```

### 5.2 Logging 攔截器

**檔案：`internal/server/interceptors/logging.go`**

```go
package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"github.com/chslin/my-grpc-service/internal/logger"
)

// LoggingUnaryInterceptor 記錄所有 Unary RPC 呼叫的詳細資訊
// 包括方法名、執行時間和是否發生錯誤
func LoggingUnaryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// 執行 RPC 方法
		resp, err := handler(ctx, req)

		// 計算執行時間
		duration := time.Since(start)

		// 記錄結果
		if err != nil {
			log.Error("RPC failed",
				"method", info.FullMethod,
				"error", err.Error(),
				"duration_ms", duration.Milliseconds(),
			)
		} else {
			log.Info("RPC completed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
			)
		}

		return resp, err
	}
}

// LoggingStreamInterceptor 記錄所有串流 RPC 呼叫的詳細資訊
func LoggingStreamInterceptor(log logger.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// 執行串流 RPC 方法
		err := handler(srv, ss)

		// 計算執行時間
		duration := time.Since(start)

		// 記錄結果
		if err != nil {
			log.Error("stream RPC failed",
				"method", info.FullMethod,
				"error", err.Error(),
				"duration_ms", duration.Milliseconds(),
			)
		} else {
			log.Info("stream RPC completed",
				"method", info.FullMethod,
				"duration_ms", duration.Milliseconds(),
			)
		}

		return err
	}
}
```

## 步驟 6：服務實現

**檔案：`internal/services/greeting.go`**

```go
package services

import (
	"context"
	"fmt"

	v1 "github.com/chslin/my-grpc-service/api/hello/v1"
)

// GreetingService 實現 hello.v1.GreetingService
type GreetingService struct {
	v1.UnimplementedGreetingServiceServer
}

// NewGreetingService 建立新的 GreetingService 實例
func NewGreetingService() *GreetingService {
	return &GreetingService{}
}

// Greet 實現 Unary RPC：接收一個名字，返回問候訊息
func (s *GreetingService) Greet(ctx context.Context, req *v1.GreetRequest) (*v1.GreetResponse, error) {
	// 驗證輸入
	if req.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	return &v1.GreetResponse{
		Message: fmt.Sprintf("Hello, %s!", req.Name),
	}, nil
}

// GreetStream 實現串流 RPC：接收一個名字，持續返回問候訊息
func (s *GreetingService) GreetStream(req *v1.GreetRequest, stream v1.GreetingService_GreetStreamServer) error {
	// 驗證輸入
	if req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// 發送多個響應
	messages := []string{
		fmt.Sprintf("Hello, %s!", req.Name),
		fmt.Sprintf("Welcome, %s!", req.Name),
		fmt.Sprintf("Goodbye, %s!", req.Name),
	}

	for _, msg := range messages {
		if err := stream.Send(&v1.GreetResponse{Message: msg}); err != nil {
			return err
		}
	}

	return nil
}
```

## 步驟 7：gRPC 伺服器設置

**檔案：`internal/server/grpc.go`**

```go
package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"github.com/chslin/my-grpc-service/internal/logger"
	"github.com/chslin/my-grpc-service/internal/server/interceptors"
	"github.com/chslin/my-grpc-service/internal/services"
	v1 "github.com/chslin/my-grpc-service/api/hello/v1"
)

// GRPCServer 封裝 gRPC 伺服器的啟動、執行和關閉邏輯
type GRPCServer struct {
	grpcServer *grpc.Server
	listener   net.Listener
	address    string
	log        logger.Logger
}

// NewGRPCServer 建立新的 gRPC 伺服器
// 設置攔截器鏈並註冊所有服務
func NewGRPCServer(address string, log logger.Logger) (*GRPCServer, error) {
	// 1. 建立 TCP 監聽器
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	// 2. 建立 gRPC 伺服器並配置攔截器鏈
	// 攔截器按順序執行：Recovery → Logging
	// 這樣可確保 panic 被捕獲，所有呼叫都被記錄
	grpcServer := grpc.NewServer(
		// Unary RPC 攔截器鏈
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryUnaryInterceptor(log),
			interceptors.LoggingUnaryInterceptor(log),
		),
		// 串流 RPC 攔截器鏈
		grpc.ChainStreamInterceptor(
			interceptors.RecoveryStreamInterceptor(log),
			interceptors.LoggingStreamInterceptor(log),
		),
	)

	// 3. 建立並註冊服務
	greetingService := services.NewGreetingService()
	v1.RegisterGreetingServiceServer(grpcServer, greetingService)

	return &GRPCServer{
		grpcServer: grpcServer,
		listener:   listener,
		address:    address,
		log:        log,
	}, nil
}

// Start 阻塞式啟動伺服器
// 此方法會阻塞直到伺服器停止或發生錯誤
func (s *GRPCServer) Start() error {
	s.log.Info("Starting gRPC server", "address", s.address)
	return s.grpcServer.Serve(s.listener)
}

// StartAsync 非阻塞式啟動伺服器
// 在背景 goroutine 中運行伺服器
func (s *GRPCServer) StartAsync() error {
	go func() {
		if err := s.Start(); err != nil {
			s.log.Error("gRPC server error", "error", err.Error())
		}
	}()
	return nil
}

// Stop 優雅關閉伺服器
// 在給定的上下文超時內關閉伺服器，如果超時則強制關閉
func (s *GRPCServer) Stop(ctx context.Context) error {
	s.log.Info("Stopping gRPC server")

	done := make(chan error, 1)
	go func() {
		s.grpcServer.GracefulStop()
		done <- nil
	}()

	select {
	case <-ctx.Done():
		s.log.Warn("graceful shutdown timeout, forcing stop")
		s.grpcServer.Stop()  // 強制關閉
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// WaitForReady 等待伺服器完全啟動
// 持續嘗試連接到伺服器，直到成功或超時
func (s *GRPCServer) WaitForReady(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := net.Dial("tcp", s.address)
			if err == nil {
				conn.Close()
				s.log.Info("gRPC server is ready")
				return nil
			}
		}
	}
}
```

## 步驟 8：主程式

**檔案：`cmd/server/main.go`**

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chslin/my-grpc-service/internal/logger"
	"github.com/chslin/my-grpc-service/internal/server"
)

func main() {
	// 1. 初始化 Logger
	// 使用標準 log/slog，輸出到標準輸出
	log := logger.NewSlogAdapter(
		slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	)

	// 2. 建立 gRPC 伺服器
	grpcServer, err := server.NewGRPCServer(":50051", log)
	if err != nil {
		log.Error("failed to create gRPC server", "error", err.Error())
		os.Exit(1)
	}

	// 3. 非阻塞式啟動伺服器（在背景 goroutine 中運行）
	if err := grpcServer.StartAsync(); err != nil {
		log.Error("failed to start gRPC server", "error", err.Error())
		os.Exit(1)
	}

	// 4. 等待伺服器啟動完成（最多等待 5 秒）
	readyCtx, readyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := grpcServer.WaitForReady(readyCtx); err != nil {
		readyCancel()
		log.Error("gRPC server failed to become ready", "error", err.Error())
		os.Exit(1)
	}
	readyCancel()

	log.Info("gRPC server started successfully", "address", ":50051")

	// 5. 設置優雅關閉信號處理
	// 監聽 SIGINT (Ctrl+C) 和 SIGTERM 信號
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	<-ctx.Done()  // 等待中止信號

	// 6. 優雅關閉伺服器（30 秒超時）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := grpcServer.Stop(shutdownCtx); err != nil {
		log.Error("error during shutdown", "error", err.Error())
		os.Exit(1)
	}

	log.Info("gRPC server stopped gracefully")
}
```

## 步驟 9：測試客戶端

**檔案：`cmd/client/main.go`**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "github.com/chslin/my-grpc-service/api/hello/v1"
)

func main() {
	// 1. 連接到伺服器
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// 2. 建立客戶端
	client := v1.NewGreetingServiceClient(conn)

	// 3. 測試 Unary RPC
	fmt.Println("=== Testing Unary RPC ===")
	testUnaryRPC(client)

	// 4. 測試串流 RPC
	fmt.Println("\n=== Testing Streaming RPC ===")
	testStreamingRPC(client)
}

// testUnaryRPC 測試 Unary RPC 呼叫
func testUnaryRPC(client v1.GreetingServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.Greet(ctx, &v1.GreetRequest{Name: "World"})
	if err != nil {
		log.Printf("Unary RPC failed: %v", err)
		return
	}

	fmt.Printf("Unary Response: %s\n", resp.Message)
}

// testStreamingRPC 測試串流 RPC 呼叫
func testStreamingRPC(client v1.GreetingServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.GreetStream(ctx, &v1.GreetRequest{Name: "Alice"})
	if err != nil {
		log.Printf("Stream RPC failed: %v", err)
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		fmt.Printf("Stream Response: %s\n", resp.Message)
	}
}
```

## 步驟 10：Makefile

**檔案：`Makefile`**

```makefile
.PHONY: help generate build run-server run-client test clean

help:
	@echo "Available commands:"
	@echo "  make generate   - Generate gRPC code from proto files"
	@echo "  make build      - Build server and client binaries"
	@echo "  make run-server - Run gRPC server"
	@echo "  make run-client - Run gRPC client (requires server running)"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean generated files"

# 安裝 protoc 編譯工具
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 使用 protoc 生成 gRPC 程式碼
generate:
	protoc \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		api/hello/v1/hello.proto

# 或使用 buf 生成（需先安裝 buf：brew install buf）
generate-buf:
	buf generate

# 下載依賴
deps:
	go mod download
	go mod tidy

# 建構伺服器
build-server:
	go build -o bin/server ./cmd/server

# 建構客戶端
build-client:
	go build -o bin/client ./cmd/client

# 建構所有二進制文件
build: build-server build-client
	@echo "Build complete: bin/server and bin/client"

# 執行伺服器
run-server: build-server
	./bin/server

# 執行客戶端
run-client: build-client
	./bin/client

# 執行所有測試
test:
	go test -v ./...

# 執行測試並生成覆蓋率報告
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 清理生成的檔案
clean:
	rm -rf bin/
	find . -name "*.pb.go" -delete
	find . -name "coverage.*" -delete

# 執行 linter
lint:
	golangci-lint run ./...

# 格式化程式碼
fmt:
	go fmt ./...
	goimports -w .

# 完整開發流程
dev: deps generate build-server
	@echo "Ready for development!"
```

## 步驟 11：配置檔案

**檔案：`configs/config.toml`**

```toml
# gRPC 伺服器配置
[grpc]
address = ":50051"

# 日誌配置
[log]
level = "info"     # 日誌等級: debug, info, warn, error
format = "plain"   # 日誌格式: plain 或 json
```

## 步驟 12：Go Module 配置

**檔案：`go.mod`**

```
module github.com/chslin/my-grpc-service

go 1.21

require (
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da29e248c // indirect
)
```

## 快速開始

### 1. 生成 gRPC 代碼

```bash
make generate
# 或使用 buf
make generate-buf
```

### 2. 啟動伺服器

```bash
make run-server
# 輸出應該顯示：
# time=2024-01-01T12:00:00.000Z level=INFO msg="gRPC server started successfully" address=:50051
```

### 3. 在另一個終端運行客戶端

```bash
make run-client
# 輸出應該顯示：
# === Testing Unary RPC ===
# Unary Response: Hello, World!
#
# === Testing Streaming RPC ===
# Stream Response: Hello, Alice!
# Stream Response: Welcome, Alice!
# Stream Response: Goodbye, Alice!
```

### 4. 停止伺服器

```bash
# 在伺服器終端按 Ctrl+C
# 輸出應該顯示：
# time=2024-01-01T12:00:05.000Z level=INFO msg="Stopping gRPC server"
# time=2024-01-01T12:00:05.000Z level=INFO msg="gRPC server stopped gracefully"
```

## 進階功能

### 添加健康檢查

```bash
go get google.golang.org/grpc/health
```

然後修改 `internal/server/grpc.go`：

```go
import (
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
)

func NewGRPCServer(address string, log logger.Logger) (*GRPCServer, error) {
    // ... 現有代碼 ...

    // 註冊健康檢查服務
    healthServer := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

    // 設置初始狀態
    healthServer.SetServingStatus("hello.v1.GreetingService", grpc_health_v1.HealthCheckResponse_SERVING)

    return &GRPCServer{
        grpcServer: grpcServer,
        listener:   listener,
        address:    address,
        log:        log,
    }, nil
}
```

### 測試健康檢查

```bash
# 使用 grpcurl
brew install grpcurl

# 檢查服務健康狀態
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

## 與 arx-platform-core 的對應

| 功能 | my-grpc-service | arx-platform-core |
|------|-----------------|-------------------|
| Proto 定義 | `api/hello/v1/hello.proto` | `buf.gen.yaml` + 外部 repo |
| 服務實現 | `internal/services/greeting.go` | `internal/services/node/system.go` |
| 攔截器 | `internal/server/interceptors/` | `internal/server/interceptors/` |
| 伺服器設置 | `internal/server/grpc.go` | `internal/server/grpc_setup.go` |
| 主程式 | `cmd/server/main.go` | `cmd/server/main.go` |
| 配置管理 | `configs/config.toml` | `configs/config.toml` |

## 常見問題

### Q: 如何添加更多服務？

在 `internal/services/` 目錄下建立新的服務檔案，然後在 `internal/server/grpc.go` 的 `NewGRPCServer` 中註冊：

```go
// 建立新服務
newService := services.NewMyService()

// 註冊服務
v1.RegisterMyServiceServer(grpcServer, newService)
```

### Q: 如何添加自定義攔截器？

在 `internal/server/interceptors/` 目錄下建立新檔案，實現 `grpc.UnaryServerInterceptor` 或 `grpc.StreamServerInterceptor` 介面，然後在 `NewGRPCServer` 中加入攔截器鏈。

### Q: 如何進行單元測試？

參考 `cmd/client/main.go` 的模式，建立測試檔案如 `internal/services/greeting_test.go`：

```go
package services_test

import (
    "context"
    "testing"

    "github.com/chslin/my-grpc-service/internal/services"
    v1 "github.com/chslin/my-grpc-service/api/hello/v1"
)

func TestGreet(t *testing.T) {
    service := services.NewGreetingService()

    resp, err := service.Greet(context.Background(), &v1.GreetRequest{Name: "Test"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if resp.Message != "Hello, Test!" {
        t.Errorf("expected 'Hello, Test!', got '%s'", resp.Message)
    }
}
```

## 延伸閱讀

- [gRPC Go 官方文檔](https://grpc.io/docs/languages/go/)
- [Protocol Buffers 官方文檔](https://developers.google.com/protocol-buffers)
- [Uber Go Style Guide](https://github.com/uber-go/guide)
- [Effective Go](https://go.dev/doc/effective_go)
