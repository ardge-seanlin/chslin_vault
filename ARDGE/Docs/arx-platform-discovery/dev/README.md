# arx_mdns Package

Enterprise-grade mDNS (Multicast DNS) service registration and discovery package implementing RFC 6762/6763.

## Features

- **Advertiser**: Register and broadcast services on the local network via mDNS
  - Dynamic TXT record updates without service restart
  - Automatic hostname and IP resolution
  - Support for multiple network interfaces
  - Context-aware lifecycle management

- **Scanner**: Discover services advertised via mDNS
  - One-time scanning with configurable timeout
  - Continuous streaming discovery
  - Automatic device information parsing

- **TXT Records**: Encode/decode device metadata in standard key=value format
  - Automatic validation of record sizes (RFC 6763 compliance)
  - Support for common device attributes (ID, model, version, etc.)
  - IPv4 address management

## Installation

```bash
go get github.com/ardge-labs/arx-platform-discover/pkg/arx_mdns
```

## Quick Start

### Advertising a Service

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "os"
    "os/signal"
    "time"

    v1 "github.com/ardge-labs/arx-platform-discover/api/arx/node/v1"
    "github.com/ardge-labs/arx-platform-discover/pkg/arx_mdns"
)

func main() {
    // Create logger
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    // Configure advertiser
    cfg := &arx_mdns.AdvertiserConfig{
        ServiceName: "_arx-edge._tcp",
        ServicePort: 8080,
        Domain:      "local.",
        TTL:         120, // 2 minutes
    }

    // Define device information
    device := &v1.EdgeDevice{
        Hostname:        "edge-device-01",
        DeviceId:        "edge-001",
        ModelName:       "Jetson Xavier NX",
        BiosVersion:     "1.0.0",
        PlatformVersion: "Ubuntu 20.04",
        ApiPort:         8080,
        TlsSupport:      true,
        CpuCores:        6,
        Memory:          8589934592, // 8GB
        Interfaces: []*v1.NetworkInterface{
            {
                Name: "eth0",
                Ipv4: "192.168.1.100",
                Role: v1.NetworkInterfaceRole_NETWORK_INTERFACE_ROLE_PRIMARY,
            },
        },
    }

    // Create advertiser
    advertiser, err := arx_mdns.NewAdvertiser(cfg, device, logger)
    if err != nil {
        log.Fatalf("Failed to create advertiser: %v", err)
    }

    // Start advertising
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := advertiser.Start(ctx); err != nil {
        log.Fatalf("Failed to start advertiser: %v", err)
    }
    defer advertiser.Stop()

    log.Println("Service advertising started. Press Ctrl+C to stop.")

    // Wait for interrupt signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Shutting down...")
}
```

### Discovering Services

```go
package main

import (
    "context"
    "errors"
    "log"
    "log/slog"
    "os"
    "time"

    "github.com/ardge-labs/arx-platform-discover/pkg/arx_mdns"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    // Create scanner
    scanner, err := arx_mdns.NewScanner(
        "_arx-edge._tcp",
        "local.",
        3*time.Second,
        logger,
    )
    if err != nil {
        log.Fatalf("Failed to create scanner: %v", err)
    }

    // Perform single scan
    ctx := context.Background()
    devices, err := scanner.Scan(ctx)
    if err != nil && !errors.Is(err, arx_mdns.ErrScanTimeout) {
        log.Fatalf("Scan failed: %v", err)
    }

    log.Printf("Found %d devices:", len(devices))
    for _, device := range devices {
        log.Printf("  - %s (%s) - Model: %s, IP: %s",
            device.Hostname,
            device.DeviceId,
            device.ModelName,
            getPrimaryIP(device.Interfaces),
        )
    }
}

func getPrimaryIP(interfaces []*v1.NetworkInterface) string {
    if len(interfaces) > 0 {
        return interfaces[0].Ipv4
    }
    return "N/A"
}
```

### Continuous Discovery (Streaming)

```go
package main

import (
    "context"
    "errors"
    "log"
    "log/slog"
    "os"
    "time"

    "github.com/ardge-labs/arx-platform-discover/pkg/arx_mdns"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    scanner, err := arx_mdns.NewScanner(
        "_arx-edge._tcp",
        "local.",
        2*time.Second,
        logger,
    )
    if err != nil {
        log.Fatalf("Failed to create scanner: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    resultCh := make(chan arx_mdns.ScanResult, 10)

    // Start continuous scanning
    go scanner.StreamScan(ctx, 5*time.Second, resultCh)

    // Process results
    for {
        select {
        case result := <-resultCh:
            if result.Error != nil {
                if !errors.Is(result.Error, arx_mdns.ErrScanTimeout) {
                    log.Printf("Scan error: %v", result.Error)
                }
                continue
            }

            log.Printf("Discovered %d devices:", len(result.Devices))
            for _, device := range result.Devices {
                log.Printf("  - %s (%s)", device.Hostname, device.DeviceId)
            }

        case <-ctx.Done():
            log.Println("Scan completed")
            return
        }
    }
}
```

### Dynamic Device Updates

```go
// Update device information without restarting the service
updatedDevice := &v1.EdgeDevice{
    Hostname:        "edge-device-01",
    DeviceId:        "edge-001",
    ModelName:       "Jetson Xavier NX",
    BiosVersion:     "1.1.0",  // Updated version
    PlatformVersion: "Ubuntu 22.04",  // Updated OS
    ApiPort:         8080,
    TlsSupport:      true,
    CpuCores:        6,
    Memory:          8589934592,
}

if err := advertiser.UpdateDevice(updatedDevice); err != nil {
    log.Printf("Failed to update device: %v", err)
}
```

## Configuration

### AdvertiserConfig

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| ServiceName | string | mDNS service name (e.g., "_arx-edge._tcp") | Yes |
| ServicePort | int | Port number (1-65535) | Yes |
| Domain | string | mDNS domain (default: "local.") | No |
| TTL | uint32 | Time-to-live for records (default: 120s) | No |
| Interfaces | []string | Network interfaces to advertise on | No |

### ScannerConfig

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| ServiceName | string | mDNS service name to search for | Yes |
| Domain | string | mDNS domain (default: "local.") | No |
| ScanTimeout | time.Duration | Timeout for each scan | Yes |

## Error Handling

The package provides comprehensive error handling with custom error types:

### Sentinel Errors

- `ErrScanTimeout`: Scan timeout reached (expected behavior)
- `ErrAlreadyStarted`: Advertiser already running
- `ErrNotStarted`: Advertiser not started
- `ErrInvalidConfig`: Configuration validation failed
- `ErrInvalidDevice`: Device validation failed
- `ErrInvalidTXTRecord`: TXT record parsing failed

### Custom Error Types

- `ValidationError`: Configuration or input validation errors
- `TXTRecordError`: TXT record encoding/decoding errors

### Error Checking Examples

```go
// Check for specific error
if errors.Is(err, arx_mdns.ErrScanTimeout) {
    // Timeout is expected, handle gracefully
}

// Check for error type
var validationErr *arx_mdns.ValidationError
if errors.As(err, &validationErr) {
    log.Printf("Validation failed for field %s: %s",
        validationErr.Field,
        validationErr.Message)
}
```

## Best Practices

### 1. Always Use Context

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := advertiser.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### 2. Validate Configuration Early

```go
// Validation happens automatically in constructors
advertiser, err := arx_mdns.NewAdvertiser(cfg, device, logger)
if err != nil {
    // Handle validation error
    log.Fatalf("Invalid configuration: %v", err)
}
```

### 3. Handle Timeouts Gracefully

```go
devices, err := scanner.Scan(ctx)
if err != nil {
    if errors.Is(err, arx_mdns.ErrScanTimeout) {
        // Partial results are still returned
        log.Printf("Scan timed out, found %d devices", len(devices))
    } else {
        // Actual error occurred
        log.Fatalf("Scan failed: %v", err)
    }
}
```

### 4. Use Structured Logging

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

advertiser, _ := arx_mdns.NewAdvertiser(cfg, device, logger)
```

### 5. Clean Shutdown

```go
defer func() {
    if err := advertiser.Stop(); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
}()
```

## Thread Safety

All public methods are thread-safe and can be called concurrently:

- `Advertiser.Start()`, `Stop()`, `UpdateDevice()` are protected by mutexes
- `Scanner.Scan()` and `StreamScan()` can run concurrently
- Test with `go test -race` for race condition detection

## Performance Considerations

- **Scan Timeout**: Shorter timeouts improve responsiveness but may miss devices
- **Stream Interval**: Balance between discovery latency and network overhead
- **TXT Record Size**: Limited to 255 bytes per record (RFC 6763)
- **Network Interfaces**: Specifying interfaces reduces multicast traffic

## Limitations

- IPv6 support is pending (currently IPv4 only)
- Maximum TXT record size: 255 bytes
- Service name must contain "_tcp" or "_udp"
- Port range: 1-65535

## License

Proprietary - Ardge Labs

## Contributing

Internal package - contact the platform team for contributions or issues.
