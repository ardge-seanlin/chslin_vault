## 語言規則

### 回應語言
- **強制要求**：無論問題使用何種語言，回應時必須使用繁體中文（繁體中文）
- **術語規範**：使用台灣風格的術語和正確的專有名詞：
  - 「軟體」而非「软件」
  - 「程式」而非「程序」
  - 「伺服器」而非「服务器」
  - 「網路」而非「网络」
  - 「資料庫」而非「数据库」
  - 「檔案」而非「文件」

### 禁用語言
- **嚴格禁止**：簡體中文（簡體中文）字符
- **嚴格禁止**：中國大陸術語（例如「鼠标、软件、硬件、文件夹」）

### 程式碼註解
- **強制要求**：所有程式碼註解必須使用英文書寫
- **禁止**：在程式碼註解中使用中文或其他非英文語言
- 範例：
  - 良好：`// Initialize database connection`
  - 不良：`// 初始化資料庫連線`

### 特殊註解標記

#### 1. TODO
- **用途**：標記不完整的功能或計劃的改進
- **緊急程度**：低 - 用於未來的工作
- **格式**：`// TODO: <清晰的描述>`

#### 2. FIXME
- **用途**：標記已知的錯誤或不正確的實現
- **緊急程度**：中到高
- **格式**：`// FIXME: <清晰的描述>`

#### 3. XXX
- **用途**：嚴重問題、臨時代碼或變通方法的強力警告
- **緊急程度**：中到高
- **格式**：`// XXX: <清晰的警告和說明>`

**最佳實踐：**
- 始終提供清晰、可執行的描述
- 優先選擇立即修復問題而不是添加標記

## Go 編碼標準

遵循官方 Go 風格指南和社區最佳實踐。

**參考資源：**
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### 1. 命名慣例

**基本規則：**
- 使用 **MixedCaps** 或 **mixedCaps**，不使用 snake_case
- 縮寫應該全部大寫：`HTTP`、`URL`、`ID`、`API`
- 簡潔但清晰

**套件（Packages）：** 短、小寫、單字名稱（例如 `user`、`http`、`auth`）

**函數（Functions）：** 導出函數使用 MixedCaps，未導出函數使用 mixedCaps

**變數（Variables）：** 小範圍內使用短名稱（`i`、`n`、`err`），較大範圍內使用描述性名稱

**常數（Constants）：** MixedCaps，不是 ALL_CAPS（例如 `MaxRetries`）

**介面（Interfaces）：** 單一方法介面以 `-er` 結尾（例如 `Reader`、`Writer`）

### 2. 程式碼組織

**檔案結構：**
- 每個檔案只包含一個主要概念
- 順序：類型 → 建構函數 → 方法 → 輔助函數

**匯入分組：** 分為三組並用空白行隔開：
```go
import (
    // 標準函式庫
    "context"
    "fmt"

    // 外部依賴
    "github.com/gin-gonic/gin"

    // 內部套件
    "yourproject/pkg/auth"
)
```

### 3. 函數指南

- 盡可能讓函數保持在 50 行以內
- 限制參數數量為 3-4 個；如果更多則使用結構體
- 將錯誤作為最後一個回傳值
- Context 應該始終是第一個參數

### 4. 方法和接收者

**接收者命名：**
- 使用一致、簡短的接收者名稱（1-2 個字符）
- 在同一類型的所有方法中使用相同的接收者名稱

**指標 vs 值：**
- 當方法需要修改接收者或結構體很大時使用指標接收者
- 對於小的、不可變的類型使用值接收者
- 保持一致：如果任何方法使用指標，所有方法都應該使用

**Context：**
- Context 應該始終是第一個參數
- 永遠不要在結構體中儲存 Context
- 通過呼叫鏈顯式傳遞 Context

### 5. 常見 Go 慣用法

**提前回傳：**
```go
func ProcessData(data []byte) error {
    if len(data) == 0 {
        return errors.New("empty data")
    }
    // 主要邏輯
    return nil
}
```

**使用 Defer 進行清理：**
```go
func ReadFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    return io.ReadAll(f)
}
```

**接受介面，回傳結構體：**
```go
func ProcessData(r io.Reader) (*Result, error) {
    // ...
}
```

## Go 錯誤處理規則

**參考資源：**
- [Effective Go - Errors](https://go.dev/doc/effective_go#errors)
- [Uber Go Style Guide - Error Handling](https://github.com/uber-go/guide/blob/master/style.md#error-handling)

### 核心原則

1. **始終檢查錯誤**：除非有充分理由，否則不要使用 `_` 忽略錯誤

2. **錯誤包裝**：使用 `%w` 維持錯誤鏈以支援 `errors.Is()` 和 `errors.As()`
   ```go
   return fmt.Errorf("process data failed: %w", err)
   ```

3. **錯誤消息**：使用小寫，無尾部標點符號，提供上下文

4. **Sentinel 錯誤**：為預期的錯誤定義套件層級的錯誤變數
   ```go
   var (
       ErrNotFound     = errors.New("resource not found")
       ErrUnauthorized = errors.New("unauthorized access")
   )
   ```

5. **自訂錯誤類型**：對於需要額外上下文的錯誤，實現 `Error()` 和 `Unwrap()`

6. **何時 Panic**：僅用於無法恢復的程式設計錯誤或初始化失敗

### 常見模式

**提前回傳：**
```go
func ProcessRequest(req *Request) error {
    if req == nil {
        return errors.New("request is nil")
    }
    if err := validateRequest(req); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return nil
}
```

**使用 Context 重試：**
```go
func DoWithRetry(ctx context.Context, maxRetries int, fn func() error) error {
    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        if ctx.Err() != nil {
            return ctx.Err()
        }
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }
        // 指數退避加抖動
        if attempt < maxRetries-1 {
            backoff := time.Duration(100*(1<<uint(attempt))) * time.Millisecond
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
    return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
```

**使用 Context 設定超時：**
```go
func FetchWithTimeout(userID string) (*User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return FetchUser(ctx, userID)
}
```

## Go 測試規則

### 最佳實踐
- 測試必須清晰、有意義且易於維護
- 對多個相似的案例使用表驅動測試
- 每個測試應該關注單一行為

### 測試組織

**單元測試：**
- 獨立測試單一函數/方法
- 無外部依賴
- 快速執行（毫秒級）
- 檔案命名：`*_test.go`

**整合測試：**
- 測試組件之間的互動
- 可能使用外部依賴
- 使用編譯標籤：`// +build integration`
- 單獨執行：`go test -tags=integration ./...`

### 表驅動測試

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid", "user@example.com", false},
        {"invalid", "not-an-email", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Mock 哲學

**首先手動編寫 Mock：**
- Go 的介面設計使得手動 mock 很直接
- 僅對大介面（10+ 個方法）使用 mock 工具

**簡單 Mock 範例：**
```go
type mockUserRepository struct {
    users map[string]*User
    err   error
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.users[id], nil
}
```

### 覆蓋率哲學
- **不要**為了增加覆蓋率百分比而編寫測試
- **只有**測試有意義的情景來驗證實際行為
- 專注於業務邏輯、邊界案例和錯誤處理
- 覆蓋率指標是指標，不是目標

### 驗證工作流程
- **有 Makefile 時**：執行 `make lint` 和 `make test`
- **沒有 Makefile 時**：執行 `golangci-lint run` 和 `go test -v ./...`
- **要求**：在認為工作完成前，兩者都必須通過

## 程式碼重構規則

### 核心原則
- **禁止**：無測試的重構
- **強制要求**：在每次重構前編寫測試
- **理念**：小步驟、頻繁提交、始終可運作的程式碼

### 何時重構（程式碼壞味）

1. **重複程式碼** - 多個地方有相同邏輯
2. **冗長函數** - 函數 > 50 行或做多件事
3. **長參數列表** - 超過 3-4 個參數
4. **大型結構體** - 過多字段/方法
5. **深層嵌套** - 超過 3 層的 if/for
6. **功能依賴** - 函數使用來自另一結構體的大量資料

### 測試驅動重構 (TDR) 工作流程

**步驟 1：首先編寫測試**
- 為要重構的程式碼編寫全面測試
- 測試必須涵蓋現有功能和邊界案例
- 執行測試以確保全部通過（基線）

**步驟 2：分小步重構**
- 一次進行一項改進
- 每一步後執行測試
- 每次成功重構後提交

**步驟 3：驗證並提交**
- 執行 `make lint && make test`
- 確保所有測試仍然通過
- 使用 `refactor:` 類型提交

### 常見重構技術

| 技術 | 何時使用 |
|-----------|-------------|
| **提取函數** | 函數 > 50 行或做多件事 |
| **提取介面** | 需要多態性或可測試性 |
| **引入參數物件** | 超過 3-4 個參數 |
| **用常數替換魔數** | 沒有上下文的硬編碼數字 |

### 範圍控制

**黃金規則**：一個程式碼壞味，一個 PR

- 每次會話專注於一個特定問題
- 避免「既然我在這裡，也讓我修復...」（範圍蔓延）
- 為其他問題建立單獨的分支

### SOLID 原則參考
- **S (單一責任)**：每個函數/結構體只做一件事
- **O (開/閉)**：可擴展而無需修改現有程式碼
- **L (Liskov 替換)**：子類型可替代父類型
- **I (介面隔離)**：小的、專注的介面
- **D (依賴反轉)**：依賴抽象而非具體實現

### 何時停止重構
- 程式碼壞味已解決
- 程式碼清晰且易於維護
- 進一步的更改沒有增加實際價值
- **YAGNI**：你不會需要它（You Aren't Gonna Need It）

## Git 操作規則

### 1. Git Flow 分支類型

**長期分支：**
- `main` - 生產環境
- `develop` - 開發整合
- `support/*` - 支援多個版本

**臨時分支：**
- `feature/*` - 新功能開發
- `release/*` - 發布準備
- `hotfix/*` - 生產緊急修復

**分支命名：** 使用小寫和連字符：`<type>/<descriptive-name>`

範例：`feature/user-authentication`、`fix/database-connection-leak`

### 2. 提交消息標準（Conventional Commits）

#### 結構
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### 提交類型
- `feat:` - 新功能（SemVer 中的 MINOR）
- `fix:` - 錯誤修復（SemVer 中的 PATCH）
- `docs:` - 文件變更
- `style:` - 程式碼風格變更（格式化，無程式碼變更）
- `refactor:` - 程式碼重構
- `perf:` - 性能改進
- `test:` - 添加或更新測試
- `build:` - 構建系統或依賴項變更
- `ci:` - CI/CD 配置變更
- `chore:` - 其他變更

#### 破壞性變更
- 在類型/範圍後添加 `!`：`feat(api)!: breaking change`
- 或使用頁腳：`BREAKING CHANGE: description`（SemVer 中的 MAJOR）

#### 提交消息範例

**範例 1：測試**
```
test: add comprehensive tests for binfile and generator packages

Add extensive unit and integration tests for core functionality:

pkg/binfile:
- Unit tests for Writer and Reader (28 tests)
- Integration tests for write-read cycles (10 tests)

pkg/manifest/generator:
- Unit tests for template processing (14 tests)
- HTTP download tests with retry mechanism (9 tests)

All tests pass with race detection enabled.
```

**範例 2：功能**
```
feat(api): add user authentication and authorization system

Implement JWT-based authentication with role-based access control:

pkg/auth:
- JWT token generation and validation
- Token refresh mechanism with sliding expiration

pkg/middleware:
- Authentication middleware for protected routes
- Role-based authorization middleware

Database migrations included for user and role tables.
```

**範例 3：修復**
```
fix(database): resolve connection pool exhaustion under high load

Fix goroutine leak causing connection pool exhaustion:

Root Cause:
- Database connections not properly released in error paths
- Context cancellation not properly handled

Solution:
- Add deferred connection close in all query functions
- Implement proper context cancellation handling
- Add connection pool metrics for monitoring

Testing:
- Add stress tests simulating 1000 concurrent requests
- Verify no connection leaks under error conditions
```

**範例 4：性能**
```
perf(cache): optimize Redis operations for frequently accessed data

Implement caching layer to reduce database load:

Changes:
- Add Redis-based caching for user profile data
- Implement cache-aside pattern with TTL of 5 minutes
- Add cache invalidation on profile updates

Performance Impact:
- User profile API response time: 250ms → 15ms (94% improvement)
- Database query reduction: 85% for cached endpoints
- Redis hit rate: 92% after warmup
```

### 3. 檔案狀態管理
- **禁止**：`git add`、`git reset`、`git restore`、`git stash`，除非明確要求
- **條件**：除非要求，否則不執行影響檔案暫存狀態的操作

### 4. 提交前工作流程
當指定特定檔案用於提交時：
1. 執行 `git diff --staged <specified files>` 以確認變更
2. 根據變更提供 gitflow 分支名稱建議
3. 等待確認後再繼續

### 5. 提交執行
- **強制要求**：使用 `git commit <specified files>` 格式
- **用途**：確保提交範圍精確限制於指定檔案
- **禁止**：使用不指定檔案的 `git commit`

### 6. 提交消息要求
- **語言**：所有提交消息必須以英文書寫
- **標題長度**：主題行必須 72 個字符或更少
- **內容基礎**：僅基於 `git diff --staged` 的實際變更提供建議
- **格式**：嚴格遵循 Conventional Commits 規範
- **簽名**：不要添加自動簽名（例如「Generated with Claude Code」、「Co-Authored-By」）到提交消息

### 7. 檔案移動和重命名
- **強制要求**：對 git 管理的檔案使用 `git mv` 而非常規的 `mv`
- **用途**：保留 git 歷史記錄並確保正確追蹤
