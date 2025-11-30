// ============================================================================
// Interface Contracts - 介面契約定義
// 此檔案定義所有元件之間的介面契約，確保設計如實作
// ============================================================================

package contracts

import (
	"context"
	"time"
)

// ============================================================================
// 1. Domain Models - 領域模型
// ============================================================================

// TunnelStatus 表示 Tunnel 在 Cloudflare 端的狀態
type TunnelStatus string

const (
	TunnelStatusInactive TunnelStatus = "inactive" // 未連線
	TunnelStatusHealthy  TunnelStatus = "healthy"  // 健康（4 個連線）
	TunnelStatusDegraded TunnelStatus = "degraded" // 降級（部分連線）
	TunnelStatusDown     TunnelStatus = "down"     // 離線
)

// ProcessState 表示本地 cloudflared 進程狀態
type ProcessState string

const (
	ProcessStateIdle     ProcessState = "idle"     // 未啟動
	ProcessStateStarting ProcessState = "starting" // 啟動中
	ProcessStateRunning  ProcessState = "running"  // 運行中
	ProcessStateStopping ProcessState = "stopping" // 停止中
	ProcessStateStopped  ProcessState = "stopped"  // 已停止
	ProcessStateFailed   ProcessState = "failed"   // 失敗
)

// Tunnel 代表一個 Cloudflare Tunnel
type Tunnel struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	AccountID string       `json:"account_id"`
	Status    TunnelStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	DeletedAt *time.Time   `json:"deleted_at,omitempty"`
}

// TunnelToken 敏感資料，獨立結構
type TunnelToken struct {
	TunnelID string
	Token    string
	// 不實作 json 序列化，防止意外洩漏
}

// IngressRule 定義 Tunnel 的路由規則
type IngressRule struct {
	Hostname      string         `json:"hostname,omitempty"`
	Path          string         `json:"path,omitempty"`
	Service       string         `json:"service"`
	OriginRequest *OriginRequest `json:"originRequest,omitempty"`
}

// OriginRequest 定義連接後端服務的選項
type OriginRequest struct {
	// 連線選項
	ConnectTimeout Duration `json:"connectTimeout,omitempty"`
	TLSTimeout     Duration `json:"tlsTimeout,omitempty"`
	TCPKeepAlive   Duration `json:"tcpKeepAlive,omitempty"`

	// HTTP 選項
	HTTPHostHeader         string `json:"httpHostHeader,omitempty"`
	OriginServerName       string `json:"originServerName,omitempty"`
	DisableChunkedEncoding bool   `json:"disableChunkedEncoding,omitempty"`

	// TLS 選項 (僅 HTTPS 有效)
	NoTLSVerify bool   `json:"noTLSVerify,omitempty"`
	CAPool      string `json:"caPool,omitempty"`
}

// TunnelConfiguration 完整的 Tunnel 配置
type TunnelConfiguration struct {
	Ingress     []IngressRule      `json:"ingress"`
	WarpRouting *WarpRoutingConfig `json:"warp-routing,omitempty"`
}

// WarpRoutingConfig WARP 路由配置
type WarpRoutingConfig struct {
	Enabled bool `json:"enabled"`
}

// Connection 代表 Tunnel 到 Cloudflare Edge 的連線
type Connection struct {
	ID                 string    `json:"id"`
	ColoName           string    `json:"colo_name"`
	OriginIP           string    `json:"origin_ip"`
	OpenedAt           time.Time `json:"opened_at"`
	ClientVersion      string    `json:"client_version"`
	IsPendingReconnect bool      `json:"is_pending_reconnect"`
}

// Process 代表本地 cloudflared 進程
type Process struct {
	TunnelID  string       `json:"tunnel_id"`
	PID       int          `json:"pid"`
	State     ProcessState `json:"state"`
	StartedAt time.Time    `json:"started_at"`
	StoppedAt *time.Time   `json:"stopped_at,omitempty"`
	ExitCode  *int         `json:"exit_code,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// Duration 自訂時間類型，支援 JSON "30s" 格式
type Duration struct {
	time.Duration
}

// ============================================================================
// 2. API DTOs - 請求/回應資料傳輸物件
// ============================================================================

// --- Requests ---

// CreateTunnelRequest 建立 Tunnel 請求
type CreateTunnelRequest struct {
	Name    string        `json:"name" validate:"required,min=1,max=64,alphanum_dash"`
	Ingress []IngressRule `json:"ingress,omitempty" validate:"omitempty,dive"`
}

// UpdateConfigRequest 更新配置請求
type UpdateConfigRequest struct {
	Ingress []IngressRule `json:"ingress" validate:"required,min=1,dive,required"`
}

// --- Responses ---

// TunnelResponse 單一 Tunnel 回應
type TunnelResponse struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Status      TunnelStatus  `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	Connections []Connection  `json:"connections,omitempty"`
	Process     *ProcessInfo  `json:"process,omitempty"`
}

// ProcessInfo 進程資訊（回應用）
type ProcessInfo struct {
	State     ProcessState `json:"state"`
	PID       int          `json:"pid,omitempty"`
	StartedAt *time.Time   `json:"started_at,omitempty"`
}

// ListTunnelsResponse 列表回應
type ListTunnelsResponse struct {
	Tunnels []TunnelResponse `json:"tunnels"`
	Total   int              `json:"total"`
}

// ConfigResponse 配置回應
type ConfigResponse struct {
	TunnelID string              `json:"tunnel_id"`
	Config   TunnelConfiguration `json:"config"`
}

// StatusResponse 狀態回應
type StatusResponse struct {
	TunnelID    string       `json:"tunnel_id"`
	CloudStatus TunnelStatus `json:"cloud_status"`
	Process     ProcessInfo  `json:"process"`
}

// ErrorResponse 錯誤回應
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 錯誤詳情
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ============================================================================
// 3. Service Interfaces - 服務層介面
// ============================================================================

// TunnelService 定義 Tunnel 管理的業務邏輯介面
type TunnelService interface {
	// === Tunnel 生命週期 ===

	// Create 建立新的 Tunnel
	// 1. 呼叫 Cloudflare API 建立 Tunnel
	// 2. 如果提供 Ingress，更新配置
	// 3. 回傳 Tunnel 資訊（不含 Token）
	Create(ctx context.Context, req CreateTunnelRequest) (*TunnelResponse, error)

	// Get 取得單一 Tunnel 詳情
	// 包含 Cloudflare 狀態 + 本地進程狀態
	Get(ctx context.Context, tunnelID string) (*TunnelResponse, error)

	// List 列出所有 Tunnels
	List(ctx context.Context) (*ListTunnelsResponse, error)

	// Delete 刪除 Tunnel
	// 1. 停止本地進程（如果運行中）
	// 2. 清理 Cloudflare 連線
	// 3. 刪除 Tunnel
	Delete(ctx context.Context, tunnelID string) error

	// === 配置管理 ===

	// GetConfig 取得 Tunnel 配置
	GetConfig(ctx context.Context, tunnelID string) (*ConfigResponse, error)

	// UpdateConfig 更新 Tunnel 配置
	// 配置會即時生效（cloudflared 會自動重載）
	UpdateConfig(ctx context.Context, tunnelID string, req UpdateConfigRequest) error

	// === 進程控制 ===

	// Start 啟動 cloudflared 進程
	// 1. 檢查是否已運行
	// 2. 取得 Tunnel Token
	// 3. 啟動 cloudflared
	Start(ctx context.Context, tunnelID string) error

	// Stop 停止 cloudflared 進程
	// 發送 SIGTERM，等待優雅關閉
	Stop(ctx context.Context, tunnelID string) error

	// Restart 重啟 cloudflared 進程
	// Stop + Start
	Restart(ctx context.Context, tunnelID string) error

	// GetStatus 取得綜合狀態
	GetStatus(ctx context.Context, tunnelID string) (*StatusResponse, error)
}

// ============================================================================
// 4. Repository Interfaces - 資料存取介面
// ============================================================================

// CloudflareAPIClient 定義與 Cloudflare API 互動的介面
type CloudflareAPIClient interface {
	// === Tunnel CRUD ===

	// CreateTunnel 建立 Tunnel（遠端管理模式）
	// POST /accounts/{account_id}/cfd_tunnel
	CreateTunnel(ctx context.Context, name string) (*Tunnel, error)

	// GetTunnel 取得 Tunnel 資訊
	// GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}
	GetTunnel(ctx context.Context, tunnelID string) (*Tunnel, error)

	// ListTunnels 列出所有 Tunnels
	// GET /accounts/{account_id}/cfd_tunnel
	ListTunnels(ctx context.Context) ([]Tunnel, error)

	// DeleteTunnel 刪除 Tunnel
	// DELETE /accounts/{account_id}/cfd_tunnel/{tunnel_id}
	DeleteTunnel(ctx context.Context, tunnelID string) error

	// === Token ===

	// GetTunnelToken 取得 Tunnel Token（用於 cloudflared 連線）
	// GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/token
	// 注意：Token 為敏感資料，不應記錄到日誌
	GetTunnelToken(ctx context.Context, tunnelID string) (*TunnelToken, error)

	// === Configuration ===

	// GetTunnelConfiguration 取得 Tunnel 配置
	// GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurations
	GetTunnelConfiguration(ctx context.Context, tunnelID string) (*TunnelConfiguration, error)

	// UpdateTunnelConfiguration 更新 Tunnel 配置
	// PUT /accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurations
	UpdateTunnelConfiguration(ctx context.Context, tunnelID string, cfg TunnelConfiguration) error

	// === Connection Management ===

	// GetConnections 取得 Tunnel 連線資訊
	// GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/connections
	GetConnections(ctx context.Context, tunnelID string) ([]Connection, error)

	// CleanupConnections 清理 Tunnel 連線
	// DELETE /accounts/{account_id}/cfd_tunnel/{tunnel_id}/connections
	CleanupConnections(ctx context.Context, tunnelID string) error
}

// ============================================================================
// 5. Process Manager Interface - 進程管理介面
// ============================================================================

// ProcessOptions 進程啟動選項
type ProcessOptions struct {
	LogLevel    string        // debug | info | warn | error | fatal
	Protocol    string        // auto | quic | http2
	EdgeIPVer   string        // auto | 4 | 6
	GracePeriod time.Duration // 優雅關閉等待時間
	MetricsAddr string        // Prometheus metrics 地址 (可選)
}

// DefaultProcessOptions 預設選項
func DefaultProcessOptions() ProcessOptions {
	return ProcessOptions{
		LogLevel:    "info",
		Protocol:    "quic",
		EdgeIPVer:   "auto",
		GracePeriod: 30 * time.Second,
	}
}

// ProcessManager 定義 cloudflared 進程管理介面
type ProcessManager interface {
	// Start 啟動 cloudflared 進程
	// token: Tunnel Token（從 CloudflareAPIClient.GetTunnelToken 取得）
	// opts: 進程啟動選項
	// 實作需求：
	// 1. 檢查是否已運行（防止重複啟動）
	// 2. 使用環境變數傳遞 Token（不使用命令列參數）
	// 3. 啟動監控 goroutine
	Start(ctx context.Context, tunnelID string, token TunnelToken, opts ProcessOptions) error

	// Stop 停止 cloudflared 進程
	// 實作需求：
	// 1. 發送 SIGTERM
	// 2. 等待 GracePeriod
	// 3. 如果超時，發送 SIGKILL
	Stop(ctx context.Context, tunnelID string) error

	// GetProcess 取得進程狀態
	// 回傳 nil 如果進程不存在
	GetProcess(tunnelID string) *Process

	// ListProcesses 列出所有進程
	ListProcesses() []Process

	// StopAll 停止所有進程（用於優雅關閉）
	StopAll(ctx context.Context) error
}

// ProcessEventType 進程事件類型
type ProcessEventType string

const (
	ProcessEventStarted  ProcessEventType = "started"
	ProcessEventStopped  ProcessEventType = "stopped"
	ProcessEventFailed   ProcessEventType = "failed"
	ProcessEventCrashed  ProcessEventType = "crashed"
)

// ProcessEvent 進程事件（用於監控/通知）
type ProcessEvent struct {
	Type      ProcessEventType `json:"type"`
	TunnelID  string           `json:"tunnel_id"`
	PID       int              `json:"pid,omitempty"`
	ExitCode  *int             `json:"exit_code,omitempty"`
	Error     string           `json:"error,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}

// ProcessEventHandler 進程事件處理器（可選實作）
type ProcessEventHandler interface {
	OnProcessEvent(event ProcessEvent)
}

// ============================================================================
// 6. Error Definitions - 錯誤定義
// ============================================================================

// ErrorCode 錯誤碼
type ErrorCode string

const (
	// 認證錯誤 (401)
	ErrAuthMissingKey ErrorCode = "AUTH_MISSING_KEY"
	ErrAuthInvalidKey ErrorCode = "AUTH_INVALID_KEY"

	// 限流錯誤 (429)
	ErrRateLimited ErrorCode = "AUTH_RATE_LIMITED"

	// 資源錯誤 (404)
	ErrTunnelNotFound ErrorCode = "TUNNEL_NOT_FOUND"

	// 衝突錯誤 (409)
	ErrTunnelAlreadyExists ErrorCode = "TUNNEL_ALREADY_EXISTS"
	ErrTunnelRunning       ErrorCode = "TUNNEL_RUNNING"
	ErrTunnelNotRunning    ErrorCode = "TUNNEL_NOT_RUNNING"

	// 驗證錯誤 (400)
	ErrConfigInvalid       ErrorCode = "CONFIG_INVALID"
	ErrConfigMissingCatch  ErrorCode = "CONFIG_MISSING_CATCHALL"
	ErrInvalidTunnelName   ErrorCode = "INVALID_TUNNEL_NAME"

	// 外部錯誤 (502)
	ErrCloudflareAPI ErrorCode = "CLOUDFLARE_ERROR"

	// 內部錯誤 (500)
	ErrProcessStartFailed ErrorCode = "PROCESS_START_FAILED"
	ErrProcessStopFailed  ErrorCode = "PROCESS_STOP_FAILED"
	ErrInternalError      ErrorCode = "INTERNAL_ERROR"
)

// AppError 應用程式錯誤
type AppError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Details any
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError 建立應用程式錯誤
func NewAppError(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// WithCause 附加原因
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithDetails 附加詳情
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// ============================================================================
// 7. Configuration - 設定結構
// ============================================================================

// Config 應用程式設定
type Config struct {
	Server      ServerConfig      `json:"server"`
	Cloudflare  CloudflareConfig  `json:"cloudflare"`
	Cloudflared CloudflaredConfig `json:"cloudflared"`
	Logging     LoggingConfig     `json:"logging"`
}

// ServerConfig HTTP Server 設定
type ServerConfig struct {
	ListenAddr string     `json:"listen_addr"` // :8443
	TLS        *TLSConfig `json:"tls,omitempty"`
	RateLimit  RateLimitConfig `json:"rate_limit"`
}

// TLSConfig TLS 設定
type TLSConfig struct {
	CertFile   string `json:"cert_file"`
	KeyFile    string `json:"key_file"`
	MinVersion string `json:"min_version"` // 1.3
}

// RateLimitConfig 限流設定
type RateLimitConfig struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	Burst             int     `json:"burst"`
}

// CloudflareConfig Cloudflare 設定
type CloudflareConfig struct {
	// APIToken 從環境變數 CF_API_TOKEN 讀取
	AccountID string `json:"account_id"`
	ZoneID    string `json:"zone_id,omitempty"`
}

// CloudflaredConfig cloudflared 設定
type CloudflaredConfig struct {
	Path           string         `json:"path"` // /usr/local/bin/cloudflared
	DefaultOptions ProcessOptions `json:"default_options"`
}

// LoggingConfig 日誌設定
type LoggingConfig struct {
	Level  string `json:"level"`  // debug | info | warn | error
	Format string `json:"format"` // json | text
	Output string `json:"output"` // stdout | file path
}

// ============================================================================
// 8. Secret Provider Interface - 秘密提供者介面
// ============================================================================

// SecretProvider 定義秘密存取介面
type SecretProvider interface {
	// GetSecret 取得秘密值
	// key: 秘密鍵名
	GetSecret(ctx context.Context, key string) (string, error)

	// GetSecretWithDefault 取得秘密值，不存在時回傳預設值
	GetSecretWithDefault(ctx context.Context, key, defaultValue string) string
}

// 常用秘密鍵名
const (
	SecretKeyCFAPIToken  = "CF_API_TOKEN"
	SecretKeyAPIKeyHash  = "API_KEY_HASH"
	SecretKeyTLSCert     = "TLS_CERT"
	SecretKeyTLSKey      = "TLS_KEY"
)
