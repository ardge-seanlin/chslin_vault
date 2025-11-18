# gRPC 快速參考指南

## 一頁紙速查表

### 專案結構速查

```
api/           → Proto 定義和生成的代碼
cmd/           → 應用程式入口點（server、client）
internal/      → 內部實現
  ├─logger/    → 日誌介面
  ├─config/    → 配置管理
  ├─server/    → gRPC 伺服器（包含攔截器）
  ├─services/  → 業務邏輯實現
  └─client/    → 客戶端連接池
configs/       → 配置檔案
```

### 常用命令

```bash
make generate      # 生成 gRPC 代碼
make build         # 編譯二進制
make run-server    # 啟動伺服器
make run-client    # 運行客戶端
make test          # 運行測試
make lint          # 代碼檢查
make clean         # 清理檔案
```

### Proto 基本語法

```protobuf
syntax = "proto3";
package hello.v1;
option go_package = "github.com/xxx/api/hello/v1";

// Unary RPC：一請求一響應
service GreetingService {
  rpc Greet(GreetRequest) returns (GreetResponse);
}

// 串流 RPC 類型
service StreamService {
  // 伺服器端串流
  rpc ServerStream(Request) returns (stream Response);

  // 客戶端串流
  rpc ClientStream(stream Request) returns (Response);

  // 雙向串流
  rpc BiStream(stream Request) returns (stream Response);
}

message GreetRequest {
  string name = 1;
}

message GreetResponse {
  string message = 1;
}
```

### 服務實現框架

```go
package services

import (
    "context"
    v1 "github.com/xxx/api/hello/v1"
)

type GreetingService struct {
    v1.UnimplementedGreetingServiceServer
    // 添加依賴
    log logger.Logger
}

func NewGreetingService(log logger.Logger) *GreetingService {
    return &GreetingService{log: log}
}

// 實現 RPC 方法
func (s *GreetingService) Greet(ctx context.Context,
    req *v1.GreetRequest) (*v1.GreetResponse, error) {

    // 業務邏輯
    return &v1.GreetResponse{Message: "Hello, " + req.Name}, nil
}

// 伺服器端串流
func (s *GreetingService) ServerStream(req *v1.Request,
    stream v1.Service_ServerStreamServer) error {

    for i := 0; i < 3; i++ {
        if err := stream.Send(&v1.Response{...}); err != nil {
            return err
        }
    }
    return nil
}

// 客戶端串流
func (s *GreetingService) ClientStream(
    stream v1.Service_ClientStreamServer) error {

    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&v1.Response{...})
        }
        if err != nil {
            return err
        }
        // 處理 req
    }
}

// 雙向串流
func (s *GreetingService) BiStream(
    stream v1.Service_BiStreamServer) error {

    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        if err := stream.Send(&v1.Response{...}); err != nil {
            return err
        }
    }
}
```

### 攔截器實現框架

```go
package interceptors

import (
    "context"
    "google.golang.org/grpc"
)

// Unary 攔截器
func MyUnaryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // 前置邏輯

        // 執行 RPC
        resp, err := handler(ctx, req)

        // 後置邏輯
        return resp, err
    }
}

// 串流攔截器
func MyStreamInterceptor(log logger.Logger) grpc.StreamServerInterceptor {
    return func(
        srv interface{},
        ss grpc.ServerStream,
        info *grpc.StreamServerInfo,
        handler grpc.StreamHandler,
    ) error {
        // 前置邏輯

        // 執行串流 RPC
        err := handler(srv, ss)

        // 後置邏輯
        return err
    }
}
```

### gRPC 伺服器設置框架

```go
package server

import (
    "google.golang.org/grpc"
    v1 "github.com/xxx/api/hello/v1"
)

func NewGRPCServer(address string, log logger.Logger) (*GRPCServer, error) {
    listener, err := net.Listen("tcp", address)
    if err != nil {
        return nil, err
    }

    // 建立伺服器並配置攔截器
    grpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            interceptors.RecoveryUnaryInterceptor(log),
            interceptors.LoggingUnaryInterceptor(log),
        ),
        grpc.ChainStreamInterceptor(
            interceptors.RecoveryStreamInterceptor(log),
            interceptors.LoggingStreamInterceptor(log),
        ),
    )

    // 建立並註冊服務
    svc := services.NewGreetingService(log)
    v1.RegisterGreetingServiceServer(grpcServer, svc)

    return &GRPCServer{
        grpcServer: grpcServer,
        listener:   listener,
        address:    address,
        log:        log,
    }, nil
}

// 啟動伺服器
func (s *GRPCServer) Start() error {
    return s.grpcServer.Serve(s.listener)
}

// 非阻塞啟動
func (s *GRPCServer) StartAsync() error {
    go s.Start()
    return nil
}

// 優雅關閉
func (s *GRPCServer) Stop(ctx context.Context) error {
    done := make(chan error, 1)
    go func() {
        s.grpcServer.GracefulStop()
        done <- nil
    }()

    select {
    case <-ctx.Done():
        s.grpcServer.Stop()
        return ctx.Err()
    case err := <-done:
        return err
    }
}
```

### 主程式框架

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    // 1. 初始化 Logger
    log := logger.NewSlogAdapter(
        slog.New(slog.NewTextHandler(os.Stdout, nil)),
    )

    // 2. 建立伺服器
    srv, err := server.NewGRPCServer(":50051", log)
    if err != nil {
        log.Error("failed to create server", "error", err)
        os.Exit(1)
    }

    // 3. 非阻塞啟動
    srv.StartAsync()

    // 4. 等待就緒
    readyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    srv.WaitForReady(readyCtx)
    cancel()

    // 5. 監聽關閉信號
    ctx, cancel := signal.NotifyContext(
        context.Background(),
        syscall.SIGINT, syscall.SIGTERM,
    )
    <-ctx.Done()
    cancel()

    // 6. 優雅關閉
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    srv.Stop(shutdownCtx)
    cancel()
}
```

### gRPC 客戶端框架

```go
package main

import (
    "context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    v1 "github.com/xxx/api/hello/v1"
)

func main() {
    // 連接伺服器
    conn, err := grpc.NewClient(
        "localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // 建立客戶端
    client := v1.NewGreetingServiceClient(conn)

    // 呼叫 Unary RPC
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    resp, err := client.Greet(ctx, &v1.GreetRequest{Name: "World"})
    cancel()

    if err != nil {
        panic(err)
    }
    println(resp.Message)
}
```

### 單元測試框架

```go
package services_test

import (
    "context"
    "testing"
    v1 "github.com/xxx/api/hello/v1"
)

// 表格驅動測試
func TestGreet(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"success", "Alice", "Hello, Alice!", false},
        {"empty", "", "", true},
    }

    svc := services.NewGreetingService()

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp, err := svc.Greet(
                context.Background(),
                &v1.GreetRequest{Name: tt.input},
            )

            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, want %v", err, tt.wantErr)
            }

            if !tt.wantErr && resp.Message != tt.want {
                t.Errorf("got %q, want %q", resp.Message, tt.want)
            }
        })
    }
}
```

### 常見錯誤處理

```go
import "google.golang.org/grpc/status"
import "google.golang.org/grpc/codes"

// 返回 gRPC 錯誤
func (s *Service) Method(ctx context.Context, req *Request) (*Response, error) {
    // 驗證
    if req.Name == "" {
        return nil, status.Error(codes.InvalidArgument, "name is required")
    }

    // 未找到
    if !exists {
        return nil, status.Error(codes.NotFound, "resource not found")
    }

    // 內部錯誤
    if err != nil {
        return nil, status.Error(codes.Internal, "internal server error")
    }

    // 超時
    if timeout {
        return nil, status.Error(codes.DeadlineExceeded, "request timeout")
    }

    return &Response{}, nil
}
```

### gRPC 錯誤碼對照

```
codes.OK                 = 0   // 成功
codes.Cancelled          = 1   // 已取消
codes.Unknown            = 2   // 未知錯誤
codes.InvalidArgument    = 3   // 無效參數
codes.DeadlineExceeded   = 4   // 截止期限超過
codes.NotFound           = 5   // 未找到
codes.AlreadyExists      = 6   // 已存在
codes.PermissionDenied   = 7   // 權限被拒絕
codes.ResourceExhausted  = 8   // 資源耗盡
codes.FailedPrecondition = 9   // 前置條件失敗
codes.Aborted            = 10  // 已中止
codes.OutOfRange         = 11  // 超出範圍
codes.Unimplemented      = 12  // 未實現
codes.Internal           = 13  // 內部錯誤
codes.Unavailable        = 14  // 不可用
codes.DataLoss           = 15  // 資料遺失
codes.Unauthenticated    = 16  // 未認證
```

### Context 用法

```go
// 設置超時
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 檢查超時
if ctx.Err() == context.DeadlineExceeded {
    log.Println("request timeout")
}

// 在 context 中存儲值
ctx = context.WithValue(ctx, "request_id", "123")

// 取得值
requestID := ctx.Value("request_id").(string)

// 檢查取消
select {
case <-ctx.Done():
    return ctx.Err()
}
```

### 配置檔案參考

```toml
[grpc]
address = ":50051"
slow_threshold_ms = 1000

[log]
level = "info"      # debug, info, warn, error
format = "plain"    # plain, json
```

### 環境變數設置

```bash
# 啟用 gRPC 詳細日誌
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info

# 禁用 DNS 解析
export GRPC_GO_DISCOVERY=static

# 設置 keepalive
export GRPC_GO_KEEPALIVE_TIME_MS=30000
```

### 偵錯技巧

```go
// 打印請求詳情
log.Printf("method=%s, request=%+v", info.FullMethod, req)

// 記錄 context 值
ctx := context.WithValue(context.Background(), "debug", true)

// 捕獲 panic
defer func() {
    if r := recover(); r != nil {
        log.Printf("panic: %v", r)
    }
}()

// 檢查連接狀態
state := conn.GetState()
log.Printf("connection state: %v", state)
```

### 性能優化建議

1. **連接復用** - 使用連接池而不是新建連接
2. **批量操作** - 合並多個請求為單個批量操作
3. **緩存** - 緩存頻繁查詢的結果
4. **異步處理** - 使用非阻塞 I/O
5. **調整緩衝區** - 根據需要調整 gRPC 緩衝大小
6. **監控慢查詢** - 使用性能攔截器捕獲慢請求

### 常用工具命令

```bash
# 使用 grpcurl 測試 RPC
grpcurl -plaintext localhost:50051 list

# 查看服務信息
grpcurl -plaintext localhost:50051 describe hello.v1.GreetingService

# 調用 RPC 方法
grpcurl -plaintext -d '{"name":"World"}' \
  localhost:50051 hello.v1.GreetingService/Greet

# 檢查健康狀態
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# 使用 Evans（互動式 gRPC 客戶端）
brew install evans
evans -r localhost:50051
```

### Git 提交範本

```
feat(service): add new gRPC method

- Implemented Greet RPC method
- Added unit tests with 100% coverage
- Tested with client integration

Closes #123
```

### 檢查清單

- [ ] Proto 定義完整
- [ ] 生成了 gRPC 代碼
- [ ] 實現了所有 RPC 方法
- [ ] 添加了錯誤處理
- [ ] 編寫了單元測試
- [ ] 測試通過且覆蓋率 > 80%
- [ ] 代碼通過 linter 檢查
- [ ] 文檔已更新

---

**提示**：將此頁面加入書籤以快速查閱！
