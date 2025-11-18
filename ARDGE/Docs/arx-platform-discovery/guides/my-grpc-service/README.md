# My gRPC Service

一個完整的 gRPC 伺服器實現，基於 arx-platform-core 的架構和最佳實踐。

## 特點

- ✅ **完整的 gRPC 設置** - Proto 定義、Unary 和 Stream RPC
- ✅ **攔截器鏈** - Recovery、Logging、Request ID、Performance 監控
- ✅ **配置管理** - TOML 檔案配置系統
- ✅ **健康檢查** - gRPC 標準健康檢查協議
- ✅ **優雅啟動/關閉** - 完整的生命周期管理
- ✅ **連接池** - 用於後端服務的連接複用
- ✅ **多路復用** - cmux 在同一端口處理 gRPC 和 HTTP
- ✅ **結構化日誌** - 標準 log/slog 集成
- ✅ **性能監控** - 慢請求警告和追蹤
- ✅ **測試示例** - 單元測試和集成測試

## 快速開始

### 前置需求

- Go 1.21+
- protoc（用於編譯 proto 檔案）

### 安裝依賴

```bash
# 安裝 protoc 編譯工具（macOS）
brew install protobuf

# 安裝 Go protoc 外掛
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 下載 Go 依賴
go mod download
```

### 專案初始化

```bash
# 1. 生成 gRPC 代碼
make generate

# 2. 啟動伺服器
make run-server

# 3. 在另一個終端運行客戶端
make run-client
```

## 目錄結構

```
my-grpc-service/
├── api/                          # 生成的 gRPC 代碼和 proto 定義
│   └── hello/
│       └── v1/
│           ├── hello.proto       # Proto 服務定義
│           ├── hello.pb.go       # 生成的 Protobuf 代碼
│           └── hello_grpc.pb.go  # 生成的 gRPC 代碼
│
├── cmd/                          # 應用程式入口點
│   ├── server/
│   │   └── main.go              # 伺服器主程式
│   └── client/
│       └── main.go              # 測試客戶端
│
├── internal/                     # 內部包（不對外導出）
│   ├── logger/
│   │   └── logger.go            # 日誌介面和實現
│   ├── config/
│   │   └── config.go            # 配置管理
│   ├── server/
│   │   ├── grpc.go              # gRPC 伺服器設置
│   │   ├── health.go            # 健康檢查
│   │   ├── cmux_setup.go        # 多路復用設置
│   │   └── interceptors/
│   │       ├── recovery.go      # Panic 恢復攔截器
│   │       ├── logging.go       # 日誌攔截器
│   │       ├── request_id.go    # Request ID 攔截器
│   │       └── performance.go   # 性能監控攔截器
│   ├── services/
│   │   ├── greeting.go          # 服務實現
│   │   └── greeting_test.go     # 服務單元測試
│   └── client/
│       └── connector.go         # 連接池實現
│
├── configs/
│   └── config.toml              # 伺服器配置檔案
│
├── go.mod                        # Go 模組定義
├── go.sum                        # Go 依賴版本鎖定
├── Makefile                      # 開發工具命令
├── SETUP_GUIDE.md               # 詳細設置指南
├── ADVANCED_FEATURES.md         # 進階功能文檔
└── README.md                    # 本檔案
```

## 核心概念

### 1. Proto 定義

服務在 `api/hello/v1/hello.proto` 中定義，包含兩種 RPC 類型：

```protobuf
service GreetingService {
  rpc Greet(GreetRequest) returns (GreetResponse);           // Unary RPC
  rpc GreetStream(GreetRequest) returns (stream GreetResponse);  // 伺服器端串流
}
```

### 2. 攔截器鏈

攔截器按順序執行，提供橫切關注點：

```
請求 → RequestID → Recovery → Performance → Logging → 處理程序
                                                        ↓
                                                    回應
```

### 3. 服務實現

每個服務實現 `UnimplementedXxxServiceServer` 並覆寫所需方法：

```go
type GreetingService struct {
    v1.UnimplementedGreetingServiceServer
}

func (s *GreetingService) Greet(ctx context.Context, req *v1.GreetRequest) (*v1.GreetResponse, error) {
    return &v1.GreetResponse{Message: fmt.Sprintf("Hello, %s!", req.Name)}, nil
}
```

### 4. gRPC 伺服器生命周期

```
初始化 → 啟動 → 就緒檢查 → 運行中 → 停止信號 → 優雅關閉 → 資源清理
```

## 常見操作

### 編譯和運行

```bash
# 生成所有 gRPC 代碼
make generate

# 構建二進制
make build

# 運行伺服器
make run-server

# 運行客戶端
make run-client

# 同時運行伺服器和客戶端
make run-server &  # 在背景執行
make run-client
```

### 測試

```bash
# 運行所有測試
make test

# 生成覆蓋率報告
make test-coverage

# 執行性能測試
make test-bench

# 執行 linter 檢查
make lint

# 格式化代碼
make fmt
```

### 清理

```bash
# 刪除生成的檔案和二進制
make clean

# 清除所有生成的 *.pb.go 檔案
find . -name "*.pb.go" -delete
```

## 添加新服務

### 第 1 步：定義 Proto

編輯 `api/hello/v1/hello.proto`：

```protobuf
service MyNewService {
  rpc MyMethod(MyRequest) returns (MyResponse);
}

message MyRequest {
  string input = 1;
}

message MyResponse {
  string output = 1;
}
```

### 第 2 步：生成代碼

```bash
make generate
```

### 第 3 步：實現服務

建立 `internal/services/my_service.go`：

```go
package services

import (
    "context"
    v1 "github.com/chslin/my-grpc-service/api/hello/v1"
)

type MyService struct {
    v1.UnimplementedMyNewServiceServer
}

func NewMyService() *MyService {
    return &MyService{}
}

func (s *MyService) MyMethod(ctx context.Context, req *v1.MyRequest) (*v1.MyResponse, error) {
    return &v1.MyResponse{Output: req.Input}, nil
}
```

### 第 4 步：註冊服務

修改 `internal/server/grpc.go`：

```go
// 在 NewGRPCServer 中添加
myService := services.NewMyService()
v1.RegisterMyNewServiceServer(grpcServer, myService)
```

### 第 5 步：測試

建立測試檔案並運行：

```bash
make test
```

## 配置管理

### 預設配置位置

`configs/config.toml`

### 配置選項

```toml
[grpc]
address = ":50051"              # 監聽地址和端口
slow_threshold_ms = 1000        # 慢請求閾值（毫秒）

[log]
level = "info"                  # 日誌等級：debug, info, warn, error
format = "plain"                # 日誌格式：plain 或 json
```

### 運行時使用配置

```bash
# 修改配置檔案後，重新啟動伺服器
make run-server
```

## 健康檢查

### 使用 grpcurl

```bash
# 檢查整體健康狀態
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# 檢查特定服務
grpcurl -plaintext -d '{"service":"hello.v1.GreetingService"}' \
  localhost:50051 grpc.health.v1.Health/Check
```

### 使用 Go 客戶端

```go
import "google.golang.org/grpc/health/grpc_health_v1"

conn, _ := grpc.NewClient("localhost:50051", grpc.WithInsecure())
client := grpc_health_v1.NewHealthClient(conn)
resp, _ := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
fmt.Println(resp.Status)
```

## 性能監控

### 慢請求警告

當 RPC 執行時間超過 `slow_threshold_ms` 時，會記錄警告日誌：

```
time=2024-01-01T12:00:00Z level=WARN msg="RPC call exceeded slow threshold"
  request_id=abc123 method=/hello.v1.GreetingService/Greet
  duration_ms=1500 threshold_ms=1000
```

### pprof 性能分析

如果啟用了 cmux 和 pprof：

```bash
# 查看堆積分析
go tool pprof http://localhost:50051/debug/pprof/heap

# 查看 CPU 分析（30 秒採樣）
go tool pprof http://localhost:50051/debug/pprof/profile?seconds=30

# 查看 goroutine 分析
go tool pprof http://localhost:50051/debug/pprof/goroutine
```

## 日誌輸出示例

### 正常執行

```
time=2024-01-01T12:00:00.000Z level=INFO msg="gRPC server started successfully" address=:50051
time=2024-01-01T12:00:01.234Z level=INFO msg="RPC completed" request_id=abc-123 method=/hello.v1.GreetingService/Greet duration_ms=45
```

### 發生錯誤

```
time=2024-01-01T12:00:02.567Z level=ERROR msg="RPC failed" request_id=def-456 method=/hello.v1.GreetingService/Greet error="name cannot be empty" duration_ms=12
```

### 慢請求

```
time=2024-01-01T12:00:03.890Z level=WARN msg="RPC call exceeded slow threshold" request_id=ghi-789 method=/hello.v1.GreetingService/GreetStream duration_ms=1523 threshold_ms=1000
```

## 故障排查

### 問題：連接被拒絕

```
Error: failed to connect: connection refused
```

**解決方案**：確保伺服器已啟動：
```bash
make run-server
```

### 問題：Proto 編譯失敗

```
failed to load proto file
```

**解決方案**：檢查 protoc 是否已安裝：
```bash
protoc --version
brew install protobuf
```

### 問題：模組版本衝突

```
missing go.sum entry
```

**解決方案**：更新依賴：
```bash
go mod tidy
go mod download
```

### 問題：Panic 導致伺服器當機

**解決方案**：Recovery 攔截器應已捕獲，檢查日誌：
```bash
# 查看錯誤日誌
make run-server 2>&1 | grep "panic recovered"
```

## 對應 arx-platform-core

本專案結構直接映射 arx-platform-core：

| 元件 | my-grpc-service | arx-platform-core |
|------|-----------------|-------------------|
| **Proto** | `api/hello/v1/hello.proto` | 來自外部 arx-platform-proto |
| **Logger** | `internal/logger/logger.go` | `internal/logger/logger.go` |
| **攔截器** | `internal/server/interceptors/` | `internal/server/interceptors/` |
| **服務** | `internal/services/greeting.go` | `internal/services/{core,marketplace,node}/` |
| **伺服器** | `internal/server/grpc.go` | `internal/server/grpc_setup.go` |
| **主程式** | `cmd/server/main.go` | `cmd/server/main.go` |
| **配置** | `configs/config.toml` | `configs/config.toml` |
| **健康檢查** | `internal/server/health.go` | `internal/server/health/` |
| **連接池** | `internal/client/connector.go` | `internal/client/connector.go` |

## 深入學習

- [SETUP_GUIDE.md](./SETUP_GUIDE.md) - 詳細的設置和基礎概念
- [ADVANCED_FEATURES.md](./ADVANCED_FEATURES.md) - 進階功能和優化技巧
- [gRPC 官方文檔](https://grpc.io/docs/languages/go/)
- [Protobuf 官方文檔](https://developers.google.com/protocol-buffers)
- [Effective Go](https://go.dev/doc/effective_go)
- [Uber Go Style Guide](https://github.com/uber-go/guide)

## 開發工作流

### 日常開發

```bash
# 1. 修改 proto 或程式碼
# 2. 生成代碼
make generate

# 3. 運行測試
make test

# 4. 檢查代碼質量
make lint

# 5. 啟動伺服器開發
make run-server
```

### 添加功能

```bash
# 1. 更新 proto
vim api/hello/v1/hello.proto

# 2. 生成代碼
make generate

# 3. 實現服務
vim internal/services/your_service.go

# 4. 寫測試
vim internal/services/your_service_test.go

# 5. 運行測試
make test

# 6. 手動測試
make run-server &
make run-client
```

### 性能優化

```bash
# 1. 運行基準測試
make test-bench

# 2. 分析性能瓶頸
go tool pprof -http=:8080 cpu.prof

# 3. 檢查內存使用
make test-coverage
```

## 貢獻指南

1. 建立 feature 分支：`git checkout -b feature/your-feature`
2. 提交更改：`git commit -m "feat: add your feature"`
3. 運行測試：`make test`
4. 推送分支：`git push origin feature/your-feature`
5. 建立 Pull Request

## 許可證

MIT License

## 聯繫方式

如有問題或建議，歡迎提交 Issue 或 PR。

---

**最後更新**：2024 年 1 月
**Go 版本**：1.21+
**gRPC 版本**：v1.77.0+
