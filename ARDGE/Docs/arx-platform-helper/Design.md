# 遠端支援系統架構設計報告

## 目錄

1. [專案目標](#一專案目標)
2. [技術方案選型](#二技術方案選型)
3. [系統架構](#三系統架構)
4. [Cloudflare Tunnel 介紹](#四cloudflare-tunnel-介紹)
5. [角色分工](#五角色分工)
6. [關鍵功能](#六關鍵功能)
7. [SSH 帳號設計與生命週期](#七ssh-帳號設計與生命週期)
8. [密碼傳遞方案](#八密碼傳遞方案)
9. [抽象層設計（降低移植成本）](#九抽象層設計降低移植成本)
10. [待釐清問題](#十待釐清問題)
11. [潛在風險與問題](#十一潛在風險與問題)
12. [成本估算](#十二成本估算)
13. [時程估算](#十三時程估算)
14. [決策建議](#十四決策建議)
15. [附錄](#十五附錄)

---

## 一、專案目標

### 一句話說明

> 讓客戶按一個按鈕就能讓我們的工程師透過瀏覽器遠端操作設備的 Web Desktop 和 SSH，進行系統維護和除錯。

### 使用場景

```
客戶遇到問題
    ↓
客戶打電話給技術支援
    ↓
客戶在 Web Desktop 內的 Helper 頁面按「啟動支援」
    ↓
Support Engineer 用瀏覽器連入，操作 Web Desktop 和 SSH
    ↓
問題解決，客戶按「停止支援」
    ↓
連線中斷，通道關閉
```

### 支援需求

| 需求 | 說明 |
|------|------|
| 看系統 log | 需要 SSH 存取 |
| 改系統設定 | 需要適當權限 |
| 重啟服務 | 需要 sudo 權限 |
| 操作 Web Desktop | 需要 HTTP 存取 |

---

## 二、技術方案選型

### 業界方案比較

| 廠商 | 技術方案 | 一句話說明 |
|------|----------|-----------|
| **QNAP** | SSH Reverse Tunnel | NAS 主動 SSH 連到 QNAP 伺服器，技術人員反向連入 |
| **Synology** | SSH + 臨時帳號 | 給密碼和識別碼，Synology 建立連線，14天後失效 |
| **TeamViewer** | NAT 穿透 + P2P/Relay | 中央伺服器配對 ID，嘗試 P2P，失敗則中繼 |

### 方案選擇

| 方案 | 代表廠商 | 優點 | 缺點 |
|------|----------|------|------|
| **SSH Reverse Tunnel** | QNAP, Synology | 簡單穩定、完全自控 | 要自建伺服器、無 Browser SSH |
| **NAT 穿透 + P2P** | TeamViewer | 低延遲、省頻寬 | 技術複雜、要自建中繼伺服器 |
| **Cloudflare Tunnel** | 本方案 | 免建伺服器、功能豐富 | 依賴第三方 |

### 建議方案

**前期：Cloudflare Tunnel**
- 零伺服器成本
- 內建 Browser SSH、IdP 整合、Audit Log
- 快速上線

**後期（規模擴大後）：自建**
- 透過抽象層設計，降低移植成本
- 只需實作新的 Provider，不改主要邏輯

---

## 三、系統架構

### 整體架構圖

```
客戶 LAN                                        你們的基礎設施
┌────────────────────────────────┐              ┌────────────────────────┐
│                                │              │                        │
│  客戶設備                       │              │  （可選）後台系統       │
│  ┌──────────────────────────┐  │              │  ┌──────────────────┐  │
│  │ Web Desktop (:8080)      │  │              │  │ Database         │  │
│  │ SSH (:22)                │  │   HTTPS      │  │ • Audit Log      │  │
│  │                          │  │ ◄────────────┼─►│ • 設定管理       │  │
│  │ Helper App               │  │              │  │                  │  │
│  │ ├── TunnelManager ───────┼──┼──────────────┼──►                     │
│  │ ├── AccountManager       │  │              │  └──────────────────┘  │
│  │ └── UI (in Web Desktop)  │  │              │                        │
│  │                          │  │              │                        │
│  └──────────────────────────┘  │              └────────────────────────┘
│                                │
│  ┌──────────────────────────┐  │
│  │ cloudflared (Docker)     │  │
│  │                          │  │
│  └────────────┬─────────────┘  │
│               │                │
└───────────────┼────────────────┘
                │
                │ Tunnel
                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Cloudflare                                      │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐       │
│  │ Tunnel Service   │  │ Access (Auth)    │  │ API              │       │
│  │ • 接收 tunnel     │  │ • IdP 整合       │  │ • Tunnel CRUD    │       │
│  │ • 路由流量        │  │ • Policy 檢查     │  │ • DNS 管理       │        │
│  │ • Browser SSH    │  │ • Session 管理    │  │                  │       │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘       │
└─────────────────────────────────────────────────────────────────────────┘
                                    ▲
                                    │ HTTPS
┌─────────────────────────────────────────────────────────────────────────┐
│                      Support Engineer                                   │
│  • 瀏覽器開 Web Desktop                                                   │
│  • 瀏覽器開 SSH Terminal                                                  │
│  • 不需安裝任何軟體                                                        │
└─────────────────────────────────────────────────────────────────────────┘
```

### 元件說明

| 元件 | 位置 | 功能 |
|------|------|------|
| **Web Desktop** | 客戶設備 | 你們的平台管理介面（既有） |
| **SSH daemon** | 客戶設備 | 系統內建 SSH 服務（既有） |
| **Helper App** | 客戶設備 | 管理 Tunnel 的啟停、帳號管理 |
| **cloudflared** | 客戶設備 (Docker) | Cloudflare 的 tunnel client |
| **Cloudflare** | 雲端 | Tunnel 路由、身份驗證、Browser SSH |

---

## 四、Cloudflare Tunnel 介紹

### 核心概念一句話介紹

| 環節 | 一句話介紹 |
|------|-----------|
| **Cloudflare Tunnel** | 一種讓內部服務安全暴露到網路的技術，不需要公網 IP 或開放防火牆端口 |
| **cloudflared** | 跑在你機器上的輕量程式，負責建立從內部到 Cloudflare 的出站連線 |
| **Connector** | 實際執行中的 cloudflared 程序，一個 Tunnel 可以有多個 Connector 做負載均衡 |
| **Ingress Rules** | 定義流量怎麼路由的規則，決定哪個 hostname 對應到哪個本地服務 |
| **Public Hostname** | 對外公開的網址，使用者透過這個網址存取你的內部服務 |

### 管理模式

| 環節 | 一句話介紹 |
|------|-----------|
| **Remote 模式（Dashboard）** | 設定存在 Cloudflare 雲端，本地只需要一個 Token 就能啟動 Tunnel |
| **Local 模式（設定檔）** | 設定存在本地的 config.yml，適合版本控制和固定部署環境 |
| **Tunnel Token** | 一串加密字串，讓 cloudflared 能夠認證並連上指定的 Tunnel |

### 建立方式

| 環節 | 一句話介紹 |
|------|-----------|
| **Dashboard 建立** | 在 Cloudflare Zero Trust 網頁介面點幾下就能建立 Tunnel，適合手動操作 |
| **API 建立** | 透過 HTTP API 程式化建立 Tunnel，適合自動化和動態場景 |
| **CLI 建立** | 用 `cloudflared tunnel create` 指令建立，適合開發者本地測試 |

### 支援的協定

| 環節 | 一句話介紹 |
|------|-----------|
| **HTTP/HTTPS** | 最基本的用法，把網頁服務暴露到公網 |
| **SSH** | 讓遠端使用者安全連線到伺服器的命令列介面 |
| **VNC** | 讓遠端使用者看到並操作伺服器的圖形桌面（Linux 常用） |
| **RDP** | 讓遠端使用者連線到 Windows 的遠端桌面 |
| **TCP（任意）** | 可以 tunnel 任意 TCP 協定，但需要客戶端配合 |

### SSH 連線方式

| 環節 | 一句話介紹 |
|------|-----------|
| **Browser-rendered SSH** | 使用者開瀏覽器就能看到終端機畫面，不需裝任何東西 |
| **Client-side cloudflared** | 使用者在自己電腦裝 cloudflared，用原生 SSH client 連線 |
| **WARP + Access for Infrastructure** | 使用者裝 WARP client，像在內網一樣直接 SSH，還有細緻的權限控制 |

### 存取控制（Cloudflare Access）

| 環節 | 一句話介紹 |
|------|-----------|
| **Cloudflare Access** | Cloudflare 的 Zero Trust 身份驗證服務，決定誰能存取你的應用 |
| **Access Application** | 定義一個受保護的應用，綁定 hostname 和存取政策 |
| **Access Policy** | 設定誰能進（Allow）、誰不能進（Block）、或用 Service Token 認證 |
| **Identity Provider (IdP)** | 驗證使用者身份的來源，例如 Google、GitHub、Okta 等 |
| **Service Token** | 給自動化程式用的認證方式，不需要人工登入 |

### Browser Rendering

| 環節 | 一句話介紹 |
|------|-----------|
| **Browser Rendering** | Cloudflare 的功能，讓 SSH/VNC/RDP 可以直接在瀏覽器裡操作，不需安裝客戶端 |
| **限制** | 只支援 public hostname，不支援 private IP；email 前綴要符合伺服器帳號名稱 |

---

## 五、角色分工

### 職責總覽

| 角色 | 一句話職責 |
|------|-----------|
| **Cloudflare Manager** | 設定好「舞台」— IdP、Policy、權限，然後持續監控 |
| **Helper Developer** | 寫程式讓客戶按一個按鈕就能建立 Tunnel 並啟動連線 |
| **Support Engineer** | 開瀏覽器登入就能操作客戶設備 |
| **客戶** | 按一個按鈕啟動/停止支援 |

### Cloudflare Manager 職責

| 任務 | 做什麼 | 頻率 |
|------|--------|------|
| 帳號設定 | 註冊 Cloudflare 帳號、加入域名 | 一次 |
| IdP 整合 | 設定 Google/GitHub 等登入方式 | 一次 |
| Access Policy | 定義誰可以存取（例如 @yourcompany.com） | 一次，偶爾調整 |
| Browser Rendering | 為 SSH/VNC 應用啟用瀏覽器 rendering | 一次 |
| API Token | 建立給 Helper App 用的 API Token | 一次 |
| 監控 Log | 查看存取記錄、處理異常 | 持續 |
| Session 管理 | 撤銷離職員工權限等 | 需要時 |

### Helper Developer 職責

| 任務 | 做什麼 | 說明 |
|------|--------|------|
| 呼叫 Cloudflare API | 動態建立/刪除 Tunnel | 客戶啟動支援時建立 |
| 取得 Tunnel Token | 從 API 拿到 Token 給 cloudflared 用 | 每次建立 Tunnel |
| 設定 Public Hostname | 透過 API 設定 ingress | 建立 Tunnel 時 |
| 管理 cloudflared | 啟動/停止 cloudflared 程序 | 客戶按按鈕時 |
| UI 開發 | 在 Web Desktop 裡做 Helper 管理頁面 | 一次 |
| 顯示連線資訊 | 給客戶看 Session ID 或狀態 | 每次支援 |

### Cloudflare Manager 給 Helper Developer 的東西

| 項目 | 說明 | 範例 |
|------|------|------|
| **API Token** | 有權限建立 Tunnel 和 DNS 的 token | `xxxxxxxxxxxxxxx` |
| **Account ID** | Cloudflare 帳號 ID | `abc123def456` |
| **Zone ID** | 域名的 ID | `xyz789` |
| **域名** | 用於 public hostname 的域名 | `support.yourcompany.com` |

### API Token 需要的權限

| 權限 | 用途 |
|------|------|
| `Account.Cloudflare Tunnel:Edit` | 建立/刪除 Tunnel |
| `Account.Cloudflare Tunnel:Read` | 讀取 Tunnel 資訊 |
| `Zone.DNS:Edit` | 建立 DNS 記錄 |

### 控制權限層級

```
                    誰可以中斷連線？

┌────────────────────────────────────────────────────────┐
│                                                        │
│   客戶         → 可以停止支援（刪除 Tunnel）               │
│                                                        │
│   Helper App   → 可以刪除自己建立的 Tunnel                │
│                                                        │
│   Cloudflare   → 可以做任何事：                          │
│   Manager        • 刪除 Tunnel                         │
│                  • 撤銷連線                            │
│                  • 禁止特定人存取                        │
│                  • 關閉整個應用                          │
│                                                        │
└────────────────────────────────────────────────────────┘
```

---

## 六、關鍵功能

### 功能總覽

| 功能 | 說明 | 誰受益 |
|------|------|--------|
| **IdP 整合** | 員工用公司 Google/GitHub 帳號登入 | 管理員、工程師 |
| **Browser SSH** | 瀏覽器內直接操作 SSH，不需裝軟體 | 工程師 |
| **Audit Log** | 記錄誰、何時、從哪裡存取 | 管理員、客戶 |
| **On-demand Tunnel** | 客戶需要時才建立，結束後刪除 | 安全性 |

### 1. IdP 整合（Google/GitHub 等）

**設定步驟：**

1. 登入 Cloudflare Zero Trust Dashboard
2. Settings → Authentication → Add new
3. 選擇 IdP 類型（Google、GitHub 等）
4. 設定 OAuth credentials
5. 設定 Access Policy

**常見 IdP 選項：**

| IdP | 適合 | 設定難度 |
|-----|------|---------|
| Google | 用 Google Workspace 的公司 | 簡單 |
| GitHub | 開發團隊 | 簡單 |
| Okta | 企業級 SSO | 中等 |
| Azure AD | 用 Microsoft 365 的公司 | 中等 |
| One-time PIN | 臨時存取、外部人員 | 最簡單 |

### 2. Browser SSH（純瀏覽器操作）

**設定步驟：**

1. Access → Applications → Add an application
2. 選擇 Self-hosted
3. 設定 Application domain
4. 設定 Access Policy（只有 Allow 或 Block）
5. Advanced settings → Browser rendering settings → 選擇 SSH
6. Save

**使用者體驗：**

1. Support Engineer 開瀏覽器訪問 SSH URL
2. Cloudflare Access 登入（用 Google 等 IdP）
3. 登入成功後，瀏覽器直接顯示 SSH 終端機畫面
4. 輸入 SSH 使用者名稱和密碼
5. 開始操作

### 3. Audit Log 和連線管理

**Audit Log 功能：**

Cloudflare Access audit logs 記錄身份驗證事件和對受保護 URI 路徑和基礎設施目標的請求。可以透過 API 查詢帳號的 audit logs，並可按行為類型、執行者電子郵件、IP 地址和時間框架進行篩選。

**功能位置：**

| 功能 | 位置 | 用途 |
|------|------|------|
| IdP 整合 | Settings → Authentication | 員工用公司帳號登入 |
| Browser SSH | Access → Applications → Browser rendering | 瀏覽器內操作 SSH |
| Audit Log | Manage Account → Audit Logs | 查看所有帳戶活動和存取記錄 |
| Access Log | Logs → Access Requests | 查看誰存取了什麼應用 |
| User 管理 | My Team → Users | 管理使用者權限 |
| Logpush | Logs → Logpush | 匯出 log 到外部系統（SIEM 等） |

**連線操作：**

| 操作 | 位置 | 效果 |
|------|------|------|
| 撤銷某使用者所有 Session | My Team → Users → Revoke all sessions | 該使用者在所有應用的連線都被踢出 |
| 撤銷某應用所有 Session | Access → Applications → Revoke existing tokens | 所有連到該應用的人都被踢出 |
| 刪除 Tunnel | Networks → Tunnels → Delete | 整個通道關閉，所有連線中斷 |

---

## 七、SSH 帳號設計與生命週期

### 方案比較

| 方案 | 安全性 | 便利性 | 管理複雜度 | 適合場景 |
|------|--------|--------|-----------|----------|
| A. 固定共用帳號 | 低 | 高 | 低 | 內部測試 |
| B. 每次動態產生帳號 | 高 | 中 | 高 | 高安全需求 |
| **C. 預建專用帳號 + 動態密碼** | 中高 | 中高 | 中 | **推薦** |
| D. SSH Key 認證 | 高 | 低 | 高 | 進階場景 |
| E. Cloudflare Access for Infrastructure | 最高 | 高 | 中 | 企業級 |

### 推薦方案：預建專用帳號 + 動態密碼

**生命週期：**

```
[設備出廠/部署]
      │
      ▼
┌─────────────────────────────────────────┐
│ 1. 建立 kaiden_support 帳號            │
│ 2. 設定必要權限（sudo 等）             │
│ 3. 鎖定帳號                            │
│                                         │
│ 狀態：帳號存在但無法登入               │
└─────────────────────────────────────────┘
      │
      │ (設備正常運作中，帳號保持鎖定)
      │
      ▼
[客戶請求支援，按下「啟動支援」]
      │
      ▼
┌─────────────────────────────────────────┐
│ Helper App:                             │
│ 1. 解鎖帳號                            │
│ 2. 產生隨機密碼（16+ 字元）            │
│ 3. 設定密碼                            │
│ 4. 啟動 Tunnel                         │
│ 5. 在客戶畫面顯示連線資訊              │
│                                         │
│ 狀態：帳號可登入，密碼已設定           │
└─────────────────────────────────────────┘
      │
      ▼
[Support Engineer 連線操作]
      │
      ▼
[支援結束，客戶按「停止支援」或 timeout]
      │
      ▼
┌─────────────────────────────────────────┐
│ Helper App:                             │
│ 1. 踢掉所有 SSH session                │
│ 2. 鎖定帳號                            │
│ 3. 設定隨機密碼（讓舊密碼失效）        │
│ 4. 清理 shell history                  │
│ 5. 停止 Tunnel                         │
│                                         │
│ 狀態：帳號再次鎖定，密碼已失效         │
└─────────────────────────────────────────┘
```

### 設備初始化（出廠時）

```bash
# 建立支援專用帳號
useradd -m -s /bin/bash kaiden_support

# 加入必要的群組（視需求）
usermod -aG sudo kaiden_support      # 如果需要 sudo
usermod -aG docker kaiden_support    # 如果需要操作 docker

# 鎖定帳號（無法登入）
passwd -l kaiden_support
```

### 安全措施清單

| 措施 | 實作方式 |
|------|----------|
| 帳號平時鎖定 | `passwd -l` |
| 密碼每次不同 | 隨機產生 |
| 密碼客戶畫面顯示 | 在 Helper 頁面顯示 |
| Session 結束清理 | 踢掉連線 + 鎖定帳號 + 清歷史 |
| 雙層認證 | Cloudflare Access + SSH 密碼 |
| 操作追蹤 | Cloudflare Audit Log |

---

## 八、密碼傳遞方案

### 方案比較

| 方案 | 需要自建 | 安全性 | 複雜度 | 適合階段 |
|------|----------|--------|--------|----------|
| **A. 客戶畫面顯示** | 無 | 中 | 最簡單 | **推薦** |
| B. Workers 中繼 | Worker 腳本 | 中高 | 中 | 不想讓客戶看到密碼時 |
| C. Access for Infrastructure | sshd 設定 | 最高 | 高 | 企業級 |

### 方案 A：顯示在客戶畫面（推薦）

```
┌─────────────────────────────────────────────────────────────────┐
│                      遠端支援                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ✅ 支援已啟動                                                  │
│                                                                 │
│  請將以下資訊提供給技術支援人員：                               │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Session ID: ABC-123-XYZ                                 │   │
│  │                                                         │   │
│  │ SSH 登入資訊:                                           │   │
│  │   帳號: kaiden_support                                  │   │
│  │   密碼: ●●●●●●●●●●●●  [顯示] [複製]                     │   │
│  │   URL: https://ssh.support.yourcompany.com/ABC-123-XYZ │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ⚠️ 請勿將此資訊分享給其他人                                    │
│  ⏱️ 此 session 將於 4 小時後自動過期                            │
│                                                                 │
│                        [停止支援]                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**流程：**

```
[客戶按「啟動支援」]
         │
         ▼
Helper App（在客戶設備上）
    1. 呼叫 Cloudflare API 建立 Tunnel
    2. 取得 Tunnel Token
    3. 解鎖帳號
    4. 產生隨機密碼
    5. 設定密碼
    6. 啟動 cloudflared
         │
         ▼
[客戶畫面顯示連線資訊]
    • Session ID
    • SSH 帳號
    • SSH 密碼
    • SSH URL
    • 過期時間
         │
         ▼
[客戶告知工程師]
    或透過其他通道（電話、工單系統）
         │
         ▼
[Support Engineer 用 SSH URL 或帳密連線]
```

### 安全考量

| 考量 | 方案 |
|------|------|
| 密碼傳輸加密 | HTTPS（Cloudflare 負責） |
| 防止重放攻擊 | Timeout 機制 + Session ID 驗證 |
| 客戶隱私 | 密碼只在客戶設備和 SSH 連線間傳輸 |

---

## 九、抽象層設計（降低移植成本）

### 設計原則

```
┌─────────────────────────────────────────────────────────────────┐
│                    你的應用層                                    │
├─────────────────────────────────────────────────────────────────┤
│   Helper App          Support Portal          監控管理            │
└───────────────────────────┬─────────────────────────────────────┘
                            │ 統一介面
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Tunnel Provider Interface                    │
│                    (抽象層)                                      │
├─────────────────────────────────────────────────────────────────┤
│   CreateTunnel()    DeleteTunnel()    GetTunnelStatus()         │
│   ListActiveSessions()  RevokeSession()                         │
└───────────────────────────┬─────────────────────────────────────┘
                            │
              ┌─────────────┴─────────────┐
              ▼                           ▼
┌──────────────────────┐    ┌──────────────────────┐
│ CloudflareProvider   │    │ SelfHostedProvider   │
│ (前期)               │    │ (後期)               │
└──────────────────────┘    └──────────────────────┘
```

### Tunnel Provider Interface

```go
type TunnelProvider interface {
    // Tunnel 管理
    CreateTunnel(req CreateTunnelRequest) (*CreateTunnelResponse, error)
    DeleteTunnel(tunnelID string) error
    GetTunnel(tunnelID string) (*TunnelInfo, error)
    ListTunnels() ([]TunnelInfo, error)
    
    // Session 查詢
    ListActiveSessions(tunnelID string) ([]Session, error)
    RevokeSession(sessionID string) error
    RevokeAllSessions(tunnelID string) error
    
    // 健康檢查
    HealthCheck() error
}
```

### 切換 Provider

```go
// 根據環境變數決定用哪個 provider
var provider tunnel.TunnelProvider

switch os.Getenv("TUNNEL_PROVIDER") {
case "cloudflare":
    provider = tunnel.NewCloudflareProvider(config)
case "selfhosted":
    provider = tunnel.NewSelfHostedProvider(config)
}

// Helper App 不在乎底層是什麼
tunnelManager := manager.NewTunnelManager(provider)
```

### 移植成本評估

| 項目 | 成本 |
|------|------|
| **不用改** | |
| Helper App 主邏輯 | 0 |
| 前端 UI | 0 |
| 帳號管理邏輯 | 0 |
| **要改** | |
| 新增 SelfHostedProvider | 中（幾天） |
| 自建 Relay Server | 高（1-2 週） |
| 自建 Web Proxy | 高（1-2 週） |
| 自建 Auth Server | 中（1 週） |
| 自建 Browser SSH | 高（可用 Guacamole 降低） |

---

## 十、待釐清問題

### A. 商業/策略層面

| 問題 | 需要決定 | 影響 |
|------|----------|------|
| **域名** | 用哪個域名？需要購買新域名嗎？ | 必須有域名才能用完整功能 |
| **Cloudflare 方案** | Free / Pro / Enterprise？ | 功能和費用差異 |
| **支援對象規模** | 預計同時有多少客戶需要支援？ | 影響架構和成本 |
| **合規要求** | 客戶對資料經過第三方有疑慮嗎？ | 可能需要更早自建 |
| **SLA 要求** | 遠端支援的可用性要求？ | 影響是否需要備援方案 |

### B. 技術層面

| 問題 | 需要決定 | 影響 |
|------|----------|------|
| **客戶設備 OS** | Linux / Windows / 兩者都有？ | cloudflared 部署方式 |
| **Docker 可用性** | 客戶設備上有 Docker 嗎？ | cloudflared 用 Docker 或直接安裝 |
| **SSH 帳號** | 用什麼帳號 SSH 進去？root？專用帳號？ | 需要在設備上預先設定 |
| **網路環境** | 客戶設備能連外網嗎？有 proxy 嗎？ | 可能影響 tunnel 連線 |
| **Helper 整合** | Helper 頁面整合到現有 Web Desktop 還是獨立？ | 開發方式 |

### C. 安全層面

| 問題 | 需要決定 | 影響 |
|------|----------|------|
| **存取範圍** | 工程師需要完整 root 權限嗎？ | 安全風險 vs 除錯需求 |
| **Session 時效** | 一次支援 session 多久過期？ | 安全性 vs 便利性 |
| **客戶同意流程** | 客戶需要每次明確同意嗎？ | 合規、UI 設計 |
| **操作記錄** | 需要錄影嗎？還是只要 log？ | 儲存成本、合規 |

### D. 帳號設計相關

| 問題 | 選項 | 建議 |
|------|------|------|
| **SSH 帳號名稱** | `support` / `kaiden_support` / 其他 | `kaiden_support`（品牌識別） |
| **帳號權限** | 一般使用者 / sudo / root | sudo（需要重啟服務等） |
| **密碼長度** | 8 / 12 / 16 / 更長 | 16 字元以上 |
| **密碼字元集** | 純英數 / 含特殊字元 | 含特殊字元（更安全） |
| **Session 時效** | 1小時 / 8小時 / 24小時 / 無限 | 24 小時（可調整） |
| **閒置 timeout** | 無 / 30分鐘 / 1小時 | 1 小時（自動踢出） |

---

## 十一、潛在風險與問題

### A. Cloudflare 依賴風險

| 風險 | 機率 | 影響 | 緩解方案 |
|------|------|------|----------|
| Cloudflare 服務中斷 | 低 | 高（無法支援） | 監控 + 備援方案規劃 |
| Cloudflare 漲價 | 中 | 中 | 抽象層設計，可換自建 |
| Cloudflare 功能變更 | 低 | 中 | 鎖定 API 版本 |
| 資料經過第三方 | - | 視客戶而定 | 溝通或提前自建 |

### B. 技術風險

| 風險 | 機率 | 影響 | 緩解方案 |
|------|------|------|----------|
| 客戶網路阻擋出站連線 | 低 | 高 | cloudflared 支援 proxy |
| Browser SSH 延遲高 | 中 | 中 | 可改用 native SSH |
| SSH 帳號密碼管理 | - | 中 | 預設帳號或整合 Access for Infrastructure |
| Tunnel 建立失敗 | 低 | 高 | 錯誤處理 + 重試機制 |

### C. 使用者體驗風險

| 風險 | 機率 | 影響 | 緩解方案 |
|------|------|------|----------|
| 客戶不會操作 | 中 | 中 | 簡化 UI，一鍵啟動 |
| 工程師不熟悉流程 | 中 | 低 | 文件 + 訓練 |
| 密碼/Session ID 溝通問題 | 中 | 低 | 簡化流程，盡量自動化 |

---

## 十二、成本估算

### 前期（Cloudflare 方案）

| 項目 | 費用 | 備註 |
|------|------|------|
| Cloudflare Free Plan | $0/月 | 50 位使用者免費 |
| Cloudflare Pro（如需要） | $20/月 | 更多功能 |
| 域名 | ~$10-15/年 | 如需購買 |
| 開發人力 | 內部 | Helper App 開發 |

### 後期（自建方案，僅估算）

| 項目 | 費用 | 備註 |
|------|------|------|
| Relay Server | ~$50-100/月 | 雲端 VM |
| Web Proxy Server | ~$50-100/月 | 可合併 |
| Auth Server | ~$20-50/月 | 可合併 |
| 開發人力 | 內部 | 預估 4-8 週 |

---

## 十三、時程估算

### Phase 1：POC（1-2 週）

| 任務 | 時間 |
|------|------|
| Cloudflare 帳號 + 域名設定 | 1 天 |
| 手動建立 Tunnel + 測試 HTTP | 1 天 |
| 測試 Browser SSH | 1 天 |
| 設定 Access + IdP | 1 天 |
| 用 API 建立 Tunnel | 2-3 天 |
| 整合驗證 | 2-3 天 |

### Phase 2：開發（2-3 週）

| 任務 | 時間 |
|------|------|
| Helper App 核心邏輯（API 整合） | 1 週 |
| Helper App UI（Web Desktop 整合） | 1 週 |
| 帳號管理和密碼生成 | 3-5 天 |
| 測試 + 修正 | 3-5 天 |

### Phase 3：部署 + 上線（1-2 週）

| 任務 | 時間 |
|------|------|
| 整合到現有 Web Desktop | 3-5 天 |
| 文件 + 訓練 | 2-3 天 |
| 小規模試用 | 3-5 天 |

---

## 十四、決策建議

### 建議採用方案

```
Phase 1: Cloudflare Tunnel（立即）
    ↓
Phase 2: 評估使用量和客戶回饋（3-6 個月後）
    ↓
Phase 3: 決定是否自建（視規模和需求）
```

### 立即需要的行動

| 優先序 | 行動 | 負責人 |
|--------|------|--------|
| 1 | 確認域名（現有或購買） | 管理層 |
| 2 | 註冊 Cloudflare 帳號 | IT/DevOps |
| 3 | 啟動 POC | 開發團隊 |
| 4 | 釐清待決問題（第十節） | 產品/管理層 |

---

## 十五、附錄

### 術語對照表

| 術語 | 說明 |
|------|------|
| **Cloudflare Tunnel** | Cloudflare 的安全通道服務，不需公網 IP |
| **cloudflared** | Cloudflare 的 tunnel client 程式 |
| **Zero Trust** | Cloudflare 的安全產品線名稱 |
| **Access** | Cloudflare 的身份驗證服務 |
| **IdP (Identity Provider)** | 身份提供者，如 Google、GitHub |
| **Browser Rendering** | 在瀏覽器內 render SSH/VNC 畫面 |
| **Ingress** | 定義流量如何路由到本地服務 |
| **Tunnel Token** | 讓 cloudflared 認證的密鑰 |

### 參考資料

- [Cloudflare Tunnel 文件](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)
- [Cloudflare Access 文件](https://developers.cloudflare.com/cloudflare-one/policies/access/)
- [Browser-rendered SSH](https://developers.cloudflare.com/cloudflare-one/applications/non-http/browser-rendering/)
- [Cloudflare API](https://developers.cloudflare.com/api/)