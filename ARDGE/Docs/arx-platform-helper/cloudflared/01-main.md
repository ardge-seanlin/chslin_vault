# cloudflared 全域選項詳細說明

## 概述

cloudflared 是 Cloudflare 的命令列工具，用於建立隧道、代理和存取控制。本文詳細說明各個全域選項的使用方式。

---

## 全域選項詳解

### 1. `--output value` 日誌輸出格式

**用途：** 設定日誌的輸出格式

```bash
# 預設格式（人類可讀）
cloudflared tunnel run --token <TOKEN>

# JSON 格式（適合日誌聚合系統）
cloudflared --output json tunnel run --token <TOKEN>
```

**適用情景：**
- `default`：本地開發、手動監控
- `json`：ELK Stack、Datadog、Splunk 等日誌系統

---

### 2. `--credentials-file, --cred-file value` 凭証檔案路徑

**用途：** 指定儲存隧道凭証的檔案位置

```bash
# 使用自訂凭証路徑
cloudflared tunnel run --credentials-file /etc/cloudflare/tunnel-cred.json my-tunnel

# 環境變數設定
export TUNNEL_CRED_FILE=/var/lib/cloudflare/cred.json
cloudflared tunnel run my-tunnel
```

**適用情景：**
- 多個隧道管理
- Docker/K8s 環境（凭証以卷掛載）
- 安全政策要求特定路徑

---

### 3. `--region value` Cloudflare Edge 區域

**用途：** 指定連線到哪個 Cloudflare 邊界位置

```bash
# 連線到全球最優區域（預設）
cloudflared tunnel run --token <TOKEN>

# 連線到特定區域
cloudflared --region wnam tunnel run --token <TOKEN>
cloudflared --region easia tunnel run --token <TOKEN>
cloudflared --region sam tunnel run --token <TOKEN>
```

**可用區域：**
- `wnam`：西北美洲
- `enam`：東北美洲
- `weur`：西歐
- `easia`：東亞
- `ocea`：大洋洲
- `sam`：南美洲
- `afr`：非洲

**適用情景：**
- 優化延遲（選擇最近的區域）
- 遵守資料駐留法規
- 負載均衡多個隧道

---

### 4. `--edge-ip-version value` Edge IP 版本

**用途：** 指定使用 IPv4 或 IPv6 連線到 Cloudflare

```bash
# 使用 IPv4（預設）
cloudflared tunnel run --token <TOKEN>

# 使用 IPv6
cloudflared --edge-ip-version 6 tunnel run --token <TOKEN>

# 自動選擇（優先 IPv6，若失敗則 IPv4）
cloudflared --edge-ip-version auto tunnel run --token <TOKEN>
```

**適用情景：**
- `4`：大多數環境、向後相容
- `6`：僅 IPv6 基礎設施
- `auto`：混合環境

---

### 5. `--edge-bind-address value` Edge 綁定位址

**用途：** 指定本地哪個 IP 位址用於連線到 Cloudflare Edge

```bash
# 使用特定本地 IP 連線（多網卡情況）
cloudflared --edge-bind-address 192.168.1.100 tunnel run --token <TOKEN>

# 使用特定 IPv6 位址
cloudflared --edge-bind-address 2001:db8::1 tunnel run --token <TOKEN>
```

**適用情景：**
- 多網路介面卡伺服器
- 網路政策要求特定出站 IP
- 負載均衡情況下控制路由

---

### 6. `--label value` 連接器標籤

**用途：** 為隧道連接器設定易於辨識的名稱

```bash
# 預設：使用主機名 + UUID
cloudflared tunnel run my-tunnel

# 自訂標籤
cloudflared --label "Production-East-1" tunnel run my-tunnel
cloudflared --label "staging-connector-02" tunnel run my-tunnel
```

**優點：**
- 在 Cloudflare 儀表板中易於識別
- 多個連接器時清楚區分
- 改善運維可觀測性

**在儀表板顯示：**
```
Connector Name: Production-East-1
Connector ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

---

### 7. `--post-quantum, --pq` 後量子密碼學

**用途：** 啟用實驗性後量子安全加密

```bash
# 標準 TLS（預設）
cloudflared tunnel run --token <TOKEN>

# 啟用後量子密碼學
cloudflared --post-quantum tunnel run --token <TOKEN>
```

**特點：**
- ✅ 對抗未來量子電腦威脅
- ❌ 仍為實驗性，性能可能稍差
- 適合高安全性要求的環境

---

### 8. `--management-diagnostics` 診斷路由

**用途：** 啟用內部診斷和性能監控端點

```bash
# 預設啟用
cloudflared tunnel run --token <TOKEN>

# 禁用診斷（生產環境安全考量）
cloudflared --management-diagnostics=false tunnel run --token <TOKEN>
```

**提供的診斷端點：**
- `/debug/pprof`：CPU 和記憶體性能分析
- `/metrics`：Prometheus 格式指標
- `/debug/traces`：分佈式追蹤

**適用情景：**
- 開發/測試：保持啟用
- 生產：根據安全政策決定

---

### 9. `--overwrite-dns, -f` 覆寫 DNS 記錄

**用途：** 自動覆寫現有 DNS 記錄（若已存在）

```bash
# 正常（若 DNS 已存在則報錯）
cloudflared tunnel route dns my-tunnel example.com

# 強制覆寫現有 DNS
cloudflared --overwrite-dns tunnel route dns my-tunnel example.com
cloudflared -f tunnel route dns my-tunnel example.com
```

**適用情景：**
- 更新隧道指向
- 自動化部署腳本
- 避免手動 DNS 管理

---

## 實際使用範例

### 場景 1：生產環境隧道

```bash
cloudflared \
  --credentials-file /etc/cloudflare/tunnel.json \
  --region wnam \
  --label "Production-Main" \
  --output json \
  tunnel run my-prod-tunnel
```

**特點：**
- 使用凭証檔案方式（傳統）
- 指定西北美洲區域
- JSON 日誌用於監控系統
- 清楚的標籤便於識別

---

### 場景 2：開發環境測試

```bash
cloudflared \
  --output default \
  --label "Dev-Testing" \
  tunnel run --token <TOKEN>
```

**特點：**
- 使用 token 方式（推薦）
- 人類可讀的日誌格式
- 簡單快速的本地開發

---

### 場景 3：多區域部署

```bash
cloudflared \
  --region easia \
  --edge-ip-version auto \
  --label "Asia-Replica-1" \
  tunnel run --token <TOKEN>
```

**特點：**
- 部署在東亞區域
- 自動選擇 IP 版本以提高相容性
- 清楚標識為亞洲副本

---

### 場景 4：安全敏感環境

```bash
cloudflared \
  --post-quantum \
  --management-diagnostics=false \
  --label "High-Security" \
  tunnel run --token <TOKEN>
```

**特點：**
- 啟用後量子密碼學
- 禁用診斷以減少攻擊面
- 適合高度安全敏感的環境

---

## 環境變數設定

你也可以使用環境變數設定這些選項，便於容器化和自動化部署：

```bash
export TUNNEL_LOG_OUTPUT=json
export TUNNEL_CRED_FILE=/etc/cloudflare/tunnel.json
export TUNNEL_REGION=wnam
export TUNNEL_EDGE_IP_VERSION=4
export TUNNEL_EDGE_BIND_ADDRESS=192.168.1.100
export TUNNEL_POST_QUANTUM=true
export TUNNEL_MANAGEMENT_DIAGNOSTICS=false

cloudflared tunnel run my-tunnel
```

---

## 常見問題

### Q：應該使用 token 還是凭証檔案？
**A：** 推薦使用 token，因為：
- 更安全（可設定過期時間）
- 無需保存長期凭証檔案
- 更易於 CI/CD 整合

### Q：如何在 Docker 中使用？
**A：** 透過環境變數或掛載凭証卷：
```bash
docker run -e TUNNEL_TOKEN=<TOKEN> \
  -e TUNNEL_REGION=wnam \
  cloudflare/cloudflared:latest \
  tunnel run --token $TUNNEL_TOKEN
```

### Q：多個隧道如何並行運行？
**A：** 使用不同的標籤和凭証檔案分別執行，或在 systemd/Docker 中配置多個實例。

---

## 相關文件

- [Cloudflare Tunnel 官方文檔](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)
- cloudflared 版本：2025.11.1
