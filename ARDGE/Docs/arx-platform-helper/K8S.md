# 遠端支援平台 - 完整架構設計

## 一、系統總覽

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              整體架構                                        │
└─────────────────────────────────────────────────────────────────────────────┘

                         ┌─────────────────────────────────┐
                         │     Kaiden Cloud Platform       │
                         │        (你們的雲服務)            │
                         │                                 │
                         │  ┌───────────────────────────┐  │
                         │  │    Kubernetes Cluster     │  │
                         │  │                           │  │
                         │  │  ┌─────┐ ┌─────┐ ┌─────┐  │  │
                         │  │  │ API │ │Help │ │Ticket│  │  │
                         │  │  │ GW  │ │ Svc │ │ Svc │  │  │
                         │  │  └──┬──┘ └──┬──┘ └──┬──┘  │  │
                         │  │     │       │       │     │  │
                         │  │  ┌──┴───────┴───────┴──┐  │  │
                         │  │  │   PostgreSQL / Redis │  │  │
                         │  │  └─────────────────────┘  │  │
                         │  └───────────────────────────┘  │
                         │                                 │
                         │         Cloudflare API          │
                         └───────────────┬─────────────────┘
                                         │
                              Cloudflare Edge Network
                              (Tunnel + Access + DNS)
                                         │
          ┌──────────────────────────────┼──────────────────────────────┐
          │                              │                              │
          ▼                              ▼                              ▼
┌─────────────────────┐      ┌─────────────────────┐      ┌─────────────────────┐
│   Customer A Site   │      │   Customer B Site   │      │   Customer C Site   │
│                     │      │                     │      │                     │
│  ┌───────────────┐  │      │  ┌───────────────┐  │      │  ┌───────────────┐  │
│  │  K8S Cluster  │  │      │  │  K8S Cluster  │  │      │  │  K8S Cluster  │  │
│  │               │  │      │  │               │  │      │  │               │  │
│  │ ┌───────────┐ │  │      │  │ ┌───────────┐ │  │      │  │ ┌───────────┐ │  │
│  │ │  Kaiden   │ │  │      │  │ │  Kaiden   │ │  │      │  │ │  Kaiden   │ │  │
│  │ │  Agent    │ │  │      │  │ │  Agent    │ │  │      │  │ │  Agent    │ │  │
│  │ │(DaemonSet)│ │  │      │  │ │(DaemonSet)│ │  │      │  │ │(DaemonSet)│ │  │
│  │ └───────────┘ │  │      │  │ └───────────┘ │  │      │  │ └───────────┘ │  │
│  │               │  │      │  │               │  │      │  │               │  │
│  │ Node1  Node2  │  │      │  │ Node1  Node2  │  │      │  │ Node1  Node2  │  │
│  │ Node3  Node4  │  │      │  │ Node3  ...    │  │      │  │ ...          │  │
│  └───────────────┘  │      │  └───────────────┘  │      │  └───────────────┘  │
└─────────────────────┘      └─────────────────────┘      └─────────────────────┘
```

---

## 二、雲端服務架構 (Kaiden Cloud)

### 2.1 服務組件

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Kaiden Cloud - K8S Namespace                         │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              Ingress (Cloudflare)                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    │                  │                  │
                    ▼                  ▼                  ▼
            ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
            │  API Gateway │   │   Web App    │   │   Support    │
            │  (Kong/     │   │  (Dashboard) │   │   Portal     │
            │   Traefik)   │   │              │   │  (Engineer)  │
            └──────┬───────┘   └──────────────┘   └──────────────┘
                   │
     ┌─────────────┼─────────────┬─────────────────┐
     │             │             │                 │
     ▼             ▼             ▼                 ▼
┌─────────┐  ┌──────────┐  ┌──────────┐    ┌─────────────┐
│ Cluster │  │  Ticket  │  │  Tunnel  │    │  Recording  │
│ Service │  │  Service │  │  Service │    │  Service    │
│         │  │          │  │          │    │  (Optional) │
└────┬────┘  └────┬─────┘  └────┬─────┘    └──────┬──────┘
     │            │             │                 │
     └────────────┴─────────────┴─────────────────┘
                          │
          ┌───────────────┼───────────────┐
          │               │               │
          ▼               ▼               ▼
    ┌──────────┐   ┌──────────┐   ┌──────────────┐
    │PostgreSQL│   │  Redis   │   │ Object Store │
    │          │   │ (Cache/  │   │ (S3/R2)      │
    │          │   │  PubSub) │   │ (Recordings) │
    └──────────┘   └──────────┘   └──────────────┘
```

### 2.2 服務職責

| 服務 | 職責 | 技術選型 |
|------|------|----------|
| **API Gateway** | 路由、認證、限流 | Kong / Traefik |
| **Cluster Service** | 客戶 Cluster 註冊與管理 | Go / Rust |
| **Ticket Service** | 工單 CRUD、狀態流轉 | Go / Rust |
| **Tunnel Service** | Cloudflare Tunnel 動態管理 | Go / Rust |
| **Recording Service** | Session 錄製存儲（可選） | Go + FFmpeg |
| **Web App** | 客戶 Dashboard | React / Vue |
| **Support Portal** | 工程師操作介面 | React / Vue |

---

## 三、客戶端架構 (Customer Site)

### 3.1 K8S 部署模式

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Customer K8S Cluster                                    │
└─────────────────────────────────────────────────────────────────────────────┘

Namespace: kaiden-system
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    Kaiden Agent (DaemonSet)                          │    │
│  │                    每個 Node 一個 Pod                                 │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    Kaiden Controller (Deployment)                    │    │
│  │                    整個 Cluster 一個                                  │    │
│  │                    • 與雲端通訊                                       │    │
│  │                    • 管理 Tunnel 生命週期                             │    │
│  │                    • 協調 Agent                                       │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    ConfigMap / Secret                                │    │
│  │                    • cluster_id                                      │    │
│  │                    • cloud_endpoint                                  │    │
│  │                    • credentials                                     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 組件職責

| 組件 | 部署方式 | 職責 |
|------|----------|------|
| **Kaiden Controller** | Deployment (1 replica) | 與雲端通訊、Tunnel 管理、心跳上報 |
| **Kaiden Agent** | DaemonSet (每 Node 1 個) | 節點資訊收集、SSH 服務、Web Terminal |

### 3.3 支援請求時的 Tunnel 建立

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     Tunnel 建立流程（支援請求時）                             │
└─────────────────────────────────────────────────────────────────────────────┘

                    Kaiden Cloud                          Customer Site
                         │                                      │
  Engineer               │                                      │
     │                   │                                      │
     │ 1. 選擇 Ticket    │                                      │
     │    + Node         │                                      │
     │ ─────────────────>│                                      │
     │                   │                                      │
     │                   │ 2. 透過 WebSocket 通知               │
     │                   │    Controller                        │
     │                   │ ────────────────────────────────────>│
     │                   │                                      │
     │                   │                                      │ 3. Controller
     │                   │                                      │    指示 Agent
     │                   │                                      │    啟動 Tunnel
     │                   │                                      │
     │                   │                       ┌──────────────┴──────────────┐
     │                   │                       │  Agent on Target Node       │
     │                   │                       │  • 啟動 cloudflared         │
     │                   │                       │  • 暴露 SSH (22)            │
     │                   │                       │  • 暴露 Web Desktop (8080)  │
     │                   │                       └──────────────┬──────────────┘
     │                   │                                      │
     │                   │ 4. 回報 Tunnel 已建立                │
     │                   │    + public hostname                 │
     │                   │ <────────────────────────────────────│
     │                   │                                      │
     │ 5. 回傳連線資訊   │                                      │
     │ <─────────────────│                                      │
     │                   │                                      │
     │ 6. 透過 Cloudflare Access 連線                           │
     │ ═══════════════════════════════════════════════════════> │
     │                   │                                      │
```

---

## 四、資料模型

### 4.1 核心 Entity

```sql
-- 客戶
CREATE TABLE customers (
    id              UUID PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW()
);

-- 客戶的 K8S Cluster
CREATE TABLE clusters (
    id              UUID PRIMARY KEY,
    customer_id     UUID REFERENCES customers(id),
    name            VARCHAR(255) NOT NULL,
    
    -- 認證
    api_key_hash    VARCHAR(255) NOT NULL,  -- 用於 Controller 連線認證
    
    -- 狀態
    status          VARCHAR(50) DEFAULT 'pending', -- pending/online/offline
    last_heartbeat  TIMESTAMP,
    
    -- Metadata
    k8s_version     VARCHAR(50),
    node_count      INT,
    
    created_at      TIMESTAMP DEFAULT NOW()
);

-- Cluster 內的 Node
CREATE TABLE nodes (
    id              UUID PRIMARY KEY,
    cluster_id      UUID REFERENCES clusters(id),
    name            VARCHAR(255) NOT NULL,       -- K8S node name
    
    -- 狀態
    status          VARCHAR(50) DEFAULT 'unknown', -- ready/not-ready/unknown
    internal_ip     VARCHAR(45),
    
    -- Agent 狀態
    agent_status    VARCHAR(50) DEFAULT 'offline', -- online/offline
    agent_version   VARCHAR(50),
    
    -- 資源資訊
    cpu_capacity    VARCHAR(50),
    memory_capacity VARCHAR(50),
    
    last_seen       TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(cluster_id, name)
);

-- 支援工單
CREATE TABLE tickets (
    id              UUID PRIMARY KEY,
    ticket_number   VARCHAR(50) UNIQUE NOT NULL,  -- 人類可讀編號 TKT-20241124-001
    
    customer_id     UUID REFERENCES customers(id),
    cluster_id      UUID REFERENCES clusters(id),
    
    -- 工單內容
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    priority        VARCHAR(20) DEFAULT 'normal', -- low/normal/high/urgent
    status          VARCHAR(50) DEFAULT 'open',   -- open/in_progress/resolved/closed
    
    -- 指派
    assigned_to     UUID REFERENCES users(id),
    
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    resolved_at     TIMESTAMP
);

-- 支援 Session（一個 Ticket 可能有多次連線）
CREATE TABLE support_sessions (
    id              UUID PRIMARY KEY,
    ticket_id       UUID REFERENCES tickets(id),
    node_id         UUID REFERENCES nodes(id),
    
    -- Tunnel 資訊
    tunnel_id       VARCHAR(255),              -- Cloudflare tunnel ID
    public_hostname VARCHAR(255),              -- xxx.support.kaiden.com
    
    -- 時間
    started_at      TIMESTAMP DEFAULT NOW(),
    ended_at        TIMESTAMP,
    
    -- 操作者
    engineer_id     UUID REFERENCES users(id),
    
    -- 錄影（可選）
    recording_url   VARCHAR(500)
);

-- 工單評論/日誌
CREATE TABLE ticket_comments (
    id              UUID PRIMARY KEY,
    ticket_id       UUID REFERENCES tickets(id),
    
    -- 評論內容
    content         TEXT NOT NULL,
    comment_type    VARCHAR(50) DEFAULT 'comment', -- comment/system/status_change
    
    -- 作者
    author_id       UUID REFERENCES users(id),
    author_type     VARCHAR(50),                   -- engineer/customer/system
    
    created_at      TIMESTAMP DEFAULT NOW()
);

-- 使用者（工程師 + 客戶）
CREATE TABLE users (
    id              UUID PRIMARY KEY,
    email           VARCHAR(255) UNIQUE NOT NULL,
    name            VARCHAR(255) NOT NULL,
    role            VARCHAR(50) NOT NULL,         -- admin/engineer/customer
    customer_id     UUID REFERENCES customers(id), -- NULL for engineers
    created_at      TIMESTAMP DEFAULT NOW()
);
```

### 4.2 Entity 關係

```
┌──────────────────────────────────────────────────────────────────┐
│                        Entity Relationship                        │
└──────────────────────────────────────────────────────────────────┘

┌──────────┐       ┌──────────┐       ┌──────────┐
│ Customer │──1:N──│ Cluster  │──1:N──│   Node   │
└──────────┘       └──────────┘       └──────────┘
     │                  │                  │
     │                  │                  │
     │ 1:N              │ 1:N              │ 1:N
     ▼                  ▼                  ▼
┌──────────┐       ┌──────────┐    ┌──────────────┐
│  User    │       │  Ticket  │────│   Support    │
│(customer)│       │          │1:N │   Session    │
└──────────┘       └──────────┘    └──────────────┘
                        │
                        │ 1:N
                        ▼
                  ┌──────────┐
                  │ Comment  │
                  └──────────┘
```

---

## 五、API 設計

### 5.1 對外 API（Dashboard / Portal）

```yaml
# Cluster 管理
GET    /api/v1/clusters                    # 列出所有 clusters
GET    /api/v1/clusters/{id}               # 取得 cluster 詳情
GET    /api/v1/clusters/{id}/nodes         # 列出 cluster 的 nodes
POST   /api/v1/clusters                    # 註冊新 cluster（回傳 credentials）

# 工單管理
GET    /api/v1/tickets                     # 列出工單（支援 filter）
POST   /api/v1/tickets                     # 建立工單
GET    /api/v1/tickets/{id}                # 取得工單詳情
PATCH  /api/v1/tickets/{id}                # 更新工單
POST   /api/v1/tickets/{id}/comments       # 新增評論

# 支援 Session
POST   /api/v1/tickets/{id}/sessions       # 建立支援 Session（啟動 Tunnel）
DELETE /api/v1/sessions/{id}               # 結束 Session（關閉 Tunnel）
GET    /api/v1/sessions/{id}               # 取得 Session 狀態

# 錄影（可選）
GET    /api/v1/sessions/{id}/recording     # 取得錄影 URL
```

### 5.2 Agent API（Controller ↔ Cloud）

```yaml
# WebSocket 連線
WS     /api/v1/agent/connect               # Controller 長連線

# WebSocket Messages
## Cloud → Controller
{
  "type": "start_tunnel",
  "payload": {
    "session_id": "xxx",
    "node_name": "node-01",
    "tunnel_token": "xxx",      # Cloudflare tunnel token
    "services": ["ssh", "web"]  # 要暴露的服務
  }
}

{
  "type": "stop_tunnel",
  "payload": {
    "session_id": "xxx"
  }
}

## Controller → Cloud
{
  "type": "heartbeat",
  "payload": {
    "cluster_id": "xxx",
    "nodes": [
      {"name": "node-01", "status": "ready", "agent_status": "online"},
      {"name": "node-02", "status": "ready", "agent_status": "online"}
    ]
  }
}

{
  "type": "tunnel_ready",
  "payload": {
    "session_id": "xxx",
    "public_hostname": "sess-xxx.support.kaiden.com"
  }
}

{
  "type": "tunnel_error",
  "payload": {
    "session_id": "xxx",
    "error": "Failed to start cloudflared"
  }
}
```

---

## 六、Cloudflare 整合

### 6.1 使用的 Cloudflare 服務

| 服務 | 用途 |
|------|------|
| **Cloudflare Tunnel** | 建立安全通道到客戶 Node |
| **Cloudflare Access** | 工程師身份驗證（IdP 整合） |
| **Cloudflare DNS** | 動態建立 session subdomain |
| **Cloudflare R2** | 儲存錄影檔案（可選） |

### 6.2 Tunnel 動態建立流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Tunnel Service 運作流程                                 │
└─────────────────────────────────────────────────────────────────────────────┘

Tunnel Service                 Cloudflare API               Customer Agent
      │                              │                            │
      │ 1. Create Tunnel             │                            │
      │    POST /tunnels             │                            │
      │ ────────────────────────────>│                            │
      │                              │                            │
      │ 2. tunnel_id + credentials   │                            │
      │ <────────────────────────────│                            │
      │                              │                            │
      │ 3. Create DNS Record         │                            │
      │    sess-xxx.support.kaiden.com                            │
      │ ────────────────────────────>│                            │
      │                              │                            │
      │ 4. Configure Access Policy   │                            │
      │    (只允許特定 engineer)      │                            │
      │ ────────────────────────────>│                            │
      │                              │                            │
      │ 5. 透過 WebSocket 傳送 tunnel_token 給 Controller         │
      │ ───────────────────────────────────────────────────────────>
      │                              │                            │
      │                              │              6. 啟動 cloudflared
      │                              │                 with token │
      │                              │                            │
      │                              │ 7. Tunnel Connected        │
      │                              │ <══════════════════════════│
      │                              │                            │
```

### 6.3 Access Policy 設定

```json
{
  "name": "Support Session sess-xxx",
  "decision": "allow",
  "include": [
    {
      "email": {
        "email": "engineer@kaiden.com"
      }
    }
  ],
  "require": [
    {
      "login_method": ["otp", "google"]
    }
  ]
}
```

---

## 七、部署架構

### 7.1 雲端 K8S 部署

```yaml
# 建議的 namespace 結構
namespaces:
  - kaiden-system      # 核心服務
  - kaiden-monitoring  # Prometheus, Grafana
  - kaiden-logging     # ELK / Loki

# 核心服務部署
deployments:
  - name: api-gateway
    replicas: 2
    resources:
      cpu: 500m
      memory: 512Mi
      
  - name: cluster-service
    replicas: 2
    resources:
      cpu: 500m
      memory: 512Mi
      
  - name: ticket-service
    replicas: 2
    resources:
      cpu: 500m
      memory: 512Mi
      
  - name: tunnel-service
    replicas: 2
    resources:
      cpu: 1000m      # Tunnel 管理較吃資源
      memory: 1Gi
      
  - name: web-app
    replicas: 2
    resources:
      cpu: 200m
      memory: 256Mi

# 資料庫
statefulsets:
  - name: postgresql
    replicas: 1        # 或用 managed service
    storage: 100Gi
    
  - name: redis
    replicas: 1
    storage: 10Gi
```

### 7.2 客戶端 K8S 部署

```yaml
# Kaiden Controller
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kaiden-controller
  namespace: kaiden-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kaiden-controller
  template:
    spec:
      containers:
      - name: controller
        image: kaiden/controller:latest
        env:
        - name: CLUSTER_ID
          valueFrom:
            secretKeyRef:
              name: kaiden-credentials
              key: cluster_id
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: kaiden-credentials
              key: api_key
        - name: CLOUD_ENDPOINT
          value: "wss://api.kaiden.com/agent/connect"

---
# Kaiden Agent (DaemonSet)
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
    spec:
      hostNetwork: true        # 需要存取 host 網路
      hostPID: true            # 需要存取 host 程序
      containers:
      - name: agent
        image: kaiden/agent:latest
        securityContext:
          privileged: true     # 需要特權模式
        volumeMounts:
        - name: host-root
          mountPath: /host
          readOnly: true
      volumes:
      - name: host-root
        hostPath:
          path: /
```

---

## 八、完整流程

### 8.1 Cluster 註冊流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Cluster 註冊流程                                     │
└─────────────────────────────────────────────────────────────────────────────┘

1. 在 Dashboard 建立 Cluster
   └── 系統產生 cluster_id + api_key
   
2. 下載安裝 YAML
   └── 包含 cluster_id, api_key, cloud_endpoint

3. 在客戶 K8S 執行
   $ kubectl apply -f kaiden-install.yaml
   
4. Controller 啟動，連線到 Cloud
   └── WebSocket 長連線
   └── 定期心跳 + Node 狀態上報
   
5. Dashboard 顯示 Cluster Online
```

### 8.2 支援請求完整流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         支援請求完整流程                                     │
└─────────────────────────────────────────────────────────────────────────────┘

1. 客戶建立工單
   POST /api/v1/tickets
   {
     "cluster_id": "xxx",
     "title": "Node-02 無法啟動服務",
     "description": "..."
   }
   
2. 工程師查看工單，選擇目標 Node，啟動 Session
   POST /api/v1/tickets/{id}/sessions
   {
     "node_id": "node-02-uuid"
   }
   
3. Tunnel Service:
   a. 呼叫 Cloudflare API 建立 Tunnel
   b. 建立 DNS Record
   c. 設定 Access Policy
   d. 透過 WebSocket 通知 Controller
   
4. Controller 收到指令:
   a. 找到目標 Node 的 Agent
   b. 指示 Agent 啟動 cloudflared + SSH
   
5. Agent 回報 Tunnel Ready
   
6. 工程師透過瀏覽器連線:
   https://sess-xxx.support.kaiden.com
   └── Cloudflare Access 驗證
   └── 進入 Web Terminal 或 SSH
   
7. 工程師完成支援，結束 Session
   DELETE /api/v1/sessions/{id}
   
8. 系統清理:
   a. 停止 cloudflared
   b. 刪除 DNS Record
   c. 刪除 Tunnel
   d. 儲存錄影（如果有）
   
9. 工程師在工單加入評論，關閉工單
```

---

## 九、技術選型建議

| 層面 | 建議 | 原因 |
|------|------|------|
| **雲端語言** | Go | K8S 生態、效能、並發 |
| **客戶端 Agent** | Go / Rust | 輕量、跨平台、系統程式 |
| **前端** | React + TypeScript | 成熟、生態好 |
| **資料庫** | PostgreSQL | 可靠、功能完整 |
| **快取/PubSub** | Redis | WebSocket 狀態同步 |
| **API Gateway** | Kong / Traefik | K8S 原生支援 |
| **監控** | Prometheus + Grafana | K8S 標準 |
| **日誌** | Loki + Grafana | 輕量、與 K8S 整合好 |
