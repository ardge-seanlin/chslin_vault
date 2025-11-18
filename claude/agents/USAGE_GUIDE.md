# Rust Senior Engineer Agent - 使用指南

## 概述

Rust Senior Engineer 是一個專業的 Rust 代理，專門設計用於：

- 🏗️ 架構設計和系統評審
- 🚀 性能優化和分析
- 🔒 安全性審查
- 📝 代碼審查和重構指導
- ✅ 測試策略和品質保證
- 🔧 並發和異步模式

## 安裝位置

配置檔案已存放於：
```
~/.claude/agents/
├── rust-senior-engineer.yaml          # 基本配置
├── rust-senior-engineer-advanced.yaml # 進階配置
├── README.md                           # 代理說明
└── USAGE_GUIDE.md                     # 本檔案
```

## 如何使用

### 方法 1: 直接通過 Task 工具調用

```python
from anthropic import Task

agent = Task(
    subagent_type="rust-senior-engineer",
    description="Review Rust code for safety and performance",
    prompt="分析 src/main.rs 中的並發邏輯，並提供優化建議"
)
```

### 方法 2: 在 Claude Code 中使用

```bash
# 使用 rust-architect 別名
@agent-rust-architect 分析這個 Tokio 應用的架構

# 或在任務中使用
Task(subagent_type="rust-senior-engineer", prompt="...")
```

### 方法 3: 自動觸發

Claude Code 會在以下情況自動使用此代理：
- 分析複雜的 Rust 系統架構
- 進行深度代碼審查
- 優化性能關鍵代碼
- 評估安全性和並發性

## 使用場景

### 場景 1: 代碼審查

```
請求：請針對我的 Tokio 應用進行全面的代碼審查，重點關注：
1. 內存安全性
2. 線程安全性
3. 錯誤處理
4. 性能瓶頸
```

**代理會提供：**
- 結構化的安全分析
- 架構改進建議
- 性能優化機會
- 具體的代碼示例

### 場景 2: 架構設計

```
請求：我正在設計一個高性能的分散式系統。
請幫我評估以下架構選擇：
1. Tokio vs rayon
2. 同步 vs 異步 I/O
3. 事件驅動 vs 線程池
```

**代理會提供：**
- 比較分析和權衡
- 推薦的架構模式
- 實現示例
- 潛在的陷阱和注意事項

### 場景 3: 性能優化

```
請求：我的應用處理 100k req/s，但有記憶體洩漏。
請幫我找出瓶頸並提供優化建議。
```

**代理會提供：**
- 性能分析框架
- 分析工具建議
- 優化技術
- 基準測試策略

### 場景 4: 安全審查

```
請求：審查我的 unsafe 代碼塊，並建議如何消除不安全性。
```

**代理會提供：**
- unsafe 代碼審查
- 替代方案分析
- 安全性驗證
- 最佳實踐建議

## 代理的優勢

| 特性 | 說明 |
|------|------|
| **深度分析** | 使用 Claude Sonnet 4.5 進行複雜分析 |
| **最佳實踐** | 遵循最新的 Rust 社群標準 |
| **實戰經驗** | 基於 10+ 年的系統程式設計經驗 |
| **結構化輸出** | 清晰的分析框架和建議 |
| **教育性** | 解釋設計權衡和推理 |
| **工具整合** | 整合 cargo、clippy、criterion 等 |

## 代理檢查清單

代理在審查代碼時會檢查：

### 🔒 內存安全
- ✓ 未初始化的記憶體
- ✓ 生命週期註解正確性
- ✓ unsafe 代碼的安全使用
- ✓ Use-after-free 檢查
- ✓ 緩衝區溢出防護

### 🧵 線程安全
- ✓ Send 和 Sync 邊界
- ✓ 同步機制正確性
- ✓ 數據競爭檢測
- ✓ 死鎖分析
- ✓ Arc/Mutex 使用

### ⚠️ 錯誤處理
- ✓ 完整的錯誤情況
- ✓ Result/Option 使用
- ✓ 錯誤上下文保留
- ✓ 優雅降級策略
- ✓ 恢復路徑

### 🚀 性能
- ✓ 時間複雜度
- ✓ 空間複雜度
- ✓ 不必要的分配
- ✓ 抽象合理性
- ✓ 編譯優化

### 📚 可維護性
- ✓ 變數命名清晰度
- ✓ 模組組織邏輯
- ✓ API 文檔完整性
- ✓ 類型安全最大化
- ✓ SOLID 原則應用

## 配置選項

### 基本配置 (rust-senior-engineer.yaml)

適合：
- 標準 Rust 項目審查
- 一般性建築設計
- 日常指導和建議

```yaml
model: claude-sonnet-4-5-20250929
instructions: [詳細的 Rust 專家指令]
tools: [read, write, edit, bash, grep, glob]
```

### 進階配置 (rust-senior-engineer-advanced.yaml)

適合：
- 大規模系統設計
- 深度性能分析
- 複雜的並發系統
- 安全關鍵應用

包含額外的：
- 專業領域知識
- 詳細的審查清單
- 最佳實踐指南
- 版本管理

## 與 Phase 1 的整合

這個代理可以與您的改進計畫整合：

```rust
// 使用代理審查 PR-1 的改動
@agent-rust-architect
請審查 PR-1 的 daemon 生命週期管理改動。
重點檢查：
1. Arc<Mutex<>> 的使用是否安全
2. shutdown channel 的設計是否合理
3. 是否有任何並發問題或死鎖風險

並提供性能和安全性的全面評估。
```

## 常見命令

### 審查代碼變更
```
@agent-rust-architect
請審查檔案 crates/mdns/src/service.rs 的最新改動
```

### 設計建議
```
@agent-rust-architect
我需要為服務發現系統設計一個配置管理器。
請建議最佳架構模式和實現方式。
```

### 性能分析
```
@agent-rust-architect
分析這個異步函式的性能特性，
並建議優化機會。
```

### 安全審查
```
@agent-rust-architect
審查 unsafe 塊並建議如何消除不安全性。
```

## 最佳實踐

1. **詳細描述任務**
   - 提供充分的上下文
   - 說明目標和約束
   - 指定關注領域

2. **提供相關代碼**
   - 完整的代碼片段
   - 上下文路徑
   - 相關檔案

3. **明確期望**
   - 期望的輸出格式
   - 優先級和限制
   - 時間框架（如果有）

4. **反覆迭代**
   - 提出後續問題
   - 測試建議
   - 獲取更多細節

## 故障排除

### 代理回應不充分

**解決方案**：
- 提供更多上下文
- 更具體地描述問題
- 附加相關代碼片段

### 建議不適用

**解決方案**：
- 解釋您的約束條件
- 說明為什麼建議不適用
- 要求替代方案

### 需要不同的視角

**解決方案**：
- 切換到進階配置
- 指定不同的分析角度
- 請求替代架構

## 文件和資源

- 📖 [Rust 官方手冊](https://doc.rust-lang.org/)
- 📚 [Rust API 指南](https://rust-lang.github.io/api-guidelines/)
- 🔍 [Clippy 檢查](https://doc.rust-lang.org/clippy/)
- ⚙️ [Cargo 文檔](https://doc.rust-lang.org/cargo/)

## 支援和回饋

如果您遇到任何問題或有改進建議，請：

1. 檢查配置檔案的語法
2. 驗證檔案位置 (~/.claude/agents/)
3. 確保有適當的存取權限
4. 提供詳細的錯誤信息

## 更新歷史

| 版本 | 日期 | 描述 |
|------|------|------|
| 1.0 | 2025-11-12 | 初始版本，包括基本和進階配置 |

---

**最後更新**: 2025-11-12
**狀態**: ✅ 已準備好使用
