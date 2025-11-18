# 完整檔案範本集合

這個文檔包含了所有必需檔案的完整代碼，可直接複製使用。

## 檔案清單

1. `go.mod` - Go 模組定義
2. `api/hello/v1/hello.proto` - Proto 定義
3. `buf.yaml` - Buf 配置
4. `buf.gen.yaml` - Buf 生成配置
5. `internal/logger/logger.go` - Logger 介面實現
6. `internal/server/interceptors/recovery.go` - Recovery 攔截器
7. `internal/server/interceptors/logging.go` - Logging 攔截器
8. `internal/services/greeting.go` - 服務實現
9. `internal/server/grpc.go` - gRPC 伺服器設置
10. `cmd/server/main.go` - 伺服器主程式
11. `cmd/client/main.go` - 客戶端測試程式
12. `Makefile` - 開發工具命令
13. `configs/config.toml` - 配置檔案

---

## 1. go.mod

```go
module github.com/chslin/my-grpc-service

go 1.21

require (
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/golang/protobuf v1.5.4
	golang.org/x/net v0.33.0
	golang.org/x/sys v0.27.0
	golang.org/x/text v0.21.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da29e248c
)
```

---

## 2. api/hello/v1/hello.proto

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

---

## 3. buf.yaml

```yaml
version: v2

modules:
  - path: api
```

---

## 4. buf.gen.yaml

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

---

## 5. internal/logger/logger.go

```go
package logger

import (
	"log/slog"
)

// Logger 定義日誌記錄介面
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

---

## 6. internal/server/interceptors/recovery.go

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

---

## 7. internal/server/interceptors/logging.go

```go
package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"github.com/chslin/my-grpc-service/internal/logger"
)

// LoggingUnaryInterceptor 記錄所有 Unary RPC 呼叫的詳細資訊
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

---

## 8. internal/services/greeting.go

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

---

## 9. internal/server/grpc.go

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
func NewGRPCServer(address string, log logger.Logger) (*GRPCServer, error) {
	// 1. 建立 TCP 監聽器
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	// 2. 建立 gRPC 伺服器並配置攔截器鏈
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
func (s *GRPCServer) Start() error {
	s.log.Info("Starting gRPC server", "address", s.address)
	return s.grpcServer.Serve(s.listener)
}

// StartAsync 非阻塞式啟動伺服器
func (s *GRPCServer) StartAsync() error {
	go func() {
		if err := s.Start(); err != nil {
			s.log.Error("gRPC server error", "error", err.Error())
		}
	}()
	return nil
}

// Stop 優雅關閉伺服器
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
		s.grpcServer.Stop()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// WaitForReady 等待伺服器完全啟動
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

---

## 10. cmd/server/main.go

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

	// 3. 非阻塞式啟動伺服器
	if err := grpcServer.StartAsync(); err != nil {
		log.Error("failed to start gRPC server", "error", err.Error())
		os.Exit(1)
	}

	// 4. 等待伺服器啟動完成
	readyCtx, readyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := grpcServer.WaitForReady(readyCtx); err != nil {
		readyCancel()
		log.Error("gRPC server failed to become ready", "error", err.Error())
		os.Exit(1)
	}
	readyCancel()

	log.Info("gRPC server started successfully", "address", ":50051")

	// 5. 設置優雅關閉信號處理
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	<-ctx.Done()

	// 6. 優雅關閉伺服器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := grpcServer.Stop(shutdownCtx); err != nil {
		log.Error("error during shutdown", "error", err.Error())
		os.Exit(1)
	}

	log.Info("gRPC server stopped gracefully")
}
```

---

## 11. cmd/client/main.go

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

---

## 12. Makefile

```makefile
.PHONY: help generate build run-server run-client test clean install-tools

help:
	@echo "Available commands:"
	@echo "  make generate       - Generate gRPC code from proto files"
	@echo "  make build          - Build server and client binaries"
	@echo "  make run-server     - Run gRPC server"
	@echo "  make run-client     - Run gRPC client (requires server running)"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean generated files"
	@echo "  make install-tools  - Install required tools"

install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

generate:
	protoc \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		api/hello/v1/hello.proto

deps:
	go mod download
	go mod tidy

build-server:
	mkdir -p bin
	go build -o bin/server ./cmd/server

build-client:
	mkdir -p bin
	go build -o bin/client ./cmd/client

build: build-server build-client
	@echo "Build complete: bin/server and bin/client"

run-server: build-server
	./bin/server

run-client: build-client
	./bin/client

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	find . -name "*.pb.go" -delete
	find . -name "coverage.*" -delete

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	goimports -w .
```

---

## 13. configs/config.toml

```toml
# gRPC 伺服器配置
[grpc]
address = ":50051"
slow_threshold_ms = 1000

# 日誌配置
[log]
level = "info"
format = "plain"
```

---

## 目錄建立命令

```bash
# 建立所有必需的目錄
mkdir -p api/hello/v1
mkdir -p cmd/server
mkdir -p cmd/client
mkdir -p internal/logger
mkdir -p internal/server/interceptors
mkdir -p internal/services
mkdir -p internal/client
mkdir -p configs
mkdir -p bin
```

---

## 快速部署步驟

```bash
# 1. 建立專案目錄
mkdir my-grpc-service && cd my-grpc-service

# 2. 初始化 Git 和 Go 模組
git init
go mod init github.com/chslin/my-grpc-service

# 3. 建立目錄結構
mkdir -p api/hello/v1 cmd/server cmd/client internal/{logger,server/interceptors,services,client} configs

# 4. 複製所有檔案（使用上面提供的範本）
# - 複製 go.mod 到根目錄
# - 複製 api/hello/v1/hello.proto
# - 複製所有其他檔案到對應位置

# 5. 下載依賴
go mod download
go mod tidy

# 6. 生成 gRPC 代碼
make generate

# 7. 測試啟動
make run-server &
make run-client

# 停止伺服器
pkill -f "bin/server"
```

---

## 檔案權限

如果在 Linux/macOS 上執行，確保 Makefile 有執行權限：

```bash
chmod +x Makefile
```

---

## 驗證安裝

```bash
# 檢查 Go 安裝
go version

# 檢查 protoc 安裝
protoc --version

# 檢查 gRPC 外掛
protoc-gen-go --version
protoc-gen-go-grpc --version
```

如果任何工具缺失，按照 SETUP_GUIDE.md 中的「前置需求」部分安裝。

---

**提示**：複製完成後，執行 `make generate && make run-server` 來驗證設置！
