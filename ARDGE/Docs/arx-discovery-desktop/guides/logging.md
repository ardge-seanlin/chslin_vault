# Tauri 應用程式日誌配置指南

本指南詳細說明如何在 Tauri 應用程式中配置和使用日誌系統，包括輸出到檔案、控制台和即時監控。

## 目錄

- [基本概念](#基本概念)
- [依賴項配置](#依賴項配置)
- [初始化設定](#初始化設定)
- [日誌輸出目標](#日誌輸出目標)
- [滾動策略](#滾動策略)
- [日誌級別控制](#日誌級別控制)
- [格式化選項](#格式化選項)
- [效能考量](#效能考量)
- [實際應用範例](#實際應用範例)
- [故障排除](#故障排除)
- [最佳實踐](#最佳實踐)

## 基本概念

Tauri 應用程式使用 `tracing` 生態系統提供強大的日誌記錄功能：

- **tracing**：核心日誌記錄框架
- **tracing-subscriber**：日誌訂閱和格式化管理
- **tracing-appender**：檔案輸出和滾動管理

## 依賴項配置

### 在 Cargo.toml 中新增依賴

```toml
[dependencies]
# Logging and tracing
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter", "fmt", "json"] }
tracing-appender = "0.2"

# Optional: for structured logging
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
```

### 功能特性說明

| 特性 | 說明 |
|------|------|
| `env-filter` | 支援環境變數控制日誌級別 |
| `fmt` | 格式化輸出層 |
| `json` | JSON 格式輸出 |

## 初始化設定

### 1. 基礎初始化

在 `src-tauri/src/lib.rs` 中初始化日誌系統：

```rust
use tracing_subscriber::{fmt, prelude::*, EnvFilter};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    // Initialize logging with default info level
    init_logger();

    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![greet])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn init_logger() {
    // Filter level: environment variable or default to info
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    // Basic console output
    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer().with_writer(std::io::stdout))
        .init();
}
```

### 2. 包含檔案輸出的初始化

```rust
use tracing_subscriber::{fmt, prelude::*, EnvFilter};
use tracing_appender::rolling;

fn init_logger() -> Result<(), Box<dyn std::error::Error>> {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    // Create daily rolling log file
    let file_appender = rolling::daily("./logs", "app.log");
    let (non_blocking_file, _guard) = tracing_appender::non_blocking(file_appender);

    // Output to both console and file
    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer().with_writer(std::io::stdout)) // Console
        .with(fmt::layer().with_writer(non_blocking_file)) // File
        .init();

    // Keep the guard alive for the application lifetime
    Box::leak(_guard);

    Ok(())
}
```

### 3. 使用 Tauri 應用資料目錄

```rust
use tauri::Manager;
use tracing_subscriber::{fmt, prelude::*, EnvFilter};
use tracing_appender::rolling;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            // Get app-specific log directory
            let log_dir = app.path().app_log_dir()?;
            std::fs::create_dir_all(&log_dir)?;

            // Initialize logger
            init_logger_with_path(log_dir)?;

            Ok(())
        })
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![greet])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn init_logger_with_path(log_dir: impl AsRef<std::path::Path>) -> Result<(), Box<dyn std::error::Error>> {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    let file_appender = rolling::daily(log_dir, "app.log");
    let (non_blocking_file, _guard) = tracing_appender::non_blocking(file_appender);

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer().with_writer(std::io::stdout))
        .with(fmt::layer().with_writer(non_blocking_file))
        .init();

    Box::leak(_guard);
    Ok(())
}
```

## 日誌輸出目標

### 1. 僅輸出到控制台

```rust
fn init_logger_console_only() {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer().with_writer(std::io::stdout))
        .init();
}
```

### 2. 僅輸出到檔案

```rust
fn init_logger_file_only() -> Result<(), Box<dyn std::error::Error>> {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    let file_appender = rolling::daily("./logs", "app.log");
    let (non_blocking_file, _guard) = tracing_appender::non_blocking(file_appender);

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer().with_writer(non_blocking_file))
        .init();

    Box::leak(_guard);
    Ok(())
}
```

### 3. 多個檔案輸出

```rust
fn init_logger_multiple_files() -> Result<(), Box<dyn std::error::Error>> {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    // General log file
    let app_appender = rolling::daily("./logs", "app.log");
    let (non_blocking_app, _app_guard) = tracing_appender::non_blocking(app_appender);

    // Error log file
    let error_appender = rolling::daily("./logs", "error.log");
    let (non_blocking_error, _error_guard) = tracing_appender::non_blocking(error_appender);

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer().with_writer(non_blocking_app))
        .with(fmt::layer().with_writer(non_blocking_error))
        .init();

    Box::leak(_app_guard);
    Box::leak(_error_guard);
    Ok(())
}
```

## 滾動策略

### 1. 每日滾動（推薦）

```rust
// Creates files: app.log, app.log.2025-01-15, app.log.2025-01-14, etc.
let file_appender = rolling::daily("./logs", "app.log");
```

**特點**：
- 自動每天建立新檔案
- 舊檔案自動備份
- 容易按日期查看日誌

### 2. 每小時滾動

```rust
// Creates hourly log files
let file_appender = rolling::hourly("./logs", "app.log");
```

**適用場景**：
- 高流量應用程式
- 需要細粒度時間分割
- 日誌檔案可能非常大

### 3. 永不滾動

```rust
// All logs go to app.log
let file_appender = rolling::never("./logs", "app.log");
```

**使用場景**：
- 開發環境
- 低日誌量應用程式
- 手動日誌輪轉

### 4. 自訂滾動邏輯

```rust
// Example: rotate every 100MB
use std::sync::Arc;

fn init_logger_with_size_rotation() -> Result<(), Box<dyn std::error::Error>> {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    let file_appender = rolling::daily("./logs", "app.log");
    let (non_blocking_file, _guard) = tracing_appender::non_blocking(file_appender);

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer().with_writer(non_blocking_file))
        .init();

    Box::leak(_guard);
    Ok(())
}
```

## 日誌級別控制

### 1. 日誌級別階層

從最詳細到最簡潔：

| 級別 | 說明 | 何時使用 |
|------|------|---------|
| `trace` | 最詳細的訊息 | 深度除錯 |
| `debug` | 除錯訊息 | 開發期間 |
| `info` | 一般資訊訊息 | 標準執行 |
| `warn` | 警告訊息 | 潛在問題 |
| `error` | 錯誤訊息 | 問題發生 |

### 2. 透過環境變數控制

```bash
# Run with debug level
RUST_LOG=debug cargo tauri dev

# Run with trace level
RUST_LOG=trace cargo tauri dev

# Run with error level only
RUST_LOG=error cargo tauri dev
```

### 3. 程式碼中設定全局級別

```rust
fn init_logger_debug() {
    let env_filter = EnvFilter::new("debug");

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer())
        .init();
}
```

### 4. 模組級別控制

```rust
fn init_logger_module_levels() {
    // Set specific module levels while maintaining base level
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| {
            EnvFilter::new("info,arx_discovery::mdns=debug,arx_discovery::discovery=trace")
        });

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer())
        .init();
}
```

環境變數等效用法：

```bash
RUST_LOG=info,arx_discovery::mdns=debug,arx_discovery::discovery=trace cargo tauri dev
```

### 5. 執行時控制

```rust
use tracing::{info, warn, error, debug, trace};

pub fn example_logging() {
    trace!("Detailed trace information");
    debug!("Debug message: {}", "details");
    info!("Application started successfully");
    warn!("Warning: something might be wrong");
    error!("Error occurred: {}", "error details");
}
```

## 格式化選項

### 1. 簡潔格式

```rust
fn init_logger_compact() {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer()
            .with_thread_ids(false)
            .with_file(false)
            .with_line_number(false)
        )
        .init();
}
```

### 2. 詳細格式

```rust
fn init_logger_detailed() {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("debug"));

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer()
            .with_thread_ids(true)
            .with_thread_names(true)
            .with_file(true)
            .with_line_number(true)
            .with_target(true)
        )
        .init();
}
```

### 3. JSON 格式

```rust
fn init_logger_json() {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    let file_appender = rolling::daily("./logs", "app.json.log");
    let (non_blocking_file, _guard) = tracing_appender::non_blocking(file_appender);

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer()
            .with_writer(non_blocking_file)
            .json()
        )
        .init();

    Box::leak(_guard);
}
```

JSON 輸出範例：

```json
{
  "timestamp": "2025-01-15T10:30:45.123456Z",
  "level": "INFO",
  "target": "arx_discovery",
  "message": "Service started",
  "module_path": "arx_discovery::service",
  "file": "service.rs",
  "line": 42
}
```

### 4. 自訂時間戳格式

```rust
use tracing_subscriber::fmt::time::SystemTime;

fn init_logger_custom_time() {
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt::layer()
            .with_timer(SystemTime::default())
        )
        .init();
}
```

## 效能考量

### 1. 非阻塞寫入

```rust
// Using non_blocking wrapper for better performance
let file_appender = rolling::daily("./logs", "app.log");
let (non_blocking_file, _guard) = tracing_appender::non_blocking(file_appender);

// This prevents disk I/O from blocking application threads
tracing_subscriber::registry()
    .with(fmt::layer().with_writer(non_blocking_file))
    .init();
```

### 2. 緩衝設定

```rust
// Non-blocking by default uses internal buffering
let file_appender = rolling::daily("./logs", "app.log");
let (non_blocking_file, _guard) = tracing_appender::non_blocking(file_appender);

// Specify buffer capacity (optional)
// Default is usually 128KB
```

### 3. 選擇性級別過濾

```rust
// Only enable trace in specific modules to reduce overhead
let env_filter = EnvFilter::new("info,arx_discovery::mdns=trace");

tracing_subscriber::registry()
    .with(env_filter)
    .with(fmt::layer())
    .init();
```

## 實際應用範例

### 完整的生產環境配置

```rust
use tracing_subscriber::{fmt, prelude::*, EnvFilter, filter::LevelFilter};
use tracing_appender::rolling;
use tauri::Manager;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            // Get app-specific directories
            let log_dir = app.path().app_log_dir()?;
            std::fs::create_dir_all(&log_dir)?;

            // Initialize logging system
            init_production_logger(log_dir)?;

            Ok(())
        })
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![greet])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn init_production_logger(log_dir: impl AsRef<std::path::Path>) -> Result<(), Box<dyn std::error::Error>> {
    // Environment-based configuration
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    // Create separate log files
    let general_appender = rolling::daily(log_dir.as_ref(), "app.log");
    let (non_blocking_general, _general_guard) = tracing_appender::non_blocking(general_appender);

    let error_appender = rolling::daily(log_dir.as_ref(), "error.log");
    let (non_blocking_error, _error_guard) = tracing_appender::non_blocking(error_appender);

    // Build layered subscriber
    tracing_subscriber::registry()
        .with(env_filter)
        .with(
            fmt::layer()
                .with_writer(std::io::stdout)
                .with_thread_ids(true)
                .with_line_number(true)
        )
        .with(
            fmt::layer()
                .with_writer(non_blocking_general)
                .with_thread_ids(true)
                .with_line_number(true)
        )
        .with(
            fmt::layer()
                .with_writer(non_blocking_error)
                .with_level(true)
                .with_target(true)
        )
        .init();

    Box::leak(_general_guard);
    Box::leak(_error_guard);

    tracing::info!("Logger initialized successfully");
    Ok(())
}

// Usage in application code
use tracing::{info, warn, error, debug};

pub fn handle_device_discovery() {
    info!("Starting device discovery");

    match discover_devices() {
        Ok(devices) => {
            info!("Found {} devices", devices.len());
            for device in devices {
                debug!(device_id = %device.id, "Processing device");
            }
        }
        Err(e) => {
            error!("Discovery failed: {}", e);
            warn!("Will retry in 5 seconds");
        }
    }
}

fn discover_devices() -> Result<Vec<Device>, String> {
    // Device discovery logic
    Ok(vec![])
}

struct Device {
    id: String,
}
```

### mDNS 服務與日誌整合

```rust
use tracing::{info, debug, warn, error};

pub fn init_mdns_with_logging() {
    info!("Initializing mDNS service discovery");

    // mDNS initialization code
    debug!("mDNS service browser created");
}

pub fn handle_service_event(event_type: &str) {
    match event_type {
        "service_found" => {
            info!("New service discovered");
        }
        "service_lost" => {
            warn!("Service disappeared from network");
        }
        "service_updated" => {
            debug!("Service information updated");
        }
        _ => {
            error!("Unknown service event: {}", event_type);
        }
    }
}
```

## 故障排除

### 1. 日誌未顯示

**問題**：執行應用程式時看不到日誌輸出

**解決方案**：

```bash
# Check if RUST_LOG is set correctly
RUST_LOG=debug cargo tauri dev

# Ensure logger is initialized before any logging calls
# Check initialization code is called in setup hook
```

### 2. 日誌檔案無法建立

**問題**：`permission denied` 或路徑無效

**解決方案**：

```rust
// Ensure directory exists before creating appender
let log_dir = "./logs";
std::fs::create_dir_all(log_dir)?;

let file_appender = rolling::daily(log_dir, "app.log");
```

### 3. 效能下降

**問題**：應用程式變慢，特別是在高日誌量情況下

**解決方案**：

```rust
// Use non_blocking wrapper
let file_appender = rolling::daily("./logs", "app.log");
let (non_blocking_file, _guard) = tracing_appender::non_blocking(file_appender);

// Raise minimum log level
let env_filter = EnvFilter::new("warn");

// Disable unnecessary fields
.with_file(false)
.with_line_number(false)
```

### 4. 日誌檔案過大

**問題**：日誌檔案佔用過多磁碟空間

**解決方案**：

```rust
// Use daily rolling instead of never
let file_appender = rolling::daily("./logs", "app.log");

// Raise minimum level
let env_filter = EnvFilter::new("info");

// Only log warnings and errors in production
#[cfg(not(debug_assertions))]
let env_filter = EnvFilter::new("warn");
```

### 5. 日誌級別設定不生效

**問題**：環境變數設定的級別無法生效

**解決方案**：

```rust
// Correct order matters: set environment variable BEFORE running
RUST_LOG=debug cargo tauri dev

// Verify module name in logs matches filter
// Example: if logs show "arx_discovery::" use that exact prefix
RUST_LOG=arx_discovery=debug cargo tauri dev

// Use wildcard for all modules
RUST_LOG=trace cargo tauri dev
```

## 最佳實踐

### 1. 日誌級別使用規範

```rust
use tracing::{trace, debug, info, warn, error};

// trace: 極詳細的執行路徑追蹤
trace!("Entering function with args: {}", args);

// debug: 開發除錯訊息
debug!(user_id = %id, "User logged in");

// info: 重要事件
info!("Service started on port 8080");

// warn: 潛在問題
warn!("High memory usage detected: {} MB", usage);

// error: 錯誤情況
error!("Failed to connect to database: {}", err);
```

### 2. 結構化日誌

```rust
use tracing::info;

// Good: structured fields
info!(
    user_id = %user.id,
    email = %user.email,
    duration_ms = elapsed.as_millis(),
    "User registration completed"
);

// Avoid: unstructured concatenation
// info!("User {} with email {} registered in {}ms", user.id, user.email, elapsed.as_millis());
```

### 3. 敏感訊息處理

```rust
use tracing::debug;

// Avoid logging passwords or tokens
// debug!("User password: {}", password);

// Instead, log only necessary identifiers
debug!(user_id = %user.id, "Authentication attempt");

// For errors, be cautious with sensitive data
// error!("API request failed: {}", response_body);
```

### 4. 性能優化建議

- 在生產環境使用 `warn` 或 `error` 級別
- 在開發環境可使用 `debug` 級別
- 使用非阻塞寫入避免 I/O 堵塞
- 定期檢查和清理舊日誌檔案
- 監控日誌檔案大小

### 5. 日誌檔案維護

```bash
# List current log files
ls -lh logs/

# View recent logs
tail -f logs/app.log

# Search in logs
grep "error" logs/app.log

# Clean old logs (keep last 7 days)
find logs/ -name "*.log.*" -mtime +7 -delete
```

### 6. 整合監控

建議使用日誌聚合工具處理生產環境日誌：

- 本地檔案：簡單應用程式或開發環境
- ELK Stack：中型應用程式
- 雲端日誌服務：生產環境（CloudWatch, Stackdriver 等）

---

## 相關資源

- [Rust tracing 文檔](https://docs.rs/tracing/)
- [tracing-subscriber 指南](https://docs.rs/tracing-subscriber/)
- [Tauri 官方文檔](https://tauri.app/)
- [Rust 日誌最佳實踐](https://docs.rust-embedded.org/book/unsupported/logger.html)
