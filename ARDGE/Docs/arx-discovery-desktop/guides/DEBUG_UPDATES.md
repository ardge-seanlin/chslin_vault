# Debug 更新對話框指南

## 🎯 目標
確認應用程式檢查到更新時能正確顯示對話框。

---

## 📋 檢查清單

### 1️⃣ **檢查 tauri.conf.json 配置**

```bash
cat src-tauri/tauri.conf.json | grep -A 10 "updater"
```

應該看到：
```json
"updater": {
  "active": true,
  "dialog": true,
  "pubkey": "dW50...",
  "endpoints": ["https://192.168.5.166:80/updates/{{target}}-{{arch}}/latest.json"]
}
```

**檢查項目：**
- ✅ `"active": true`
- ✅ `"endpoints"` 正確配置（包含 {{target}} 和 {{arch}}）
- ✅ `"pubkey"` 不為空

### 2️⃣ **檢查版本設定**

```bash
grep '"version"' src-tauri/tauri.conf.json
```

應該看到：
```json
"version": "0.1.0"
```

### 3️⃣ **檢查前端程式碼**

確認 `src/main.ts` 有以下程式碼：

```typescript
// 應該有 showUpdateDialog() 函數
// 應該在 checkForUpdates() 中呼叫 showUpdateDialog(update)
// 應該有 await update.downloadAndInstall()
```

---

## 🚀 測試更新流程（本地測試）

### 步驟 1: 啟動本地 HTTP 伺服器

```bash
mkdir -p ./updates/darwin-aarch64
cd ./updates/darwin-aarch64

# 建立測試的 latest.json
cat > latest.json <<'EOF'
{
  "version": "0.1.1",
  "notes": "Test update",
  "pub_date": "2024-11-21T14:00:00Z",
  "platforms": {
    "darwin-aarch64": {
      "signature": "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6IEREQzc1OEZCOUZGMzgwQzAKUldUQWdQT2YrMWpIM1JxVTExRDZWbjlIWm9DOEZ2TzB5Mkh6VnVHejU4M1pqMEFYeTByVXBMV3gK",
      "url": "http://localhost:8000/updates/darwin-aarch64/arx-finder_0.1.1_darwin-aarch64.tar.gz"
    }
  }
}
EOF

# 啟動 HTTP 伺服器
python3 -m http.server 8000
```

### 步驟 2: 修改 tauri.conf.json 指向本地伺服器

```bash
# 編輯 src-tauri/tauri.conf.json
# 將 endpoints 改為:
"endpoints": ["http://localhost:8000/updates/{{target}}-{{arch}}/latest.json"]
```

### 步驟 3: 開發模式執行應用

```bash
npm run tauri dev
```

### 步驟 4: 打開瀏覽器開發者工具

按 `Cmd+Option+I` (macOS) 打開開發者工具

**Console 標籤下應該看到：**

```
✅ Update available: 0.1.1
✅ Current version: 0.1.0
✅ Update dialog displayed
```

**Network 標籤下應該看到：**

```
GET http://localhost:8000/updates/darwin-aarch64/latest.json
  Status: 200 OK
  Response: { "version": "0.1.1", ... }
```

### 步驟 5: 確認對話框彈出

應該看到一個確認對話框：
```
┌─────────────────────────────────────┐
│  New version available: 0.1.1       │
│                                     │
│  Would you like to update now?      │
│                                     │
│  [Cancel]    [OK]                   │
└─────────────────────────────────────┘
```

---

## 🐛 常見問題 Debug

### 問題 1: 對話框沒有出現

**可能原因：**

| 原因 | 檢查方式 | 解決方案 |
|------|---------|---------|
| updater 未啟用 | `"active": true`? | 在 tauri.conf.json 設為 true |
| 無法連接伺服器 | Console 中有 error? | 檢查 endpoint URL 和伺服器狀態 |
| 版本相同 | 是否 0.1.0 == 0.1.0? | 在 latest.json 中使用 0.1.1 |
| 沒有呼叫檢查 | checkForUpdates() 被呼叫? | 確認在 DOMContentLoaded 中呼叫 |

**Debug 步驟：**

```bash
# 1. 檢查 Console 日誌
# 打開開發者工具 (Cmd+Option+I)
# 看是否有錯誤訊息

# 2. 檢查網路請求
# Network 標籤 → 搜尋 "latest.json"
# 狀態碼應該是 200

# 3. 檢查 endpoint URL 替換
# Console 執行:
# navigator.userAgent  # 查看使用者代理
# 確認 {{target}} 和 {{arch}} 正確替換
```

### 問題 2: 收到簽名驗證錯誤

```
Error: Failed to verify signature
```

**原因：** 簽名檔案內容不匹配

**解決方案：**

```bash
# 重新生成簽名檔案
export TAURI_SIGNING_PRIVATE_KEY_PATH=~/.tauri/arx-finder.key
export TAURI_SIGNING_PRIVATE_KEY_PASSWORD=your-password

npm run tauri build

# 複製正確的簽名
cp src-tauri/target/release/bundle/macos/*.sig ./updates/darwin-aarch64/

# 更新 latest.json 中的 signature 欄位
cat ./updates/darwin-aarch64/*.sig
```

### 問題 3: 無法下載更新檔案

```
Error: Failed to download update
```

**原因：** URL 錯誤或檔案不存在

**解決方案：**

```bash
# 1. 測試 URL 是否可訪問
curl http://localhost:8000/updates/darwin-aarch64/arx-finder_0.1.1_darwin-aarch64.tar.gz

# 2. 檢查檔案是否存在
ls -lh ./updates/darwin-aarch64/

# 3. 確認 latest.json 中的 URL 正確
cat ./updates/darwin-aarch64/latest.json | grep url
```

---

## 🔍 Console 日誌範例

### ✅ 成功流程

```javascript
// 應用啟動
DOMContentLoaded event fired
Automatically start device discovery on app startup
refreshServices()

// 檢查更新
Check for application updates
checkForUpdates()
Update available: 0.1.1
Current version: 0.1.0
Show update dialog
```

### ❌ 失敗流程

```javascript
// 伺服器無法連接
Update check failed: Failed to fetch latest.json
Error: Network error

// 簽名驗證失敗
Update available: 0.1.1
Failed to install update: Signature verification failed

// 版本相同（無新更新）
Application is up to date
```

---

## 📝 設定環境變數

### macOS/Linux

```bash
# 設定臨時環境變數（僅當前 session）
export RUST_LOG=debug
npm run tauri dev

# 或設定到 .env 檔案
echo "RUST_LOG=debug" >> .env
```

### 啟用 Tauri 日誌

```bash
# 開發模式顯示詳細日誌
RUST_LOG=tauri=debug npm run tauri dev
```

---

## ✅ 驗證清單

在提交前確保：

- [ ] `tauri.conf.json` 中 `"active": true`
- [ ] endpoint 配置正確
- [ ] 前端有 `showUpdateDialog()` 函數
- [ ] `checkForUpdates()` 在應用啟動時被呼叫
- [ ] 本地測試能看到對話框彈出
- [ ] Console 沒有錯誤訊息
- [ ] Network 標籤能看到 latest.json 的 200 請求

---

## 🆘 求助資源

- [Tauri 官方文檔 - Updater](https://v2.tauri.app/plugin/updater/)
- [Tauri Discord 社區](https://discord.com/invite/tauri)
- 查看應用程式日誌：`~/Library/Logs/com.ardge.arx-discovery/`

