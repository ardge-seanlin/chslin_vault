# mDNS 訊息處理開發指南

> 本文檔為 AI 平台 mDNS 服務發現的開發參考指南。涵蓋訊息格式、實作要點和最佳實踐。

---

## 目錄

1. [快速參考](#快速參考)
2. [訊息結構](#訊息結構)
3. [伺服器實作](#伺服器實作-服務公告)
4. [客戶端實作](#客戶端實作-服務發現)
5. [常見實作模式](#常見實作模式)
6. [除錯指南](#除錯指南)

---

## 快速參考

### mDNS 基本資訊

| 項目 | 值 |
|------|-----|
| **多播地址 (IPv4)** | `224.0.0.251:5353` |
| **多播地址 (IPv6)** | `[ff02::fb]:5353` |
| **協議** | UDP 多播（不使用 TCP） |
| **訊息大小限制** | 1200 字節（標準） |
| **TXT 記錄單一鍵值對最大值** | 255 字元 |
| **標準 TTL** | 4500 秒（75 分鐘） |
| **短期 TTL** | 120 秒（用於 A 記錄） |

### 服務命名約定

```
服務格式: <instance>.<service type>.<proto>.<domain>

範例：
  ai-server-01._ai-platform._tcp.local
  ↑            ↑                ↑     ↑
  instance     service type     proto  domain
```

### DNS 記錄類型速查表

| 類型 | 代碼 | 用途 | 您的應用 |
|------|------|------|---------|
| **A** | 0x0001 | IPv4 位址對應 | 主機名 → IPv4 |
| **AAAA** | 0x001C | IPv6 位址對應 | 主機名 → IPv6 |
| **SRV** | 0x0021 | 服務定位 | 服務實例 → 埠號+主機 |
| **TXT** | 0x0010 | 文字屬性 | 設備詳細資訊 |
| **PTR** | 0x000C | 指標（反向查詢） | 服務發現 |

### Header 標誌速查表

| 標誌 | 含義 | 查詢值 | 應答值 |
|------|------|--------|--------|
| **QR** | 查詢(0)/應答(1) | 0 | 1 |
| **AA** | 權威應答 | 0 | 1 (mDNS) |
| **TC** | 訊息截斷 | 0 | 通常 0 |
| **RD** | 遞迴查詢 | 0 (mDNS) | 0 |
| **RA** | 遞迴可用 | 0 (mDNS) | 0 |
| **AD** | 認証資料 | 0 | 0 |
| **CD** | 檢查禁用 | 0 | 0 |

---

## 訊息結構

### 完整訊息佈局

```
mDNS Message (UDP)
├─ Header (12 bytes, 固定)
├─ Questions (可變, 典型 25-50 bytes)
├─ Answers (可變, 典型 100-300 bytes)
├─ Authority (mDNS 通常為空)
└─ Additional (mDNS 通常為空)

總大小: 通常 200-500 bytes (< 1200 byte 限制)
```

### Header 二進制格式

```
位元位置:  0              15 16 19 20 21 22 23 24 25 26 27 28 31
          ┌──────────────┬────┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
          │      ID      │ QR │OP│AA│TC│RD│RA│Z │AD│CD│RC│
          └──────────────┴────┴──┴──┴──┴──┴──┴──┴──┴──┴──┘

查詢範例:  0x0000 0x0000
應答範例:  0x0000 0x8400
```

### Flags 計算公式

```go
// 構建 Flags (16 bits)
flags := uint16(0)
flags |= (qr << 15)      // QR: bit 15
flags |= (opcode << 11)  // OpCode: bits 11-14
flags |= (aa << 10)      // AA: bit 10
flags |= (tc << 9)       // TC: bit 9
flags |= (rd << 8)       // RD: bit 8
flags |= (ra << 7)       // RA: bit 7
flags |= (z << 6)        // Z: bit 6
flags |= (ad << 5)       // AD: bit 5
flags |= (cd << 4)       // CD: bit 4
flags |= rcode           // RCode: bits 0-3

// 查詢: flags = 0x0000
// 應答: flags = 0x8400 (10000100 00000000)
```

### 域名編碼（DNS Label Compression）

```
域名: ai-server-01._ai-platform._tcp.local

壓縮編碼流程:
1. 分解標籤: [ai-server-01] [_ai-platform] [_tcp] [local]
2. 編碼每個標籤: <長度><字元><長度><字元>...
3. 結尾: 0x00 (終止符)

十六進制範例:
┌─────┬─────────────────┬────┬──────────────────┬────┬──────┬────┐
│ 10  │ ai-server-01    │ 0b │ _ai-platform     │ 04 │ _tcp │ 05 │
├─────┼─────────────────┼────┼──────────────────┼────┼──────┼────┤
│ 0x09 0x61 0x69 ... 01 │ ... │ ...              │ ... │ local│ 00 │
└─────┴─────────────────┴────┴──────────────────┴────┴──────┴────┘

指標壓縮 (Pointer):
如果該標籤已編碼，使用指標替代:
  0xC0 <offset>   // 指向訊息中的位置

作用: 減少訊息大小（特別是重複域名）
```

### Resource Record (RR) 結構

```
通用 RR 格式:

┌──────────┬──────┬────────┬──────┬───────────┬──────┐
│ NAME     │ TYPE │ CLASS  │ TTL  │ RDLENGTH  │ RDATA│
│ (變數)   │ 2B   │ 2B     │ 4B   │ 2B        │ 變數 │
└──────────┴──────┴────────┴──────┴───────────┴──────┘

CLASS 編碼:
  0x0001  →  IN (Standard)
  0x4001  →  IN | Cache Flush (設定最高位)
  0x8001  →  IN | Legacy Unicast

TTL 解釋:
  4500   →  標準 TTL (75 分鐘)
  120    →  短期 TTL (IP 位址)
  0      →  立即移除（Goodbye）
```

### 各類型 RDATA 格式

#### A 記錄 (IPv4)
```
RDLENGTH: 4 bytes
RDATA: 4 bytes IPv4 address

範例: 192.168.1.100
Hex: C0 A8 01 64
```

#### AAAA 記錄 (IPv6)
```
RDLENGTH: 16 bytes
RDATA: 16 bytes IPv6 address

範例: fd00::1
Hex: FD 00 00 00 00 00 00 00 00 00 00 00 00 00 00 01
```

#### SRV 記錄
```
RDLENGTH: variable (11+ bytes)
RDATA:
  Priority (2 bytes)   - 服務優先級
  Weight (2 bytes)     - 權重
  Port (2 bytes)       - 連接埠
  Target (variable)    - 目標主機 (域名指標)

範例: Priority=0, Weight=0, Port=8080
Hex: 00 00 00 00 1F 90 (+ target pointer)
```

#### TXT 記錄
```
RDLENGTH: variable (通常 100-500 bytes)
RDATA: Character String Array
  [Length (1B)][Data (Length bytes)] [Length][Data] ...

範例:
  "device-id=550e8400-e29b-41d4-a716-446655440000"
  "api-port=8080"
  "network-encrypted=gAAAAABl5X2m..."

Hex:
  39 64 65 76 69 63 65 2d 69 64 3d ... (57 bytes 字串)
  0D 61 70 69 2d 70 6f 72 74 3d 38 30 38 30    (13 bytes 字串)
  ...
```

#### PTR 記錄
```
RDLENGTH: variable
RDATA: Domain Name (pointer)

範例: _ai-platform._tcp.local → ai-server-01._ai-platform._tcp.local
Hex: C0 XX (指標到目標域名)
```

---

## 伺服器實作（服務公告）

### 啟動時發送的訊息

```
服務啟動流程:

[1] Conflict Detection (Probe)
    發送查詢: "ai-server-01.local" (A record)
    等待: 250ms
    如果無回應 → 名稱可用

[2] Service Announcement
    廣播多播訊息到 224.0.0.251:5353
    包含:
      - PTR: _ai-platform._tcp.local → ai-server-01._ai-platform._tcp.local
      - SRV: ai-server-01._ai-platform._tcp.local → ai-server-01.local:8080
      - A:   ai-server-01.local → 192.168.1.100
      - TXT: device-id=..., network-encrypted=...

[3] Wait for Discovery
    等待客戶端查詢
```

### 定期公告

```
時間表:

T=0s:      Initial Announcement (TTL=4500)
T=3600s:   Renewal Announcement
T=7200s:   Next Renewal
...

機制:
- 80% TTL 時自動發送再查詢
- 接收應答後，重新設定 TTL
- 目的: 保持記錄活躍，預防遺忘
```

### 服務下線

```
關閉前發送:

TTL=0 訊息到 224.0.0.251:5353
包含所有之前廣播的記錄
客戶端接收後立即清除快取

範例訊息:
  SRV: ai-server-01._ai-platform._tcp.local (TTL=0)
  A:   ai-server-01.local (TTL=0)
  TXT: device-id=... (TTL=0)
```

### 伺服器實作檢查清單

- [ ] 啟動時執行衝突檢測（Probe）
- [ ] 成功後發送初始公告（Announcement）
- [ ] 實作定期更新機制（TTL 80% 時）
- [ ] 關閉時發送 Goodbye 訊息（TTL=0）
- [ ] 設定正確的 Cache Flush 標誌（0x4001）
- [ ] TXT 記錄超過 255 字元時進行分片
- [ ] 處理服務更新（IP 變更時）
- [ ] 記錄所有廣播訊息便於除錯

---

## 客戶端實作（服務發現）

### 發現流程（推薦）

```
[Phase 1] 快速查詢 (0-130ms)
├─ 發送 PTR 查詢: _ai-platform._tcp.local
├─ 等待回應: 125ms
└─ 收集發現的服務清單

[Phase 2] 詳細查詢 (可選)
├─ 對每個服務發送 SRV 查詢
├─ 自動解析 A/AAAA 記錄
└─ 解析 TXT 記錄

[Phase 3] 持續監聽 (背景)
├─ 綁定多播接收器
├─ 監聽 224.0.0.251:5353
└─ 即時檢測服務變化

[Phase 4] 定期重新查詢 (可選)
├─ 每 3600s 執行一次
└─ 防止快取過期和遺漏
```

### 查詢訊息構建

```go
// 構建 PTR 查詢
func buildPTRQuery(serviceName string) []byte {
    // Header: QR=0 (Query), OpCode=0, AA=0, TC=0, RD=0
    // QDCOUNT=1, ANCOUNT=0, NSCOUNT=0, ARCOUNT=0

    // Question:
    // QNAME: _service._tcp.local (encoded)
    // QTYPE: 0x000C (PTR)
    // QCLASS: 0x0001 (IN)
}

// 構建 SRV 查詢
func buildSRVQuery(instanceName string) []byte {
    // Header: QR=0
    // Question:
    // QNAME: instance._service._tcp.local
    // QTYPE: 0x0021 (SRV)
    // QCLASS: 0x0001
}
```

### 應答處理

```go
// 接收多播訊息的標準流程
func handleMDNSResponse(data []byte) {
    // 1. 解析 Header
    header := parseHeader(data)
    if header.QR != 1 {  // 確保是應答
        return
    }

    // 2. 解析 Answer 部分
    for i := 0; i < header.AnCount; i++ {
        rr := parseResourceRecord(data)

        switch rr.Type {
        case 0x000C:  // PTR
            handlePTR(rr)
        case 0x0021:  // SRV
            handleSRV(rr)
        case 0x0001:  // A
            handleA(rr)
        case 0x001C:  // AAAA
            handleAAAA(rr)
        case 0x0010:  // TXT
            handleTXT(rr)
        }
    }

    // 3. TTL 為 0 時，移除服務
    if rr.TTL == 0 {
        removeService(rr.Name)
    }
}
```

### 客戶端實作檢查清單

- [ ] 實作多播 Socket（UDP，非 TCP）
- [ ] 加入多播群組（224.0.0.251:5353）
- [ ] 設定 IP_MULTICAST_LOOPBACK=false（不接收自己的訊息）
- [ ] 實作 DNS 域名解碼（標籤壓縮）
- [ ] 處理 TTL=0 的 Goodbye 訊息
- [ ] 實作服務快取（防止重複查詢）
- [ ] 監聽多播訊息（背景協程）
- [ ] 定期清理過期快取（超過 TTL）

---

## 常見實作模式

### 模式 1：簡單服務公告

```go
package main

import (
    "log"
    "net"
    "time"
)

// PublishService 公告服務
func PublishService(
    serviceName string,
    port int,
    hostIP string,
    txtRecords map[string]string,
) error {
    // 1. 構建訊息
    message := buildAnnouncement(serviceName, port, hostIP, txtRecords)

    // 2. 發送到多播地址
    addr := net.UDPAddr{
        IP:   net.ParseIP("224.0.0.251"),
        Port: 5353,
    }

    conn, err := net.DialUDP("udp4", nil, &addr)
    if err != nil {
        return err
    }
    defer conn.Close()

    _, err = conn.Write(message)
    return err
}

// StartAnnounceTimer 定期公告
func StartAnnounceTimer(
    serviceName string,
    port int,
    hostIP string,
    txtRecords map[string]string,
) {
    ticker := time.NewTicker(75 * time.Minute) // TTL 的 80%

    go func() {
        for range ticker.C {
            PublishService(serviceName, port, hostIP, txtRecords)
        }
    }()
}
```

### 模式 2：服務發現監聽

```go
package main

import (
    "net"
    "time"
)

// StartServiceDiscovery 啟動服務發現
func StartServiceDiscovery(
    serviceName string,
    onFound func(service ServiceInfo),
    onLost func(name string),
) error {
    // 1. 建立多播監聽
    addr := net.UDPAddr{
        Port: 5353,
        IP:   net.ParseIP("0.0.0.0"),
    }

    conn, err := net.ListenUDP("udp4", &addr)
    if err != nil {
        return err
    }

    // 2. 加入多播群組
    group := net.ParseIP("224.0.0.251")
    iface, _ := net.InterfaceByName("en0")
    conn.SetMulticastInterface(iface)
    conn.JoinGroup(group)

    // 3. 背景監聽
    go func() {
        buffer := make([]byte, 1500)

        for {
            n, _, err := conn.ReadFromUDP(buffer)
            if err != nil {
                break
            }

            // 4. 解析訊息
            response := parseMDNSResponse(buffer[:n])

            // 5. 調用回調
            for _, rr := range response.Answers {
                if rr.TTL == 0 {
                    onLost(rr.Name)
                } else {
                    onFound(convertToServiceInfo(rr))
                }
            }
        }
    }()

    return nil
}
```

### 模式 3：TXT 記錄加密

```go
package main

import (
    "crypto/aes"
    "crypto/cipher"
    "encoding/base64"
)

// EncryptTXTData 加密 TXT 資料
func EncryptTXTData(
    plaintext []byte,
    key []byte,
) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTXTData 解密 TXT 資料
func DecryptTXTData(
    ciphertext string,
    key []byte,
) ([]byte, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return nil, err
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := data[:gcm.NonceSize()]
    plaintext, err := gcm.Open(nil, nonce, data[gcm.NonceSize():], nil)

    return plaintext, err
}

// 使用範例
func ExampleTXTEncryption() {
    key := []byte("32-byte-key-for-aes256-encryp!!") // 32 bytes for AES-256

    plaintext := []byte(`{"cpu_cores": 16, "memory_gb": 64}`)

    encrypted, _ := EncryptTXTData(plaintext, key)
    txtRecord := map[string]string{
        "hardware-encrypted": encrypted,
    }

    // ... 廣播 txtRecord
}
```

### 模式 4：服務狀態管理

```go
package main

import (
    "sync"
    "time"
)

// ServiceRegistry 服務註冊表
type ServiceRegistry struct {
    mu       sync.RWMutex
    services map[string]*DiscoveredService
}

type DiscoveredService struct {
    Name      string
    IP        string
    Port      int
    TXT       map[string]string
    ExpiresAt time.Time
}

// Add 添加服務
func (r *ServiceRegistry) Add(service DiscoveredService) {
    r.mu.Lock()
    defer r.mu.Unlock()

    service.ExpiresAt = time.Now().Add(time.Duration(getTTL()) * time.Second)
    r.services[service.Name] = &service
}

// Remove 移除服務
func (r *ServiceRegistry) Remove(name string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    delete(r.services, name)
}

// GetActive 取得活躍服務
func (r *ServiceRegistry) GetActive() []DiscoveredService {
    r.mu.RLock()
    defer r.mu.RUnlock()

    now := time.Now()
    active := make([]DiscoveredService, 0)

    for _, service := range r.services {
        if service.ExpiresAt.After(now) {
            active = append(active, *service)
        }
    }

    return active
}

// Cleanup 定期清理過期服務
func (r *ServiceRegistry) Cleanup() {
    ticker := time.NewTicker(1 * time.Minute)

    go func() {
        for range ticker.C {
            r.mu.Lock()
            now := time.Now()

            for name, service := range r.services {
                if service.ExpiresAt.Before(now) {
                    delete(r.services, name)
                }
            }

            r.mu.Unlock()
        }
    }()
}
```

---

## 除錯指南

### 使用 Wireshark 檢查訊息

```
Wireshark 過濾器:

mdns                  # 所有 mDNS 訊息
mdns.resp == 1        # 只看應答
mdns.qry == 1         # 只看查詢
mdns.txt              # 只看 TXT 記錄
ip.dst == 224.0.0.251 # 只看多播訊息
```

### 常見問題排查

| 問題 | 檢查項目 | 解決方案 |
|------|--------|--------|
| **服務無法被發現** | 1. 訊息是否發送到 224.0.0.251:5353 | 檢查防火牆 |
| | 2. 是否設定了 Cache Flush 標誌 | 設定 0x4001 |
| | 3. TTL 是否為 0（代表刪除） | 應為 4500 或 120 |
| **收不到多播訊息** | 1. 是否加入多播群組 | 呼叫 JoinGroup() |
| | 2. IP_MULTICAST_LOOPBACK 設定 | 應為 false |
| | 3. 網卡是否支援多播 | 檢查網卡狀態 |
| **訊息格式錯誤** | 1. Header 標誌是否正確 | 查詢用 0x0000，應答用 0x8400 |
| | 2. 域名編碼是否正確 | 驗證標籤壓縮 |
| | 3. RDLENGTH 是否與實際資料匹配 | 檢查十六進制 |
| **TXT 記錄超大** | 1. 單一鍵值對是否超過 255 字元 | 進行分片 |
| | 2. 整個 TXT 是否超過訊息大小 | 使用外部資源指標 |
| **服務過期問題** | 1. TTL 管理是否正確 | 實作 TTL 計時器 |
| | 2. 是否發送了再查詢 | 80% TTL 時發送 |

### 日誌記錄範例

```go
// 記錄所有 mDNS 活動
func LogMDNSEvent(eventType string, details map[string]interface{}) {
    log.Printf(
        "[mDNS] %s | Service: %v, IP: %v, Port: %v, TTL: %v",
        eventType,
        details["service"],
        details["ip"],
        details["port"],
        details["ttl"],
    )
}

// 使用:
LogMDNSEvent("SERVICE_FOUND", map[string]interface{}{
    "service": "ai-server-01",
    "ip":      "192.168.1.100",
    "port":    8080,
    "ttl":     4500,
})
```

### 性能監控指標

```
應監控的指標:

1. 查詢延遲
   - 從發送查詢到接收應答的時間
   - 目標: < 50ms

2. 訊息大小
   - 發送/接收訊息的大小
   - 目標: < 500 bytes

3. 服務發現時間
   - 從啟動到發現所有服務的時間
   - 目標: < 500ms

4. 多播丟包率
   - 發送訊息數 / 接收訊息數
   - 目標: < 5%

5. 快取命中率
   - 來自快取的查詢 / 總查詢
   - 目標: > 80%
```

---

## 實作檢查清單

### 伺服器端

- [ ] 實作衝突檢測（Probe）機制
- [ ] 發送初始公告（Announcement）
- [ ] 實作定期更新（TTL 80% 時）
- [ ] 正確設定 Cache Flush 標誌（0x4001）
- [ ] 處理服務更新（IP 變更）
- [ ] 發送 Goodbye 訊息（TTL=0）
- [ ] 驗證訊息格式（使用 Wireshark）
- [ ] 處理 TXT 記錄超大情況（分片或指標）
- [ ] 加密敏感的 TXT 資訊
- [ ] 記錄所有廣播訊息
- [ ] 測試網路隔離場景

### 客戶端

- [ ] 建立 UDP 多播 Socket
- [ ] 加入多播群組（224.0.0.251:5353）
- [ ] 實作 DNS 域名解碼（標籤壓縮）
- [ ] 正確解析各種記錄類型
- [ ] 處理 TTL=0（Goodbye 訊息）
- [ ] 實作服務快取機制
- [ ] 背景監聽多播訊息
- [ ] 定期清理過期快取
- [ ] 解密 TXT 記錄
- [ ] 驗證應答訊息（Header 檢查）
- [ ] 測試服務動態加入/移除

### 安全相關

- [ ] 驗證訊息來源（多播域內）
- [ ] 實作 TXT 記錄加密（AES-256）
- [ ] 實作金鑰管理機制
- [ ] 定期輪換加密金鑰
- [ ] 防止 mDNS 轟炸攻擊
- [ ] 限制服務發現範圍（僅本地網路）

---

## 參考資源

- **RFC 6762**: Multicast DNS (mDNS) 標準
- **RFC 6763**: DNS Service Discovery (DNS-SD)
- **Wireshark**: 網路封包分析工具
- **mDNS 記錄類型**: https://tools.ietf.org/html/rfc1035

---

## 版本歷史

| 版本 | 日期 | 更新 |
|------|------|------|
| 1.0 | 2025-11-14 | 初始版本 |

