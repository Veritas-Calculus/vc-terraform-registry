// Package models defines data structures for the application.
package models

import (
	"time"

	"gorm.io/gorm"
)

// SourceType represents the source of a provider.
type SourceType string

const (
	// SourceUpload indicates the provider was manually uploaded.
	SourceUpload SourceType = "upload"
	// SourceMirror indicates the provider was mirrored from upstream.
	SourceMirror SourceType = "mirror"
)

// Provider represents a Terraform provider.
type Provider struct {
	ID          uint               `gorm:"primarykey" json:"id"`
	Namespace   string             `gorm:"index:idx_provider,unique;not null" json:"namespace"`
	Name        string             `gorm:"index:idx_provider,unique;not null" json:"name"`
	Version     string             `gorm:"index:idx_provider,unique;not null" json:"version"`
	Description string             `json:"description"`
	SourceType  SourceType         `gorm:"type:varchar(20);default:'upload'" json:"source_type"`
	SourceURL   string             `json:"source_url"`
	Protocols   string             `json:"protocols"` // JSON array of protocol versions
	Published   time.Time          `json:"published"`
	Downloads   int64              `json:"downloads"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	DeletedAt   gorm.DeletedAt     `gorm:"index" json:"-"`
	Platforms   []ProviderPlatform `gorm:"foreignKey:ProviderID" json:"platforms,omitempty"`
}

// Module represents a Terraform module.
type Module struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Namespace   string         `gorm:"index:idx_module,unique;not null" json:"namespace"`
	Name        string         `gorm:"index:idx_module,unique;not null" json:"name"`
	Provider    string         `gorm:"index:idx_module,unique;not null" json:"provider"`
	Version     string         `gorm:"index:idx_module,unique;not null" json:"version"`
	Description string         `json:"description"`
	Source      string         `json:"source"`
	Published   time.Time      `json:"published"`
	Downloads   int64          `json:"downloads"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// User represents a system user.
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Username  string         `gorm:"uniqueIndex;not null" json:"username"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"not null" json:"-"`
	Role      string         `gorm:"not null;default:'user'" json:"role"`
	APIToken  string         `gorm:"uniqueIndex" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ProviderPlatform represents platform-specific provider binaries.
type ProviderPlatform struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	ProviderID uint           `gorm:"not null;index" json:"provider_id"`
	OS         string         `gorm:"not null" json:"os"`
	Arch       string         `gorm:"not null" json:"arch"`
	Filename   string         `gorm:"not null" json:"filename"`
	FilePath   string         `gorm:"not null" json:"file_path"`
	SHA256Sum  string         `gorm:"not null" json:"sha256sum"`
	FileSize   int64          `json:"file_size"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// MirrorConfig represents configuration for mirroring from upstream.
type MirrorConfig struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Namespace   string         `gorm:"not null" json:"namespace"`
	Name        string         `gorm:"not null" json:"name"`
	UpstreamURL string         `gorm:"not null;default:'https://registry.terraform.io'" json:"upstream_url"`
	AutoSync    bool           `gorm:"default:false" json:"auto_sync"`
	LastSyncAt  *time.Time     `json:"last_sync_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Settings represents global application settings.
type Settings struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	AllowOnlineSearch  bool      `gorm:"default:true" json:"allow_online_search"`
	DefaultUpstreamURL string    `gorm:"default:'https://registry.terraform.io'" json:"default_upstream_url"`
	RegistryURL        string    `gorm:"default:''" json:"registry_url"` // Custom registry URL for Terraform config
	ProxyEnabled       bool      `gorm:"default:false" json:"proxy_enabled"`
	ProxyURL           string    `gorm:"default:''" json:"proxy_url"`
	ProxyType          string    `gorm:"default:'http'" json:"proxy_type"` // http, socks5
	UpdatedAt          time.Time `json:"updated_at"`
}

// SyncSchedule represents a scheduled sync task for a provider.
type SyncSchedule struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	Namespace  string         `gorm:"not null;index:idx_sync_provider" json:"namespace"`
	Name       string         `gorm:"not null;index:idx_sync_provider" json:"name"`
	CronExpr   string         `gorm:"not null" json:"cron_expr"`
	Enabled    bool           `gorm:"default:true" json:"enabled"`
	SyncOS     string         `gorm:"default:'all'" json:"sync_os"`
	SyncArch   string         `gorm:"default:'all'" json:"sync_arch"`
	LastRunAt  *time.Time     `json:"last_run_at"`
	LastStatus string         `json:"last_status"`
	LastError  string         `json:"last_error"`
	NextRunAt  *time.Time     `json:"next_run_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
