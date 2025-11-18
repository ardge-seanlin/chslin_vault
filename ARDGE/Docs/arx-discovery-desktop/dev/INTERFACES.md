# mDNS 服務發現 - Interfaces 互動圖

## 1. 整體系統架構互動

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Frontend (Tauri UI)                          │
│                                                                      │
│  listen("add_service") ← Backend emits DiscoveredService            │
│  listen("clear") ← Backend emits when registry clears              │
│  invoke("get_devices") → Trigger refresh cycle                      │
└──────────────────────────────────┬──────────────────────────────────┘
                                   │ Tauri IPC
                                   ↓
┌──────────────────────────────────────────────────────────────────────┐
│              Tauri Backend Application (lib.rs)                      │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ run() function:                                             │   │
│  │ - Initialize MdnsClient + ServiceRegistry                  │   │
│  │ - Start discovery                                          │   │
│  │ - Setup event listener task                                │   │
│  │ - Register commands (get_devices)                          │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ Event Listener Task (async):                               │  │
│  │                                                              │  │
│  │ loop {                                                       │  │
│  │   receiver.recv_async() → ServiceEvent                      │  │
│  │                                                              │  │
│  │   ServiceEvent::ServiceAdded {                              │  │
│  │     ├─ registry.add_service(service)                        │  │
│  │     └─ app_handle.emit("add_service", service)             │  │
│  │                                                              │  │
│  │   ServiceEvent::ServicesCleared {                           │  │
│  │     └─ app_handle.emit("clear", ())                         │  │
│  │                                                              │  │
│  │   Other events ← ignored for MVP                            │  │
│  │ }                                                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ Tauri Commands:                                             │  │
│  │                                                              │  │
│  │ get_devices(client, registry) → Result<(), String> {        │  │
│  │   1. client.stop_discovery()                                │  │
│  │   2. registry.clear()                                       │  │
│  │   3. client.start_discovery()                               │  │
│  │   ← Returns immediately, events flow through listener       │  │
│  │ }                                                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Managed State:                                                      │
│  - Mutex<MdnsClient>        (exclusive mutable access)              │
│  - ServiceRegistry          (Arc-wrapped for cloning)               │
└──────────────────────────────────┬──────────────────────────────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    ↓                             ↓
        ┌─────────────────────┐      ┌──────────────────────┐
        │   MdnsClient        │      │ ServiceRegistry      │
        │   (Business Logic)  │      │ (Business Logic)     │
        └─────────────────────┘      └──────────────────────┘
```

---

## 2. MdnsClient Interface 詳細互動

```
MdnsClient {
    config: DiscoveryConfig
    daemon: Option<ServiceDaemon>  (from mdns-sd crate)
    event_channel: (Sender, Receiver<ServiceEvent>)
    is_running: Arc<AtomicBool>
}

┌─────────────────────────────────────────────────────────────┐
│ MdnsClient 生命週期                                          │
└─────────────────────────────────────────────────────────────┘

1. new(config: DiscoveryConfig) → Result<MdnsClient>
   ├─ Validates config
   ├─ Creates flume channel for ServiceEvent
   └─ Returns instance (discovery not started yet)

2. start_discovery() → Result<()>
   ├─ Creates ServiceDaemon
   ├─ Starts browsing for service_type
   ├─ Spawns event handler task (async loop)
   ├─ Emits ServiceEvent::DiscoveryStarted
   └─ Sets is_running = true

3. get_receiver() → Receiver<ServiceEvent>
   └─ Returns cloneable receiver for the channel

4. Event Handler Task (background):

   Loop {
     Listen to mdns-sd MdnsEvent

     MdnsEvent::ServiceResolved(fullname, service_info)
       ├─ Resolver::resolve(fullname, service_info)
       │  └─ Returns DiscoveredService
       ├─ Emit ServiceEvent::ServiceAdded { service }
       └─ Event flows to receiver

     MdnsEvent::ServiceRemoved(fullname)
       ├─ Extract service name
       └─ Emit ServiceEvent::ServiceRemoved { id, name }

     ... (other events ignored for MVP)
   }

5. stop_discovery() → Result<()>
   ├─ Stops ServiceDaemon
   ├─ Emits ServiceEvent::DiscoveryStopped
   └─ Sets is_running = false
```

---

## 3. ServiceRegistry Interface 詳細互動

```
ServiceRegistry {
    services: Arc<RwLock<HashMap<ServiceId, DiscoveredService>>>
    last_access: Arc<RwLock<HashMap<ServiceId, SystemTime>>>
}

┌─────────────────────────────────────────────────────────────┐
│ ServiceRegistry 操作                                         │
└─────────────────────────────────────────────────────────────┘

1. new() → ServiceRegistry
   └─ Creates empty HashMaps wrapped in Arc<RwLock<>>

2. add_service(service: DiscoveredService) → Result<()>

   Acquires write locks on both maps:
   ├─ services.write() → Insert or update service
   ├─ last_access.write() → Update timestamp to now()
   └─ Returns Ok(())

   Error cases:
   └─ RwLock poison → AppError::internal()

3. list_all_services() → Result<Vec<DiscoveredService>>

   Acquires read lock:
   ├─ services.read() → Clone all values
   └─ Returns Vec<DiscoveredService>

   Multiple threads can read simultaneously

4. clear() → Result<()>

   Acquires write locks on both maps:
   ├─ services.write() → services.clear()
   ├─ last_access.write() → last_access.clear()
   └─ Returns Ok(())

   Atomic operation: both maps cleared together

5. Thread Safety Pattern:

   Arc<RwLock<T>> allows:
   ├─ Multiple concurrent readers (read locks)
   ├─ Exclusive writer (write lock)
   └─ No deadlocks (simple lock ordering)
```

---

## 4. Event Flow 詳細互動

```
┌─────────────────────────────────────────────────────────────┐
│ 事件流向 (Event Flow Chain)                                  │
└─────────────────────────────────────────────────────────────┘

User opens app:
  ↓
run() in lib.rs:
  ├─ Create MdnsClient with DiscoveryConfig
  ├─ Create ServiceRegistry
  ├─ Call client.start_discovery()
  │  └─ ServiceEvent::DiscoveryStarted (emitted internally)
  └─ Spawn event listener task

  ↓
Event Listener Task (async loop):
  while let Ok(event) = receiver.recv_async().await {

    ServiceEvent::ServiceAdded { service } ──┐
      │                                        │
      ├─ registry.add_service(service)       │ (from MdnsClient)
      │  ├─ Acquires RwLock::write()         │
      │  ├─ Inserts into HashMap             │
      │  └─ Updates timestamp                │
      │                                       │
      └─ app_handle.emit("add_service", service)
         └─ Frontend receives event

    ServiceEvent::ServicesCleared ────────────┐
      │                                        │
      └─ app_handle.emit("clear", ())       │ (from Tauri command)
         └─ Frontend receives event

    Other events → ignored (MVP)
  }

Frontend interaction:

User clicks "Refresh" button:
  ↓
invoke("get_devices"):

  1. Client.stop_discovery()
     ├─ Stops browsing
     └─ Emits ServiceEvent::DiscoveryStopped

  2. Registry.clear()
     ├─ Clears all services
     ├─ Clears all timestamps
     └─ Returns Ok(())

  3. Client.start_discovery()
     ├─ Starts browsing again
     └─ Emits ServiceEvent::DiscoveryStarted

  ↓ (Command returns immediately)

  Event Listener continues receiving events:
  ├─ ServiceEvent::ServiceAdded events → emit("add_service", ...)
  └─ ServiceEvent::ServicesCleared → emit("clear", ...)

  ↓
Frontend receives events:
  ├─ listen("add_service") → Add to UI list
  └─ listen("clear") → Clear UI list
```

---

## 5. 資料流向 (Data Flow)

```
┌─────────────────────────────────────────────────────────────┐
│ 資料型別和轉換                                               │
└─────────────────────────────────────────────────────────────┘

mDNS Network
  ↓ (mdns-sd crate)
ServiceInfo (from mdns-sd)
  ↓ (Resolver::resolve())
DiscoveredService
  ├─ id: ServiceId (String)
  ├─ name: String
  ├─ service_type: String
  ├─ hostname: String
  ├─ addresses: Vec<IpAddr>
  ├─ port: u16
  ├─ txt_records: HashMap<String, String>
  ├─ priority: u16
  ├─ weight: u16
  ├─ discovered_at: SystemTime
  └─ last_seen_at: SystemTime

  ↓ (Serialized by serde)

JSON (to Frontend):
{
  "id": "service-name._arx._tcp.local.",
  "name": "My Service",
  "service_type": "_arx._tcp.local.",
  "hostname": "device.local.",
  "addresses": ["192.168.1.100"],
  "port": 8080,
  "txt_records": {"version": "1.0"},
  "priority": 0,
  "weight": 0,
  "discovered_at": "2025-11-14T10:30:00Z",
  "last_seen_at": "2025-11-14T10:30:05Z"
}


┌─────────────────────────────────────────────────────────────┐
│ ServiceEvent 型別                                           │
└─────────────────────────────────────────────────────────────┘

#[serde(tag = "type", content = "payload")]
pub enum ServiceEvent {
    ServiceAdded { service: DiscoveredService },
    ServiceRemoved { id: ServiceId, name: String },
    DiscoveryStarted,
    DiscoveryStopped,
    DiscoveryError { message: String },
    ServicesCleared,
}

序列化範例:
{
  "type": "ServiceAdded",
  "payload": { /* DiscoveredService JSON */ }
}

或

{
  "type": "ServiceRemoved",
  "payload": {
    "id": "service-id",
    "name": "service-name"
  }
}


┌─────────────────────────────────────────────────────────────┐
│ 錯誤流向 (Error Flow)                                        │
└─────────────────────────────────────────────────────────────┘

AppError enum:
├─ MdnsError(String)        → mDNS protocol failures
├─ InvalidData(String)      → Configuration/data validation
└─ InternalError(String)    → Lock poisoning, system issues

Flow:
MdnsClient::start_discovery()
  └─ Error → Err(AppError::MdnsError(...))
              └─ Converted to String in Tauri command
                 └─ Sent to frontend

Registry::add_service()
  └─ RwLock::write() fails
     └─ Err(AppError::InternalError(...))
        └─ Logged to stderr
           └─ Event not emitted to frontend
```

---

## 6. 模組依賴關係圖

```
┌─────────────────────────────────────────────────────────────┐
│ 模組層級 (Module Hierarchy)                                  │
└─────────────────────────────────────────────────────────────┘

lib.rs (Entry Point)
├─ pub mod commands
│  └─ pub mod get_devices
│     └─ Uses: MdnsClient, ServiceRegistry (via State)
│
├─ pub mod mdns (Zero Tauri dependencies)
│  ├─ client.rs → MdnsClient
│  │  ├─ Uses: DiscoveryConfig, ServiceEvent
│  │  ├─ Uses: mdns-sd crate (external)
│  │  └─ Uses: flume channel
│  │
│  ├─ resolver.rs → Resolver
│  │  ├─ Uses: ServiceInfo (from mdns-sd)
│  │  └─ Produces: DiscoveredService
│  │
│  ├─ parser.rs → TxtRecordParser
│  │  └─ Parses TXT records from mDNS
│  │
│  └─ mod.rs (exports)
│
├─ pub mod service (Zero Tauri dependencies)
│  ├─ registry.rs → ServiceRegistry
│  │  ├─ Uses: DiscoveredService, ServiceId
│  │  ├─ Uses: Arc, RwLock for thread safety
│  │  └─ Stores: HashMap<ServiceId, DiscoveredService>
│  │
│  └─ mod.rs (exports)
│
├─ pub mod models (Data structures, zero Tauri dependencies)
│  ├─ service.rs
│  │  ├─ DiscoveredService (struct)
│  │  ├─ ServiceId (type alias: String)
│  │  └─ ServiceEvent (enum)
│  │
│  ├─ config.rs
│  │  └─ DiscoveryConfig (struct)
│  │
│  └─ mod.rs (exports)
│
└─ pub mod error (Error handling, zero Tauri dependencies)
   └─ error.rs
      ├─ AppError (enum)
      ├─ AppResult<T> (type alias)
      └─ Error conversion implementations


┌─────────────────────────────────────────────────────────────┐
│ 依賴方向 (Dependency Direction)                             │
└─────────────────────────────────────────────────────────────┘

Tauri Layer (只在這裡):
  ↓ (depends on)
Business Logic Layer (零 Tauri 依賴):
  ├─ MdnsClient
  ├─ ServiceRegistry
  └─ Models + Error handling

External Dependencies (零耦合):
  ├─ mdns-sd (for mDNS protocol)
  ├─ flume (for channels)
  ├─ serde (for serialization)
  └─ tokio (for async runtime)

✓ 單向依賴: Tauri → Business Logic
✓ 零循環依賴
✓ 業務邏輯可獨立測試
✓ 易於遷移到 gRPC
```

---

## 7. 並發和執行模型

```
┌─────────────────────────────────────────────────────────────┐
│ 執行執行緒和任務 (Execution Threads/Tasks)                  │
└─────────────────────────────────────────────────────────────┘

Main Thread (Tauri):
  ├─ Runs Tauri event loop
  ├─ Handles frontend IPC
  └─ Processes command invocations

MdnsClient Event Handler (background async task):
  ├─ Spawned in lib.rs run() setup
  ├─ Listens to mdns-sd events (background)
  └─ Sends ServiceEvents to flume channel

Event Listener Task (async, Tauri runtime):
  ├─ Spawned in lib.rs .setup()
  ├─ Awaits on receiver.recv_async()
  ├─ Calls registry.add_service() (thread-safe)
  └─ Calls app_handle.emit() (from Tauri runtime)

mDNS Library Thread (mdns-sd internal):
  └─ Background network listening


┌─────────────────────────────────────────────────────────────┐
│ 同步機制 (Synchronization)                                   │
└─────────────────────────────────────────────────────────────┘

MdnsClient → Event Listener:
  └─ flume::unbounded() channel
     ├─ Sender in MdnsClient
     └─ Receiver in Event Listener Task
     └─ Thread-safe, message passing

Event Listener → ServiceRegistry:
  └─ Arc<RwLock<T>>
     ├─ Multiple readers (list_all_services)
     └─ Exclusive writer (add_service, clear)

MdnsClient access:
  └─ Mutex<MdnsClient> in Tauri state
     └─ Exclusive access via get_devices command

No Deadlocks:
  ├─ Simple lock ordering (registry only)
  ├─ No nested locks
  └─ Timeout not needed (MVP)
```

---

## 8. 生產流程總結

```
┌─────────────────────────────────────────────────────────────┐
│ 完整的服務發現週期                                          │
└─────────────────────────────────────────────────────────────┘

1️⃣  應用啟動:
    lib.rs::run()
      → Create DiscoveryConfig (default)
      → Create MdnsClient(config)
      → Create ServiceRegistry
      → client.start_discovery()
      → Setup event listener task
      → Setup Tauri commands & state

2️⃣  mDNS 掃描 (background):
    ServiceDaemon listens to network
      → ServiceInfo found
      → Resolver converts to DiscoveredService
      → Emit ServiceEvent::ServiceAdded
      → Flow to receiver channel

3️⃣  事件監聽和處理:
    Event Listener Task receives event
      → match ServiceEvent::ServiceAdded { service }
      → registry.add_service(service)  [RwLock::write]
      → app_handle.emit("add_service", service)

4️⃣  前端接收事件:
    Frontend listen("add_service")
      → Display service in UI

5️⃣  使用者點擊刷新:
    invoke("get_devices")
      → Stop discovery
      → Clear registry (RwLock::write)
      → Start discovery again
      → Return immediately

6️⃣  新一輪掃描:
    重複步驟 2-4

7️⃣  應用關閉:
    client.stop_discovery()
      → Shutdown daemon
      → Clean up resources
```

---

## 邊界和責任

```
┌─────────────────────────────────────────────────────────────┐
│ 各層責任分工                                                 │
└─────────────────────────────────────────────────────────────┘

Frontend (Tauri UI):
  ✓ User interaction handling
  ✓ Display services in list
  ✓ Invoke refresh command
  ✗ No business logic
  ✗ No direct mDNS access

Tauri Backend (lib.rs, commands):
  ✓ Application setup
  ✓ State management (Mutex, Arc)
  ✓ Event listener task
  ✓ Command registration
  ✓ IPC bridge to frontend
  ✗ No mDNS protocol details
  ✗ No network operations

Business Logic (mdns/, service/):
  ✓ mDNS discovery (MdnsClient)
  ✓ Service resolution (Resolver)
  ✓ Service storage (ServiceRegistry)
  ✓ Error handling
  ✗ No Tauri dependencies
  ✗ No UI concerns
  ✗ No IPC

External Libraries:
  ✓ mdns-sd: mDNS protocol
  ✓ flume: Message passing
  ✓ tokio: Async runtime
  ✓ serde: Serialization


┌─────────────────────────────────────────────────────────────┐
│ 可移植性 (Portability)                                       │
└─────────────────────────────────────────────────────────────┘

當前: Tauri + Tauri IPC
  MdnsClient ← (不變)
  ServiceRegistry ← (不變)
  Event Listener → app_handle.emit()

遷移到 gRPC:
  MdnsClient ← (不變)
  ServiceRegistry ← (不變)
  Event Listener → grpc_stream.send()

✓ 只需改變事件發送機制
✓ 業務邏輯完全相同
✓ 無需重新測試 mDNS 邏輯
```
