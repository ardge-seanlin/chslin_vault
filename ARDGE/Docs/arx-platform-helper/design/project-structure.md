# 專案目錄結構

```
tunnel-manager/
├── cmd/
│   └── server/
│       └── main.go                 # 應用程式進入點
│
├── internal/                       # 私有套件（不對外暴露）
│   ├── config/
│   │   ├── config.go              # 設定結構與載入
│   │   ├── config_test.go
│   │   └── secret.go              # Secret Provider 實作
│   │
│   ├── domain/                    # 領域模型（核心業務）
│   │   ├── tunnel.go              # Tunnel 實體
│   │   ├── process.go             # Process 實體
│   │   ├── ingress.go             # IngressRule 實體
│   │   └── errors.go              # 領域錯誤定義
│   │
│   ├── service/                   # 業務邏輯層
│   │   ├── tunnel_service.go      # TunnelService 實作
│   │   ├── tunnel_service_test.go
│   │   └── interfaces.go          # 服務介面定義
│   │
│   ├── cloudflare/               # Cloudflare API Client
│   │   ├── client.go             # CloudflareAPIClient 實作
│   │   ├── client_test.go
│   │   ├── tunnel.go             # Tunnel 相關 API
│   │   ├── config.go             # Configuration API
│   │   └── types.go              # API 請求/回應類型
│   │
│   ├── process/                  # 進程管理
│   │   ├── manager.go            # ProcessManager 實作
│   │   ├── manager_test.go
│   │   ├── options.go            # ProcessOptions
│   │   └── monitor.go            # 進程監控
│   │
│   ├── api/                      # HTTP API 層
│   │   ├── server.go             # HTTP Server
│   │   ├── router.go             # 路由設定
│   │   ├── middleware/
│   │   │   ├── auth.go           # 認證 middleware
│   │   │   ├── ratelimit.go      # 限流 middleware
│   │   │   ├── logging.go        # 日誌 middleware
│   │   │   └── recovery.go       # Panic 恢復
│   │   ├── handler/
│   │   │   ├── tunnel.go         # Tunnel handlers
│   │   │   ├── process.go        # Process handlers
│   │   │   └── health.go         # Health check
│   │   ├── dto/
│   │   │   ├── request.go        # 請求 DTO
│   │   │   ├── response.go       # 回應 DTO
│   │   │   └── error.go          # 錯誤回應
│   │   └── validator/
│   │       ├── tunnel.go         # Tunnel 驗證
│   │       └── ingress.go        # Ingress 驗證
│   │
│   └── pkg/                      # 內部共用套件
│       ├── logger/
│       │   ├── logger.go         # 結構化日誌
│       │   └── redact.go         # 敏感資料遮蔽
│       ├── crypto/
│       │   └── compare.go        # Constant-time 比較
│       └── duration/
│           └── duration.go       # JSON Duration 類型
│
├── pkg/                          # 公開套件（可被其他專案使用）
│   └── contracts/
│       ├── interfaces.go         # 公開介面定義
│       └── types.go              # 公開類型定義
│
├── api/                          # API 規格
│   └── openapi.yaml              # OpenAPI 3.0 規格
│
├── deployments/                  # 部署配置
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yaml
│   ├── kubernetes/
│   │   ├── base/
│   │   │   ├── deployment.yaml
│   │   │   ├── service.yaml
│   │   │   ├── configmap.yaml
│   │   │   └── kustomization.yaml
│   │   └── overlays/
│   │       ├── dev/
│   │       └── prod/
│   └── systemd/
│       └── tunnel-manager.service
│
├── configs/                      # 設定檔範本
│   ├── config.yaml.example
│   └── env.example
│
├── scripts/                      # 工具腳本
│   ├── generate-api-key.sh       # 產生 API Key
│   ├── generate-certs.sh         # 產生憑證
│   └── health-check.sh           # 健康檢查腳本
│
├── test/                         # 測試相關
│   ├── integration/              # 整合測試
│   │   ├── api_test.go
│   │   └── flow_test.go
│   ├── mocks/                    # Mock 定義
│   │   ├── cloudflare.go
│   │   └── process.go
│   └── fixtures/                 # 測試資料
│       └── tunnels.json
│
├── docs/                         # 文檔
│   ├── design/
│   │   ├── system-design.md      # 系統設計文檔
│   │   ├── sequence-diagrams.md  # 時序圖
│   │   └── security.md           # 安全設計
│   ├── api/
│   │   └── README.md             # API 使用說明
│   └── deployment/
│       └── README.md             # 部署指南
│
├── .github/
│   └── workflows/
│       ├── ci.yaml               # CI pipeline
│       ├── release.yaml          # Release pipeline
│       └── security.yaml         # Security scanning
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── .gitignore
```

---

## 檔案職責說明

### cmd/server/main.go
```go
// 應用程式進入點
// 職責：
// 1. 載入設定
// 2. 初始化依賴（DI）
// 3. 啟動 HTTP Server
// 4. 處理優雅關閉
```

### internal/domain/
```go
// 領域層 - 核心業務邏輯
// 原則：
// 1. 不依賴任何外部套件（純 Go）
// 2. 定義業務規則和驗證
// 3. 可獨立測試
```

### internal/service/
```go
// 服務層 - 業務流程編排
// 職責：
// 1. 協調多個 repository/client
// 2. 實作業務用例
// 3. 處理交易邊界
```

### internal/cloudflare/
```go
// Cloudflare API Client
// 職責：
// 1. 封裝 Cloudflare API 呼叫
// 2. 處理認證
// 3. 錯誤轉換
```

### internal/process/
```go
// 進程管理
// 職責：
// 1. cloudflared 進程生命週期
// 2. 進程狀態追蹤
// 3. 並發安全
```

### internal/api/
```go
// HTTP API 層
// 職責：
// 1. 請求處理和回應
// 2. 認證授權
// 3. 輸入驗證
// 4. 錯誤處理
```

---

## 依賴關係圖

```
                    ┌─────────────┐
                    │   main.go   │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │   config    │ │     api     │ │   logger    │
    └─────────────┘ └──────┬──────┘ └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   service   │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │ cloudflare  │ │   process   │ │   domain    │
    │   client    │ │   manager   │ │  (models)   │
    └─────────────┘ └─────────────┘ └─────────────┘

依賴方向：上層依賴下層，下層不知道上層存在
domain 層為最底層，不依賴任何其他內部套件
```

---

## Makefile 目標

```makefile
.PHONY: all build test lint clean

# 預設目標
all: lint test build

# 編譯
build:
	go build -o bin/tunnel-manager ./cmd/server

# 測試
test:
	go test -v -race -cover ./...

test-integration:
	INTEGRATION_TEST=true go test -v ./test/integration/...

# Lint
lint:
	golangci-lint run

# 產生
generate:
	go generate ./...

# 清理
clean:
	rm -rf bin/
	go clean -cache

# Docker
docker-build:
	docker build -t tunnel-manager:latest .

docker-push:
	docker push tunnel-manager:latest

# 開發
dev:
	air -c .air.toml

# 安全掃描
security:
	gosec ./...
	trivy fs .

# API 文檔
docs:
	swag init -g cmd/server/main.go -o api/
```

---

## 相依套件建議

```go
// go.mod
module github.com/kaiden/tunnel-manager

go 1.22

require (
    // Cloudflare SDK
    github.com/cloudflare/cloudflare-go v0.86.0
    
    // HTTP Router (標準庫即可，Go 1.22+ 支援路由參數)
    // 或使用 chi/echo/gin
    
    // 驗證
    github.com/go-playground/validator/v10 v10.18.0
    
    // 設定
    github.com/spf13/viper v1.18.2
    
    // 日誌 (使用標準庫 log/slog，Go 1.21+)
    
    // 測試
    github.com/stretchr/testify v1.8.4
    github.com/golang/mock v1.6.0
)
```
