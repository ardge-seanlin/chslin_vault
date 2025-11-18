# mDNS Service Discovery Application - Architecture Document

## Overview

This is a Tauri-based desktop application for discovering mDNS services on a local network. The architecture is designed with clear separation of concerns to be highly portable to other backends (e.g., gRPC servers) without modifying the core business logic.

---

## System Architecture

### High-Level Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (Tauri UI)                       │
│                                                                  │
│  [Service List]  ← listen("add_service") ← MdnsClient events  │
│  [Clear Button]  ← listen("clear")                             │
└──────────────────────────────────┬──────────────────────────────┘
                                   │
                                   ↓
                        [Tauri IPC Bridge]
                                   │
                    ┌──────────────┴──────────────┐
                    ↓                             ↓
          get_devices()                  Event Listener Task
          (Command)                       (Async Background)
                    │                             │
         ┌──────────┴──────────┐        ┌────────┴─────────┐
         ↓                     ↓        ↓                  ↓
    Stop mDNS           Clear Registry  Listen to         Emit Events
    (client)            (registry)      Channels          to Frontend
         │                     │        (MdnsClient)
         └─────────────────────┴────────┘
                     │
                     ↓
         ┌─────────────────────────┐
         │     Start mDNS          │
         │   (client.listen)       │
         └────────────┬────────────┘
                      │
                      ↓
         ┌─────────────────────────┐
         │  Backend Services       │
         │  (Business Logic Layer) │
         └─────────────────────────┘
```

---

## Core Components

### 1. MdnsClient (`src/mdns/client.rs`)

**Responsibility**: Manage mDNS discovery lifecycle and emit service events

**Key Methods**:
- `new(config: DiscoveryConfig) -> Result<Self>`
  - Create client with configuration
  - Initialize event channel

- `start_discovery() -> Result<()>`
  - Start browsing for services
  - Spawn event handler task
  - Emit `ServiceEvent::DiscoveryStarted`

- `stop_discovery() -> Result<()>`
  - Stop browsing
  - Clean up resources
  - Emit `ServiceEvent::DiscoveryStopped`

- `get_receiver() -> Receiver<ServiceEvent>`
  - Get cloneable receiver for event channel
  - Multiple receivers can be created

**Event Emission Points**:
```rust
// When new service is discovered
ServiceEvent::ServiceAdded { service: DiscoveredService }

// When service disappears
ServiceEvent::ServiceRemoved { id: ServiceId, name: String }

// Lifecycle events
ServiceEvent::DiscoveryStarted
ServiceEvent::DiscoveryStopped
ServiceEvent::DiscoveryError { message: String }
```

**Design Notes**:
- Uses `mdns-sd` crate for underlying mDNS protocol
- Event channel is unbounded (capacity: 100)
- Thread-safe: `Arc<AtomicBool>` for state management
- No Tauri dependencies - completely decoupled

---

### 2. ServiceRegistry (`src/service/registry.rs`)

**Responsibility**: Thread-safe in-memory storage for discovered services

**Key Methods**:
- `new() -> Self`
  - Create empty registry with two HashMaps
  - One for services, one for last_access timestamps

- `add_service(service: DiscoveredService) -> Result<()>`
  - Insert or update service
  - Update last_access timestamp
  - Returns error only if lock acquisition fails

- `list_all_services() -> Result<Vec<DiscoveredService>>`
  - Get snapshot of all services
  - Read-only operation (RwLock::read)

- `clear() -> Result<()>`
  - Remove all services and timestamps
  - Atomic operation (both maps cleared together)

- `count() -> Result<usize>`
  - Get current number of services

**Internal Structure**:
```rust
pub struct ServiceRegistry {
    services: Arc<RwLock<HashMap<ServiceId, DiscoveredService>>>,
    last_access: Arc<RwLock<HashMap<ServiceId, SystemTime>>>,
}
```

**Thread Safety**:
- `Arc<RwLock<T>>` for concurrent read access
- Multiple threads can read simultaneously
- Exclusive write access when adding/clearing

**Design Notes**:
- No Tauri dependencies - pure business logic
- Can be shared across application
- Future: TTL cleanup can be added without changing public API

---

### 3. Tauri Commands (`src/commands/get_devices.rs`)

**Responsibility**: Bridge between frontend and backend service discovery

#### Command: `get_devices()`

**Purpose**: Refresh service discovery by restarting the search

**Signature**:
```rust
#[tauri::command]
pub fn get_devices(
    client: State<'_, Mutex<MdnsClient>>,
    registry: State<'_, ServiceRegistry>,
) -> Result<(), String>
```

**Parameters**:
- `client`: Mutable state containing MdnsClient
- `registry`: Shared state containing ServiceRegistry

**Execution Flow**:
```
1. Acquire client lock
2. Call client.stop_discovery()
   └─> Stops browsing, emits DiscoveryStopped
3. Call registry.clear()
   └─> Removes all cached services
4. Call client.start_discovery()
   └─> Starts browsing, emits DiscoveryStarted
5. Return Ok(())
   └─> Command completes immediately (async listening handles events)
```

**Return Value**:
- `Ok(())` - Refresh cycle completed successfully
- `Err(String)` - Error message if any step failed

**Design Notes**:
- Synchronous command (blocks until refresh is initiated)
- Does NOT wait for services to be discovered
- Does NOT return service list
- Frontend receives updates via event listeners

**Error Handling**:
```rust
// Errors that can be returned:
- "Failed to lock client" - Mutex poisoning
- "Failed to stop discovery" - mDNS operation failed
- "Failed to clear registry" - Lock acquisition failed
- "Failed to start discovery" - mDNS operation failed
```

---

## Event System

### Event Listener Architecture

The event listener runs as a background Tauri task and bridges MdnsClient events to the frontend.

**Responsibility**:
- Listen to MdnsClient event channel
- Update registry with discovered services
- Emit frontend events via Tauri

**Conceptual Flow**:
```
MdnsClient
  ↓
Emits ServiceEvent
  ↓
Event Listener receives via receiver
  ↓
  ├─ ServiceEvent::ServiceAdded
  │   ├─ Call registry.add_service()
  │   └─ Emit "add_service" event to frontend
  │
  ├─ ServiceEvent::ServicesCleared
  │   └─ Emit "clear" event to frontend
  │
  └─ Other events (ignored for MVP)
```

### Frontend Events

**Event: "add_service"**
- **Payload**: `DiscoveredService`
- **Trigger**: When MdnsClient discovers a new service
- **Frontend Action**: Add service to list UI
- **Format**:
```typescript
interface DiscoveredService {
  id: string;
  name: string;
  service_type: string;
  hostname: string;
  addresses: string[];
  port: number;
  txt_records: Record<string, string>;
  priority: number;
  weight: number;
  discovered_at: string;  // ISO timestamp
  last_seen_at: string;   // ISO timestamp
}
```

**Event: "clear"**
- **Payload**: `null` (no data)
- **Trigger**: When registry is cleared via `get_devices` command
- **Frontend Action**: Clear all services from list UI

---

## Data Flow Scenarios

### Scenario 1: Initial Service Discovery

```
User opens app
  ↓
main.rs: Initialize MdnsClient
  ↓
main.rs: Start event listener task
  ↓
MdnsClient.start_discovery()
  ↓
mDNS network
  ↓
Service found: "My Printer._ipp._tcp.local."
  ↓
MdnsEvent::ServiceResolved
  ↓
Resolver.resolve() → DiscoveredService
  ↓
ServiceEvent::ServiceAdded { service }
  ↓
Event Listener receives event
  ├─ registry.add_service(service)
  └─ app.emit("add_service", service)
  ↓
Frontend: listen("add_service")
  ↓
UI: Add service to list
```

### Scenario 2: Refresh Service List

```
User clicks "Refresh" button
  ↓
Frontend: invoke("get_devices")
  ↓
Command: get_devices()
  ├─ client.stop_discovery()
  ├─ registry.clear()
  └─ client.start_discovery()
  ↓
Event Listener receives multiple events:
  ├─ DiscoveryStopped (ignored in MVP)
  ├─ DiscoveryStarted (ignored in MVP)
  └─ ServiceAdded events (for each service found)
  ↓
Frontend: listen("clear") [from clear() call]
  ↓
UI: Clear list
  ↓
Frontend: listen("add_service") [multiple times]
  ↓
UI: Repopulate list with new services
```

---

## Architecture Principles

### 1. Separation of Concerns

```
┌─────────────────────────────────────────────┐
│ Tauri Frontend (UI Layer)                   │
│ - Handles user interaction                  │
│ - Manages UI state                          │
│ - No business logic                         │
└─────────────────────────────────────────────┘
                    ↑↓ (Tauri IPC)
┌─────────────────────────────────────────────┐
│ Tauri Backend (Integration Layer)           │
│ - Commands (get_devices)                    │
│ - Event listening                           │
│ - State management (AppHandle)              │
│ - No Tauri deps in business logic           │
└─────────────────────────────────────────────┘
                    ↑↓
┌─────────────────────────────────────────────┐
│ Business Logic Layer (Core)                 │
│ - MdnsClient                                │
│ - ServiceRegistry                           │
│ - Models (ServiceEvent, DiscoveredService) │
│ - NO Tauri dependencies                     │
│ - Completely portable                       │
└─────────────────────────────────────────────┘
```

### 2. No Circular Dependencies

- MdnsClient emits events but doesn't know about Tauri
- ServiceRegistry stores data but doesn't know about Tauri
- Tauri integration only at the outermost layer

### 3. Portability to gRPC

The architecture supports easy migration to gRPC without changing core logic:

```
Current (Tauri):
MdnsClient → Receiver → Event Listener → app.emit()

With gRPC:
MdnsClient → Receiver → Event Listener → grpc_stream.send()
```

Only the event listener implementation would change; MdnsClient and ServiceRegistry remain identical.

---

## Module Structure

```
src/
├── commands/
│   ├── mod.rs              # Commands module
│   └── get_devices.rs      # get_devices command implementation
│
├── mdns/
│   ├── mod.rs              # mDNS module
│   ├── client.rs           # MdnsClient
│   ├── resolver.rs         # ServiceInfo → DiscoveredService conversion
│   └── parser.rs           # TXT record parsing
│
├── service/
│   ├── mod.rs              # Service module
│   └── registry.rs         # ServiceRegistry
│
├── models/
│   ├── mod.rs              # Models module
│   ├── service.rs          # ServiceEvent, DiscoveredService
│   ├── config.rs           # DiscoveryConfig
│   └── error.rs            # Error types
│
├── error.rs                # Error handling
└── lib.rs                  # Root module (exports public API)
```

---

## Public API

### From `lib.rs`:

```rust
// Structs
pub use mdns::MdnsClient;
pub use service::ServiceRegistry;
pub use models::{DiscoveredService, DiscoveryConfig, ServiceEvent, ServiceId};

// Error types
pub use error::{AppError, AppResult};
```

### Commands (Tauri):

```rust
#[tauri::command]
pub fn get_devices(
    client: State<'_, Mutex<MdnsClient>>,
    registry: State<'_, ServiceRegistry>,
) -> Result<(), String>
```

### Events (Tauri):

```typescript
// Emitted from backend to frontend
listen("add_service", (service: DiscoveredService) => { ... })
listen("clear", () => { ... })
```

---

## State Management

### Tauri Application State

In `main.rs`:
```rust
tauri::Builder::default()
    .manage(Mutex::new(client))  // MdnsClient
    .manage(Arc::new(registry))  // ServiceRegistry
    .invoke_handler(tauri::generate_handler![get_devices])
    // ... event listener setup
    .run(...)
```

**Why Mutex for MdnsClient**:
- MdnsClient has mutable state (daemon, event handler)
- Only one mutable access at a time is safe
- Multiple concurrent start/stop calls must be serialized

**Why Arc for ServiceRegistry**:
- ServiceRegistry is inherently thread-safe (uses RwLock internally)
- Multiple tasks/commands can access simultaneously
- Arc allows cheap cloning

---

## Error Handling Strategy

### Error Types

1. **AppError::MdnsError** - mDNS protocol failures
2. **AppError::InvalidData** - Configuration or data validation
3. **AppError::InternalError** - Lock poisoning, system issues

### Error Propagation

```
Command layer (Tauri) → Result<_, String>
  ↓
Maps AppError → String for frontend consumption
  ↓
Frontend displays error to user
```

### Example Error Flow

```rust
registry.clear()
  ↓ (RwLock::write fails)
  ↓
AppError::internal("Failed to acquire lock")
  ↓ (converted to String in command)
  ↓
"Failed to clear registry: Failed to acquire lock"
  ↓ (returned to frontend)
  ↓
Frontend shows error message
```

---

## Design Decisions & Rationale

### 1. Why No ServiceManager?

- **Original Design**: ServiceManager as coordinator
- **Issue**: Adds abstraction layer without value in MVP
- **Decision**: Remove ServiceManager, keep only ServiceRegistry
- **Benefit**: Simpler architecture, less coupling
- **Trade-off**: If future needs arise (TTL cleanup, persistence), ServiceManager can be re-added

### 2. Why Registry, Not Just Return Services?

- **Alternative**: Commands return Vec<DiscoveredService>
- **Decision**: Use persistent registry updated by event listener
- **Benefit**:
  - Services remain available between discovery cycles
  - Can query at any time without re-scanning
  - Supports future pagination/filtering

### 3. Why Event Listener, Not Synchronous Processing?

- **Alternative**: Block on discovery wait
- **Decision**: Async background task emits events
- **Benefit**:
  - Frontend responsive (no blocking on commands)
  - Real-time updates as services are discovered
  - Can support multiple concurrent discoveries

### 4. Why emit("clear") Instead of registry.clear()?

- **Question**: When should "clear" event be sent?
- **Decision**: Event listener monitors registry changes
- **Benefit**: Frontend gets explicit notification of state changes
- **Current**: Manually in command (future: could be automatic)

---

## Testing Strategy

### Unit Tests (55 tests)
- MdnsClient: lifecycle, receiver cloning
- ServiceRegistry: add, clear, concurrent operations
- Models: serialization, validation
- Error handling: error creation, conversion

### Integration Tests (Not yet)
- Full discovery cycle with real mDNS
- Command execution with state management

### Doc Tests (6 tests)
- API examples in documentation

---

## Future Extensibility

### 1. TTL Cleanup

```rust
// In ServiceManager (to be re-added)
pub fn start_cleanup_task(ttl_secs: u64) -> JoinHandle {
    // Periodically check last_access timestamps
    // Remove expired services
    // Emit "service_expired" events
}
```

### 2. gRPC Server

```rust
// Would only change this layer:
// Business logic (MdnsClient, ServiceRegistry) unchanged

pub async fn start_grpc_server(client, registry) {
    // Listen on port
    // For each client connection:
    //   - Subscribe to receiver
    //   - Send events via gRPC stream
}
```

### 3. Service Filtering

```rust
// In ServiceRegistry:
pub fn query_services(filter: ServiceFilter) -> Result<Vec<DiscoveredService>> {
    // Filter by name, type, address, TXT records
}
```

### 4. Multi-Network Support

```rust
// In MdnsClient:
pub fn browse_network_interfaces() -> Result<Vec<Interface>> {
    // Discover available interfaces
}

pub fn browse_on_interface(iface: Interface) -> Result<()> {
    // Separate discovery per interface
}
```

---

## Performance Considerations

### Memory
- **Registry**: O(n) where n = number of services
- **Event Channel**: Bounded to 100 messages
- **Typical**: < 50MB for 1000+ services

### CPU
- **Receiver polling**: Yields when empty (minimal CPU)
- **Lock contention**: Low (quick operations)
- **Network**: Background listening (no polling)

### Concurrency
- **RwLock allows**: Multiple concurrent reads
- **Mutex serializes**: MdnsClient operations
- **No deadlocks**: Simple lock ordering

---

## Summary

This architecture provides:
1. ✅ **Clear separation** between UI, integration, and business logic
2. ✅ **High portability** to other backends (gRPC, HTTP, etc.)
3. ✅ **Type safety** with Rust and strong error handling
4. ✅ **Concurrency safety** with Arc, RwLock, and Mutex
5. ✅ **Responsive UI** with async event-driven updates
6. ✅ **Minimal MVP** without unnecessary abstractions
7. ✅ **Room to grow** with clear extension points

The design prioritizes **clarity** and **portability** while keeping the MVP **simple**.
