# 設計哲學與開發路線圖

本文件描述 Goodman 的設計哲學、開發方法與演進歷程。

## 核心開發哲學

```
MVP 先行 → 測試覆蓋 → 功能擴展 → 使用者體驗
```

Goodman 採用漸進式開發方法，優先交付可運作的軟體而非完整規劃，同時透過持續測試與重構維持程式碼品質。

## 設計原則

| 原則 | 實踐 |
|------|------|
| **MVP 先行** | 先交付核心功能，細節後續完善 |
| **測試先行重構** | 沒有測試覆蓋就不重構 |
| **漸進式複雜度** | 先建立簡單功能，逐步擴展 |
| **介面抽象** | 先定義契約，再實作 |
| **單一職責** | 每個 PR 只處理一件事 |
| **文件即程式碼** | 功能變更同時更新文件 |

## 開發階段

### 階段一：MVP 驗證

**目標**：驗證概念可行

```
Initial commit
    ↓
核心實作 (types, executor, runner)
    ↓
範例與文件
```

**關鍵決策**：
- YAML 而非 JSON → Git diff 友善
- 外部工具 (curl/grpcurl) 而非自建 client → 借助成熟工具
- Table-driven 測試 → 減少樣板程式碼

**核心資料模型**：
```
Collection
├── Environments[]
└── Suites[]
    └── Tests[]
        ├── Request
        ├── Assertions[]
        └── Table[]
```

### 階段二：使用者體驗

**目標**：讓工具好用

**問題**：長時間測試套件需要等待所有測試完成才能看到結果。

**解決方案**：串流輸出，即時回饋。

```
之前：等待所有測試 → 顯示結果
之後：✓ test-1 (5ms)    ← 即時
      ✓ test-2 (3ms)    ← 即時
      ✗ test-3 (4ms)    ← 即時
```

**增強功能**：
- 串流報告器搭配 flush 機制
- Table-driven 測試聚合顯示 `(passed/total)`
- 彩色輸出與狀態指示器

### 階段三：技術債清償

**目標**：為未來開發建立穩固基礎

**方法**：先測試，再重構

```
新增單元測試 (types, assertion, config, report, loader, executor)
    ↓
抽取硬編碼常數
    ↓
安心重構
```

**規則**：沒有測試就不重構

**成果**：
- 所有套件完整測試覆蓋
- 常數抽取至 `internal/constants/`
- 透過介面抽象降低耦合

### 階段四：橫向擴展 - API 規格

**目標**：驗證測試對 API 規格的覆蓋率

**問題**：如何知道所有 API 端點都有測試？

**解決方案**：解析 API 規格並與測試套件比對

```
                ┌─ proto parser ──┐
API 規格 ───────┼─ protoc parser ─┼──→ 統一 IR ──→ 驗證器
                └─ swagger parser ┘
```

**架構設計**：
1. 在 `pkg/spec` 定義統一的中間表示 (IR)
2. 實作各格式解析器轉換至 IR
3. 建立基於 IR 運作的驗證器

**支援格式**：
| 格式 | 說明 |
|------|------|
| `proto` | 原生 .proto 檔案 |
| `protoc` | protoc-gen-doc JSON 輸出 |
| `swagger` | Swagger/OpenAPI 2.0 |

### 階段五：縱向擴展 - gRPC Streaming

**目標**：支援所有 gRPC 通訊模式

**方法**：漸進式複雜度

```
Unary (1:1) → Server Stream (1:N) → Client Stream (N:1) → Bidirectional (N:N)
   簡單            中等                 中等                  複雜
```

每一步建立在前一步之上：

| 模式 | 輸入 | 輸出 |
|------|------|------|
| `unary` | `payload` | `response` |
| `server_streaming` | `payload` | `messages[]` |
| `client_streaming` | `payloads[]` | `response` |
| `bidirectional` | `payloads[]` | `messages[]` |

**附加功能**：
- `max_messages` 提前終止串流
- gRPC reflection 支援
- 壓縮設定

### 階段六：生產就緒

**目標**：企業級可靠性

**功能**：

| 功能 | 解決的問題 |
|------|------------|
| Retry Policy | 網路不穩定、暫時性失敗 |
| TLS 設定 | 安全連線、mTLS |
| 條件執行 | 環境特定測試 |
| Fail Fast | CI/CD 效率 |
| Shuffle | 偵測測試順序相依 |

**指數退避重試**：
```yaml
request:
  retries: 3
  retry_delay: 100ms
  retry_backoff: exponential
  retry_on: [502, 503, 504]
```

**條件執行**：
```yaml
tests:
  - name: prod-only-test
    when: "{{env}} == 'prod'"
```

### 階段七：開發者體驗

**目標**：降低撰寫測試的門檻

**問題**：YAML 錯誤只有在執行時才會發現

**解決方案**：JSON Schema 驗證 + lint 命令

```
JSON Schema 產生器
    ↓
Lint 命令 (pre-commit, CI)
    ↓
IDE 整合 (自動完成、驗證)
```

**功能**：
- 透過反射從 Go 型別產生 Schema
- 錯誤訊息對應行號
- 未知欄位的錯字建議
- Claude Code skill 整合 AI 輔助

## 版本里程碑

| 版本 | 關鍵功能 |
|------|----------|
| **v0.1.0** | 核心框架、HTTP/gRPC 執行、串流輸出 |
| **v0.2.0** | 單元測試、hook captures、全域 headers |
| **v0.3.0** | API 規格解析器、驗證器、proto/swagger 支援 |
| **v0.4.0** | gRPC streaming (server/client/bidirectional)、reflection |
| **v0.5.0** | Retry policy、條件執行、JSON Schema lint、Claude Code skill |

## 架構演進

### 初始架構 (v0.1)
```
CLI → Loader → Runner → Executor → Reporter
                           ↓
                      Assertion
```

### 目前架構 (v0.5)
```
┌─────────────────────────────────────────────────────────────────┐
│                            CLI                                   │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐ ┌───────┐        │
│  │  run   │ │  list  │ │  lint  │ │ validate │ │ skill │        │
│  └────────┘ └────────┘ └────────┘ └──────────┘ └───────┘        │
└─────────────────────────────────────────────────────────────────┘
        │           │           │           │
        ▼           │           ▼           ▼
┌──────────────┐    │    ┌──────────┐ ┌──────────────┐
│    Loader    │    │    │  Schema  │ │    Parser    │
│  Collection  │    │    │Validator │ │ proto/swagger│
│    Suite     │    │    └──────────┘ └──────────────┘
│ Environment  │    │                        │
└──────────────┘    │                        ▼
        │           │                 ┌──────────────┐
        ▼           │                 │   spec.IR    │
┌──────────────┐    │                 └──────────────┘
│    Runner    │    │                        │
│  • Retry     │    │                        ▼
│  • Condition │    │                 ┌──────────────┐
│  • Shuffle   │    │                 │  Validator   │
└──────────────┘    │                 │  • Coverage  │
        │           │                 │  • Naming    │
        ▼           │                 └──────────────┘
┌──────────────┐    │
│   Executor   │    │
│  • HTTP      │    │
│  • gRPC      │    │
│  • Streaming │    │
└──────────────┘    │
        │           │
        ▼           │
┌──────────────┐    │
│  Assertion   │◄───┘
│  • JSONPath  │
│  • Operators │
└──────────────┘
        │
        ▼
┌──────────────┐
│   Reporter   │
│  • CLI       │
│  • JSON      │
└──────────────┘
```

## 未來方向

未來開發的潛在領域：

1. **原生 HTTP Client** - 取代 curl 相依以提升效能
2. **原生 gRPC Client** - 取代 grpcurl 以改善 streaming
3. **OpenAPI 3.0 支援** - 擴展解析器能力
4. **平行測試執行** - 在套件內並行執行測試
5. **測試產生** - 從 API 規格產生測試骨架
6. **外掛系統** - 自訂執行器與報告器

## 經驗總結

1. **從簡單開始**：外部工具 (curl/grpcurl) 降低初期複雜度
2. **及早測試**：測試覆蓋讓重構有信心
3. **快速迭代**：單一職責的小 PR 更容易審查
4. **持續文件化**：程式碼同步文件避免文件債
5. **傾聽使用者**：串流輸出來自真實使用回饋
6. **適時抽象**：介面從具體實作中浮現
