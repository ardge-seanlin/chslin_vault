# Cloudflare Access CLI 完整使用指南

`cloudflared access` 提供命令列工具，讓您透過 Cloudflare Access 安全地存取受保護的應用程式和服務。

---

## 一、HTTP 應用程式存取

### 1. 登入 (`login`)

啟動瀏覽器進行身份驗證：

```sh
cloudflared access login https://app.example.com
```

### 2. 使用 curl 發送請求 (`curl`)

自動注入驗證權杖：

```sh
cloudflared access curl https://app.example.com/api/data
```

### 3. 取得權杖 (`token`)

取得權杖供其他工具使用：

```sh
cloudflared access token -app=https://app.example.com
```

**實用技巧 - 將權杖存為環境變數：**

```sh
# 儲存權杖
export TOKEN=$(cloudflared access token -app=https://app.example.com)

# 在 curl 請求中使用
curl -H "cf-access-token: $TOKEN" https://app.example.com/api/data
```

---

## 二、SSH 連線

### 用戶端設定

1. **編輯 SSH 設定檔** (`~/.ssh/config`)：

```
Host ssh.example.com
    ProxyCommand /usr/local/bin/cloudflared access ssh --hostname %h
```

> macOS Homebrew 路徑：`/opt/homebrew/bin/cloudflared`

2. **連線方式**：

```sh
ssh username@ssh.example.com
```

執行後會自動開啟瀏覽器進行身份驗證。

---

## 三、RDP 遠端桌面

### 用戶端連線

```sh
cloudflared access rdp --hostname rdp.example.com --url rdp://localhost:3389
```

**連線步驟：**

1. 執行上述命令（保持運作）
2. 開啟 RDP 用戶端（如 Microsoft Remote Desktop）
3. 連線至 `localhost:3389`

> Windows 若 3389 已被佔用，可改用其他連接埠如 `rdp://localhost:3390`

---

## 四、任意 TCP 連線

適用於資料庫、自訂服務等非標準協定。

### 伺服器端（建立通道）

```sh
cloudflared tunnel --hostname tcp.example.com --url tcp://localhost:7870
```

### 用戶端（連線存取）

```sh
cloudflared access tcp --hostname tcp.example.com --url localhost:9210
```

連線後，本地應用程式可透過 `localhost:9210` 存取遠端服務。

---

## 五、SMB 檔案分享

### 用戶端連線

```sh
cloudflared access smb --hostname smb.example.com --url localhost:8445
```

**掛載網路磁碟：**

- Windows：`\\localhost\sharename`
- macOS/Linux：`smb://localhost:8445/sharename`

---

## 六、常用參數說明

| 參數 | 說明 |
|------|------|
| `--hostname` | Access 保護的公開主機名稱 |
| `--url` | 本地監聽位址和連接埠 |
| `--fedramp` | 用於 FedRAMP 帳戶 |

---

## 七、使用情境範例

### 情境 1：開發人員存取內部 API

```sh
# 登入取得權杖
cloudflared access login https://api.internal.example.com

# 使用 curl 測試 API
cloudflared access curl https://api.internal.example.com/v1/users

# 或用權杖搭配其他工具
export TOKEN=$(cloudflared access token -app=https://api.internal.example.com)
curl -H "cf-access-token: $TOKEN" https://api.internal.example.com/v1/users
```

### 情境 2：遠端 SSH 維護伺服器

```sh
# 設定 SSH config 後直接連線
ssh admin@server.example.com

# 瀏覽器會自動開啟進行驗證
```

### 情境 3：存取遠端資料庫

```sh
# 建立 TCP 通道至 PostgreSQL
cloudflared access tcp --hostname db.example.com --url localhost:5432

# 使用 psql 連線（另開終端機）
psql -h localhost -p 5432 -U dbuser -d mydb
```

---

## 資料來源

- [Connect through Cloudflare Access using a CLI](https://developers.cloudflare.com/cloudflare-one/tutorials/cli/)
- [Client-side cloudflared Authentication](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/non-http/cloudflared-authentication/)
- [SSH with cloudflared](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/use-cases/ssh/ssh-cloudflared-authentication/)
- [RDP with cloudflared](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/use-cases/rdp/rdp-cloudflared-authentication/)
- [Arbitrary TCP](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/non-http/cloudflared-authentication/arbitrary-tcp/)
- [SMB](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/use_cases/smb/)
