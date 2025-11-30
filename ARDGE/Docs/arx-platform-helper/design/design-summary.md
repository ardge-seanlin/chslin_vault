# Cloudflare Tunnel Manager - 設計總覽

## 文檔索引

| # | 文檔 | 內容 | 用途 |
|---|------|------|------|
| 1 | **System Design Document** | 架構、資料結構、安全、通訊、部署 | 整體設計藍圖 |
| 2 | **Sequence Diagrams** | 元件互動時序圖、狀態流轉、並發控制 | 理解運作流程 |
| 3 | **Interface Contracts** | Go 介面定義、資料模型、錯誤碼 | 實作契約 |
| 4 | **Design Summary** | 本文檔 - 快速總覽 | 快速參考 |

---

## 核心架構

```
┌─────────────────────────────────────────────────────────┐
│                  Tunnel Manager Server                   │
│                                                          │
│  ┌──────────┐    ┌──────────┐    ┌────────────────┐    │
│  │ HTTP API │───►│ Service  │───►│ CF API Client  │────┼──► Cloudflare API
│  │ Handler  │    │  Layer   │    └────────────────┘    │
│  └──────────┘    │          │    ┌────────────────┐    │
│                  │          │───►│ Process Manager│────┼──► cloudflared
│                  └──────────┘    └────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

---

## 關鍵設計決策

### 1. 認證策略

| 層級 | 方式 | 說明 |
|------|------|------|
| 內部 API | API Key + HTTPS | 簡單有效，搭配 IP 白名單 |
| Cloudflare API | API Token | 最小權限原則 |
| cloudflared | Tunnel Token | 執行時取得，不持久化 |

### 2. 通訊協定

```
Client ──HTTPS/TLS1.3──► Manager ──HTTPS──► Cloudflare API
                              │
                              └──Process──► cloudflared ──QUIC──► CF Edge
```

### 3. Token 處理原則

```
❌ 不做：
   - 命令列參數傳遞 Token（會洩漏到 /proc）
   - 持久化 Tunnel Token
   - 日誌記錄 Token

✅ 要做：
   - 環境變數傳遞
   - 執行時從 API 取得
   - 記憶體中短暫保存
```

### 4. Ingress 配置規則

```yaml
# 必須有 catch-all 規則
ingress:
  - hostname: app.example.com
    service: http://localhost:8080    # HTTP 連本地 ✅
  - hostname: secure.example.com  
    service: https://localhost:8443   # HTTPS 連本地 ✅
    originRequest:
      noTLSVerify: true               # 自簽憑證
      # caPool: /path/to/ca.crt       # 或指定 CA（僅 HTTPS 有效）
  - service: http_status:404          # catch-all 必須 ✅
```

---

## 資料模型摘要

### 核心實體

```
Tunnel
├── ID (UUID)
├── Name
├── Status: inactive | healthy | degraded | down
├── CreatedAt
└── [1:N] IngressRule
         ├── Hostname
         ├── Service (http:// | https:// | ssh:// | rdp://)
         └── OriginRequest (連線選項)

Process (本地狀態)
├── TunnelID
├── PID
├── State: idle | starting | running | stopping | stopped | failed
└── StartedAt
```

### 狀態對應

| Cloudflare Status | Process State | 說明 |
|-------------------|---------------|------|
| inactive | idle | Tunnel 存在，未連線 |
| healthy | running | 正常運作 |
| degraded | running | 部分連線異常 |
| down | failed | 全部連線失敗 |
| - | stopped | 主動停止 |

---

## API 端點摘要

```
Health (No Auth)
  GET  /health
  GET  /version

Tunnels (Auth Required)
  POST   /api/v1/tunnels              建立
  GET    /api/v1/tunnels              列表
  GET    /api/v1/tunnels/:id          詳情
  DELETE /api/v1/tunnels/:id          刪除

Configuration
  GET    /api/v1/tunnels/:id/config   取得配置
  PUT    /api/v1/tunnels/:id/config   更新配置

Process Control
  POST   /api/v1/tunnels/:id/start    啟動
  POST   /api/v1/tunnels/:id/stop     停止
  POST   /api/v1/tunnels/:id/restart  重啟
  GET    /api/v1/tunnels/:id/status   狀態
```

---

## 錯誤碼對照

| Code | HTTP | 說明 |
|------|------|------|
| AUTH_MISSING_KEY | 401 | 缺少 API Key |
| AUTH_INVALID_KEY | 401 | API Key 無效 |
| TUNNEL_NOT_FOUND | 404 | Tunnel 不存在 |
| TUNNEL_RUNNING | 409 | 已在運行 |
| TUNNEL_NOT_RUNNING | 409 | 未運行 |
| CONFIG_INVALID | 400 | 配置錯誤 |
| CONFIG_MISSING_CATCHALL | 400 | 缺少 catch-all |
| CLOUDFLARE_ERROR | 502 | Cloudflare API 錯誤 |
| PROCESS_START_FAILED | 500 | 進程啟動失敗 |

---

## 安全檢查清單

### 必要項目

- [ ] **TLS 1.3** - Server 強制使用
- [ ] **API Token 最小權限** - 僅 Tunnel Edit + DNS Edit
- [ ] **Token 環境變數** - 不用命令列參數
- [ ] **輸入驗證** - Tunnel name 限制 `[a-zA-Z0-9-_]`
- [ ] **Rate Limiting** - 防止暴力破解
- [ ] **敏感資料遮蔽** - 日誌不含 Token

### 建議項目

- [ ] IP 白名單
- [ ] mTLS（雙向認證）
- [ ] 審計日誌
- [ ] Token 自動輪換

---

## 實作順序建議

```
Phase 1: 核心功能
├── 1.1 Config 載入（環境變數 + 檔案）
├── 1.2 CloudflareAPIClient 實作
├── 1.3 ProcessManager 實作
└── 1.4 TunnelService 實作

Phase 2: API 層
├── 2.1 HTTP Handler + 路由
├── 2.2 認證 Middleware
├── 2.3 錯誤處理
└── 2.4 Request 驗證

Phase 3: 安全強化
├── 3.1 TLS 配置
├── 3.2 Rate Limiting
├── 3.3 日誌脫敏
└── 3.4 輸入驗證強化

Phase 4: 運維功能
├── 4.1 Health Check
├── 4.2 Metrics (Prometheus)
├── 4.3 Graceful Shutdown
└── 4.4 Docker/K8s 部署配置
```

---

## 快速參考：Cloudflare API

```bash
# 建立 Tunnel
POST https://api.cloudflare.com/client/v4/accounts/{account_id}/cfd_tunnel
Authorization: Bearer {token}
{"name": "my-tunnel", "config_src": "cloudflare"}

# 取得 Token
GET https://api.cloudflare.com/client/v4/accounts/{account_id}/cfd_tunnel/{tunnel_id}/token

# 更新配置
PUT https://api.cloudflare.com/client/v4/accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurations
{"config": {"ingress": [...]}}
```

## 快速參考：cloudflared CLI

```bash
# 使用 Token 運行（推薦）
cloudflared tunnel run --token <TOKEN>

# 或使用環境變數
TUNNEL_TOKEN=<TOKEN> cloudflared tunnel run

# 完整選項
cloudflared tunnel \
  --protocol quic \
  --loglevel info \
  --no-autoupdate \
  --grace-period 30s \
  run --token <TOKEN>
```

---

## 注意事項

1. **Ingress catch-all** - 必須有，否則 Cloudflare 拒絕配置
2. **caPool 僅限 HTTPS** - HTTP service 設定 caPool 無效
3. **Token 有效期** - Tunnel Token 長期有效，但建議定期輪換
4. **連線數量** - 健康的 Tunnel 應有 4 個連線到不同資料中心
