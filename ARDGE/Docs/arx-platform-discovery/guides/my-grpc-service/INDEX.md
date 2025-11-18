# gRPC 伺服器完整項目文檔索引

## 📚 文檔導覽

### 快速開始（新手必讀）

1. **[README.md](./README.md)** ⭐ 開始這裡
   - 專案概述和特點
   - 快速開始指南
   - 常見操作
   - FAQ 和故障排查

2. **[SETUP_GUIDE.md](./SETUP_GUIDE.md)** - 詳細設置指南
   - 完整的逐步設置
   - 每個步驟的詳細解釋
   - 代碼示例和最佳實踐
   - 與 arx-platform-core 的對應

3. **[QUICK_REFERENCE.md](./QUICK_REFERENCE.md)** - 快速查閱
   - 一頁紙速查表
   - 常見代碼片段
   - Proto 語法參考
   - gRPC 錯誤碼對照

### 進階功能（深度學習）

4. **[ADVANCED_FEATURES.md](./ADVANCED_FEATURES.md)** - 進階功能實現
   - Request ID 追蹤
   - 性能監控
   - 配置管理
   - 健康檢查
   - 連接池
   - cmux 多路復用
   - 單元測試

### 代碼模板（複製即用）

5. **[FILE_TEMPLATES.md](./FILE_TEMPLATES.md)** - 完整檔案範本
   - 所有必需檔案的完整代碼
   - 目錄結構建立命令
   - 快速部署步驟
   - 驗證安裝檢查清單

---

## 🎯 學習路徑

### 初學者路徑（Week 1）

```
Day 1-2: 閱讀 README.md
         └─ 了解專案整體結構和特點

Day 3-4: 按照 SETUP_GUIDE.md 第 1-8 步
         └─ 建立基本的 gRPC 伺服器

Day 5:   運行伺服器和客戶端
         └─ 使用 Makefile 命令

Day 6-7: 實驗修改
         └─ 修改 Proto、服務、攔截器
         └─ 觀察日誌輸出
```

### 進階開發者路徑（Week 2-3）

```
Week 2:  學習 ADVANCED_FEATURES.md
         └─ Request ID 追蹤
         └─ 性能監控
         └─ 配置管理

Week 3:  實現進階功能
         └─ 添加連接池
         └─ 設置 cmux
         └─ 編寫單元測試
```

### 生產準備路徑（Week 4+）

```
Week 4:  優化和部署準備
         └─ 性能調優
         └─ 錯誤處理完善
         └─ 日誌和監控配置

Week 5+: 部署和維護
         └─ Docker 容器化
         └─ Kubernetes 配置
         └─ CI/CD 管線
```

---

## 📂 檔案組織

```
my-grpc-service/
├── 📖 文檔（你在這裡）
│   ├── README.md              - 項目概述
│   ├── INDEX.md              - 本檔案
│   ├── SETUP_GUIDE.md        - 詳細設置
│   ├── ADVANCED_FEATURES.md  - 進階功能
│   ├── QUICK_REFERENCE.md    - 速查表
│   └── FILE_TEMPLATES.md     - 代碼範本
│
├── 📋 配置檔案
│   ├── go.mod                - Go 模組
│   ├── Makefile              - 開發命令
│   └── configs/
│       └── config.toml       - 伺服器配置
│
├── 🔧 代碼
│   ├── api/                  - Proto 和生成的代碼
│   ├── cmd/                  - 應用入口點
│   │   ├── server/main.go   - 伺服器
│   │   └── client/main.go   - 客戶端
│   └── internal/             - 內部實現
│       ├── logger/           - 日誌
│       ├── server/           - 伺服器設置
│       ├── services/         - 業務邏輯
│       └── client/           - 客戶端工具
│
└── 📦 輸出
    └── bin/                  - 編譯的二進制
```

---

## 🔍 按需求查找

### 「我想...」 查找表

#### 基礎功能

| 需求 | 文檔位置 |
|------|---------|
| 理解 gRPC 基礎 | SETUP_GUIDE.md 第 1-6 步 |
| 設置我的第一個伺服器 | SETUP_GUIDE.md 全部 |
| 添加新服務 | README.md / QUICK_REFERENCE.md |
| 定義 Proto 檔案 | QUICK_REFERENCE.md / SETUP_GUIDE.md 步驟 2 |
| 實現服務方法 | SETUP_GUIDE.md 步驟 6 / QUICK_REFERENCE.md |
| 運行測試 | README.md / SETUP_GUIDE.md 步驟 10 |

#### 進階功能

| 需求 | 文檔位置 |
|------|---------|
| 添加 Request ID 追蹤 | ADVANCED_FEATURES.md 第 1 節 |
| 監控性能和慢查詢 | ADVANCED_FEATURES.md 第 2 節 |
| 配置管理系統 | ADVANCED_FEATURES.md 第 3 節 |
| 健康檢查 | ADVANCED_FEATURES.md 第 4 節 |
| 連接池實現 | ADVANCED_FEATURES.md 第 5 節 |
| 多路復用（gRPC + HTTP） | ADVANCED_FEATURES.md 第 6 節 |
| 編寫單元測試 | ADVANCED_FEATURES.md 第 7 節 |

#### 快速查閱

| 需求 | 文檔位置 |
|------|---------|
| 常用命令速查 | QUICK_REFERENCE.md 開頭 |
| Proto 語法 | QUICK_REFERENCE.md / 服務實現框架 |
| gRPC 錯誤處理 | QUICK_REFERENCE.md / 常見錯誤處理 |
| 錯誤碼對照表 | QUICK_REFERENCE.md / gRPC 錯誤碼對照 |
| Context 使用 | QUICK_REFERENCE.md / Context 用法 |
| 性能優化建議 | QUICK_REFERENCE.md / 性能優化建議 |

#### 代碼範本

| 需求 | 文檔位置 |
|------|---------|
| 完整的檔案代碼 | FILE_TEMPLATES.md |
| 目錄結構建立 | FILE_TEMPLATES.md / 目錄建立命令 |
| 快速部署步驟 | FILE_TEMPLATES.md / 快速部署步驟 |

---

## 💡 常見問題速查

### 「我看不懂 Proto 定義」
→ 查看：QUICK_REFERENCE.md / Proto 基本語法

### 「我不知道如何添加新服務」
→ 查看：README.md / 添加新服務

### 「伺服器啟動失敗」
→ 查看：README.md / 故障排查

### 「我想了解攔截器如何工作」
→ 查看：QUICK_REFERENCE.md / 攔截器實現框架

### 「我需要快速複製一個完整項目」
→ 查看：FILE_TEMPLATES.md / 快速部署步驟

### 「我想優化性能」
→ 查看：QUICK_REFERENCE.md / 性能優化建議 或 ADVANCED_FEATURES.md / 第 2 節

### 「我不知道如何編寫測試」
→ 查看：QUICK_REFERENCE.md / 單元測試框架 或 ADVANCED_FEATURES.md / 第 7 節

---

## 📊 文檔特點對應表

| 特點 | README | SETUP | ADVANCED | QUICK | TEMPLATE |
|------|--------|-------|----------|-------|----------|
| 詳細解釋 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐ | ⭐⭐ |
| 代碼示例 | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 快速查閱 | ⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 進階內容 | ⭐⭐ | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐ |
| 最佳實踐 | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |

---

## 🎓 學習難度等級

### 初級（第 1-2 天）
- ✅ 了解 gRPC 基本概念
- ✅ 建立 Hello World 伺服器
- ✅ 執行簡單的 RPC 呼叫

**推薦文檔**：README.md + SETUP_GUIDE.md 前 6 步 + QUICK_REFERENCE.md

### 中級（第 3-7 天）
- ✅ 實現完整的服務邏輯
- ✅ 添加多個 RPC 方法
- ✅ 實現攔截器
- ✅ 寫單元測試

**推薦文檔**：SETUP_GUIDE.md 全部 + ADVANCED_FEATURES.md 第 1-2 節

### 高級（第 2-4 週）
- ✅ 實現進階功能（Request ID、性能監控、連接池）
- ✅ 配置管理系統
- ✅ 健康檢查和多路復用
- ✅ CI/CD 整合

**推薦文檔**：ADVANCED_FEATURES.md 全部 + 外部資源

### 專家（第 1 個月+）
- ✅ 優化性能
- ✅ 完善錯誤處理
- ✅ 部署到生產環境
- ✅ 監控和日誌聚合

**推薦文檔**：所有文檔 + 官方文檔 + 社區資源

---

## 🔗 外部資源

### 官方文檔
- [gRPC 官方文檔](https://grpc.io/docs/languages/go/)
- [Protocol Buffers 官方文檔](https://developers.google.com/protocol-buffers)
- [Go 官方文檔](https://golang.org/doc/)

### 參考實現
- [arx-platform-core](https://github.com/ardge-labs/arx-platform-core)
- [gRPC Go 示例](https://github.com/grpc/grpc-go/tree/master/examples)

### 最佳實踐指南
- [Effective Go](https://go.dev/doc/effective_go)
- [Uber Go Style Guide](https://github.com/uber-go/guide)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)

### 工具和框架
- [Buf](https://buf.build/) - Proto 管理工具
- [grpcurl](https://github.com/fullstorydev/grpcurl) - gRPC 命令行工具
- [Evans](https://github.com/ktr0731/evans) - 互動式 gRPC 客戶端

---

## 📝 使用建議

### 首次使用

1. 📖 閱讀 README.md（5 分鐘）
2. 🚀 按照 SETUP_GUIDE.md 建立項目（30 分鐘）
3. ▶️ 運行伺服器和客戶端（10 分鐘）
4. 🔧 修改代碼和 Proto 進行實驗（30 分鐘）

### 日常開發

- 🔍 使用 QUICK_REFERENCE.md 快速查找語法
- 📋 參考 FILE_TEMPLATES.md 複製代碼框架
- 📚 遇到複雜問題時查看 SETUP_GUIDE.md 或 ADVANCED_FEATURES.md

### 深度學習

- 📖 完整閱讀 ADVANCED_FEATURES.md
- 🔬 實現每個進階功能
- 🧪 編寫測試來驗證理解

---

## ✅ 檢查清單

### 項目設置完整性檢查

```
□ 閱讀了 README.md
□ 了解了項目結構
□ 安裝了所有前置需求
□ 建立了目錄結構
□ 複製了所有檔案
□ 生成了 gRPC 代碼
□ 伺服器成功啟動
□ 客戶端成功連接
□ 運行了所有測試
□ 代碼通過 linter 檢查
```

### 學習目標檢查

```
□ 理解 Proto 定義語法
□ 能夠實現 gRPC 服務
□ 能夠添加自定義攔截器
□ 能夠編寫單元測試
□ 了解錯誤處理方式
□ 能夠配置伺服器
□ 了解優雅啟動/關閉
□ 能夠實現 Request ID 追蹤
□ 能夠監控性能
□ 了解連接池概念
```

---

## 🚀 快速導航

### 我是完全新手
→ 從 README.md 開始 → SETUP_GUIDE.md 前 6 步

### 我想快速部署
→ FILE_TEMPLATES.md / 快速部署步驟

### 我要添加功能
→ QUICK_REFERENCE.md / 相關框架 或 SETUP_GUIDE.md / 添加新服務

### 我遇到了問題
→ README.md / 故障排查

### 我想優化性能
→ ADVANCED_FEATURES.md / 第 2 節 或 QUICK_REFERENCE.md / 性能優化建議

---

## 📞 支持和反饋

### 文檔未涵蓋的內容

建議查看：
1. [arx-platform-core](https://github.com/ardge-labs/arx-platform-core) - 生產級示例
2. [gRPC 官方文檔](https://grpc.io/docs/languages/go/)
3. [Go wiki](https://go.dev/wiki)

### 報告問題

如發現文檔錯誤或不清楚之處，請提交 Issue 或 PR。

---

## 📈 版本歷史

| 版本 | 日期 | 變更 |
|------|------|------|
| 1.0 | 2024-01-15 | 初始版本，包含基礎和進階功能 |

---

## 📄 許可證

MIT License - 可自由使用和修改

---

## 🙏 致謝

本項目基於 [arx-platform-core](https://github.com/ardge-labs/arx-platform-core) 的架構和最佳實踐。

---

**上次更新**：2024 年 1 月 15 日

**推薦使用順序**：README.md → SETUP_GUIDE.md → QUICK_REFERENCE.md → ADVANCED_FEATURES.md → FILE_TEMPLATES.md
