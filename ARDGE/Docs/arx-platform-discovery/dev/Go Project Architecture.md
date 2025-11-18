# Go 專案資料夾架構指南

  

本文件說明標準 Go 專案的資料夾結構組織，以及在使用 gRPC 伺服器或代理時的建議做法。

  

## 標準 Go 專案資料夾架構

  

```

myproject/

├── cmd/ # 可執行程式進入點

│ ├── server/

│ │ └── main.go # 伺服器主程式

│ ├── cli/

│ │ └── main.go # CLI 工具主程式

│ └── worker/

│ └── main.go # 背景工作程式

│

├── pkg/ # 可在其他專案重用的套件

│ ├── auth/

│ │ ├── auth.go

│ │ ├── jwt.go

│ │ └── *_test.go

│ ├── database/

│ │ ├── connection.go

│ │ └── *_test.go

│ └── utils/

│ ├── validator.go

│ └── *_test.go

│

├── internal/ # 專案專用的內部套件（不對外公開）

│ ├── api/

│ │ ├── http/

│ │ │ ├── handler.go

│ │ │ ├── middleware.go

│ │ │ └── *_test.go

│ │ └── grpc/ # gRPC 相關

│ │ ├── server.go

│ │ ├── interceptor.go

│ │ └── *_test.go

│ ├── service/

│ │ ├── user.go

│ │ ├── product.go

│ │ └── *_test.go

│ ├── repository/

│ │ ├── user_repo.go

│ │ ├── product_repo.go

│ │ └── *_test.go

│ ├── model/

│ │ ├── user.go

│ │ └── product.go

│ └── config/

│ └── config.go

│

├── proto/ # Protocol Buffer 檔案（如果使用 gRPC）

│ ├── user.proto

│ ├── product.proto

│ └── generate.sh # gRPC 程式碼生成腳本

│

├── migrations/ # 資料庫遷移檔案

│ ├── 001_init_schema.sql

│ └── 002_add_users_table.sql

│

├── scripts/ # 實用腳本

│ ├── build.sh

│ ├── deploy.sh

│ └── setup.sh

│

├── testdata/ # 測試用資料

│ ├── fixtures.json

│ └── sample_data.sql

│

├── docs/ # 文件

│ ├── API.md

│ ├── ARCHITECTURE.md

│ └── DEPLOYMENT.md

│

├── .github/ # GitHub 工作流程

│ └── workflows/

│ ├── test.yml

│ └── lint.yml

│

├── Makefile # 構建和測試命令

├── go.mod # Go 模組定義

├── go.sum # 依賴版本鎖定

├── Dockerfile # Docker 容器配置

├── docker-compose.yml # 本地開發環境

├── .env.example # 環境變數範本

├── .gitignore

└── README.md

```

  

### 各目錄說明

  

| 目錄 | 用途 | 說明 |

|------|------|------|

| `cmd/` | 可執行程式進入點 | 每個可執行檔案對應一個子目錄，包含獨立的 `main.go` |

| `pkg/` | 可重用套件 | 可在其他專案導入的程式碼，遵循 Go 公開 API 慣例 |

| `internal/` | 專案內部代碼 | 專案專用代碼，受 Go 編譯器保護，無法被外部導入 |

| `proto/` | Protocol Buffer 定義 | gRPC 服務定義，自動生成 Go 程式碼 |

| `migrations/` | 資料庫遷移 | SQL 遷移檔案，版本控制友善 |

| `scripts/` | 實用腳本 | 構建、部署、設定等自動化腳本 |

| `testdata/` | 測試資料 | 單元測試所需的 fixtures 和樣本資料 |

| `docs/` | 文件 | 專案文件、API 說明、架構設計 |

  

## 使用 gRPC 伺服器的架構

  

當專案包含 gRPC 伺服器時，建議如下組織：

  

```

myproject/

├── proto/ # 必須！Protocol Buffer 定義

│ ├── user.proto

│ ├── order.proto

│ └── Makefile # 自動生成 Go 程式碼

│

├── internal/

│ ├── api/

│ │ └── grpc/

│ │ ├── server.go # gRPC 伺服器主邏輯

│ │ ├── user_service.go # 實現 UserService

│ │ ├── order_service.go # 實現 OrderService

│ │ ├── interceptor.go # 驗證、日誌等中間層

│ │ └── *_test.go

│ ├── service/

│ │ ├── user.go # 業務邏輯（不依賴 gRPC）

│ │ └── order.go

│ └── repository/

│ ├── user_repo.go

│ └── order_repo.go

│

├── cmd/

│ └── grpc-server/

│ └── main.go # 啟動 gRPC 伺服器

│

└── ...

```

  

### gRPC 檔案結構最佳實踐

  

1. **分離傳輸層和業務邏輯**

- `internal/api/grpc/` - gRPC 伺服器實現（傳輸層）

- `internal/service/` - 業務邏輯層

- Service 不應知道 gRPC 的存在

  

2. **使用中間層（Interceptor）**

```go

// internal/api/grpc/interceptor.go

// 處理通用邏輯：驗證、日誌、追蹤、錯誤轉換

```

  

3. **一個 proto 檔案對應一個 service 實現**

- `user.proto` → `user_service.go`

- `order.proto` → `order_service.go`

  

4. **Proto 程式碼生成**

```makefile

# proto/Makefile

.PHONY: generate

generate:

protoc --go_out=. --go-grpc_out=. user.proto

protoc --go_out=. --go-grpc_out=. order.proto

```

  

## 使用 Agent（代理）的架構

  

當專案包含 AI 代理時，建議如下組織：

  

```

myproject/

├── internal/

│ ├── agent/ # 代理相關邏輯

│ │ ├── agent.go # 代理主結構和執行邏輯

│ │ ├── executor.go # 代理執行引擎

│ │ ├── tool/ # 代理可用工具

│ │ │ ├── tool.go # 工具介面定義

│ │ │ ├── calculator.go # 計算器工具

│ │ │ └── database.go # 資料庫查詢工具

│ │ ├── memory/ # 代理狀態和記憶

│ │ │ ├── memory.go # 記憶介面

│ │ │ └── conversation.go # 對話歷史

│ │ └── *_test.go

│ ├── llm/ # LLM 整合層

│ │ ├── client.go # LLM API 客戶端

│ │ ├── prompt.go # 提示詞管理

│ │ └── response.go # 回應解析

│ └── service/

│ └── agent_service.go # 代理業務服務

│

├── cmd/

│ └── agent/

│ └── main.go # 代理程式進入點

│

└── ...

```

  

### Agent 架構最佳實踐

  

1. **代理核心層**（`internal/agent/`）

- `Agent` 結構：代理主體和狀態管理

- `Executor` 介面：代理執行引擎

- 獨立於傳輸層

  

2. **工具定義**（`internal/agent/tool/`）

```go

// internal/agent/tool/tool.go

type Tool interface {

Name() string

Description() string

Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)

}

```

  

3. **記憶管理**（`internal/agent/memory/`）

- 對話歷史管理

- 上下文維護

- 狀態持久化

  

4. **LLM 整合層**（`internal/llm/`）

- 與語言模型的通信

- 提示詞管理

- 回應解析和驗證

  

## 組合使用 gRPC + Agent

  

當同時使用 gRPC 和代理時，架構應如下組織：

  

```

myproject/

├── proto/

│ ├── agent_service.proto # 代理 gRPC 介面

│ └── Makefile

│

├── internal/

│ ├── api/

│ │ └── grpc/

│ │ ├── server.go

│ │ └── agent_service.go # 實現 gRPC AgentService

│ ├── agent/

│ │ ├── agent.go

│ │ ├── executor.go

│ │ ├── tool/

│ │ └── memory/

│ ├── service/

│ │ └── agent_service.go # 業務邏輯層

│ └── llm/

│ └── client.go

│

├── cmd/

│ └── server/

│ └── main.go # 啟動 gRPC 伺服器（提供代理介面）

│

└── ...

```

  

### 整合重點

  

1. **依賴方向清晰**

```

gRPC API → Service 層 → Agent 核心 → LLM 客戶端

```

  

2. **層級分離**

- gRPC 伺服器只負責網路通信和協議轉換

- Service 層協調業務邏輯和代理

- Agent 層獨立於具體的通信方式

  

3. **可測試性**

```go

// Agent 邏輯可以獨立測試，無需 gRPC

agent := NewAgent(llmClient, tools)

result := agent.Execute(ctx, "query")

  

// Service 層可以單獨測試

service := NewAgentService(agent)

  

// gRPC 處理層可以單獨測試

// 使用 mock service 測試

```

  

## 關鍵設計原則

  

### 1. 依賴反轉

- 內層（核心邏輯）不依賴外層（傳輸層）

- 通過介面定義依賴關係

- 外層依賴內層的抽象

  

### 2. 單一職責

- 每個套件做好一件事

- API 層只處理協議

- Service 層只處理業務邏輯

- Agent 層只處理代理執行

  

### 3. 測試友善

- 每層獨立可測

- 使用 mock 和 stub

- 避免外部依賴（資料庫、LLM）進入單元測試

  

### 4. 內部保護

- 使用 `internal/` 目錄禁止外部導入

- 使用 `pkg/` 明確標示公開 API

- 小寫首字母表示包級私有

  

## 命名慣例

  

### 套件名稱

- 短小且有意義

- 單數形式：`user` 而非 `users`

- 小寫字母，無底線

- 例：`auth`, `database`, `service`

  

### 檔案名稱

- 小寫字母，使用底線分隔單字

- 測試檔案：`filename_test.go`

- 例：`user_service.go`, `auth_handler.go`

  

### 目錄名稱

- 複數形式用於集合：`pkg/`, `tools/`, `migrations/`

- 單數形式用於功能域：`internal/service/`, `internal/model/`

  

## 範例：mDNS 發現服務專案結構

  

```

arx-platform-discover/

├── cmd/

│ ├── server/

│ │ └── main.go # gRPC 伺服器進入點

│ └── cli/

│ └── main.go # CLI 工具進入點

│

├── proto/

│ ├── discover.proto # mDNS 發現 gRPC 定義

│ └── Makefile

│

├── internal/

│ ├── api/

│ │ └── grpc/

│ │ ├── server.go

│ │ ├── discover_service.go

│ │ └── interceptor.go

│ ├── service/

│ │ ├── discovery.go # mDNS 發現邏輯

│ │ └── resolver.go # 服務解析

│ ├── mdns/

│ │ ├── browser.go # mDNS 瀏覽器

│ │ ├── advertiser.go # mDNS 發佈者

│ │ └── model.go

│ ├── config/

│ │ └── config.go

│ └── model/

│ └── service.go

│

├── pkg/

│ └── mdnsutil/ # 可重用 mDNS 工具

│ ├── util.go

│ └── *_test.go

│

├── docs/

│ ├── ARCHITECTURE.md

│ ├── API.md

│ └── DEPLOYMENT.md

│

├── Makefile

├── go.mod

├── go.sum

└── README.md

```

  

## 進一步參考

  

- [Go 官方專案結構建議](https://github.com/golang-standards/project-layout)

- [Effective Go](https://go.dev/doc/effective_go)

- [gRPC Go 最佳實踐](https://grpc.io/docs/guides/performance-best-practices/)