# 三階段開發路徑

## 總覽

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           開發階段總覽                                       │
└─────────────────────────────────────────────────────────────────────────────┘

Phase 1                    Phase 2                    Phase 3
單機核心                    K8S 化                     雲服務整合
────────────────────────────────────────────────────────────────────────────

┌─────────┐              ┌─────────────┐            ┌─────────────────┐
│ Server  │              │   Server    │            │  Kaiden Cloud   │
│ (Local) │              │ (可部署K8S) │            │  (Multi-tenant) │
└────┬────┘              └──────┬──────┘            └────────┬────────┘
     │                          │                            │
Cloudflare               Cloudflare                   Cloudflare
     │                          │                            │
┌────┴────┐              ┌──────┴──────┐            ┌────────┴────────┐
│  Agent  │              │ K8S Cluster │            │   Customer A    │
│ (單機)  │              │ ┌─────────┐ │            │   K8S Cluster   │
└─────────┘              │ │DaemonSet│ │            ├─────────────────┤
                         │ │ Agent   │ │            │   Customer B    │
                         │ └─────────┘ │            │   K8S Cluster   │
                         └─────────────┘            └─────────────────┘

功能：                     新增功能：                  新增功能：
• Tunnel 建立             • 多 Node 管理              • 多租戶
• SSH 連線                • Node 選擇                 • Account System
• Web Desktop             • Cluster 狀態              • Ticket System
• Session 記錄            • K8S 部署方式              • 客戶 Portal
                                                     • 計費（可選）
```

---

## Phase 1：單機核心功能

### 1.1 目標

```
驗證核心流程：
Engineer → Server → Cloudflare → Agent → Device

最小組件：
• kaiden-server（跑在你們的機器）
• kaiden-agent（跑在客戶單機）
```

### 1.2 架構

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Phase 1 架構                                         │
└─────────────────────────────────────────────────────────────────────────────┘

    你們這邊                                              客戶那邊
┌───────────────────┐                              ┌───────────────────┐
│                   │                              │                   │
│  ┌─────────────┐  │                              │  ┌─────────────┐  │
│  │   kaiden    │  │                              │  │   kaiden    │  │
│  │   server    │  │        Cloudflare            │  │   agent     │  │
│  │             │  │                              │  │             │  │
│  │  ┌───────┐  │  │    ┌───────────────┐         │  │  ┌───────┐  │  │
│  │  │ API   │◄─┼──┼────┤  Tunnel       ├─────────┼──┼─►│Tunnel │  │  │
│  │  └───────┘  │  │    │  (動態建立)    │         │  │  │Client │  │  │
│  │  ┌───────┐  │  │    └───────────────┘         │  │  └───────┘  │  │
│  │  │ WS Hub│◄─┼──┼─────────────────────────────►┼──┼──│ WS    │  │  │
│  │  └───────┘  │  │    (Agent 長連線)            │  │  └───────┘  │  │
│  │  ┌───────┐  │  │                              │  │  ┌───────┐  │  │
│  │  │SQLite │  │  │    ┌───────────────┐         │  │  │ SSH   │  │  │
│  │  └───────┘  │  │    │  Access       │         │  │  │ WebD  │  │  │
│  └─────────────┘  │    │  (身份驗證)    │         │  │  └───────┘  │  │
│                   │    └───────────────┘         │  └─────────────┘  │
│  ┌─────────────┐  │           │                  │                   │
│  │  Engineer   │  │           │                  │  Linux Server     │
│  │  Browser    │◄─┼───────────┘                  │                   │
│  └─────────────┘  │                              └───────────────────┘
│                   │
│  你的開發機/雲VM   │
└───────────────────┘
```

### 1.3 資料模型

```go
// ========================================
// Phase 1 資料模型 - 極簡
// ========================================

// Device - 註冊的設備
type Device struct {
    ID           string    `json:"id" db:"id"`
    Name         string    `json:"name" db:"name"`
    
    // 認證
    SecretHash   string    `json:"-" db:"secret_hash"`
    
    // 狀態
    Status       string    `json:"status" db:"status"`       // online/offline
    LastSeen     time.Time `json:"last_seen" db:"last_seen"`
    
    // 設備資訊（Agent 回報）
    Hostname     string    `json:"hostname" db:"hostname"`
    OS           string    `json:"os" db:"os"`
    IPAddress    string    `json:"ip_address" db:"ip_address"`
    AgentVersion string    `json:"agent_version" db:"agent_version"`
    
    // ===== Phase 2 擴展欄位（先預留，Phase 1 nullable）=====
    ClusterID    *string   `json:"cluster_id,omitempty" db:"cluster_id"`
    NodeName     *string   `json:"node_name,omitempty" db:"node_name"`
    Labels       *string   `json:"labels,omitempty" db:"labels"`  // JSON string
    
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Session - 支援 Session
type Session struct {
    ID           string     `json:"id" db:"id"`
    DeviceID     string     `json:"device_id" db:"device_id"`
    
    // Tunnel
    TunnelID     string     `json:"tunnel_id" db:"tunnel_id"`
    PublicURL    string     `json:"public_url" db:"public_url"`
    
    // 狀態
    Status       string     `json:"status" db:"status"`  // pending/active/ended/failed
    
    // 操作者（Phase 1 簡化，只記名字）
    EngineerName string     `json:"engineer_name" db:"engineer_name"`
    
    // 備註（替代 Ticket）
    Notes        string     `json:"notes" db:"notes"`
    
    // 時間
    StartedAt    time.Time  `json:"started_at" db:"started_at"`
    EndedAt      *time.Time `json:"ended_at,omitempty" db:"ended_at"`
    
    // ===== Phase 3 擴展欄位 =====
    TicketID     *string    `json:"ticket_id,omitempty" db:"ticket_id"`
    EngineerID   *string    `json:"engineer_id,omitempty" db:"engineer_id"`
    RecordingURL *string    `json:"recording_url,omitempty" db:"recording_url"`
}
```

### 1.4 API

```yaml
# ========================================
# Phase 1 API
# ========================================

# ----- 設備管理 -----
POST   /api/v1/devices                    # 產生新設備（回傳 device_id + secret）
GET    /api/v1/devices                    # 列出所有設備
GET    /api/v1/devices/{id}               # 取得設備詳情
DELETE /api/v1/devices/{id}               # 刪除設備

# ----- Agent 連線 -----
WS     /api/v1/ws/agent                   # Agent WebSocket（認證用 device_id + secret）

# ----- Session 管理 -----
POST   /api/v1/sessions                   # 建立 Session
       Body: { "device_id": "xxx", "engineer_name": "John" }
       
GET    /api/v1/sessions                   # 列出 Sessions
GET    /api/v1/sessions/{id}              # 取得 Session
DELETE /api/v1/sessions/{id}              # 結束 Session
PATCH  /api/v1/sessions/{id}              # 更新 Notes
       Body: { "notes": "已修復 XXX 問題" }
```

### 1.5 核心流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Phase 1 完整流程                                          │
└─────────────────────────────────────────────────────────────────────────────┘

=== 設備註冊（一次性）===

1. Engineer 在 Server 建立 Device
   POST /api/v1/devices { "name": "Customer-A-Server" }
   → 回傳 { "device_id": "xxx", "secret": "yyy" }

2. 將 device_id + secret 設定到客戶機器的 Agent
   $ kaiden-agent --device-id xxx --secret yyy --server wss://your-server.com

3. Agent 啟動，連線到 Server
   Agent ──WebSocket──► Server
   → Server 更新 Device 狀態為 online

=== 支援連線 ===

4. Engineer 想要連線，建立 Session
   POST /api/v1/sessions { "device_id": "xxx", "engineer_name": "John" }

5. Server 處理：
   a. 呼叫 Cloudflare API 建立 Tunnel
   b. 建立 DNS Record (sess-xxx.support.kaiden.com)
   c. 設定 Access Policy
   d. 透過 WebSocket 通知 Agent

6. Agent 收到指令：
   a. 啟動 cloudflared (with tunnel token)
   b. 暴露 SSH (22) + Web Desktop (8080)
   c. 回報 Ready

7. Server 回傳給 Engineer：
   { 
     "session_id": "xxx",
     "url": "https://sess-xxx.support.kaiden.com",
     "ssh_port": 22,
     "web_port": 8080
   }

8. Engineer 開瀏覽器：
   https://sess-xxx.support.kaiden.com
   → Cloudflare Access 驗證
   → 連到客戶機器

=== 結束連線 ===

9. Engineer 完成支援，結束 Session
   DELETE /api/v1/sessions/{id}

10. Server 處理：
    a. 通知 Agent 停止 cloudflared
    b. 刪除 Cloudflare Tunnel
    c. 刪除 DNS Record
    d. 更新 Session 狀態
```

### 1.6 目錄結構

```
kaiden/
├── server/                          # Server 程式
│   ├── cmd/
│   │   └── kaiden-server/
│   │       └── main.go
│   │
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go
│   │   │
│   │   ├── api/
│   │   │   ├── handler/
│   │   │   │   ├── device.go
│   │   │   │   └── session.go
│   │   │   ├── middleware/
│   │   │   └── router.go
│   │   │
│   │   ├── model/
│   │   │   ├── device.go
│   │   │   └── session.go
│   │   │
│   │   ├── service/
│   │   │   ├── device_svc.go
│   │   │   └── session_svc.go
│   │   │
│   │   ├── tunnel/                  # Cloudflare 整合
│   │   │   ├── provider.go          # 介面
│   │   │   └── cloudflare.go        # 實作
│   │   │
│   │   ├── websocket/
│   │   │   ├── hub.go
│   │   │   └── agent.go
│   │   │
│   │   └── store/
│   │       ├── store.go             # 介面
│   │       └── sqlite.go            # SQLite 實作
│   │
│   ├── configs/
│   │   └── config.yaml
│   │
│   └── go.mod
│
├── agent/                           # Agent 程式
│   ├── cmd/
│   │   └── kaiden-agent/
│   │       └── main.go
│   │
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go
│   │   │
│   │   ├── connector/               # 與 Server 連線
│   │   │   └── websocket.go
│   │   │
│   │   ├── tunnel/                  # cloudflared 管理
│   │   │   └── manager.go
│   │   │
│   │   └── system/                  # 系統資訊
│   │       └── info.go
│   │
│   └── go.mod
│
├── proto/                           # 共用訊息定義
│   └── messages.go                  # WebSocket 訊息結構
│
└── deploy/
    ├── docker-compose.yaml          # 開發環境
    └── install-agent.sh             # Agent 安裝腳本
```

---

## Phase 2：K8S 化

### 2.1 目標

```
把 Phase 1 的架構升級為可以運行在 K8S 環境：
• Server 可部署到 K8S
• Agent 變成 DaemonSet（每個 Node 一個）
• 支援「選擇 Node」來建立 Session
• 但仍然是「單一 Cluster」概念
```

### 2.2 架構變化

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Phase 1 → Phase 2 變化                                    │
└─────────────────────────────────────────────────────────────────────────────┘

Phase 1                              Phase 2
──────────────────────               ──────────────────────

Server                               Server (可選 K8S 部署)
  └── SQLite                           └── PostgreSQL（推薦）

Agent                                Agent (DaemonSet)
  └── 單一實例                          └── 每個 Node 一個
  └── device_id 識別                    └── device_id + node_name 識別

Device 模型                          Device 模型
  └── 1 Device = 1 機器                └── 1 Device = 1 Node (in Cluster)
                                       └── cluster_id 欄位啟用
                                       └── node_name 欄位啟用
                                       └── labels 欄位啟用

Session 建立                         Session 建立
  └── 指定 device_id                   └── 指定 device_id (= node)
                                       └── 或指定 cluster_id + node 選擇

UI                                   UI
  └── 設備列表                          └── Cluster → Node 層級展示
```

### 2.3 新增/修改的部分

```go
// ========================================
// Phase 2 資料模型變化
// ========================================

// Cluster - 新增（可選）
type Cluster struct {
    ID           string    `json:"id" db:"id"`
    Name         string    `json:"name" db:"name"`
    
    // 認證（Cluster 層級）
    SecretHash   string    `json:"-" db:"secret_hash"`
    
    // 狀態
    Status       string    `json:"status" db:"status"`
    NodeCount    int       `json:"node_count" db:"node_count"`
    
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Device 變化
type Device struct {
    // ... Phase 1 欄位 ...
    
    // Phase 2 啟用
    ClusterID    *string   `json:"cluster_id" db:"cluster_id"`      // FK to Cluster
    NodeName     *string   `json:"node_name" db:"node_name"`        // K8S node name
    Labels       *string   `json:"labels" db:"labels"`              // K8S labels (JSON)
    
    // 新增
    NodeStatus   *string   `json:"node_status" db:"node_status"`    // K8S node status
    Resources    *string   `json:"resources" db:"resources"`        // CPU/Memory (JSON)
}
```

### 2.4 K8S 部署方式

```yaml
# Agent DaemonSet
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kaiden-agent
  namespace: kaiden-system
spec:
  selector:
    matchLabels:
      app: kaiden-agent
  template:
    metadata:
      labels:
        app: kaiden-agent
    spec:
      hostNetwork: true
      hostPID: true
      serviceAccountName: kaiden-agent
      containers:
      - name: agent
        image: kaiden/agent:latest
        securityContext:
          privileged: true
        env:
        - name: KAIDEN_SERVER
          value: "wss://kaiden-server.kaiden-system.svc:8080/ws/agent"
        - name: KAIDEN_CLUSTER_ID
          valueFrom:
            secretKeyRef:
              name: kaiden-credentials
              key: cluster_id
        - name: KAIDEN_SECRET
          valueFrom:
            secretKeyRef:
              name: kaiden-credentials
              key: secret
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
```

### 2.5 API 變化

```yaml
# Phase 2 新增 API

# Cluster 管理（可選，或自動建立）
POST   /api/v1/clusters                   # 建立 Cluster
GET    /api/v1/clusters                   # 列出 Clusters
GET    /api/v1/clusters/{id}              # 取得 Cluster
GET    /api/v1/clusters/{id}/nodes        # 列出 Cluster 的 Nodes

# Device API 變化
GET    /api/v1/devices?cluster_id=xxx     # 可按 Cluster 過濾

# Session API 變化
POST   /api/v1/sessions
       Body: { 
         "device_id": "xxx",              # 仍然用 device_id
         # 或
         "cluster_id": "yyy",             # Cluster + Node 選擇
         "node_name": "node-01",
         "engineer_name": "John" 
       }
```

---

## Phase 3：雲服務整合

### 3.1 目標

```
把 Phase 2 升級為多租戶 SaaS：
• 多客戶（Customer）支援
• Account System（用戶管理、認證）
• Ticket System（工單系統）
• 計費基礎（可選）
```

### 3.2 架構

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Phase 3 架構                                         │
└─────────────────────────────────────────────────────────────────────────────┘

                            Kaiden Cloud
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│   │   Account   │  │   Cluster   │  │   Ticket    │  │   Tunnel    │       │
│   │   Service   │  │   Service   │  │   Service   │  │   Service   │       │
│   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘       │
│          │                │                │                │              │
│          └────────────────┴────────────────┴────────────────┘              │
│                                    │                                        │
│                          ┌─────────┴─────────┐                              │
│                          │    PostgreSQL     │                              │
│                          └───────────────────┘                              │
│                                                                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                         Web Applications                             │   │
│   │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│   │   │  Engineer   │  │  Customer   │  │   Admin     │                 │   │
│   │   │  Portal     │  │  Portal     │  │  Dashboard  │                 │   │
│   │   └─────────────┘  └─────────────┘  └─────────────┘                 │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                              Cloudflare
                                    │
           ┌────────────────────────┼────────────────────────┐
           │                        │                        │
           ▼                        ▼                        ▼
    ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
    │ Customer A  │          │ Customer B  │          │ Customer C  │
    │ K8S Cluster │          │ K8S Cluster │          │ 單機        │
    └─────────────┘          └─────────────┘          └─────────────┘
```

### 3.3 新增資料模型

```go
// ========================================
// Phase 3 新增資料模型
// ========================================

// Customer - 客戶/租戶
type Customer struct {
    ID           string    `json:"id" db:"id"`
    Name         string    `json:"name" db:"name"`
    
    // 聯絡資訊
    Email        string    `json:"email" db:"email"`
    Phone        *string   `json:"phone,omitempty" db:"phone"`
    
    // 狀態
    Status       string    `json:"status" db:"status"`   // active/suspended
    
    // 計費（可選）
    Plan         string    `json:"plan" db:"plan"`       // free/pro/enterprise
    
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// User - 系統用戶
type User struct {
    ID           string    `json:"id" db:"id"`
    Email        string    `json:"email" db:"email"`
    Name         string    `json:"name" db:"name"`
    
    // 角色
    Role         string    `json:"role" db:"role"`       // admin/engineer/customer
    
    // 關聯
    CustomerID   *string   `json:"customer_id,omitempty" db:"customer_id"`  // NULL for engineers
    
    // 認證（或用外部 IdP）
    PasswordHash *string   `json:"-" db:"password_hash"`
    
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Ticket - 工單
type Ticket struct {
    ID           string    `json:"id" db:"id"`
    Number       string    `json:"number" db:"number"`   // TKT-20241124-001
    
    // 關聯
    CustomerID   string    `json:"customer_id" db:"customer_id"`
    ClusterID    *string   `json:"cluster_id,omitempty" db:"cluster_id"`
    
    // 內容
    Title        string    `json:"title" db:"title"`
    Description  string    `json:"description" db:"description"`
    Priority     string    `json:"priority" db:"priority"`   // low/normal/high/urgent
    Status       string    `json:"status" db:"status"`       // open/in_progress/resolved/closed
    
    // 指派
    AssignedTo   *string   `json:"assigned_to,omitempty" db:"assigned_to"`  // User ID
    
    // 時間
    CreatedBy    string    `json:"created_by" db:"created_by"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
    ResolvedAt   *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// TicketComment - 工單評論
type TicketComment struct {
    ID           string    `json:"id" db:"id"`
    TicketID     string    `json:"ticket_id" db:"ticket_id"`
    
    Content      string    `json:"content" db:"content"`
    Type         string    `json:"type" db:"type"`          // comment/system/status_change
    
    AuthorID     string    `json:"author_id" db:"author_id"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
```

### 3.4 Phase 1/2 欄位啟用

```go
// Device - Phase 3 啟用 CustomerID
type Device struct {
    // ... existing fields ...
    
    CustomerID   *string   `json:"customer_id" db:"customer_id"`  // Phase 3 啟用
}

// Cluster - Phase 3 啟用 CustomerID
type Cluster struct {
    // ... existing fields ...
    
    CustomerID   *string   `json:"customer_id" db:"customer_id"`  // Phase 3 啟用
}

// Session - Phase 3 欄位啟用
type Session struct {
    // ... existing fields ...
    
    TicketID     *string   `json:"ticket_id" db:"ticket_id"`       // Phase 3 啟用
    EngineerID   *string   `json:"engineer_id" db:"engineer_id"`   // Phase 3 啟用
}
```

---

## 介面設計（跨階段不變）

```go
// ========================================
// 核心介面 - 所有階段共用
// ========================================

// TunnelProvider - Tunnel 管理（Phase 1 就實作）
type TunnelProvider interface {
    CreateTunnel(ctx context.Context, req CreateTunnelRequest) (*Tunnel, error)
    DeleteTunnel(ctx context.Context, tunnelID string) error
    GetTunnelStatus(ctx context.Context, tunnelID string) (*TunnelStatus, error)
}

// DeviceStore - 設備存取（Phase 1 實作，Phase 2 擴展）
type DeviceStore interface {
    Create(ctx context.Context, device *Device) error
    GetByID(ctx context.Context, id string) (*Device, error)
    List(ctx context.Context, filter DeviceFilter) ([]Device, error)
    Update(ctx context.Context, device *Device) error
    Delete(ctx context.Context, id string) error
}

// SessionStore - Session 存取
type SessionStore interface {
    Create(ctx context.Context, session *Session) error
    GetByID(ctx context.Context, id string) (*Session, error)
    List(ctx context.Context, filter SessionFilter) ([]Session, error)
    Update(ctx context.Context, session *Session) error
}

// ===== Phase 2 新增 =====

// ClusterStore - Cluster 存取
type ClusterStore interface {
    Create(ctx context.Context, cluster *Cluster) error
    GetByID(ctx context.Context, id string) (*Cluster, error)
    List(ctx context.Context, filter ClusterFilter) ([]Cluster, error)
    Update(ctx context.Context, cluster *Cluster) error
    Delete(ctx context.Context, id string) error
}

// ===== Phase 3 新增 =====

// CustomerStore - 客戶存取
type CustomerStore interface {
    Create(ctx context.Context, customer *Customer) error
    GetByID(ctx context.Context, id string) (*Customer, error)
    List(ctx context.Context, filter CustomerFilter) ([]Customer, error)
    Update(ctx context.Context, customer *Customer) error
}

// UserStore - 用戶存取
type UserStore interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    List(ctx context.Context, filter UserFilter) ([]User, error)
    Update(ctx context.Context, user *User) error
}

// TicketStore - 工單存取
type TicketStore interface {
    Create(ctx context.Context, ticket *Ticket) error
    GetByID(ctx context.Context, id string) (*Ticket, error)
    List(ctx context.Context, filter TicketFilter) ([]Ticket, error)
    Update(ctx context.Context, ticket *Ticket) error
    
    // Comments
    AddComment(ctx context.Context, comment *TicketComment) error
    ListComments(ctx context.Context, ticketID string) ([]TicketComment, error)
}

// AuthProvider - 認證（Phase 3）
type AuthProvider interface {
    Authenticate(ctx context.Context, email, password string) (*User, error)
    ValidateToken(ctx context.Context, token string) (*User, error)
}
```

---

## 配置演進

```yaml
# ========================================
# Phase 1 配置
# ========================================
server:
  port: 8080

database:
  driver: sqlite
  path: ./kaiden.db

cloudflare:
  account_id: ${CF_ACCOUNT_ID}
  api_token: ${CF_API_TOKEN}
  zone_id: ${CF_ZONE_ID}
  domain: support.kaiden.com

# ========================================
# Phase 2 配置（新增）
# ========================================
server:
  port: 8080

database:
  driver: postgres                    # 改用 PostgreSQL
  dsn: postgres://...

cloudflare:
  # ... same ...

kubernetes:
  enabled: true                       # 新增
  in_cluster: true                    # 跑在 K8S 內

# ========================================
# Phase 3 配置（新增）
# ========================================
server:
  port: 8080
  mode: multi-tenant                  # 新增：多租戶模式

database:
  driver: postgres
  dsn: postgres://...

cloudflare:
  # ... same ...

kubernetes:
  enabled: true
  in_cluster: true

auth:
  provider: cloudflare_access         # 新增：認證提供者
  # 或
  provider: internal
  jwt_secret: ${JWT_SECRET}

ticket:
  enabled: true                       # 新增：啟用工單
  
features:
  customer_portal: true               # 新增：客戶 Portal
  recording: false                    # 錄影功能
```

---

## 開發時程建議

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         開發時程                                             │
└─────────────────────────────────────────────────────────────────────────────┘

Phase 1: 單機核心（4-6 週）
├── Week 1-2: Server 基礎 + Cloudflare 整合
├── Week 3-4: Agent 開發 + WebSocket 通訊
├── Week 5: Session 完整流程
└── Week 6: 簡易 UI + 測試

✓ Milestone: 可以透過 Cloudflare Tunnel 連到單一設備

Phase 2: K8S 化（3-4 週）
├── Week 1: Cluster/Node 資料模型
├── Week 2: Agent DaemonSet 改造
├── Week 3: UI 支援 Node 選擇
└── Week 4: K8S 部署 + 測試

✓ Milestone: 可以在 K8S Cluster 中選擇特定 Node 連線

Phase 3: 雲服務（4-6 週）
├── Week 1-2: Customer + User 模型 + Auth
├── Week 3-4: Ticket System
├── Week 5: Customer Portal
└── Week 6: 整合測試 + 部署

✓ Milestone: 多租戶 SaaS 服務
```
