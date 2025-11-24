# 簡化設計 - Remote Help System

## 你的需求流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              使用流程                                        │
└─────────────────────────────────────────────────────────────────────────────┘

客戶端                                              工程師端
────────                                            ────────

1. 客戶打開 Helper App
   (Web UI)
        │
        ▼
2. 點「建立支援請求」
   → 用 fake account 建立 ticket
   → 拿到 ticket_id
        │
        ▼
3. 點「開啟遠端連線」
   → 用 ticket_id 建立 tunnel
   → 啟動 cloudflared
   → 顯示「等待工程師連線...」
        │                                    4. 工程師打開 Dashboard
        │                                       看到 ticket 列表
        │                                            │
        │                                            ▼
        │                                    5. 看到這個 ticket
        │                                       狀態：Tunnel 已開啟
        │                                       連線資訊：
        │                                       • URL: xxx.support.kaiden.com
        │                                       • SSH: 可用
        │                                       • Web: 可用
        │                                            │
        │                                            ▼
        │                                    6. 點擊連線
        │         ┌──────────────────────────────────┘
        │         │
        ▼         ▼
   ════════════════════════════
   ║  Cloudflare Tunnel 連線  ║
   ════════════════════════════
```

---

## 極簡架構

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              系統架構                                        │
└─────────────────────────────────────────────────────────────────────────────┘

                              Kaiden Server
                         (一個簡單的 API 服務)
                        ┌─────────────────────┐
                        │                     │
                        │  • Ticket API       │
                        │  • Tunnel API       │
                        │  • Cloudflare 整合  │
                        │  • SQLite DB        │
                        │                     │
                        └──────────┬──────────┘
                                   │
                          Cloudflare API
                          (建立 Tunnel/DNS)
                                   │
                ┌──────────────────┴──────────────────┐
                │                                     │
                ▼                                     ▼
┌───────────────────────────┐          ┌───────────────────────────┐
│      客戶端                │          │      工程師端              │
│                           │          │                           │
│  ┌─────────────────────┐  │          │  ┌─────────────────────┐  │
│  │    Helper App       │  │          │  │    Dashboard        │  │
│  │    (Web UI)         │  │          │  │    (Web UI)         │  │
│  │                     │  │          │  │                     │  │
│  │  • 建立 Ticket      │  │          │  │  • 查看 Tickets     │  │
│  │  • 啟動 Tunnel      │  │          │  │  • 看連線資訊       │  │
│  │  • 顯示狀態         │  │          │  │  • 點擊連線         │  │
│  └─────────────────────┘  │          │  └─────────────────────┘  │
│                           │          │                           │
│  ┌─────────────────────┐  │          │                           │
│  │    cloudflared      │  │          │                           │
│  │    (Tunnel Client)  │──┼──────────┼───► Cloudflare ◄──────────┤
│  └─────────────────────┘  │          │                           │
│                           │          │                           │
│  客戶的機器               │          │  工程師的瀏覽器           │
└───────────────────────────┘          └───────────────────────────┘
```

---

## 資料模型（極簡）

```sql
-- Ticket（支援請求）
CREATE TABLE tickets (
    id              TEXT PRIMARY KEY,       -- UUID
    
    -- Fake account（Phase 1 簡化）
    customer_name   TEXT NOT NULL,          -- 客戶名稱（手動輸入）
    customer_email  TEXT,                   -- 客戶 Email（可選）
    
    -- 問題描述
    title           TEXT NOT NULL,          -- 問題標題
    description     TEXT,                   -- 問題描述
    
    -- 狀態
    status          TEXT DEFAULT 'open',    -- open / tunnel_active / resolved / closed
    
    -- Tunnel 資訊（當 tunnel 開啟時填入）
    tunnel_id       TEXT,                   -- Cloudflare tunnel ID
    tunnel_url      TEXT,                   -- https://xxx.support.kaiden.com
    tunnel_token    TEXT,                   -- 給 cloudflared 用的 token
    
    -- 時間
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    tunnel_started_at DATETIME,
    resolved_at     DATETIME
);
```

---

## API 設計

```yaml
# ========================================
# Ticket API（客戶 + 工程師共用）
# ========================================

# 建立 Ticket（客戶用）
POST /api/tickets
Body: {
  "customer_name": "王小明",
  "customer_email": "wang@example.com",  # 可選
  "title": "系統無法啟動",
  "description": "開機後卡在 loading..."
}
Response: {
  "id": "ticket-uuid-xxx",
  "status": "open",
  ...
}

# 查看 Ticket 列表（工程師用）
GET /api/tickets
Response: [
  {
    "id": "ticket-uuid-xxx",
    "customer_name": "王小明",
    "title": "系統無法啟動",
    "status": "tunnel_active",
    "tunnel_url": "https://t-xxx.support.kaiden.com",
    "created_at": "..."
  },
  ...
]

# 查看單一 Ticket
GET /api/tickets/{id}
Response: {
  "id": "ticket-uuid-xxx",
  "customer_name": "王小明",
  "title": "系統無法啟動",
  "status": "tunnel_active",
  "tunnel_url": "https://t-xxx.support.kaiden.com",
  "tunnel_ssh_command": "ssh support@t-xxx.support.kaiden.com",
  ...
}

# ========================================
# Tunnel API（客戶端用）
# ========================================

# 為 Ticket 建立 Tunnel
POST /api/tickets/{id}/tunnel
Response: {
  "tunnel_id": "cf-tunnel-xxx",
  "tunnel_token": "eyJ...",           # 給 cloudflared 用
  "tunnel_url": "https://t-xxx.support.kaiden.com"
}

# 關閉 Tunnel
DELETE /api/tickets/{id}/tunnel
Response: { "success": true }

# ========================================
# 狀態更新
# ========================================

# 更新 Ticket 狀態（工程師用）
PATCH /api/tickets/{id}
Body: {
  "status": "resolved"  # 或 "closed"
}
```

---

## 完整流程 Sequence

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         完整流程                                             │
└─────────────────────────────────────────────────────────────────────────────┘

Customer          Helper App          Server           Cloudflare        Engineer
    │                 │                  │                  │                │
    │ 1. 打開 App     │                  │                  │                │
    │────────────────>│                  │                  │                │
    │                 │                  │                  │                │
    │ 2. 填寫問題     │                  │                  │                │
    │    點「建立」   │                  │                  │                │
    │────────────────>│                  │                  │                │
    │                 │                  │                  │                │
    │                 │ 3. POST /api/tickets               │                │
    │                 │ { customer_name, title }           │                │
    │                 │─────────────────>│                  │                │
    │                 │                  │                  │                │
    │                 │ 4. { ticket_id } │                  │                │
    │                 │<─────────────────│                  │                │
    │                 │                  │                  │                │
    │ 5. 顯示 ticket  │                  │                  │                │
    │    點「開啟連線」                  │                  │                │
    │────────────────>│                  │                  │                │
    │                 │                  │                  │                │
    │                 │ 6. POST /api/tickets/{id}/tunnel   │                │
    │                 │─────────────────>│                  │                │
    │                 │                  │                  │                │
    │                 │                  │ 7. Create Tunnel │                │
    │                 │                  │─────────────────>│                │
    │                 │                  │                  │                │
    │                 │                  │ 8. Create DNS    │                │
    │                 │                  │─────────────────>│                │
    │                 │                  │                  │                │
    │                 │ 9. { tunnel_token, tunnel_url }    │                │
    │                 │<─────────────────│                  │                │
    │                 │                  │                  │                │
    │                 │ 10. 啟動 cloudflared               │                │
    │                 │     with tunnel_token              │                │
    │                 │                  │                  │                │
    │                 │ 11. Tunnel Connected ══════════════>│                │
    │                 │                  │                  │                │
    │ 12. 顯示        │                  │                  │                │
    │ 「等待連線中...」                  │                  │                │
    │<────────────────│                  │                  │                │
    │                 │                  │                  │                │
    │                 │                  │                  │ 13. 打開 Dashboard
    │                 │                  │                  │<───────────────│
    │                 │                  │                  │                │
    │                 │                  │ 14. GET /api/tickets              │
    │                 │                  │<─────────────────────────────────│
    │                 │                  │                  │                │
    │                 │                  │ 15. 回傳 tickets │                │
    │                 │                  │      (含 tunnel_url)             │
    │                 │                  │─────────────────────────────────>│
    │                 │                  │                  │                │
    │                 │                  │                  │ 16. 看到 ticket
    │                 │                  │                  │     點擊連線
    │                 │                  │                  │                │
    │                 │                  │                  │<───────────────│
    │                 │                  │                  │                │
    │                 │ 17. 透過 Cloudflare 連線 ◄══════════════════════════│
    │                 │                  │                  │                │
    │ 18. 連線成功！  │                  │                  │                │
    │<════════════════════════════════════════════════════════════════════>│
    │                 │                  │                  │                │
```

---

## 組件說明

### 1. Kaiden Server

```
職責：
• 管理 Tickets
• 呼叫 Cloudflare API 建立/刪除 Tunnel
• 提供 API 給 Helper App 和 Dashboard

技術：
• Go 或 Rust
• SQLite（簡單）
• Cloudflare API Client
```

### 2. Helper App（客戶端）

```
職責：
• 顯示 UI 讓客戶填寫問題
• 呼叫 Server API 建立 Ticket
• 呼叫 Server API 建立 Tunnel
• 執行 cloudflared（用拿到的 token）
• 顯示連線狀態

技術：
• Tauri App（你已經有的 arx-discovery）
• 或純 Web App + 命令列工具
```

### 3. Dashboard（工程師端）

```
職責：
• 列出所有 Tickets
• 顯示哪些有開啟 Tunnel
• 提供連線資訊（URL, SSH command）
• 更新 Ticket 狀態

技術：
• 簡單的 Web App
• React / Vue / 甚至純 HTML + JS
```

---

## 階段演進

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         階段演進                                             │
└─────────────────────────────────────────────────────────────────────────────┘

現在 (Phase 1)
──────────────
• Fake account（客戶填名字 + Email）
• Ticket 存在 Server 的 SQLite
• 工程師用 client tool 或瀏覽器連線

之後 (Phase 2) - K8S
────────────────────
• Helper App 可以選擇要連哪個 Node
• 一個 Ticket 可能開多個 Tunnel（多 Node）

更之後 (Phase 3) - 雲服務
─────────────────────────
• Fake account → 真的 Account System
• Ticket 整合到雲端系統
• 工程師在雲端 Ticket 頁面直接連線
• 可能加入錄影、評論等功能
```

---

## 你需要開發的東西

```
Phase 1 開發項目
────────────────

┌─────────────────────────────────────────────────────────────────┐
│ 1. Kaiden Server                                                │
│    • Ticket CRUD API                                            │
│    • Tunnel 建立/刪除（呼叫 Cloudflare API）                     │
│    • SQLite 存儲                                                │
│    預估：1-2 週                                                  │
├─────────────────────────────────────────────────────────────────┤
│ 2. Helper App 修改                                              │
│    • 新增「支援請求」頁面                                        │
│    • 整合 Tunnel 啟動（執行 cloudflared）                        │
│    • 顯示連線狀態                                               │
│    預估：1 週                                                    │
├─────────────────────────────────────────────────────────────────┤
│ 3. 簡易 Dashboard                                               │
│    • Ticket 列表頁                                              │
│    • Ticket 詳情頁（含連線資訊）                                 │
│    預估：3-5 天                                                  │
├─────────────────────────────────────────────────────────────────┤
│ 4. 測試 + 部署                                                  │
│    • 端到端測試                                                  │
│    • Server 部署（你們的機器或 cloud VM）                        │
│    預估：3-5 天                                                  │
└─────────────────────────────────────────────────────────────────┘

總計：約 3-4 週
```
