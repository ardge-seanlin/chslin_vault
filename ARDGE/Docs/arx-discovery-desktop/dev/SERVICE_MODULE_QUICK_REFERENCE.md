# Service Module Quick Reference

## 快速開始

### 基本使用

```rust
use arx_discovery_lib::{ServiceManager, DiscoveryConfig, ServiceEvent};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 1. 建立配置
    let config = DiscoveryConfig::default();

    // 2. 初始化管理器
    let mut manager = ServiceManager::new(config)?;

    // 3. 啟動自動清理任務
    manager.start_cleanup_task().await?;

    // 4. 處理事件
    let event = ServiceEvent::DiscoveryStarted;
    manager.handle_service_event(event)?;

    // 5. 查詢註冊表
    let registry = manager.get_registry();
    let services = registry.list_all_services()?;

    Ok(())
}
```

## API 總覽

### ServiceRegistry

執行緒安全的服務儲存庫。

```rust
// 建立
let registry = ServiceRegistry::new();

// 新增服務
registry.add_service(service)?;

// 取得服務
let service = registry.get_service(&id)?;

// 更新服務
registry.update_service(updated_service)?;

// 移除服務
let removed = registry.remove_service(&id)?;

// 列出所有服務
let services = registry.list_all_services()?;

// 清空所有服務
registry.clear_all_services()?;

// 清理過期服務
let expired = registry.cleanup_expired_services(ttl_secs)?;

// 檢查服務存在
let exists = registry.contains(&id)?;

// 取得服務數量
let count = registry.count()?;
```

### ServiceManager

高階服務生命週期管理器。

```rust
// 建立管理器
let config = DiscoveryConfig::new(
    "_arx._tcp.local.".to_string(),
    5,    // 解析超時（秒）
    300,  // TTL（秒）
);
let mut manager = ServiceManager::new(config)?;

// 啟動自動清理任務
manager.start_cleanup_task().await?;

// 停止清理任務
manager.stop_cleanup_task().await?;

// 處理事件
manager.handle_service_event(ServiceEvent::ServiceAdded { service })?;

// 取得註冊表
let registry = manager.get_registry();

// 手動清理
let expired = manager.cleanup_now()?;

// 重新載入配置
let new_config = DiscoveryConfig::default();
manager.reload(new_config).await?;

// 檢查清理任務狀態
let is_running = manager.is_cleanup_running().await;

// 取得配置
let config = manager.get_config();
```

## 事件處理

### ServiceEvent 類型

```rust
pub enum ServiceEvent {
    // 新服務被發現
    ServiceAdded { service: DiscoveredService },

    // 服務被移除
    ServiceRemoved { id: ServiceId, name: String },

    // 發現已啟動
    DiscoveryStarted,

    // 發現已停止
    DiscoveryStopped,

    // 發現錯誤
    DiscoveryError { message: String },

    // 所有服務已清除
    ServicesCleared,
}
```

### 事件處理範例

```rust
match event {
    ServiceEvent::ServiceAdded { service } => {
        println!("新服務: {}", service.name);
        manager.handle_service_event(event)?;
    }
    ServiceEvent::ServiceRemoved { id, name } => {
        println!("服務移除: {}", name);
        manager.handle_service_event(event)?;
    }
    ServiceEvent::DiscoveryError { message } => {
        eprintln!("錯誤: {}", message);
    }
    _ => {
        manager.handle_service_event(event)?;
    }
}
```

## 並發安全

### ServiceRegistry 並發保證

- ✅ 多個讀取操作可並行執行
- ✅ 寫入操作互斥保護
- ✅ 無死鎖風險
- ✅ 執行緒安全的克隆（共享底層資料）

```rust
let registry = Arc::new(ServiceRegistry::new());

// 多執行緒讀取
let registry_clone = Arc::clone(&registry);
tokio::spawn(async move {
    let services = registry_clone.list_all_services().unwrap();
    // ... 處理
});

// 多執行緒寫入
let registry_clone = Arc::clone(&registry);
tokio::spawn(async move {
    registry_clone.add_service(service).unwrap();
});
```

## TTL 清理機制

### 自動清理

```rust
// 啟動後，清理任務每 60 秒執行一次
manager.start_cleanup_task().await?;

// 移除超過 TTL 的服務
// TTL 在 DiscoveryConfig.service_ttl_secs 中設定
```

### 手動清理

```rust
// 立即清理過期服務
let expired_ids = manager.cleanup_now()?;
println!("移除了 {} 個過期服務", expired_ids.len());
```

## 配置管理

### 建立配置

```rust
// 使用預設值
let config = DiscoveryConfig::default();

// 自訂配置
let config = DiscoveryConfig::new(
    "_arx._tcp.local.".to_string(),  // 服務類型
    5,                                 // 解析超時（秒）
    300,                               // TTL（秒）
);

// 為特定服務類型建立
let config = DiscoveryConfig::for_service_type("_http._tcp.local.");
```

### 配置驗證

```rust
// 配置會自動驗證
match config.validate() {
    Ok(_) => println!("配置有效"),
    Err(e) => eprintln!("配置無效: {}", e),
}
```

### 動態重新載入

```rust
let new_config = DiscoveryConfig::new(
    "_http._tcp.local.".to_string(),
    10,
    600,
);

// 重新載入會自動重啟清理任務（如果之前在執行）
manager.reload(new_config).await?;
```

## 錯誤處理

所有操作返回 `AppResult<T>`：

```rust
use arx_discovery_lib::{AppResult, AppError};

fn example() -> AppResult<()> {
    let registry = ServiceRegistry::new();

    // 錯誤會自動傳播
    let service = registry.get_service(&id)?;

    // 手動錯誤處理
    match registry.add_service(service) {
        Ok(_) => println!("成功"),
        Err(AppError::MdnsError(msg)) => eprintln!("mDNS 錯誤: {}", msg),
        Err(AppError::InvalidData(msg)) => eprintln!("無效資料: {}", msg),
        Err(AppError::InternalError(msg)) => eprintln!("內部錯誤: {}", msg),
    }

    Ok(())
}
```

## 效能特性

### 時間複雜度

| 操作 | 複雜度 |
|------|--------|
| add_service | O(1) 平攤 |
| get_service | O(1) |
| remove_service | O(1) |
| list_all_services | O(n) |
| cleanup_expired | O(n) |
| contains | O(1) |
| count | O(1) |

### 記憶體使用

- 每個服務：約 200-500 bytes（取決於 TXT 記錄）
- 註冊表額外開銷：O(n)
- 管理器額外開銷：O(1)

## 測試範例

### 執行測試

```bash
# 所有測試
cargo test --lib

# 只測試 service 模組
cargo test --lib service

# 詳細輸出
cargo test --lib service -- --nocapture

# 特定測試
cargo test --lib test_add_service
```

### 執行範例

```bash
# 服務管理範例
cargo run --example service_management

# mDNS 發現範例（來自 PR #2）
cargo run --example mdns_discovery
```

## 常見模式

### 模式 1：基本服務追蹤

```rust
let config = DiscoveryConfig::default();
let mut manager = ServiceManager::new(config)?;
manager.start_cleanup_task().await?;

// 從 MdnsClient 接收事件
while let Ok(event) = event_rx.recv() {
    manager.handle_service_event(event)?;
}
```

### 模式 2：查詢特定服務

```rust
let registry = manager.get_registry();

// 檢查服務是否存在
if registry.contains(&service_id)? {
    let service = registry.get_service(&service_id)?.unwrap();
    println!("找到服務: {}", service.name);
}
```

### 模式 3：定期快照

```rust
use tokio::time::{interval, Duration};

let mut ticker = interval(Duration::from_secs(10));
loop {
    ticker.tick().await;
    let services = registry.list_all_services()?;
    println!("當前服務數: {}", services.len());
}
```

### 模式 4：整合 MdnsClient

```rust
use arx_discovery_lib::{MdnsClient, ServiceManager, DiscoveryConfig};

let config = DiscoveryConfig::default();
let mut manager = ServiceManager::new(config.clone())?;
manager.start_cleanup_task().await?;

let client = MdnsClient::new(config)?;
let event_rx = client.start_discovery().await?;

tokio::spawn(async move {
    while let Ok(event) = event_rx.recv_async().await {
        if let Err(e) = manager.handle_service_event(event) {
            eprintln!("處理事件錯誤: {}", e);
        }
    }
});
```

## 最佳實踐

### ✅ 建議

1. **始終啟動清理任務**
   ```rust
   manager.start_cleanup_task().await?;
   ```

2. **使用適當的 TTL**
   ```rust
   // 區域網路：60-300 秒
   // 穩定環境：300-600 秒
   let config = DiscoveryConfig::new(service_type, 5, 300);
   ```

3. **錯誤處理**
   ```rust
   if let Err(e) = manager.handle_service_event(event) {
       tracing::error!("事件處理失敗: {}", e);
   }
   ```

4. **資源清理**
   ```rust
   // 應用程式關閉時
   manager.stop_cleanup_task().await?;
   ```

### ❌ 避免

1. 不要在持有註冊表鎖時執行 I/O 操作
2. 不要設定過短的 TTL（< 10 秒）
3. 不要忽略事件處理錯誤
4. 不要頻繁調用 `cleanup_now()`（使用自動清理）

## 除錯建議

### 啟用日誌

```rust
// 在 main 函數開頭
tracing_subscriber::fmt::init();
```

### 檢查服務狀態

```rust
let count = registry.count()?;
let services = registry.list_all_services()?;
println!("註冊表狀態: {} 個服務", count);
for service in services {
    println!("  - {} [{}]", service.name, service.id);
}
```

### 監控清理

```rust
let expired = manager.cleanup_now()?;
if !expired.is_empty() {
    println!("清理了服務: {:?}", expired);
}
```

## 相關文件

- [PR #1: 資料模型](../src-tauri/src/models/)
- [PR #2: mDNS Client](../src-tauri/src/mdns/)
- [範例程式碼](../src-tauri/examples/)
- [完整實作總結](./PR3_IMPLEMENTATION_SUMMARY.md)
