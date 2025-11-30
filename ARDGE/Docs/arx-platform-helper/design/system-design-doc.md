# Cloudflare Tunnel Manager 系統設計文檔

> **版本:** 1.0  
> **日期:** 2025-01-XX  
> **作者:** Kaiden Intelligence  

---

## 1. 概述

### 1.1 目標

建立一個 Go Server 作為 Cloudflare Tunnel 的管理層，實現：
- 透過 Cloudflare API 管理 Tunnel 生命週期
- 控制本地 cloudflared 進程
- 提供安全的內部 API 供其他系統整合

### 1.2 設計原則

| 原則 | 說明 |
|------|------|
| 最小權限 | 每個元件只擁有必要的權限 |
| 深度防禦 | 多層安全機制，不依賴單一防線 |
| 明確分離 | 控制平面與資料平面分離 |
| 可觀測性 | 完整的日誌、監控、追蹤 |

---

## 2. 系統架構

### 2.1 高層架構圖

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Control Plane                                   │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    Tunnel Manager Server (Go)                    │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │    │
│  │  │   HTTP API  │  │  Cloudflare │  │  Cloudflared Process    │  │    │
│  │  │   Handler   │  │  API Client │  │  Manager                │  │    │
│  │  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │    │
│  │         │                │                     │                 │    │
│  │         └────────────────┼─────────────────────┘                 │    │
│  │                          │                                       │    │
│  │                   ┌──────┴──────┐                               │    │
│  │                   │   Service   │                               │    │
│  │                   │   Layer     │                               │    │
│  │                   └─────────────┘                               │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
         │                    │                         │
         │ HTTPS/mTLS         │ HTTPS                   │ Process
         ▼                    ▼                         ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐
│ Internal Client │  │  Cloudflare API │  │  cloudflared instances      │
│ (gRPC/REST)     │  │  (api.cloudflare│  │  ┌───────┐ ┌───────┐       │
└─────────────────┘  │   .com)         │  │  │ Tunnel│ │ Tunnel│ ...   │
                     └─────────────────┘  │  │   A   │ │   B   │       │
                                          │  └───────┘ └───────┘       │
                                          └─────────────────────────────┘
                                                       │
                                                       │ QUIC (encrypted)
                                                       ▼
                                          ┌─────────────────────────────┐
                                          │  Cloudflare Edge Network    │
                                          └─────────────────────────────┘
```

### 2.2 元件職責

| 元件 | 職責 | 對外介面 |
|------|------|----------|
| HTTP API Handler | 接收處理 REST 請求 | HTTPS :8443 |
| Cloudflare API Client | 與 Cloudflare API 通訊 | Outbound HTTPS |
| Process Manager | 管理 cloudflared 進程生命週期 | 系統進程 |
| Service Layer | 業務邏輯協調 | 內部呼叫 |

---

## 3. 資料結構設計

### 3.1 核心領域模型

```
┌─────────────────────────────────────────────────────────────────┐
│                        Domain Models                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐       1:N       ┌─────────────────┐            │
│  │   Tunnel    │────────────────►│  IngressRule    │            │
│  ├─────────────┤                 ├─────────────────┤            │
│  │ ID          │                 │ Hostname        │            │
│  │ Name        │                 │ Service         │            │
│  │ AccountID   │                 │ OriginRequest   │            │
│  │ Status      │                 └─────────────────┘            │
│  │ CreatedAt   │                                                │
│  │ Token       │       1:N       ┌─────────────────┐            │
│  └─────────────┘────────────────►│  Connection     │            │
│                                  ├─────────────────┤            │
│                                  │ ID              │            │
│  ┌─────────────┐                 │ ColoName        │            │
│  │  Process    │                 │ OriginIP        │            │
│  ├─────────────┤                 │ OpenedAt        │            │
│  │ TunnelID    │                 └─────────────────┘            │
│  │ PID         │                                                │
│  │ State       │                                                │
│  │ StartedAt   │                                                │
│  └─────────────┘                                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 資料結構定義

```go
// ─────────────────────────────────────────────────────────────
// Tunnel - 代表一個 Cloudflare Tunnel
// ─────────────────────────────────────────────────────────────
type Tunnel struct {
    ID        string       `json:"id"`         // UUID, Cloudflare 產生
    Name      string       `json:"name"`       // 使用者定義名稱
    AccountID string       `json:"account_id"` // Cloudflare Account ID
    Status    TunnelStatus `json:"status"`     // inactive | healthy | degraded | down
    CreatedAt time.Time    `json:"created_at"`
    DeletedAt *time.Time   `json:"deleted_at,omitempty"`
    
    // 敏感資料 - 不序列化到一般回應
    Token     string       `json:"-"`
}

type TunnelStatus string
const (
    TunnelStatusInactive TunnelStatus = "inactive"
    TunnelStatusHealthy  TunnelStatus = "healthy"
    TunnelStatusDegraded TunnelStatus = "degraded"
    TunnelStatusDown     TunnelStatus = "down"
)

// ─────────────────────────────────────────────────────────────
// IngressRule - Tunnel 的路由規則
// ─────────────────────────────────────────────────────────────
type IngressRule struct {
    Hostname      string         `json:"hostname,omitempty"` // 空 = catch-all
    Path          string         `json:"path,omitempty"`
    Service       string         `json:"service"`            // http://localhost:8080
    OriginRequest *OriginRequest `json:"originRequest,omitempty"`
}

type OriginRequest struct {
    // 通用選項
    ConnectTimeout   *Duration `json:"connectTimeout,omitempty"`   // 預設 30s
    TLSTimeout       *Duration `json:"tlsTimeout,omitempty"`       // 預設 10s
    TCPKeepAlive     *Duration `json:"tcpKeepAlive,omitempty"`     // 預設 30s
    KeepAliveTimeout *Duration `json:"keepAliveTimeout,omitempty"` // 預設 90s
    
    // HTTP 選項
    HTTPHostHeader          string `json:"httpHostHeader,omitempty"`
    OriginServerName        string `json:"originServerName,omitempty"`
    DisableChunkedEncoding  bool   `json:"disableChunkedEncoding,omitempty"`
    
    // TLS 選項 (僅 HTTPS service 有效)
    NoTLSVerify bool   `json:"noTLSVerify,omitempty"`
    CAPool      string `json:"caPool,omitempty"` // CA 憑證路徑
}

// ─────────────────────────────────────────────────────────────
// TunnelConfiguration - Tunnel 完整配置
// ─────────────────────────────────────────────────────────────
type TunnelConfiguration struct {
    TunnelID string        `json:"tunnel_id"`
    Ingress  []IngressRule `json:"ingress"`
    
    // 全域設定
    WarpRouting *WarpRoutingConfig `json:"warp-routing,omitempty"`
}

type WarpRoutingConfig struct {
    Enabled bool `json:"enabled"`
}

// ─────────────────────────────────────────────────────────────
// Connection - Tunnel 連線資訊
// ─────────────────────────────────────────────────────────────
type Connection struct {
    ID                 string    `json:"id"`
    TunnelID           string    `json:"tunnel_id"`
    ColoName           string    `json:"colo_name"`    // 資料中心代碼
    OriginIP           string    `json:"origin_ip"`
    OpenedAt           time.Time `json:"opened_at"`
    ClientID           string    `json:"client_id"`
    ClientVersion      string    `json:"client_version"`
    IsPendingReconnect bool      `json:"is_pending_reconnect"`
}

// ─────────────────────────────────────────────────────────────
// Process - 本地 cloudflared 進程狀態
// ─────────────────────────────────────────────────────────────
type Process struct {
    TunnelID  string       `json:"tunnel_id"`
    PID       int          `json:"pid"`
    State     ProcessState `json:"state"`
    StartedAt time.Time    `json:"started_at"`
    ExitCode  *int         `json:"exit_code,omitempty"`
    Error     string       `json:"error,omitempty"`
}

type ProcessState string
const (
    ProcessStateStarting ProcessState = "starting"
    ProcessStateRunning  ProcessState = "running"
    ProcessStateStopping ProcessState = "stopping"
    ProcessStateStopped  ProcessState = "stopped"
    ProcessStateFailed   ProcessState = "failed"
)
```

### 3.3 API 請求/回應結構

```go
// ─────────────────────────────────────────────────────────────
// API Request/Response DTOs
// ─────────────────────────────────────────────────────────────

// 建立 Tunnel
type CreateTunnelRequest struct {
    Name    string        `json:"name" validate:"required,min=1,max=64"`
    Ingress []IngressRule `json:"ingress,omitempty"`
}

type CreateTunnelResponse struct {
    Tunnel  *Tunnel `json:"tunnel"`
    Message string  `json:"message,omitempty"`
}

// 列出 Tunnels
type ListTunnelsResponse struct {
    Tunnels []TunnelWithProcess `json:"tunnels"`
    Total   int                 `json:"total"`
}

type TunnelWithProcess struct {
    Tunnel
    Process *Process `json:"process,omitempty"`
}

// 更新配置
type UpdateConfigRequest struct {
    Ingress []IngressRule `json:"ingress" validate:"required,min=1"`
}

// 通用錯誤回應
type ErrorResponse struct {
    Error struct {
        Code    string `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}
```

---

## 4. 元件詳細設計

### 4.1 元件互動圖

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Component Interaction                          │
└──────────────────────────────────────────────────────────────────────┘

  Internal Client                    Tunnel Manager Server
       │                                      │
       │  POST /api/v1/tunnels               │
       │─────────────────────────────────────►│
       │                                      │
       │                    ┌─────────────────┴─────────────────┐
       │                    │                                   │
       │                    ▼                                   │
       │            ┌───────────────┐                          │
       │            │  HTTP Handler │                          │
       │            └───────┬───────┘                          │
       │                    │                                   │
       │                    │ validate & parse                  │
       │                    ▼                                   │
       │            ┌───────────────┐                          │
       │            │ Service Layer │                          │
       │            └───────┬───────┘                          │
       │                    │                                   │
       │        ┌───────────┴───────────┐                      │
       │        │                       │                      │
       │        ▼                       ▼                      │
       │ ┌─────────────┐      ┌─────────────────┐             │
       │ │  CF API     │      │ Process Manager │             │
       │ │  Client     │      │                 │             │
       │ └──────┬──────┘      └────────┬────────┘             │
       │        │                      │                       │
       │        │ CreateTunnel()       │                       │
       │        ▼                      │                       │
       │   Cloudflare API              │                       │
       │        │                      │                       │
       │        │ return Tunnel        │                       │
       │        ▼                      │                       │
       │ ┌─────────────┐               │                       │
       │ │  Get Token  │               │                       │
       │ └──────┬──────┘               │                       │
       │        │                      │                       │
       │        └──────────┬───────────┘                       │
       │                   │                                   │
       │                   │ StartTunnel(token)                │
       │                   ▼                                   │
       │          ┌─────────────────┐                          │
       │          │ exec cloudflared│                          │
       │          └─────────────────┘                          │
       │                   │                                   │
       │◄──────────────────┴───────────────────────────────────┘
       │  Response: Tunnel Created
       │
```

### 4.2 Service Layer 設計

```go
// ─────────────────────────────────────────────────────────────
// Service Layer Interface
// ─────────────────────────────────────────────────────────────

type TunnelService interface {
    // Tunnel 生命週期
    Create(ctx context.Context, req CreateTunnelRequest) (*Tunnel, error)
    Get(ctx context.Context, tunnelID string) (*TunnelWithProcess, error)
    List(ctx context.Context) ([]TunnelWithProcess, error)
    Delete(ctx context.Context, tunnelID string) error
    
    // 配置管理
    UpdateConfig(ctx context.Context, tunnelID string, cfg TunnelConfiguration) error
    GetConfig(ctx context.Context, tunnelID string) (*TunnelConfiguration, error)
    
    // 進程控制
    Start(ctx context.Context, tunnelID string) error
    Stop(ctx context.Context, tunnelID string) error
    Restart(ctx context.Context, tunnelID string) error
    GetStatus(ctx context.Context, tunnelID string) (*Process, error)
}

// ─────────────────────────────────────────────────────────────
// Dependencies (Interface for testability)
// ─────────────────────────────────────────────────────────────

type CloudflareAPIClient interface {
    CreateTunnel(ctx context.Context, name string) (*Tunnel, error)
    GetTunnel(ctx context.Context, tunnelID string) (*Tunnel, error)
    ListTunnels(ctx context.Context) ([]Tunnel, error)
    DeleteTunnel(ctx context.Context, tunnelID string) error
    GetTunnelToken(ctx context.Context, tunnelID string) (string, error)
    UpdateTunnelConfiguration(ctx context.Context, tunnelID string, cfg TunnelConfiguration) error
    GetTunnelConfiguration(ctx context.Context, tunnelID string) (*TunnelConfiguration, error)
    CleanupConnections(ctx context.Context, tunnelID string) error
}

type ProcessManager interface {
    Start(ctx context.Context, tunnelID, token string, opts ProcessOptions) error
    Stop(ctx context.Context, tunnelID string) error
    GetStatus(tunnelID string) (*Process, error)
    ListAll() []Process
    StopAll(ctx context.Context) error
}

type ProcessOptions struct {
    LogLevel   string   // debug | info | warn | error
    Protocol   string   // auto | quic | http2
    EdgeIPVer  string   // auto | 4 | 6
    GracePeriod time.Duration
}
```

### 4.3 狀態機設計

```
┌─────────────────────────────────────────────────────────────────┐
│                    Process State Machine                         │
└─────────────────────────────────────────────────────────────────┘

                         ┌─────────────┐
                         │   (init)    │
                         └──────┬──────┘
                                │
                                │ Start()
                                ▼
                         ┌─────────────┐
              ┌──────────│  Starting   │──────────┐
              │          └──────┬──────┘          │
              │                 │                 │
              │ timeout/error   │ connected       │ exec failed
              │                 ▼                 │
              │          ┌─────────────┐          │
              │          │   Running   │          │
              │          └──────┬──────┘          │
              │                 │                 │
              │     ┌───────────┼───────────┐     │
              │     │           │           │     │
              │     │ Stop()    │ crash     │     │
              │     ▼           ▼           │     │
              │ ┌─────────┐ ┌─────────┐     │     │
              │ │Stopping │ │  Failed │◄────┼─────┘
              │ └────┬────┘ └─────────┘     │
              │      │                      │
              │      │ exited               │
              │      ▼                      │
              │ ┌─────────────┐             │
              └►│   Stopped   │◄────────────┘
                └─────────────┘
```

---

## 5. 安全設計

### 5.1 威脅模型

```
┌─────────────────────────────────────────────────────────────────┐
│                      Threat Model                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐                                                │
│  │  Threats    │                                                │
│  └─────────────┘                                                │
│                                                                  │
│  T1: API Key 洩漏                                               │
│      → 攻擊者可控制所有 Tunnel                                   │
│      → 緩解: Key rotation, IP 白名單, Rate limit                │
│                                                                  │
│  T2: Cloudflare API Token 洩漏                                  │
│      → 攻擊者可操作 Cloudflare 帳號資源                          │
│      → 緩解: 最小權限, 環境變數, Secret Manager                  │
│                                                                  │
│  T3: Tunnel Token 洩漏                                          │
│      → 攻擊者可啟動未授權的 cloudflared                          │
│      → 緩解: Token rotation, 監控異常連線                        │
│                                                                  │
│  T4: 命令注入                                                    │
│      → 透過 Tunnel name 或 config 注入惡意命令                   │
│      → 緩解: 輸入驗證, 參數化命令                                │
│                                                                  │
│  T5: 中間人攻擊                                                  │
│      → 攔截 API 通訊                                             │
│      → 緩解: TLS 1.3, 憑證驗證                                   │
│                                                                  │
│  T6: 未授權存取內部服務                                          │
│      → 透過 Tunnel 存取不應暴露的服務                            │
│      → 緩解: Ingress 白名單, Cloudflare Access                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 安全控制矩陣

| 層級 | 控制措施 | 說明 |
|------|----------|------|
| **網路** | TLS 1.3 | API Server 強制使用 |
| | mTLS (optional) | 雙向憑證驗證 |
| | IP 白名單 | 限制 API 存取來源 |
| **認證** | API Key | 內部 API 認證 |
| | Cloudflare API Token | 外部 API 認證，最小權限 |
| | Tunnel Token | cloudflared 認證 |
| **授權** | RBAC (future) | 角色權限控制 |
| **輸入** | Schema 驗證 | JSON Schema / Go validator |
| | 長度限制 | 防止 DoS |
| | 字元白名單 | Tunnel name 只允許 `[a-zA-Z0-9-_]` |
| **進程** | 最小權限使用者 | cloudflared 獨立使用者 |
| | Seccomp | 系統呼叫限制 |
| | 命令參數化 | 不使用 shell |
| **日誌** | 結構化日誌 | JSON 格式 |
| | 敏感資料遮蔽 | Token 不記錄 |
| | 審計日誌 | 所有變更操作 |
| **監控** | 異常偵測 | 失敗次數、異常 IP |
| | Rate Limiting | 防止暴力破解 |

### 5.3 認證流程

```
┌─────────────────────────────────────────────────────────────────┐
│                   Authentication Flow                            │
└─────────────────────────────────────────────────────────────────┘

  Client                    API Server              Cloudflare
    │                           │                       │
    │ Request + API Key         │                       │
    │──────────────────────────►│                       │
    │                           │                       │
    │                    ┌──────┴──────┐               │
    │                    │ Validate    │               │
    │                    │ API Key     │               │
    │                    │ (constant   │               │
    │                    │  time cmp)  │               │
    │                    └──────┬──────┘               │
    │                           │                       │
    │                    ┌──────┴──────┐               │
    │                    │ Rate Limit  │               │
    │                    │ Check       │               │
    │                    └──────┬──────┘               │
    │                           │                       │
    │                           │ API Token            │
    │                           │──────────────────────►│
    │                           │                       │
    │                           │◄──────────────────────│
    │                           │                       │
    │◄──────────────────────────│                       │
    │ Response                  │                       │
```

### 5.4 秘密管理架構

```
┌─────────────────────────────────────────────────────────────────┐
│                    Secret Management                             │
└─────────────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────────┐
  │                    Secret Sources                            │
  │  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │
  │  │ Environment   │  │ Vault/KMS     │  │ K8s Secrets   │   │
  │  │ Variables     │  │               │  │               │   │
  │  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘   │
  │          │                  │                  │            │
  │          └──────────────────┼──────────────────┘            │
  │                             │                               │
  │                             ▼                               │
  │                    ┌─────────────────┐                      │
  │                    │ Secret Provider │                      │
  │                    │   Interface     │                      │
  │                    └────────┬────────┘                      │
  │                             │                               │
  └─────────────────────────────┼───────────────────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │ Tunnel Manager  │
                       │ Server          │
                       └─────────────────┘

  Secrets:
  ┌────────────────────────────────────────────────────────────┐
  │ CF_API_TOKEN      │ Cloudflare API 認證                    │
  │ API_KEY_HASH      │ 內部 API 認證（已雜湊）                 │
  │ TLS_CERT/KEY      │ HTTPS 憑證                            │
  │ TUNNEL_TOKENS     │ 執行時期取得，不持久化                  │
  └────────────────────────────────────────────────────────────┘
```

---

## 6. 通訊架構

### 6.1 通訊協定總覽

```
┌─────────────────────────────────────────────────────────────────┐
│                  Communication Protocols                         │
└─────────────────────────────────────────────────────────────────┘

  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
  │  Internal   │  HTTPS  │   Tunnel    │  HTTPS  │ Cloudflare  │
  │  Client     │────────►│   Manager   │────────►│ API         │
  └─────────────┘  :8443  └─────────────┘         └─────────────┘
                   TLS 1.3      │
                   API Key      │
                                │
                                │ Process Control
                                ▼
                         ┌─────────────┐
                         │ cloudflared │
                         │ (子進程)    │
                         └──────┬──────┘
                                │
                                │ QUIC (UDP 7844)
                                │ Post-Quantum Encryption
                                ▼
                         ┌─────────────┐         ┌─────────────┐
                         │ Cloudflare  │  HTTPS  │  End User   │
                         │ Edge        │◄────────│             │
                         └─────────────┘         └─────────────┘
```

### 6.2 API 端點設計

```
┌─────────────────────────────────────────────────────────────────┐
│                      API Endpoints                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Base URL: https://<host>:8443/api/v1                           │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Health & Info (No Auth)                                    │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ GET  /health              健康檢查                         │ │
│  │ GET  /version             版本資訊                         │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Tunnel Management (Requires Auth)                          │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ POST   /tunnels           建立 Tunnel                      │ │
│  │ GET    /tunnels           列出所有 Tunnels                 │ │
│  │ GET    /tunnels/:id       取得 Tunnel 詳情                 │ │
│  │ DELETE /tunnels/:id       刪除 Tunnel                      │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Tunnel Configuration (Requires Auth)                       │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ GET    /tunnels/:id/config     取得配置                    │ │
│  │ PUT    /tunnels/:id/config     更新配置                    │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Process Control (Requires Auth)                            │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ POST   /tunnels/:id/start      啟動 cloudflared            │ │
│  │ POST   /tunnels/:id/stop       停止 cloudflared            │ │
│  │ POST   /tunnels/:id/restart    重啟 cloudflared            │ │
│  │ GET    /tunnels/:id/status     取得進程狀態                │ │
│  │ GET    /tunnels/:id/logs       取得日誌 (SSE)              │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 6.3 錯誤碼設計

```
┌─────────────────────────────────────────────────────────────────┐
│                       Error Codes                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Format: <CATEGORY>_<SPECIFIC_ERROR>                            │
│                                                                  │
│  ┌──────────────────────┬──────┬──────────────────────────────┐ │
│  │ Code                 │ HTTP │ Description                  │ │
│  ├──────────────────────┼──────┼──────────────────────────────┤ │
│  │ AUTH_MISSING_KEY     │ 401  │ 缺少 API Key                 │ │
│  │ AUTH_INVALID_KEY     │ 401  │ API Key 無效                 │ │
│  │ AUTH_RATE_LIMITED    │ 429  │ 請求過於頻繁                 │ │
│  ├──────────────────────┼──────┼──────────────────────────────┤ │
│  │ TUNNEL_NOT_FOUND     │ 404  │ Tunnel 不存在                │ │
│  │ TUNNEL_ALREADY_EXISTS│ 409  │ 名稱已存在                   │ │
│  │ TUNNEL_RUNNING       │ 409  │ Tunnel 正在運行              │ │
│  │ TUNNEL_NOT_RUNNING   │ 409  │ Tunnel 未運行                │ │
│  ├──────────────────────┼──────┼──────────────────────────────┤ │
│  │ CONFIG_INVALID       │ 400  │ 配置格式錯誤                 │ │
│  │ CONFIG_MISSING_CATCH │ 400  │ 缺少 catch-all 規則          │ │
│  ├──────────────────────┼──────┼──────────────────────────────┤ │
│  │ CLOUDFLARE_ERROR     │ 502  │ Cloudflare API 錯誤          │ │
│  │ PROCESS_START_FAILED │ 500  │ 進程啟動失敗                 │ │
│  │ INTERNAL_ERROR       │ 500  │ 內部錯誤                     │ │
│  └──────────────────────┴──────┴──────────────────────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. 部署架構

### 7.1 部署拓撲

```
┌─────────────────────────────────────────────────────────────────┐
│                    Deployment Topology                           │
└─────────────────────────────────────────────────────────────────┘

  Production Environment
  ┌─────────────────────────────────────────────────────────────┐
  │                                                              │
  │  ┌──────────────────────────────────────────────────────┐   │
  │  │                 Kubernetes Cluster                    │   │
  │  │                                                       │   │
  │  │  ┌─────────────┐      ┌─────────────┐                │   │
  │  │  │   Pod A     │      │   Pod B     │                │   │
  │  │  │ ┌─────────┐ │      │ ┌─────────┐ │   (HA)        │   │
  │  │  │ │ Manager │ │      │ │ Manager │ │                │   │
  │  │  │ └─────────┘ │      │ └─────────┘ │                │   │
  │  │  │ ┌─────────┐ │      │ ┌─────────┐ │                │   │
  │  │  │ │cloudfla-│ │      │ │cloudfla-│ │                │   │
  │  │  │ │red      │ │      │ │red      │ │                │   │
  │  │  │ └─────────┘ │      │ └─────────┘ │                │   │
  │  │  └─────────────┘      └─────────────┘                │   │
  │  │         │                    │                        │   │
  │  │         └────────┬───────────┘                        │   │
  │  │                  │                                    │   │
  │  │           ┌──────┴──────┐                            │   │
  │  │           │   Service   │                            │   │
  │  │           │ (LoadBalancer)                           │   │
  │  │           └─────────────┘                            │   │
  │  │                                                       │   │
  │  └───────────────────────────────────────────────────────┘   │
  │                                                              │
  │  ┌──────────────────┐  ┌──────────────────┐                 │
  │  │  Vault / KMS     │  │  Monitoring      │                 │
  │  │  (Secrets)       │  │  (Prometheus)    │                 │
  │  └──────────────────┘  └──────────────────┘                 │
  │                                                              │
  └─────────────────────────────────────────────────────────────┘
```

### 7.2 設定檔結構

```yaml
# config.yaml
server:
  listen_addr: ":8443"
  tls:
    cert_file: "/certs/server.crt"
    key_file: "/certs/server.key"
    min_version: "1.3"
  
  # Rate limiting
  rate_limit:
    requests_per_second: 10
    burst: 20

cloudflare:
  # 從環境變數讀取: CF_API_TOKEN
  account_id: "${CF_ACCOUNT_ID}"
  zone_id: "${CF_ZONE_ID}"

cloudflared:
  path: "/usr/local/bin/cloudflared"
  default_options:
    protocol: "quic"
    log_level: "info"
    grace_period: "30s"

logging:
  level: "info"
  format: "json"
  output: "stdout"

# 僅開發環境
development:
  disable_auth: false
  verbose_errors: false
```

---

## 8. 附錄

### 8.1 Cloudflare API 權限需求

| 權限 | 用途 |
|------|------|
| `Cloudflare Tunnel:Edit` | 建立、刪除、修改 Tunnel |
| `Cloudflare Tunnel:Read` | 讀取 Tunnel 資訊 |
| `DNS:Edit` | 建立 CNAME 記錄 (選用) |

### 8.2 cloudflared CLI 參數對照

| 功能 | CLI 參數 | 環境變數 |
|------|----------|----------|
| Token | `--token` | `TUNNEL_TOKEN` |
| 協議 | `--protocol` | `TUNNEL_TRANSPORT_PROTOCOL` |
| 日誌等級 | `--loglevel` | `TUNNEL_LOGLEVEL` |
| 日誌檔案 | `--logfile` | `TUNNEL_LOGFILE` |
| 優雅關閉 | `--grace-period` | `TUNNEL_GRACE_PERIOD` |
| 停用更新 | `--no-autoupdate` | `NO_AUTOUPDATE` |
| Metrics | `--metrics` | `TUNNEL_METRICS` |

### 8.3 待定項目

- [ ] RBAC 權限模型設計
- [ ] Multi-tenant 支援
- [ ] Webhook 通知
- [ ] 自動 Token 輪換排程
- [ ] 與 Cloudflare Access 整合
