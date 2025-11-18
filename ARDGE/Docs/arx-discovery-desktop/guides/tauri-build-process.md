# Tauri 構建程序詳解

本指南詳細說明當執行 `tauri build` 命令時，整個系統實際上執行的步驟和流程。

## 目錄

- [構建流程概觀](#構建流程概觀)
- [詳細步驟說明](#詳細步驟說明)
- [配置文件解析](#配置文件解析)
- [輸出產物](#輸出產物)
- [構建時間優化](#構建時間優化)
- [常見問題](#常見問題)
- [進階配置](#進階配置)

## 構建流程概觀

### 高階流程圖

```
tauri build
    ↓
[1] 前端構建 (Frontend Build)
    ├─ npm run build
    │   ├─ tsc (TypeScript 型別檢查)
    │   └─ vite build (打包前端資源)
    └─ 輸出到 dist/ 目錄
    ↓
[2] Rust 後端編譯 (Rust Compilation)
    ├─ cargo build --release
    │   ├─ 編譯 src-tauri/src/lib.rs
    │   ├─ 編譯 mDNS 和其他依賴項
    │   └─ 連結靜態庫
    └─ 輸出 target/release/ 執行檔
    ↓
[3] 資源整理 (Asset Bundling)
    ├─ 複製 dist/ 到應用程式包
    ├─ 複製圖標檔案
    ├─ 複製配置檔案
    └─ 準備應用程式外殼
    ↓
[4] 應用程式打包 (App Bundling)
    ├─ macOS: .app 束 + .dmg
    ├─ Windows: .exe + .msi
    └─ Linux: AppImage + .deb
    ↓
✅ 完成 - 可安裝應用程式已準備就緒
```

## 詳細步驟說明

### 步驟 1：前端構建

#### 1.1 前置設定

Tauri 根據 `tauri.conf.json` 中的配置執行構建：

```json
{
  "build": {
    "beforeBuildCommand": "npm run build",
    "frontendDist": "../dist"
  }
}
```

#### 1.2 執行構建命令

```bash
npm run build
```

根據 `package.json`，實際執行：

```json
{
  "scripts": {
    "build": "tsc && vite build"
  }
}
```

#### 1.3 TypeScript 型別檢查

```bash
tsc
```

**執行內容：**
- 根據 `tsconfig.json` 的設定
- 檢查 `src/` 目錄下所有 TypeScript 檔案的型別
- 不生成 JavaScript 檔案（`noEmit: true`）
- 驗證型別一致性

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src"]
}
```

**如果型別檢查失敗，整個構建會停止。**

#### 1.4 Vite 打包

```bash
vite build
```

**執行內容：**
- 編譯 TypeScript 為 JavaScript
- 打包模組和資源
- 壓縮程式碼和靜態資源
- 生成 source maps（如啟用）
- 優化輸出大小

**Vite 根據 `vite.config.ts` 設定：**

```typescript
export default defineConfig(async () => ({
  // Prevent Vite from obscuring rust errors
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true
  }
}));
```

**輸出產物：**

```
dist/
├── index.html                 (主 HTML 檔案)
├── assets/
│   ├── index-XXXX.js         (主應用程式 JavaScript)
│   ├── index-XXXX.css        (樣式表)
│   └── ...其他資源檔案
└── vite.svg                   (靜態資源)
```

### 步驟 2：Rust 後端編譯

#### 2.1 編譯命令

```bash
cd src-tauri
cargo build --release
```

#### 2.2 依賴項編譯

根據 `Cargo.toml`，編譯所有依賴項：

```toml
[dependencies]
tauri = { version = "2.9", features = [] }
tauri-plugin-opener = "2.5"
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
mdns-sd = "0.17"                # mDNS 服務發現
flume = "0.11"                  # 跨執行緒通道
tokio = { version = "1.48", features = ["full"] }  # 非同步執行時
thiserror = "2.0"               # 錯誤型別
tracing = "0.1"                 # 日誌記錄
tracing-subscriber = "0.3"      # 日誌訂閱
```

**編譯過程：**
- 解析依賴項樹
- 下載缺失的 crates（如需要）
- 編譯每個依賴項
- 編譯應用程式程式碼

#### 2.3 應用程式編譯

編譯 `src-tauri/src/lib.rs` 中的程式碼：

```rust
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![greet])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

**包括：**
- Tauri 應用程式初始化
- 外掛程式配置
- 命令處理程序
- 日誌系統設定

#### 2.4 連結和優化

- 連結所有物件檔案
- 應用最佳化 (`--release` 模式)
- 生成最終執行檔

**輸出產物：**

```
src-tauri/target/release/
├── arx_discovery_lib          (主應用程式二進制)
├── .deps/                      (依賴項構成物)
├── incremental/                (增量編譯資訊)
└── deps/                       (編譯的依賴項)
```

### 步驟 3：資源整理和外殼準備

#### 3.1 資源複製

Tauri 將以下資源複製到應用程式包中：

**前端資源：**
```
dist/ → Application Bundle/Resources/
├── index.html
├── assets/
│   └── *.js, *.css, ...
```

**圖標檔案：**

根據 `tauri.conf.json`：

```json
{
  "bundle": {
    "icon": [
      "icons/32x32.png",
      "icons/128x128.png",
      "icons/128x128@2x.png",
      "icons/icon.icns",
      "icons/icon.ico"
    ]
  }
}
```

複製到：
```
→ macOS: .app bundle
→ Windows: .exe 資源
→ Linux: .svg 檔案
```

#### 3.2 應用程式配置

使用 `tauri.conf.json` 中的設定：

```json
{
  "productName": "arx-discovery",
  "version": "0.1.0",
  "identifier": "com.ardge.arx-discovery",
  "app": {
    "windows": [
      {
        "title": "arx-discovery",
        "width": 800,
        "height": 600
      }
    ]
  }
}
```

**產生配置檔案：**
- macOS: Info.plist
- Windows: .exe 資源和登錄資訊
- Linux: .desktop 檔案

### 步驟 4：應用程式打包

#### 4.1 平台特定的打包

根據作業系統生成不同格式的應用程式。

**macOS 打包流程：**

```
src-tauri/target/release/
  ├── arx_discovery_lib
  └── bundle/
      └── macos/
          └── arx-discovery.app/
              ├── Contents/
              │   ├── MacOS/
              │   │   └── arx-discovery          (執行檔)
              │   ├── Resources/
              │   │   ├── index.html
              │   │   ├── assets/
              │   │   └── icons/
              │   ├── Frameworks/
              │   │   └── WebKit.framework        (系統框架)
              │   └── Info.plist
              └── arx-discovery_0.1.0_aarch64.dmg (可分配的磁碟映像)
```

**Windows 打包流程：**

```
src-tauri/target/release/
  ├── arx_discovery_lib.exe
  └── bundle/
      ├── nsis/
      │   ├── arx-discovery_0.1.0_x64-setup.exe  (NSIS 安裝程式)
      │   └── nsis_scripts/
      └── msi/
          └── arx-discovery_0.1.0_x64.msi        (MSI 安裝程式)
```

**Linux 打包流程：**

```
src-tauri/target/release/
  └── bundle/
      ├── appimage/
      │   └── arx-discovery_0.1.0_amd64.AppImage (AppImage 包)
      ├── deb/
      │   └── arx-discovery_0.1.0_amd64.deb      (Debian 包)
      └── rpm/
          └── arx-discovery-0.1.0-1.x86_64.rpm   (RPM 包)
```

## 配置文件解析

### tauri.conf.json 中的構建相關設定

```json
{
  "productName": "arx-discovery",           // 應用程式名稱
  "version": "0.1.0",                       // 版本號
  "identifier": "com.ardge.arx-discovery",  // 唯一識別符

  "build": {
    // 生產構建前執行的命令
    "beforeBuildCommand": "npm run build",

    // 開發模式使用的前端 URL
    "devUrl": "http://localhost:1420",

    // 前端構建輸出目錄
    "frontendDist": "../dist"
  },

  "app": {
    // 應用程式視窗配置
    "windows": [
      {
        "title": "arx-discovery",
        "width": 800,
        "height": 600
      }
    ],

    // 安全性設定
    "security": {
      "csp": null
    }
  },

  "bundle": {
    "active": true,                    // 啟用應用程式打包
    "targets": "all",                  // 為所有平台打包
    "icon": [
      "icons/32x32.png",
      "icons/128x128.png",
      "icons/128x128@2x.png",
      "icons/icon.icns",
      "icons/icon.ico"
    ]
  }
}
```

### 自訂構建命令

可以修改 `beforeBuildCommand` 以執行自訂邏輯：

```json
{
  "build": {
    "beforeBuildCommand": "npm run build && npm run lint",
    "beforeBuildCommand": "npm install && npm run build"
  }
}
```

## 輸出產物

### 完整的輸出結構

**構建完成後的目錄結構：**

```
arx-discovery-desktop/
├── dist/                                    (✨ 前端輸出)
│   ├── index.html
│   └── assets/
│       ├── index-XXXX.js
│       └── index-XXXX.css
│
└── src-tauri/target/release/
    ├── arx_discovery_lib                    (✨ 應用程式二進制)
    │
    ├── bundle/
    │   ├── dmg/
    │   │   ├── arx-discovery_0.1.0_x64.dmg
    │   │   └── arx-discovery_0.1.0_aarch64.dmg
    │   │
    │   ├── macos/
    │   │   └── arx-discovery.app/
    │   │       └── Contents/
    │   │           ├── MacOS/arx-discovery
    │   │           ├── Resources/
    │   │           └── Info.plist
    │   │
    │   ├── msi/
    │   │   └── arx-discovery_0.1.0_x64.msi
    │   │
    │   ├── nsis/
    │   │   └── arx-discovery_0.1.0_x64-setup.exe
    │   │
    │   ├── appimage/
    │   │   └── arx-discovery_0.1.0_amd64.AppImage
    │   │
    │   ├── deb/
    │   │   └── arx-discovery_0.1.0_amd64.deb
    │   │
    │   └── rpm/
    │       └── arx-discovery-0.1.0-1.x86_64.rpm
    │
    └── deps/                                (編譯的依賴項和暫時物件)
```

### 各平台產物詳解

| 平台 | 格式 | 說明 | 位置 |
|------|------|------|------|
| **macOS** | `.dmg` | macOS 磁碟映像 (推薦安裝方式) | `bundle/dmg/` |
| **macOS** | `.app` | macOS 應用程式束 (直接執行) | `bundle/macos/` |
| **Windows** | `.msi` | Windows 安裝程式 (推薦) | `bundle/msi/` |
| **Windows** | `.exe` | NSIS 自動安裝程式 | `bundle/nsis/` |
| **Linux** | `.AppImage` | 通用 Linux 應用程式 | `bundle/appimage/` |
| **Linux** | `.deb` | Debian 套件 | `bundle/deb/` |
| **Linux** | `.rpm` | RedHat 套件 | `bundle/rpm/` |

## 構建時間優化

### 1. 增量構建

第一次完整構建通常需要 2-5 分鐘，後續構建可更快。

```bash
# 完整清潔構建 (最慢)
rm -rf src-tauri/target/release dist/
npm run tauri build

# 增量構建 (快速，如果只改動了 src/ 檔案)
npm run tauri build

# 僅編譯 Rust，跳過前端構建 (如果知道前端未更改)
cargo build --release -C src-tauri
```

### 2. 並行構建

利用多核心加速：

```bash
# 增加並行編譯工作數
CARGO_BUILD_JOBS=8 npm run tauri build

# 或在 .cargo/config.toml 中設定
[build]
jobs = 8
```

### 3. 優化依賴項

檢查是否有不必要的依賴項：

```bash
# 查看依賴項樹
cargo tree -C src-tauri

# 查看未使用的依賴項
cargo tree -C src-tauri --unused-packages
```

### 4. 使用快取

```bash
# 利用 sccache 加速編譯
# 1. 安裝 sccache
cargo install sccache

# 2. 設定環境變數
export RUSTC_WRAPPER=sccache

# 3. 構建
npm run tauri build
```

### 5. 預建開發環境

在 CI/CD 中預先準備：

```bash
# 預先編譯依賴項
cargo build --release -C src-tauri --offline
```

## 常見問題

### Q1: 構建失敗 - 「TypeScript 型別檢查錯誤」

**症狀：**
```
error: Type 'X' is not assignable to type 'Y'
```

**解決方案：**
1. 修正 `src/` 中的型別錯誤
2. 執行 `npm run build` 進行檢查
3. 重新執行 `npm run tauri build`

### Q2: 構建失敗 - 「Rust 編譯錯誤」

**症狀：**
```
error[E0308]: mismatched types
```

**解決方案：**
1. 檢查 `src-tauri/src/lib.rs` 中的 Rust 程式碼
2. 執行 `cargo check -C src-tauri` 檢查錯誤
3. 修正後重新構建

### Q3: 應用程式產物檔案太大

**原因：**
- 包含了除錯符號
- 未啟用優化

**解決方案：**
```toml
# Cargo.toml 中設定
[profile.release]
opt-level = 3          # 最大最佳化
lto = true             # 連結時間最佳化
codegen-units = 1      # 更好的最佳化但更慢
strip = true           # 移除除錯符號
```

### Q4: 構建太慢

**原因：**
- 首次構建編譯所有依賴項
- 磁碟 I/O 緩慢
- 系統資源不足

**解決方案：**
```bash
# 1. 使用 SSD
# 2. 關閉其他應用程式
# 3. 增加並行工作數
CARGO_BUILD_JOBS=$(nproc) npm run tauri build
```

### Q5: 「前端資源未找到」

**症狀：**
應用程式執行時白畫面或找不到資源

**原因：**
`frontendDist` 設定不正確或 `npm run build` 未執行

**解決方案：**
1. 驗證 `tauri.conf.json` 中的 `frontendDist` 路徑
2. 驗證 `dist/` 目錄存在且包含檔案
3. 手動執行 `npm run build` 測試

```bash
npm run build
ls dist/
npm run tauri build
```

## 進階配置

### 自訂應用程式圖標

在 `tauri.conf.json` 中指定圖標：

```json
{
  "bundle": {
    "icon": [
      "icons/32x32.png",
      "icons/128x128.png",
      "icons/128x128@2x.png",
      "icons/icon.icns",
      "icons/icon.ico"
    ]
  }
}
```

**所需的圖標格式：**
- `icon.icns` - macOS (512x512)
- `icon.ico` - Windows (256x256)
- `.png` 檔案 - 各種大小

### 修改應用程式版本

在 `tauri.conf.json` 和 `Cargo.toml` 中更新：

```json
// tauri.conf.json
{
  "version": "0.2.0"
}
```

```toml
# Cargo.toml
[package]
version = "0.2.0"
```

### 條件式構建

根據環境使用不同的組態：

```bash
# 開發構建 (未優化)
npm run tauri build

# 生產構建 (完全優化)
NODE_ENV=production npm run tauri build
```

在 `tauri.conf.json` 中使用環境變數：

```json
{
  "build": {
    "beforeBuildCommand": "npm run build:prod"
  }
}
```

### 跳過某些平台

在 `tauri.conf.json` 中配置：

```json
{
  "bundle": {
    "targets": ["deb", "dmg", "msi"]  // 只打包這些格式
  }
}
```

## 構建流程環境變數

### 常用的環境變數

```bash
# Rust 編譯選項
RUSTFLAGS="-C opt-level=3"               # 最佳化級別
CARGO_BUILD_JOBS=4                        # 並行工作數
RUSTC_WRAPPER=sccache                     # 使用 sccache 加速

# Tauri 特定變數
TAURI_BUILD_TARGET=aarch64-apple-darwin  # 特定目標平台
TAURI_DEV_HOST=0.0.0.0                   # 開發伺服器主機

# 前端構建
NODE_ENV=production                       # 生產環境變數
```

### 完整構建命令範例

```bash
# 最佳化構建 (生產環境)
NODE_ENV=production \
RUSTFLAGS="-C opt-level=3" \
CARGO_BUILD_JOBS=$(nproc) \
npm run tauri build

# 快速增量構建
npm run tauri build

# 特定平台構建
TAURI_BUILD_TARGET=x86_64-apple-darwin npm run tauri build
```

---

## 相關資源

- [Tauri 官方文檔 - 構建](https://tauri.app/start/setup/)
- [Cargo 構建文檔](https://doc.rust-lang.org/cargo/commands/cargo-build.html)
- [Vite 構建指南](https://vitejs.dev/guide/build.html)
- [TypeScript 編譯選項](https://www.typescriptlang.org/tsconfig)
