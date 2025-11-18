# ARX Platform Discover - Architecture Design

  

## Overview

  

This document describes the architecture design for a discovery platform with two distinct server components:

  

1. **mDNS Broadcast Server (Agent)** - Broadcasts device information via mDNS for discovery on local network

2. **gRPC Server (Discover)** - Provides API interface for clients to browse/query discovered devices

  

## System Architecture

  

```

┌─────────────────┐ mDNS Broadcast ┌─────────────────┐

│ Edge Device 1 │ ◄───────────────────► │ Discover Server │

│ (Agent) │ │ │

│ ┌───────────┐ │ │ ┌──────────┐ │

│ │Advertiser │ │ │ │ Scanner │ │

│ └───────────┘ │ │ └────┬─────┘ │

└─────────────────┘ │ │ │

│ ▼ │

┌─────────────────┐ │ ┌──────────┐ │

│ Edge Device 2 │ │ │Discovery │ │

│ (Agent) │ │ │ Service │ │

└─────────────────┘ │ └────┬─────┘ │

│ │ │

│ ▼ │

│ ┌──────────┐ │

│ │ gRPC │ │

│ │ Server │ │

│ └────┬─────┘ │

└───────┼─────────┘

│

▼

┌─────────────────┐

│ gRPC Client │

└─────────────────┘

```

  

## Directory Structure

  

```

arx-platform-discover/

├── api/

│ └── core/v1/ # Generated proto files (existing)

│ ├── discovery.pb.go

│ └── discovery_grpc.pb.go

├── cmd/

│ ├── agent/ # mDNS broadcast agent (edge device)

│ │ └── main.go

│ └── discover/ # Discovery server (gRPC + mDNS scanner)

│ └── main.go

├── internal/

│ ├── config/ # Shared configuration

│ │ ├── config.go

│ │ └── config_test.go

│ ├── mdns/ # mDNS infrastructure

│ │ ├── advertiser.go # mDNS broadcast server

│ │ ├── advertiser_test.go

│ │ ├── scanner.go # mDNS device scanner

│ │ ├── scanner_test.go

│ │ ├── txtrecord.go # TXT record encoding/decoding

│ │ └── txtrecord_test.go

│ ├── device/ # Device information collection

│ │ ├── info.go # System information gathering

│ │ └── info_test.go

│ ├── grpcserver/ # gRPC server infrastructure

│ │ ├── server.go # Server setup and lifecycle

│ │ ├── server_test.go

│ │ ├── interceptors.go # Middleware (logging, recovery, metrics)

│ │ └── interceptors_test.go

│ ├── service/ # gRPC service implementations

│ │ ├── discovery.go # DiscoveryService implementation

│ │ └── discovery_test.go

│ └── app/ # Application orchestration

│ ├── agent.go # Agent app (mDNS broadcaster)

│ ├── agent_test.go

│ ├── discover.go # Discover app (gRPC + scanner)

│ └── discover_test.go

├── pkg/ # Public packages (if needed)

│ └── errx/ # Error utilities

│ └── errors.go

├── go.mod

├── go.sum

├── Makefile

└── VERSION

```

  

## Core Components

  

### 1. Configuration Management

  

**File:** `internal/config/config.go`

  

```go

package config

  

import (

"time"

)

  

// AgentConfig holds configuration for mDNS broadcast agent

type AgentConfig struct {

// ServiceName is the mDNS service name (e.g., "_arx-edge._tcp")

ServiceName string

// ServicePort is the port advertised in mDNS

ServicePort int

// Domain is the mDNS domain (default: "local.")

Domain string

// TTL is the time-to-live for mDNS records

TTL time.Duration

// InterfaceName specifies which network interface to advertise on (empty for all)

InterfaceName string

}

  

// DiscoverConfig holds configuration for discovery server

type DiscoverConfig struct {

// GRPCAddress is the address for gRPC server (e.g., ":50051")

GRPCAddress string

// ServiceName is the mDNS service name to scan for

ServiceName string

// ScanInterval is the interval between mDNS scans

ScanInterval time.Duration

// DefaultTimeout is the default discovery timeout

DefaultTimeout time.Duration

// MaxConcurrentScans limits concurrent discovery operations

MaxConcurrentScans int

}

  

// DefaultAgentConfig returns default agent configuration

func DefaultAgentConfig() *AgentConfig {

return &AgentConfig{

ServiceName: "_arx-edge._tcp",

ServicePort: 8080,

Domain: "local.",

TTL: 120 * time.Second,

InterfaceName: "",

}

}

  

// DefaultDiscoverConfig returns default discover configuration

func DefaultDiscoverConfig() *DiscoverConfig {

return &DiscoverConfig{

GRPCAddress: ":50051",

ServiceName: "_arx-edge._tcp",

ScanInterval: 5 * time.Second,

DefaultTimeout: 30 * time.Second,

MaxConcurrentScans: 10,

}

}

```

  

### 2. mDNS TXT Record Encoding/Decoding

  

**File:** `internal/mdns/txtrecord.go`

  

```go

package mdns

  

import (

"fmt"

"strconv"

"strings"

  

v1 "github.com/ardge-labs/arx-platform-discover/api/core/v1"

)

  

// TXTRecordKeys defines the keys used in mDNS TXT records

const (

KeyDeviceID = "id"

KeyModelName = "model"

KeyBiosVersion = "bios"

KeyPlatformVersion = "platform"

KeyAPIPort = "port"

KeyTLSSupport = "tls"

KeyCPUCores = "cpu"

KeyMemory = "mem"

KeyInterfaces = "ifaces"

)

  

// EncodeTXTRecords encodes EdgeDevice information into mDNS TXT records

func EncodeTXTRecords(device *v1.EdgeDevice) []string {

records := []string{

fmt.Sprintf("%s=%s", KeyDeviceID, device.DeviceId),

fmt.Sprintf("%s=%s", KeyModelName, device.ModelName),

fmt.Sprintf("%s=%s", KeyBiosVersion, device.BiosVersion),

fmt.Sprintf("%s=%s", KeyPlatformVersion, device.PlatformVersion),

fmt.Sprintf("%s=%d", KeyAPIPort, device.ApiPort),

fmt.Sprintf("%s=%t", KeyTLSSupport, device.TlsSupport),

fmt.Sprintf("%s=%d", KeyCPUCores, device.CpuCores),

fmt.Sprintf("%s=%d", KeyMemory, device.Memory),

}

  

// Encode network interfaces

if len(device.Interfaces) > 0 {

ifaces := encodeInterfaces(device.Interfaces)

records = append(records, fmt.Sprintf("%s=%s", KeyInterfaces, ifaces))

}

  

return records

}

  

// DecodeTXTRecords decodes mDNS TXT records into EdgeDevice information

func DecodeTXTRecords(txtRecords []string, hostname string) (*v1.EdgeDevice, error) {

device := &v1.EdgeDevice{

Hostname: hostname,

}

  

for _, record := range txtRecords {

parts := strings.SplitN(record, "=", 2)

if len(parts) != 2 {

continue

}

  

key, value := parts[0], parts[1]

if err := decodeField(device, key, value); err != nil {

return nil, fmt.Errorf("decode field %s: %w", key, err)

}

}

  

return device, nil

}

  

func decodeField(device *v1.EdgeDevice, key, value string) error {

switch key {

case KeyDeviceID:

device.DeviceId = value

case KeyModelName:

device.ModelName = value

case KeyBiosVersion:

device.BiosVersion = value

case KeyPlatformVersion:

device.PlatformVersion = value

case KeyAPIPort:

port, err := strconv.ParseInt(value, 10, 32)

if err != nil {

return fmt.Errorf("parse api_port: %w", err)

}

device.ApiPort = int32(port)

case KeyTLSSupport:

device.TlsSupport = value == "true"

case KeyCPUCores:

cores, err := strconv.ParseInt(value, 10, 32)

if err != nil {

return fmt.Errorf("parse cpu_cores: %w", err)

}

device.CpuCores = int32(cores)

case KeyMemory:

mem, err := strconv.ParseInt(value, 10, 64)

if err != nil {

return fmt.Errorf("parse memory: %w", err)

}

device.Memory = mem

case KeyInterfaces:

ifaces, err := decodeInterfaces(value)

if err != nil {

return fmt.Errorf("parse interfaces: %w", err)

}

device.Interfaces = ifaces

}

return nil

}

  

func encodeInterfaces(ifaces []*v1.NetworkInterface) string {

var parts []string

for _, iface := range ifaces {

part := fmt.Sprintf("%s|%s|%s|%d", iface.Name, iface.Ipv4, iface.Mac, iface.Role)

parts = append(parts, part)

}

return strings.Join(parts, ";")

}

  

func decodeInterfaces(value string) ([]*v1.NetworkInterface, error) {

if value == "" {

return nil, nil

}

  

var interfaces []*v1.NetworkInterface

parts := strings.Split(value, ";")

  

for _, part := range parts {

fields := strings.Split(part, "|")

if len(fields) != 4 {

continue

}

  

role, err := strconv.ParseInt(fields[3], 10, 32)

if err != nil {

return nil, fmt.Errorf("parse interface role: %w", err)

}

  

iface := &v1.NetworkInterface{

Name: fields[0],

Ipv4: fields[1],

Mac: fields[2],

Role: v1.NetworkInterfaceRole(role),

}

interfaces = append(interfaces, iface)

}

  

return interfaces, nil

}

```

  

### 3. mDNS Advertiser (Broadcast Server)

  

**File:** `internal/mdns/advertiser.go`

  

```go

package mdns

  

import (

"context"

"fmt"

"log/slog"

"net"

"sync"

  

v1 "github.com/ardge-labs/arx-platform-discover/api/core/v1"

"github.com/ardge-labs/arx-platform-discover/internal/config"

"github.com/hashicorp/mdns"

)

  

// Advertiser broadcasts device information via mDNS

type Advertiser struct {

cfg *config.AgentConfig

device *v1.EdgeDevice

logger *slog.Logger

  

mu sync.RWMutex

server *mdns.Server

}

  

// NewAdvertiser creates a new mDNS advertiser

func NewAdvertiser(cfg *config.AgentConfig, device *v1.EdgeDevice, logger *slog.Logger) *Advertiser {

return &Advertiser{

cfg: cfg,

device: device,

logger: logger.With("component", "mdns-advertiser"),

}

}

  

// Start begins mDNS advertisement

func (a *Advertiser) Start(ctx context.Context) error {

a.mu.Lock()

defer a.mu.Unlock()

  

if a.server != nil {

return fmt.Errorf("advertiser already running")

}

  

// Build TXT records from device information

txtRecords := EncodeTXTRecords(a.device)

  

// Get network interface for advertisement

var iface *net.Interface

if a.cfg.InterfaceName != "" {

var err error

iface, err = net.InterfaceByName(a.cfg.InterfaceName)

if err != nil {

return fmt.Errorf("get interface %s: %w", a.cfg.InterfaceName, err)

}

}

  

// Create mDNS service configuration

serviceConfig := &mdns.MDNSService{

Instance: a.device.Hostname,

Service: a.cfg.ServiceName,

Domain: a.cfg.Domain,

Port: a.cfg.ServicePort,

Info: txtRecords,

}

  

// Create mDNS server configuration

serverConfig := &mdns.Config{

Zone: serviceConfig,

}

  

if iface != nil {

serverConfig.Iface = iface

}

  

// Start mDNS server

server, err := mdns.NewServer(serverConfig)

if err != nil {

return fmt.Errorf("create mdns server: %w", err)

}

  

a.server = server

a.logger.Info("mDNS advertisement started",

"service", a.cfg.ServiceName,

"port", a.cfg.ServicePort,

"hostname", a.device.Hostname,

)

  

// Monitor context for shutdown

go func() {

<-ctx.Done()

a.Stop()

}()

  

return nil

}

  

// Stop halts mDNS advertisement

func (a *Advertiser) Stop() error {

a.mu.Lock()

defer a.mu.Unlock()

  

if a.server == nil {

return nil

}

  

if err := a.server.Shutdown(); err != nil {

a.logger.Error("failed to shutdown mdns server", "error", err)

return fmt.Errorf("shutdown mdns server: %w", err)

}

  

a.server = nil

a.logger.Info("mDNS advertisement stopped")

return nil

}

  

// UpdateDevice updates the device information being advertised

func (a *Advertiser) UpdateDevice(device *v1.EdgeDevice) error {

a.mu.Lock()

defer a.mu.Unlock()

  

a.device = device

  

// If server is running, restart with new information

if a.server != nil {

if err := a.server.Shutdown(); err != nil {

return fmt.Errorf("shutdown for update: %w", err)

}

a.server = nil

  

// Re-create with updated device info

txtRecords := EncodeTXTRecords(a.device)

serviceConfig := &mdns.MDNSService{

Instance: a.device.Hostname,

Service: a.cfg.ServiceName,

Domain: a.cfg.Domain,

Port: a.cfg.ServicePort,

Info: txtRecords,

}

  

serverConfig := &mdns.Config{

Zone: serviceConfig,

}

  

server, err := mdns.NewServer(serverConfig)

if err != nil {

return fmt.Errorf("recreate mdns server: %w", err)

}

a.server = server

}

  

a.logger.Info("device information updated", "device_id", device.DeviceId)

return nil

}

```

  

### 4. mDNS Scanner

  

**File:** `internal/mdns/scanner.go`

  

```go

package mdns

  

import (

"context"

"fmt"

"log/slog"

"sync"

"time"

  

v1 "github.com/ardge-labs/arx-platform-discover/api/core/v1"

"github.com/hashicorp/mdns"

)

  

// Scanner discovers devices via mDNS

type Scanner struct {

serviceName string

logger *slog.Logger

}

  

// NewScanner creates a new mDNS scanner

func NewScanner(serviceName string, logger *slog.Logger) *Scanner {

return &Scanner{

serviceName: serviceName,

logger: logger.With("component", "mdns-scanner"),

}

}

  

// Scan performs a single mDNS scan for devices

func (s *Scanner) Scan(ctx context.Context, timeout time.Duration) ([]*v1.EdgeDevice, error) {

entriesCh := make(chan *mdns.ServiceEntry, 16)

defer close(entriesCh)

  

var devices []*v1.EdgeDevice

var mu sync.Mutex

  

// Goroutine to process discovered entries

go func() {

for entry := range entriesCh {

device, err := s.parseEntry(entry)

if err != nil {

s.logger.Warn("failed to parse mdns entry",

"host", entry.Host,

"error", err,

)

continue

}

  

mu.Lock()

devices = append(devices, device)

mu.Unlock()

  

s.logger.Debug("discovered device",

"device_id", device.DeviceId,

"hostname", device.Hostname,

"addr", entry.AddrV4,

)

}

}()

  

// Configure and perform mDNS query

params := mdns.DefaultParams(s.serviceName)

params.Entries = entriesCh

params.Timeout = timeout

params.DisableIPv6 = true

  

if err := mdns.Query(params); err != nil {

return nil, fmt.Errorf("mdns query: %w", err)

}

  

return devices, nil

}

  

// StreamScan performs continuous scanning and streams results

func (s *Scanner) StreamScan(ctx context.Context, interval time.Duration, resultCh chan<- []*v1.EdgeDevice) {

ticker := time.NewTicker(interval)

defer ticker.Stop()

  

// Perform initial scan

if devices, err := s.Scan(ctx, 2*time.Second); err == nil && len(devices) > 0 {

select {

case resultCh <- devices:

case <-ctx.Done():

return

}

}

  

for {

select {

case <-ctx.Done():

return

case <-ticker.C:

devices, err := s.Scan(ctx, 2*time.Second)

if err != nil {

s.logger.Error("scan failed", "error", err)

continue

}

if len(devices) > 0 {

select {

case resultCh <- devices:

case <-ctx.Done():

return

}

}

}

}

}

  

func (s *Scanner) parseEntry(entry *mdns.ServiceEntry) (*v1.EdgeDevice, error) {

// Decode TXT records into device information

device, err := DecodeTXTRecords(entry.InfoFields, entry.Host)

if err != nil {

return nil, fmt.Errorf("decode txt records: %w", err)

}

  

// Set hostname from service entry if not in TXT records

if device.Hostname == "" {

device.Hostname = entry.Host

}

  

// Add discovered IP to interfaces if not already present

if entry.AddrV4 != nil {

found := false

for _, iface := range device.Interfaces {

if iface.Ipv4 == entry.AddrV4.String() {

found = true

break

}

}

if !found {

device.Interfaces = append(device.Interfaces, &v1.NetworkInterface{

Name: "mdns",

Ipv4: entry.AddrV4.String(),

Role: v1.NetworkInterfaceRole_NETWORK_INTERFACE_ROLE_PRIMARY,

})

}

}

  

return device, nil

}

```

  

### 5. gRPC Server Interceptors

  

**File:** `internal/grpcserver/interceptors.go`

  

```go

package grpcserver

  

import (

"context"

"log/slog"

"runtime/debug"

"time"

  

"google.golang.org/grpc"

"google.golang.org/grpc/codes"

"google.golang.org/grpc/status"

)

  

// UnaryLoggingInterceptor logs unary RPC calls

func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {

return func(

ctx context.Context,

req interface{},

info *grpc.UnaryServerInfo,

handler grpc.UnaryHandler,

) (interface{}, error) {

start := time.Now()

  

resp, err := handler(ctx, req)

  

duration := time.Since(start)

code := status.Code(err)

  

logger.Info("unary rpc completed",

"method", info.FullMethod,

"code", code.String(),

"duration", duration,

)

  

if err != nil {

logger.Error("unary rpc error",

"method", info.FullMethod,

"error", err,

)

}

  

return resp, err

}

}

  

// StreamLoggingInterceptor logs stream RPC calls

func StreamLoggingInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {

return func(

srv interface{},

ss grpc.ServerStream,

info *grpc.StreamServerInfo,

handler grpc.StreamHandler,

) error {

start := time.Now()

  

logger.Info("stream rpc started",

"method", info.FullMethod,

)

  

err := handler(srv, ss)

  

duration := time.Since(start)

code := status.Code(err)

  

logger.Info("stream rpc completed",

"method", info.FullMethod,

"code", code.String(),

"duration", duration,

)

  

if err != nil && code != codes.OK {

logger.Error("stream rpc error",

"method", info.FullMethod,

"error", err,

)

}

  

return err

}

}

  

// UnaryRecoveryInterceptor recovers from panics in unary RPCs

func UnaryRecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {

return func(

ctx context.Context,

req interface{},

info *grpc.UnaryServerInfo,

handler grpc.UnaryHandler,

) (resp interface{}, err error) {

defer func() {

if r := recover(); r != nil {

logger.Error("panic recovered in unary rpc",

"method", info.FullMethod,

"panic", r,

"stack", string(debug.Stack()),

)

err = status.Errorf(codes.Internal, "internal server error")

}

}()

  

return handler(ctx, req)

}

}

  

// StreamRecoveryInterceptor recovers from panics in stream RPCs

func StreamRecoveryInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {

return func(

srv interface{},

ss grpc.ServerStream,

info *grpc.StreamServerInfo,

handler grpc.StreamHandler,

) (err error) {

defer func() {

if r := recover(); r != nil {

logger.Error("panic recovered in stream rpc",

"method", info.FullMethod,

"panic", r,

"stack", string(debug.Stack()),

)

err = status.Errorf(codes.Internal, "internal server error")

}

}()

  

return handler(srv, ss)

}

}

```

  

### 6. gRPC Server Setup

  

**File:** `internal/grpcserver/server.go`

  

```go

package grpcserver

  

import (

"context"

"fmt"

"log/slog"

"net"

  

"google.golang.org/grpc"

"google.golang.org/grpc/health"

"google.golang.org/grpc/health/grpc_health_v1"

"google.golang.org/grpc/reflection"

)

  

// Server wraps gRPC server with lifecycle management

type Server struct {

grpcServer *grpc.Server

healthServer *health.Server

listener net.Listener

logger *slog.Logger

address string

}

  

// NewServer creates a new gRPC server with interceptors

func NewServer(address string, logger *slog.Logger) *Server {

log := logger.With("component", "grpc-server")

  

// Configure server options with interceptors

opts := []grpc.ServerOption{

grpc.ChainUnaryInterceptor(

UnaryRecoveryInterceptor(log),

UnaryLoggingInterceptor(log),

),

grpc.ChainStreamInterceptor(

StreamRecoveryInterceptor(log),

StreamLoggingInterceptor(log),

),

}

  

grpcServer := grpc.NewServer(opts...)

healthServer := health.NewServer()

  

// Register health service

grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

  

// Enable reflection for debugging

reflection.Register(grpcServer)

  

return &Server{

grpcServer: grpcServer,

healthServer: healthServer,

logger: log,

address: address,

}

}

  

// GRPCServer returns the underlying gRPC server for service registration

func (s *Server) GRPCServer() *grpc.Server {

return s.grpcServer

}

  

// SetServingStatus sets the health status for a service

func (s *Server) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {

s.healthServer.SetServingStatus(service, status)

}

  

// Start begins listening and serving gRPC requests

func (s *Server) Start(ctx context.Context) error {

listener, err := net.Listen("tcp", s.address)

if err != nil {

return fmt.Errorf("listen on %s: %w", s.address, err)

}

s.listener = listener

  

s.logger.Info("gRPC server starting", "address", s.address)

  

// Set overall server status to serving

s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

  

// Start server in goroutine

errCh := make(chan error, 1)

go func() {

if err := s.grpcServer.Serve(listener); err != nil {

errCh <- err

}

close(errCh)

}()

  

// Monitor for shutdown or error

select {

case <-ctx.Done():

s.Stop()

return ctx.Err()

case err := <-errCh:

if err != nil {

return fmt.Errorf("grpc serve: %w", err)

}

return nil

}

}

  

// Stop gracefully shuts down the gRPC server

func (s *Server) Stop() {

s.logger.Info("gRPC server stopping")

  

// Set status to not serving

s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

  

// Graceful stop with timeout

s.grpcServer.GracefulStop()

  

if s.listener != nil {

s.listener.Close()

}

  

s.logger.Info("gRPC server stopped")

}

  

// Address returns the server's listening address

func (s *Server) Address() string {

if s.listener != nil {

return s.listener.Addr().String()

}

return s.address

}

```

  

### 7. DiscoveryService Implementation

  

**File:** `internal/service/discovery.go`

  

```go

package service

  

import (

"context"

"fmt"

"log/slog"

"time"

  

v1 "github.com/ardge-labs/arx-platform-discover/api/core/v1"

"github.com/ardge-labs/arx-platform-discover/internal/mdns"

"google.golang.org/grpc"

)

  

// DiscoveryService implements the gRPC DiscoveryService

type DiscoveryService struct {

v1.UnimplementedDiscoveryServiceServer

  

scanner *mdns.Scanner

defaultTimeout time.Duration

maxTimeout time.Duration

logger *slog.Logger

}

  

// NewDiscoveryService creates a new discovery service

func NewDiscoveryService(scanner *mdns.Scanner, defaultTimeout, maxTimeout time.Duration, logger *slog.Logger) *DiscoveryService {

return &DiscoveryService{

scanner: scanner,

defaultTimeout: defaultTimeout,

maxTimeout: maxTimeout,

logger: logger.With("component", "discovery-service"),

}

}

  

// Discover implements the server-streaming Discover RPC

func (s *DiscoveryService) Discover(req *v1.DiscoverRequest, stream grpc.ServerStreamingServer[v1.DiscoverResponse]) error {

// Determine timeout

timeout := s.defaultTimeout

if req.Timeout != nil {

timeout = req.Timeout.AsDuration()

if timeout > s.maxTimeout {

timeout = s.maxTimeout

}

if timeout <= 0 {

timeout = s.defaultTimeout

}

}

  

s.logger.Info("discovery started", "timeout", timeout)

  

// Create context with timeout

ctx, cancel := context.WithTimeout(stream.Context(), timeout)

defer cancel()

  

// Create channel for scan results

resultCh := make(chan []*v1.EdgeDevice, 8)

  

// Start continuous scanning

go s.scanner.StreamScan(ctx, 2*time.Second, resultCh)

  

// Track discovered devices to avoid duplicates

discovered := make(map[string]*v1.EdgeDevice)

  

// Stream results back to client

for {

select {

case <-ctx.Done():

s.logger.Info("discovery completed", "devices_found", len(discovered))

return nil

case devices := <-resultCh:

if err := s.processAndSend(devices, discovered, stream); err != nil {

return fmt.Errorf("send discovery response: %w", err)

}

}

}

}

  

func (s *DiscoveryService) processAndSend(

devices []*v1.EdgeDevice,

discovered map[string]*v1.EdgeDevice,

stream grpc.ServerStreamingServer[v1.DiscoverResponse],

) error {

// Update discovered map and check for changes

hasChanges := false

for _, device := range devices {

if device.DeviceId == "" {

continue

}

  

existing, exists := discovered[device.DeviceId]

if !exists || !deviceEqual(existing, device) {

discovered[device.DeviceId] = device

hasChanges = true

s.logger.Debug("device discovered or updated",

"device_id", device.DeviceId,

"hostname", device.Hostname,

)

}

}

  

// Only send if there are changes

if hasChanges {

// Convert map to slice

deviceList := make([]*v1.EdgeDevice, 0, len(discovered))

for _, device := range discovered {

deviceList = append(deviceList, device)

}

  

resp := &v1.DiscoverResponse{

Devices: deviceList,

}

  

if err := stream.Send(resp); err != nil {

return fmt.Errorf("stream send: %w", err)

}

  

s.logger.Info("sent discovery update", "total_devices", len(deviceList))

}

  

return nil

}

  

// deviceEqual checks if two devices have the same information

func deviceEqual(a, b *v1.EdgeDevice) bool {

if a == nil || b == nil {

return a == b

}

  

return a.DeviceId == b.DeviceId &&

a.ModelName == b.ModelName &&

a.BiosVersion == b.BiosVersion &&

a.PlatformVersion == b.PlatformVersion &&

a.Hostname == b.Hostname &&

a.ApiPort == b.ApiPort &&

a.TlsSupport == b.TlsSupport &&

a.CpuCores == b.CpuCores &&

a.Memory == b.Memory

}

```

  

### 8. Device Information Collector

  

**File:** `internal/device/info.go`

  

```go

package device

  

import (

"fmt"

"net"

"os"

"runtime"

  

v1 "github.com/ardge-labs/arx-platform-discover/api/core/v1"

)

  

// InfoCollector gathers device information

type InfoCollector struct{}

  

// NewInfoCollector creates a new device information collector

func NewInfoCollector() *InfoCollector {

return &InfoCollector{}

}

  

// Collect gathers current device information

func (c *InfoCollector) Collect() (*v1.EdgeDevice, error) {

hostname, err := os.Hostname()

if err != nil {

return nil, fmt.Errorf("get hostname: %w", err)

}

  

interfaces, err := c.collectNetworkInterfaces()

if err != nil {

return nil, fmt.Errorf("collect network interfaces: %w", err)

}

  

device := &v1.EdgeDevice{

DeviceId: c.generateDeviceID(),

ModelName: c.getModelName(),

BiosVersion: c.getBiosVersion(),

PlatformVersion: c.getPlatformVersion(),

Hostname: hostname,

ApiPort: 8080, // Default API port

TlsSupport: false,

CpuCores: int32(runtime.NumCPU()),

Memory: c.getMemoryBytes(),

Interfaces: interfaces,

}

  

return device, nil

}

  

func (c *InfoCollector) collectNetworkInterfaces() ([]*v1.NetworkInterface, error) {

ifaces, err := net.Interfaces()

if err != nil {

return nil, fmt.Errorf("list interfaces: %w", err)

}

  

var result []*v1.NetworkInterface

primaryFound := false

  

for _, iface := range ifaces {

// Skip loopback and down interfaces

if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {

continue

}

  

addrs, err := iface.Addrs()

if err != nil {

continue

}

  

for _, addr := range addrs {

ipNet, ok := addr.(*net.IPNet)

if !ok {

continue

}

  

ip := ipNet.IP.To4()

if ip == nil {

continue // Skip IPv6

}

  

role := v1.NetworkInterfaceRole_NETWORK_INTERFACE_ROLE_SECONDARY

if !primaryFound {

role = v1.NetworkInterfaceRole_NETWORK_INTERFACE_ROLE_PRIMARY

primaryFound = true

}

  

netIface := &v1.NetworkInterface{

Name: iface.Name,

Ipv4: ip.String(),

Mac: iface.HardwareAddr.String(),

Role: role,

}

  

result = append(result, netIface)

}

}

  

return result, nil

}

  

func (c *InfoCollector) generateDeviceID() string {

// Generate device ID from MAC address or hostname

ifaces, err := net.Interfaces()

if err != nil {

hostname, _ := os.Hostname()

return fmt.Sprintf("device-%s", hostname)

}

  

for _, iface := range ifaces {

if iface.Flags&net.FlagLoopback == 0 && len(iface.HardwareAddr) > 0 {

return fmt.Sprintf("device-%s", iface.HardwareAddr.String())

}

}

  

hostname, _ := os.Hostname()

return fmt.Sprintf("device-%s", hostname)

}

  

func (c *InfoCollector) getModelName() string {

// Platform-specific implementation

return runtime.GOOS + "-" + runtime.GOARCH

}

  

func (c *InfoCollector) getBiosVersion() string {

// Platform-specific implementation

return "unknown"

}

  

func (c *InfoCollector) getPlatformVersion() string {

return runtime.Version()

}

  

func (c *InfoCollector) getMemoryBytes() int64 {

// Platform-specific implementation

// This is a simplified version

var m runtime.MemStats

runtime.ReadMemStats(&m)

return int64(m.Sys)

}

```

  

### 9. Application Orchestrator - Agent

  

**File:** `internal/app/agent.go`

  

```go

package app

  

import (

"context"

"fmt"

"log/slog"

"sync"

  

"github.com/ardge-labs/arx-platform-discover/internal/config"

"github.com/ardge-labs/arx-platform-discover/internal/device"

"github.com/ardge-labs/arx-platform-discover/internal/mdns"

)

  

// Agent orchestrates the mDNS broadcast agent

type Agent struct {

cfg *config.AgentConfig

logger *slog.Logger

advertiser *mdns.Advertiser

collector *device.InfoCollector

  

mu sync.Mutex

running bool

}

  

// NewAgent creates a new agent application

func NewAgent(cfg *config.AgentConfig, logger *slog.Logger) *Agent {

return &Agent{

cfg: cfg,

logger: logger.With("component", "agent"),

collector: device.NewInfoCollector(),

}

}

  

// Run starts the agent and blocks until context is cancelled

func (a *Agent) Run(ctx context.Context) error {

a.mu.Lock()

if a.running {

a.mu.Unlock()

return fmt.Errorf("agent already running")

}

a.running = true

a.mu.Unlock()

  

defer func() {

a.mu.Lock()

a.running = false

a.mu.Unlock()

}()

  

// Collect device information

deviceInfo, err := a.collector.Collect()

if err != nil {

return fmt.Errorf("collect device info: %w", err)

}

  

// Override API port from config

deviceInfo.ApiPort = int32(a.cfg.ServicePort)

  

a.logger.Info("device information collected",

"device_id", deviceInfo.DeviceId,

"hostname", deviceInfo.Hostname,

"interfaces", len(deviceInfo.Interfaces),

)

  

// Create and start advertiser

a.advertiser = mdns.NewAdvertiser(a.cfg, deviceInfo, a.logger)

if err := a.advertiser.Start(ctx); err != nil {

return fmt.Errorf("start advertiser: %w", err)

}

  

a.logger.Info("agent started successfully")

  

// Wait for shutdown signal

<-ctx.Done()

  

a.logger.Info("agent shutting down")

  

// Stop advertiser

if err := a.advertiser.Stop(); err != nil {

a.logger.Error("failed to stop advertiser", "error", err)

}

  

return nil

}

```

  

### 10. Application Orchestrator - Discover

  

**File:** `internal/app/discover.go`

  

```go

package app

  

import (

"context"

"fmt"

"log/slog"

"sync"

"time"

  

v1 "github.com/ardge-labs/arx-platform-discover/api/core/v1"

"github.com/ardge-labs/arx-platform-discover/internal/config"

"github.com/ardge-labs/arx-platform-discover/internal/grpcserver"

"github.com/ardge-labs/arx-platform-discover/internal/mdns"

"github.com/ardge-labs/arx-platform-discover/internal/service"

"google.golang.org/grpc/health/grpc_health_v1"

)

  

// Discover orchestrates the discovery server (gRPC + mDNS scanner)

type Discover struct {

cfg *config.DiscoverConfig

logger *slog.Logger

  

grpcServer *grpcserver.Server

scanner *mdns.Scanner

discoveryService *service.DiscoveryService

  

mu sync.Mutex

running bool

}

  

// NewDiscover creates a new discover application

func NewDiscover(cfg *config.DiscoverConfig, logger *slog.Logger) *Discover {

return &Discover{

cfg: cfg,

logger: logger.With("component", "discover"),

}

}

  

// Run starts the discover server and blocks until context is cancelled

func (d *Discover) Run(ctx context.Context) error {

d.mu.Lock()

if d.running {

d.mu.Unlock()

return fmt.Errorf("discover already running")

}

d.running = true

d.mu.Unlock()

  

defer func() {

d.mu.Lock()

d.running = false

d.mu.Unlock()

}()

  

// Initialize components

if err := d.initializeComponents(); err != nil {

return fmt.Errorf("initialize components: %w", err)

}

  

d.logger.Info("discover server starting",

"grpc_address", d.cfg.GRPCAddress,

"service_name", d.cfg.ServiceName,

)

  

// Run gRPC server with context

if err := d.grpcServer.Start(ctx); err != nil {

if err != context.Canceled {

return fmt.Errorf("grpc server: %w", err)

}

}

  

d.logger.Info("discover server stopped")

return nil

}

  

func (d *Discover) initializeComponents() error {

// Create mDNS scanner

d.scanner = mdns.NewScanner(d.cfg.ServiceName, d.logger)

  

// Create gRPC server

d.grpcServer = grpcserver.NewServer(d.cfg.GRPCAddress, d.logger)

  

// Create discovery service

maxTimeout := 5 * time.Minute

d.discoveryService = service.NewDiscoveryService(

d.scanner,

d.cfg.DefaultTimeout,

maxTimeout,

d.logger,

)

  

// Register discovery service

v1.RegisterDiscoveryServiceServer(d.grpcServer.GRPCServer(), d.discoveryService)

  

// Set service health status

d.grpcServer.SetServingStatus(

v1.DiscoveryService_ServiceDesc.ServiceName,

grpc_health_v1.HealthCheckResponse_SERVING,

)

  

d.logger.Info("components initialized")

return nil

}

  

// Address returns the gRPC server address

func (d *Discover) Address() string {

if d.grpcServer != nil {

return d.grpcServer.Address()

}

return d.cfg.GRPCAddress

}

```

  

### 11. Main Program - Agent

  

**File:** `cmd/agent/main.go`

  

```go

package main

  

import (

"context"

"flag"

"fmt"

"log/slog"

"os"

"os/signal"

"syscall"

  

"github.com/ardge-labs/arx-platform-discover/internal/app"

"github.com/ardge-labs/arx-platform-discover/internal/config"

)

  

var version = "0.0.0"

  

func main() {

var (

showVersion = flag.Bool("version", false, "show version and exit")

showHelp = flag.Bool("help", false, "show help and exit")

serviceName = flag.String("service", "_arx-edge._tcp", "mDNS service name")

servicePort = flag.Int("port", 8080, "service port to advertise")

interfaceName = flag.String("interface", "", "network interface name (empty for all)")

)

flag.Parse()

  

if *showHelp {

fmt.Printf("ARX Platform Agent %s\n\n", version)

fmt.Println("Usage:")

fmt.Printf(" %s [options]\n\n", os.Args[0])

fmt.Println("Options:")

flag.PrintDefaults()

return

}

  

if *showVersion {

fmt.Printf("ARX Platform Agent version v%s\n", version)

return

}

  

// Setup logger

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{

Level: slog.LevelInfo,

}))

  

logger.Info("agent starting", "version", version)

  

// Build configuration

cfg := config.DefaultAgentConfig()

cfg.ServiceName = *serviceName

cfg.ServicePort = *servicePort

cfg.InterfaceName = *interfaceName

  

// Create agent application

agent := app.NewAgent(cfg, logger)

  

// Setup signal handling

ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

defer cancel()

  

// Run agent

if err := agent.Run(ctx); err != nil {

logger.Error("agent failed", "error", err)

os.Exit(1)

}

  

logger.Info("agent stopped", "version", version)

}

```

  

### 12. Main Program - Discover

  

**File:** `cmd/discover/main.go`

  

```go

package main

  

import (

"context"

"flag"

"fmt"

"log/slog"

"os"

"os/signal"

"syscall"

"time"

  

"github.com/ardge-labs/arx-platform-discover/internal/app"

"github.com/ardge-labs/arx-platform-discover/internal/config"

)

  

var version = "0.0.0"

  

func main() {

var (

showVersion = flag.Bool("version", false, "show version and exit")

showHelp = flag.Bool("help", false, "show help and exit")

grpcAddress = flag.String("grpc-address", ":50051", "gRPC server address")

serviceName = flag.String("service", "_arx-edge._tcp", "mDNS service name to scan")

defaultTimeout = flag.Duration("timeout", 30*time.Second, "default discovery timeout")

)

flag.Parse()

  

if *showHelp {

fmt.Printf("ARX Platform Discover %s\n\n", version)

fmt.Println("Usage:")

fmt.Printf(" %s [options]\n\n", os.Args[0])

fmt.Println("Options:")

flag.PrintDefaults()

return

}

  

if *showVersion {

fmt.Printf("ARX Platform Discover version v%s\n", version)

return

}

  

// Setup logger

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{

Level: slog.LevelInfo,

}))

  

logger.Info("discover daemon starting", "version", version)

  

// Build configuration

cfg := config.DefaultDiscoverConfig()

cfg.GRPCAddress = *grpcAddress

cfg.ServiceName = *serviceName

cfg.DefaultTimeout = *defaultTimeout

  

// Create discover application

discover := app.NewDiscover(cfg, logger)

  

// Setup signal handling

ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

defer cancel()

  

// Run discover server

if err := discover.Run(ctx); err != nil {

logger.Error("discover server failed", "error", err)

os.Exit(1)

}

  

logger.Info("discover daemon stopped", "version", version)

}

```

  

## Testing Strategy

  

### Unit Tests

  

**File:** `internal/mdns/txtrecord_test.go`

  

```go

package mdns

  

import (

"testing"

  

v1 "github.com/ardge-labs/arx-platform-discover/api/core/v1"

)

  

func TestEncodeTXTRecords(t *testing.T) {

tests := []struct {

name string

device *v1.EdgeDevice

want int // expected number of records

}{

{

name: "basic device",

device: &v1.EdgeDevice{

DeviceId: "test-123",

ModelName: "TestModel",

BiosVersion: "1.0",

PlatformVersion: "2.0",

ApiPort: 8080,

TlsSupport: true,

CpuCores: 4,

Memory: 1024,

},

want: 8,

},

{

name: "device with interfaces",

device: &v1.EdgeDevice{

DeviceId: "test-456",

Interfaces: []*v1.NetworkInterface{

{

Name: "eth0",

Ipv4: "192.168.1.100",

Mac: "00:11:22:33:44:55",

Role: v1.NetworkInterfaceRole_NETWORK_INTERFACE_ROLE_PRIMARY,

},

},

},

want: 9,

},

}

  

for _, tt := range tests {

t.Run(tt.name, func(t *testing.T) {

records := EncodeTXTRecords(tt.device)

if len(records) != tt.want {

t.Errorf("EncodeTXTRecords() got %d records, want %d", len(records), tt.want)

}

})

}

}

  

func TestRoundTrip(t *testing.T) {

original := &v1.EdgeDevice{

DeviceId: "roundtrip-test",

ModelName: "TestDevice",

BiosVersion: "1.2.3",

PlatformVersion: "go1.25",

Hostname: "test.local",

ApiPort: 9090,

TlsSupport: true,

CpuCores: 8,

Memory: 16384,

Interfaces: []*v1.NetworkInterface{

{

Name: "eth0",

Ipv4: "10.0.0.1",

Mac: "AA:BB:CC:DD:EE:FF",

Role: v1.NetworkInterfaceRole_NETWORK_INTERFACE_ROLE_PRIMARY,

},

},

}

  

records := EncodeTXTRecords(original)

decoded, err := DecodeTXTRecords(records, original.Hostname)

if err != nil {

t.Fatalf("DecodeTXTRecords() error = %v", err)

}

  

if decoded.DeviceId != original.DeviceId {

t.Errorf("DeviceId mismatch: got %v, want %v", decoded.DeviceId, original.DeviceId)

}

if decoded.ApiPort != original.ApiPort {

t.Errorf("ApiPort mismatch: got %v, want %v", decoded.ApiPort, original.ApiPort)

}

}

```

  

## Dependencies

  

Update `go.mod`:

  

```go

module github.com/ardge-labs/arx-platform-discover

  

go 1.25

  

require (

github.com/hashicorp/mdns v1.0.5

google.golang.org/grpc v1.68.0

google.golang.org/protobuf v1.36.0

)

```

  

## Architecture Benefits

  

### Go Idioms

  

- **Clear package boundaries**: `internal` packages ensure implementation details don't leak

- **Interface-oriented design**: Easy to test and swap implementations

- **Context passing**: Proper lifecycle and cancellation management

- **Error wrapping**: Using `%w` to maintain error chain

  

### Separation of Concerns

  

- **Configuration**: Centralized management with defaults

- **mDNS**: Independent broadcast and scanning modules

- **gRPC**: Standard server setup with interceptors

- **Service**: Business logic separated from infrastructure

- **Application**: Component lifecycle orchestration

  

### Concurrent Operation Handling

  

- Each component runs independently

- Channels for non-blocking communication

- Context ensures coordinated shutdown

- sync.Mutex protects shared state

  

### Graceful Shutdown Coordination

  

1. Signal handling triggers context cancellation

2. Components monitor context and cleanup resources

3. Ensures all goroutines terminate properly

4. gRPC uses GracefulStop

  

### High Testability

  

- Hand-written mocks are straightforward

- Table-driven tests cover various scenarios

- Integration tests separated with build tags

- Components can be tested independently

  

## Implementation Phases

  

### Phase 1: Shared Infrastructure

1. `internal/config/config.go` - AgentConfig and DiscoverConfig

2. `internal/mdns/txtrecord.go` - EdgeDevice TXT record encoding/decoding

3. `pkg/errx/errors.go` - Sentinel errors

  

### Phase 2: mDNS Layer

4. `internal/mdns/advertiser.go` - mDNS broadcast server

5. `internal/mdns/scanner.go` - mDNS device scanner

6. `internal/device/info.go` - Device information collection

  

### Phase 3: gRPC Layer

7. `internal/grpcserver/interceptors.go` - Logging/Recovery interceptors

8. `internal/grpcserver/server.go` - gRPC server wrapper

9. `internal/service/discovery.go` - DiscoveryService implementation

  

### Phase 4: Application Orchestration

10. `internal/app/agent.go` - Agent application (mDNS broadcaster)

11. `internal/app/discover.go` - Discover application (gRPC + scanner)

  

### Phase 5: Main Entry Points

12. Update `cmd/agent/main.go` - Agent startup program

13. Update `cmd/discover/main.go` - Discover startup program

  

### Phase 6: Dependencies and Testing

14. Update `go.mod` - Add hashicorp/mdns dependency

15. Create unit tests

16. Run `go mod tidy && make lint && make test`

  

## Summary

  

This architecture provides a solid foundation for a production-ready edge device discovery platform with:

  

- **Agent**: Runs on edge devices, broadcasts device information to local network via mDNS

- **Discover**: Central server providing gRPC API for clients to browse discovered devices

- Complete error handling, logging, and graceful shutdown

- Production-ready, testable, and maintainable design following Go best practices