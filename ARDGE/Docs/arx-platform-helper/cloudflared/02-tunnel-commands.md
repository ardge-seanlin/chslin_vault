# cloudflared tunnel 命令詳細說明

## 概述

`cloudflared tunnel` 是 Cloudflare 隧道的核心命令，允許你：
- 將私人服務暴露到網際網路（使用 DNS）
- 將本地 TCP/UDP 服務暴露給 Cloudflare Zero Trust 使用者（WARP 客戶端）

詳細文檔：[Cloudflare Tunnel 指南](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/tunnel-guide/)

---

## 隧道子命令

| 命令 | 說明 |
|------|------|
| `login` | 生成配置檔案並記錄登入詳情 |
| `create` | 建立新隧道 |
| `route` | 定義流量路由（DNS、負載均衡、WARP） |
| `vnet` | 配置虛擬網路以管理重疊 IP 路由 |
| `run` | 執行隧道並代理本地網頁伺服器 |
| `list` | 列出現有隧道 |
| `ready` | 呼叫 /ready 端點並返回適當的離開代碼 |
| `info` | 列出活躍連接器詳情 |
| `delete` | 刪除現有隧道 |
| `cleanup` | 清理隧道連線 |
| `token` | 獲取隧道凭証令牌 |
| `diag` | 建立本地 cloudflared 實例的診斷報告 |
| `help, h` | 顯示命令列表或特定命令說明 |

---

## 常用子命令詳解

### 1. `cloudflared tunnel login` 登入

**用途：** 生成認證憑証，用於管理隧道

```bash
# 基本登入
cloudflared tunnel login

# 登入後會顯示 Cloudflare 授權連結
# 完成授權後，凭証儲存於 ~/.cloudflared/cert.pem
```

**說明：**
- 這是使用命令列管理隧道的必要步驟
- 如果使用 Cloudflare 儀表板建立隧道，則不需要此步驟
- 凭証檔案會自動儲存

---

### 2. `cloudflared tunnel create <NAME>` 建立隧道

**用途：** 建立新的命令列管理隧道

```bash
# 建立隧道
cloudflared tunnel create my-tunnel

# 輸出：
# Created tunnel my-tunnel with ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890
# Credentials have been saved to:
# /Users/username/.cloudflared/a1b2c3d4-e5f6-7890-abcd-ef1234567890.json
```

**說明：**
- 隧道名稱必須唯一
- UUID 是隧道的唯一識別符
- 凭証檔案用於 `cloudflared tunnel run` 命令

---

### 3. `cloudflared tunnel route dns <TUNNEL_NAME> <HOSTNAME>` 路由 DNS

**用途：** 將 DNS 記錄指向隧道

```bash
# 基本路由
cloudflared tunnel route dns my-tunnel example.com

# 路由子網域
cloudflared tunnel route dns my-tunnel api.example.com

# 強制覆寫現有 DNS 記錄
cloudflared tunnel route dns --overwrite my-tunnel example.com
```

**說明：**
- 自動在 Cloudflare DNS 中建立 CNAME 記錄
- 每個隧道可以有多個 DNS 路由
- 需要隧道名稱或 UUID

**環境需求：**
- 域名必須已被添加到 Cloudflare
- 必須有管理員權限

---

### 4. `cloudflared tunnel route ip <CIDR> <TUNNEL_NAME> [<VNET_NAME>]` 路由 IP

**用途：** 為 WARP 客戶端路由私人 IP 範圍

```bash
# 路由私人 IP 到隧道
cloudflared tunnel route ip 10.0.0.0/8 my-tunnel

# 使用虛擬網路
cloudflared tunnel route ip 192.168.0.0/16 my-tunnel my-vnet

# 多個 IP 範圍
cloudflared tunnel route ip 10.0.0.0/8 my-tunnel
cloudflared tunnel route ip 172.16.0.0/12 my-tunnel
```

**說明：**
- WARP 客戶端可以透過隧道存取這些 IP
- 需要在 Zero Trust 後台配置
- 支援 IPv4 和 IPv6

---

### 5. `cloudflared tunnel vnet` 虛擬網路

**用途：** 管理虛擬網路以處理重疊的 IP 位址

```bash
# 列出虛擬網路
cloudflared tunnel vnet list

# 建立虛擬網路
cloudflared tunnel vnet create my-vnet

# 刪除虛擬網路
cloudflared tunnel vnet delete my-vnet
```

**說明：**
- 用於多個隧道有相同私人 IP 範圍的情況
- 每個虛擬網路可包含多個隧道

---

### 6. `cloudflared tunnel run [TUNNEL_UUID|TUNNEL_NAME]` 執行隧道

**用途：** 啟動隧道並開始代理流量

```bash
# 使用隧道名稱執行
cloudflared tunnel run my-tunnel

# 使用 UUID 執行
cloudflared tunnel run a1b2c3d4-e5f6-7890-abcd-ef1234567890

# 執行並指定源地址
cloudflared tunnel run --url http://localhost:8080 my-tunnel

# Hello World 測試伺服器
cloudflared tunnel run --hello-world my-tunnel
```

**說明：**
- 預設本地源為 `http://localhost:8080`
- 使用 Ctrl+C 停止隧道
- 可與全域選項組合使用

---

### 7. `cloudflared tunnel list` 列出隧道

**用途：** 顯示所有命令列管理的隧道

```bash
# 列出隧道
cloudflared tunnel list

# 輸出範例：
# ID                                   | Name        | Created              | Connections
# a1b2c3d4-e5f6-7890-abcd-ef1234567890 | my-tunnel   | 2025-01-15T10:30:00Z | 2/2
```

---

### 8. `cloudflared tunnel info <TUNNEL_UUID|TUNNEL_NAME>` 隧道資訊

**用途：** 顯示活躍連接器的詳細資訊

```bash
# 顯示隧道資訊
cloudflared tunnel info my-tunnel

# 輸出範例：
# Connector ID: abcd1234-ef56-7890-abcd-ef1234567890
# Connector Name: Production-East-1
# Status: HEALTHY
# Last Seen: 5 seconds ago
# IP: 203.0.113.42
# Version: 2025.11.1
```

---

### 9. `cloudflared tunnel delete <TUNNEL_UUID|TUNNEL_NAME>` 刪除隧道

**用途：** 刪除隧道及其所有配置

```bash
# 刪除隧道
cloudflared tunnel delete my-tunnel

# 刪除前會要求確認
# 這會刪除所有 DNS 路由和隧道凭証
```

**警告：**
- 此操作不可逆
- 會刪除相關的 DNS 記錄
- 確保沒有依賴此隧道的服務

---

### 10. `cloudflared tunnel token <TUNNEL_UUID|TUNNEL_NAME>` 獲取令牌

**用途：** 取得可用於執行隧道的臨時令牌

```bash
# 取得隧道令牌
cloudflared tunnel token my-tunnel

# 輸出範例：
# eyJhIjoiNzA4ZDc1YzEtZWY1Ni00ZDU1LWFiY2QtZWYxMjM0NTY3ODkwIiwid...

# 使用令牌執行隧道
cloudflared tunnel run --token eyJhIjoiNzA4ZDc1YzEtZWY1Ni00ZDU1LWFiY2QtZWYxMjM0NTY3ODkwIiwid...
```

**優點：**
- 無需儲存凭証檔案
- 令牌可設定過期時間
- 更適合自動化和容器化環境

---

## cloudflared tunnel run 選項詳解

### 源伺服器配置選項

#### `--url value` 源伺服器 URL

```bash
# 連線到本地 HTTP 伺服器
cloudflared tunnel run --url http://localhost:8080 my-tunnel

# 連線到本地 HTTPS 伺服器
cloudflared tunnel run --url https://localhost:8443 my-tunnel

# 連線到本地 WebSocket 伺服器
cloudflared tunnel run --url http://localhost:3000 my-tunnel
```

**預設值：** `http://localhost:8080`

**適用情景：**
- HTTP/HTTPS Web 應用程式
- WebSocket 服務
- API 伺服器

---

#### `--hello-world` Hello World 測試

```bash
# 執行測試伺服器
cloudflared tunnel run --hello-world my-tunnel

# 訪問隧道會顯示 Cloudflare Hello World 頁面
# 無需本地伺服器
```

**用途：** 驗證隧道連線正常

---

#### `--unix-socket value` Unix Socket

```bash
# 連線到 Unix Socket
cloudflared tunnel run --unix-socket /var/run/myapp.sock my-tunnel

# 用於容器或 systemd socket activation
```

---

### 代理選項

#### `--socks5` SOCKS5 代理

```bash
# 執行 SOCKS5 代理
cloudflared tunnel run --url http://localhost:8080 --socks5 my-tunnel
```

**用途：** 透過隧道提供 SOCKS5 代理

---

#### `--proxy-connect-timeout` 連線超時

```bash
# 設定連線超時為 60 秒
cloudflared tunnel run --url http://localhost:8080 --proxy-connect-timeout 60s my-tunnel
```

**預設值：** 30s

---

#### `--proxy-tls-timeout` TLS 超時

```bash
# 設定 TLS 握手超時為 15 秒
cloudflared tunnel run --url https://localhost:8443 --proxy-tls-timeout 15s my-tunnel
```

**預設值：** 10s

---

#### `--proxy-tcp-keepalive` TCP Keep-Alive

```bash
# 設定 TCP keep-alive 為 60 秒
cloudflared tunnel run --url http://localhost:8080 --proxy-tcp-keepalive 60s my-tunnel
```

**預設值：** 30s

---

#### `--proxy-keepalive-connections` Keep-Alive 連線池大小

```bash
# 設定最大 keep-alive 連線數為 200
cloudflared tunnel run --url http://localhost:8080 --proxy-keepalive-connections 200 my-tunnel
```

**預設值：** 100

---

#### `--proxy-keepalive-timeout` Keep-Alive 閒置超時

```bash
# 設定閒置超時為 3 分鐘
cloudflared tunnel run --url http://localhost:8080 --proxy-keepalive-timeout 3m my-tunnel
```

**預設值：** 1m30s

---

### HTTP 頭部選項

#### `--http-host-header` HTTP Host 頭部

```bash
# 設定 Host 頭部
cloudflared tunnel run --url http://localhost:8080 --http-host-header example.com my-tunnel

# 對於虛擬主機很有用
```

---

#### `--origin-server-name` 源伺服器名稱

```bash
# 設定 SNI（Server Name Indication）
cloudflared tunnel run --url https://localhost:8443 --origin-server-name origin.example.com my-tunnel
```

**用途：** 驗證 HTTPS 源伺服器的憑証

---

### TLS 和安全選項

#### `--no-tls-verify` 禁用 TLS 驗證

```bash
# 跳過源伺服器的 TLS 驗證（開發環境）
cloudflared tunnel run --url https://localhost:8443 --no-tls-verify my-tunnel
```

**警告：** 僅用於開發環境

---

#### `--origin-ca-pool` 自訂 CA 憑証

```bash
# 使用自訂 CA 驗證源伺服器
cloudflared tunnel run --url https://localhost:8443 --origin-ca-pool /path/to/ca.pem my-tunnel
```

**用途：** 源伺服器使用自簽或私人憑証

---

#### `--no-chunked-encoding` 禁用分塊編碼

```bash
# 禁用 HTTP 分塊傳輸編碼（某些 WSGI 伺服器需要）
cloudflared tunnel run --url http://localhost:8080 --no-chunked-encoding my-tunnel
```

**適用情景：** Python WSGI 伺服器

---

### 日誌和診斷選項

#### `--loglevel value` 日誌級別

```bash
# 詳細日誌（顯示 URL、方法、頭部）
cloudflared tunnel run --loglevel debug my-tunnel

# 標準日誌
cloudflared tunnel run --loglevel info my-tunnel

# 僅警告和錯誤
cloudflared tunnel run --loglevel warn my-tunnel
```

**可用級別：** debug, info, warn, error, fatal

**預設值：** info

**注意：** debug 會暴露敏感資訊

---

#### `--transport-loglevel` 傳輸日誌級別

```bash
# 詳細的協定日誌
cloudflared tunnel run --transport-loglevel debug my-tunnel
```

**用途：** 診斷隧道連線問題

---

#### `--logfile value` 日誌檔案

```bash
# 將日誌寫入檔案
cloudflared tunnel run --logfile /var/log/cloudflared.log my-tunnel
```

---

#### `--log-directory value` 日誌目錄

```bash
# 將日誌寫入目錄（用於旋轉日誌）
cloudflared tunnel run --log-directory /var/log/cloudflared/ my-tunnel
```

---

#### `--trace-output value` 跟蹤輸出

```bash
# 停止時生成跟蹤檔案
cloudflared tunnel run --trace-output /tmp/trace.out my-tunnel
```

**用途：** 性能分析

---

### DNS 代理選項

#### `--proxy-dns` DNS over HTTPS

```bash
# 執行 DNS over HTTPS 代理
cloudflared tunnel run --proxy-dns my-tunnel

# 監聽埠 53
```

**說明：**
- 需要在系統 DNS 設定中指向此 cloudflared 實例
- 支援 DNS over HTTPS (DoH) 和 DoT

---

#### `--proxy-dns-address` DNS 監聽位址

```bash
# 在特定位址監聽
cloudflared tunnel run --proxy-dns --proxy-dns-address 0.0.0.0 my-tunnel
```

**預設值：** localhost

---

#### `--proxy-dns-port` DNS 監聽埠

```bash
# 在自訂埠監聽
cloudflared tunnel run --proxy-dns --proxy-dns-port 5353 my-tunnel
```

**預設值：** 53

---

#### `--proxy-dns-upstream` 上游 DNS

```bash
# 使用自訂上游 DNS
cloudflared tunnel run --proxy-dns --proxy-dns-upstream https://8.8.8.8/dns-query my-tunnel

# 多個上游
cloudflared tunnel run --proxy-dns \
  --proxy-dns-upstream https://1.1.1.1/dns-query \
  --proxy-dns-upstream https://8.8.8.8/dns-query \
  my-tunnel
```

**預設值：** Cloudflare DNS (1.1.1.1)

---

### 配置檔案和高級選項

#### `--config value` 配置檔案

```bash
# 使用 YAML 配置檔案
cloudflared tunnel run --config /etc/cloudflare/config.yaml my-tunnel
```

**說明：**
- 配置檔案格式優先於命令列選項
- 適合複雜配置

---

#### `--name value` 隧道名稱快速建立

```bash
# 一次命令建立、路由和執行隧道
cloudflared tunnel run --name my-tunnel --url http://localhost:8080
```

**說明：**
- 用於快速測試
- 生產環境建議分別執行 create 和 run 命令

---

### 性能和監控選項

#### `--metrics value` 指標監聽位址

```bash
# 在自訂位址暴露 Prometheus 指標
cloudflared tunnel run --metrics 0.0.0.0:9090 my-tunnel
```

**預設值：** localhost:20241-20245（輪循）

**提供的指標：**
- 連線數、流量、延遲等 Prometheus 格式指標

---

#### `--metrics-update-freq value` 指標更新頻率

```bash
# 每 10 秒更新一次指標
cloudflared tunnel run --metrics-update-freq 10s my-tunnel
```

**預設值：** 5s

---

#### `--retries value` 重試次數

```bash
# 最多重試 10 次
cloudflared tunnel run --retries 10 my-tunnel
```

**預設值：** 5

**用途：** 網路不穩定環境

---

#### `--grace-period value` 優雅關閉超時

```bash
# 優雅關閉超時設為 60 秒
cloudflared tunnel run --grace-period 60s my-tunnel
```

**預設值：** 30s

**說明：**
- 接收 SIGTERM 後等待進行中請求完成的時間

---

### 負載均衡和路由

#### `--hostname value` 指定主機名

```bash
# 指定隧道暴露的主機名
cloudflared tunnel run --hostname example.com my-tunnel
```

---

#### `--lb-pool value` 負載均衡池

```bash
# 添加到負載均衡池
cloudflared tunnel run --lb-pool my-pool my-tunnel
```

---

### 其他選項

#### `--autoupdate-freq value` 自動更新頻率

```bash
# 每 48 小時檢查更新
cloudflared tunnel run --autoupdate-freq 48h my-tunnel

# 禁用自動更新
cloudflared tunnel run --no-autoupdate my-tunnel
```

**預設值：** 24h0m0s

---

#### `--http2-origin` HTTP/2 源伺服器

```bash
# 啟用 HTTP/2 源伺服器支援
cloudflared tunnel run --http2-origin --url http://localhost:8080 my-tunnel
```

---

#### `--bastion` 堡壘模式

```bash
# 執行為跳躍主機
cloudflared tunnel run --bastion my-tunnel
```

**用途：** 充當其他系統的代理

---

#### `--compression-quality` 壓縮品質

```bash
# 啟用高質量壓縮
cloudflared tunnel run --compression-quality 3 my-tunnel
```

**值：**
- `0`：關閉
- `1`：低
- `2`：中
- `>=3`：高

**預設值：** 0（關閉）

---

## 實際使用範例

### 場景 1：基本 Web 應用

```bash
# 步驟 1：登入
cloudflared tunnel login

# 步驟 2：建立隧道
cloudflared tunnel create my-web-app

# 步驟 3：路由 DNS
cloudflared tunnel route dns my-web-app example.com

# 步驟 4：執行隧道
cloudflared tunnel run --url http://localhost:3000 my-web-app
```

---

### 場景 2：開發環境測試

```bash
# 快速測試（使用 Hello World）
cloudflared tunnel run --hello-world my-test

# 或使用本地伺服器
cloudflared tunnel run --url http://localhost:8080 my-dev
```

---

### 場景 3：多個應用負載均衡

```bash
# 伺服器 1
cloudflared tunnel create app1
cloudflared tunnel route dns app1 app.example.com
cloudflared tunnel run --url http://localhost:3000 --lb-pool production app1

# 伺服器 2
cloudflared tunnel create app2
cloudflared tunnel route dns app2 app.example.com
cloudflared tunnel run --url http://localhost:3000 --lb-pool production app2
```

---

### 場景 4：生產環境配置

```bash
cloudflared tunnel run \
  --config /etc/cloudflare/config.yaml \
  --logfile /var/log/cloudflared.log \
  --loglevel info \
  --metrics 0.0.0.0:9090 \
  --label "Production-Main" \
  --region wnam \
  production-tunnel
```

---

## 配置檔案範例

建立 `~/.cloudflared/config.yaml`：

```yaml
tunnel: my-tunnel
credentials-file: /Users/username/.cloudflared/a1b2c3d4-e5f6-7890-abcd-ef1234567890.json

# 源伺服器配置
url: http://localhost:8080

# 日誌
loglevel: info
logfile: /var/log/cloudflared.log

# 效能
retries: 5
grace-period: 30s

# 指標
metrics: 127.0.0.1:9090

# 進階配置
originRequest:
  connectTimeout: 30s
  tlsTimeout: 10s
  tcpKeepAlive: 30s
  noHappyEyeballs: false

# Ingress 規則（用於進階路由）
ingress:
  - hostname: api.example.com
    service: http://localhost:3000
  - hostname: static.example.com
    path: /static/*
    service: http://localhost:8000
  - service: http://localhost:8080
```

執行配置檔案：

```bash
cloudflared tunnel run --config ~/.cloudflared/config.yaml
```

---

## 常見問題

### Q：如何在後台執行隧道？

**A：** 使用 systemd 或 Docker：

```bash
# Systemd
sudo systemctl start cloudflared@my-tunnel

# Docker
docker run -d --name cloudflared \
  -e TUNNEL_TOKEN=<TOKEN> \
  cloudflare/cloudflared:latest \
  tunnel run
```

---

### Q：如何監控隧道狀態？

**A：** 使用 `cloudflared tunnel info` 命令或訪問指標端點：

```bash
# 檢視隧道資訊
cloudflared tunnel info my-tunnel

# 訪問 Prometheus 指標
curl http://localhost:9090/metrics
```

---

### Q：隧道支援 WebSocket 嗎？

**A：** 是的，WebSocket 無需特殊配置即可正常工作。

---

### Q：如何使用自訂憑証？

**A：** 使用 `--origin-ca-pool` 選項：

```bash
cloudflared tunnel run \
  --url https://localhost:8443 \
  --origin-ca-pool /path/to/ca.pem \
  --origin-server-name origin.internal \
  my-tunnel
```

---

## 相關資源

- [Cloudflare Tunnel 官方文檔](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)
- [Tunnel Run Parameters](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/configure-tunnels/cloudflared-parameters/run-parameters/)
- [Useful Commands](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/do-more-with-tunnels/local-management/tunnel-useful-commands/)
- cloudflared 版本：2025.11.1
