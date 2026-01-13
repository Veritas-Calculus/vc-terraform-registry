// Package scheduler provides background sync scheduling.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/Veritas-Calculus/vc-terraform-registry/internal/proxy"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler manages scheduled sync tasks.
type Scheduler struct {
	db          *gorm.DB
	storagePath string
	cron        *cron.Cron
	jobs        map[uint]cron.EntryID
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// New creates a new Scheduler.
func New(db *gorm.DB, storagePath string) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		db:          db,
		storagePath: storagePath,
		cron:        cron.New(),
		jobs:        make(map[uint]cron.EntryID),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the scheduler.
func (s *Scheduler) Start() error {
	if err := s.loadSchedules(); err != nil {
		return err
	}
	s.cron.Start()
	go s.watchForChanges()
	log.Println("Scheduler started")
	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.cancel()
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("Scheduler stopped")
}

func (s *Scheduler) loadSchedules() error {
	var schedules []models.SyncSchedule
	if err := s.db.Where("enabled = ?", true).Find(&schedules).Error; err != nil {
		return err
	}
	for _, schedule := range schedules {
		if err := s.addJob(schedule); err != nil {
			log.Printf("Failed to add schedule %d: %v", schedule.ID, err)
		}
	}
	return nil
}

func (s *Scheduler) addJob(schedule models.SyncSchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[schedule.ID]; exists {
		s.cron.Remove(entryID)
	}

	scheduleID := schedule.ID
	entryID, err := s.cron.AddFunc(schedule.CronExpr, func() {
		s.runSync(scheduleID)
	})
	if err != nil {
		return err
	}

	s.jobs[schedule.ID] = entryID
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next
	s.db.Model(&models.SyncSchedule{}).Where("id = ?", schedule.ID).Update("next_run_at", nextRun)
	return nil
}

func (s *Scheduler) removeJob(scheduleID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, scheduleID)
	}
}

func (s *Scheduler) runSync(scheduleID uint) {
	var schedule models.SyncSchedule
	if err := s.db.First(&schedule, scheduleID).Error; err != nil {
		log.Printf("Schedule %d not found: %v", scheduleID, err)
		return
	}

	log.Printf("Running sync for %s/%s", schedule.Namespace, schedule.Name)

	now := time.Now()
	schedule.LastRunAt = &now
	schedule.LastStatus = "running"
	s.db.Save(&schedule)

	proxyService := proxy.NewProxyService(s.storagePath, "")
	err := s.mirrorProvider(proxyService, schedule.Namespace, schedule.Name, "", schedule.SyncOS, schedule.SyncArch)
	finishTime := time.Now()

	if err != nil {
		schedule.LastStatus = "failed"
		schedule.LastError = err.Error()
		log.Printf("Sync failed for %s/%s: %v", schedule.Namespace, schedule.Name, err)
	} else {
		schedule.LastStatus = "success"
		schedule.LastError = ""
		log.Printf("Sync completed for %s/%s", schedule.Namespace, schedule.Name)
	}

	schedule.LastRunAt = &finishTime

	s.mu.RLock()
	if entryID, exists := s.jobs[scheduleID]; exists {
		entry := s.cron.Entry(entryID)
		schedule.NextRunAt = &entry.Next
	}
	s.mu.RUnlock()

	s.db.Save(&schedule)
}

func (s *Scheduler) mirrorProvider(proxyService *proxy.ProxyService, namespace, name, version, osType, arch string) error {
	platforms, resolvedVersion, err := s.getPlatformsToMirror(proxyService, namespace, name, version, osType, arch)
	if err != nil {
		return err
	}

	for _, platform := range platforms {
		s.downloadAndSavePlatform(proxyService, namespace, name, resolvedVersion, platform.OS, platform.Arch)
	}

	return nil
}

// getPlatformsToMirror fetches version info and returns matching platforms.
func (s *Scheduler) getPlatformsToMirror(proxyService *proxy.ProxyService, namespace, name, version, osType, arch string) ([]struct{ OS, Arch string }, string, error) {
	versions, err := proxyService.GetProviderVersions(namespace, name)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get versions: %w", err)
	}
	if len(versions.Versions) == 0 {
		return nil, "", fmt.Errorf("no versions available")
	}

	if version == "" {
		version = versions.Versions[0].Version
	}

	var platforms []struct{ OS, Arch string }
	for _, v := range versions.Versions {
		if v.Version == version {
			for _, p := range v.Platforms {
				if (osType == "all" || osType == p.OS) && (arch == "all" || arch == p.Arch) {
					platforms = append(platforms, struct{ OS, Arch string }{p.OS, p.Arch})
				}
			}
			break
		}
	}

	if len(platforms) == 0 {
		return nil, "", fmt.Errorf("no matching platforms found")
	}

	return platforms, version, nil
}

// downloadAndSavePlatform downloads a platform and saves it to the database.
func (s *Scheduler) downloadAndSavePlatform(proxyService *proxy.ProxyService, namespace, name, version, osType, arch string) {
	filePath, sha256sum, err := proxyService.DownloadAndCacheProvider(namespace, name, version, osType, arch)
	if err != nil {
		log.Printf("Failed to download %s_%s: %v", osType, arch, err)
		return
	}

	var provider models.Provider
	result := s.db.Where("namespace = ? AND name = ? AND version = ?", namespace, name, version).First(&provider)
	if result.Error == gorm.ErrRecordNotFound {
		provider = models.Provider{
			Namespace:  namespace,
			Name:       name,
			Version:    version,
			SourceType: models.SourceMirror,
			Protocols:  `["5.0"]`,
		}
		s.db.Create(&provider)
	}

	var existingPlatform models.ProviderPlatform
	if err := s.db.Where("provider_id = ? AND os = ? AND arch = ?", provider.ID, osType, arch).First(&existingPlatform).Error; err != nil {
		platformModel := models.ProviderPlatform{
			ProviderID: provider.ID,
			OS:         osType,
			Arch:       arch,
			Filename:   filepath.Base(filePath),
			FilePath:   filePath,
			SHA256Sum:  sha256sum,
		}
		s.db.Create(&platformModel)
	}
}

func (s *Scheduler) watchForChanges() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.refreshSchedules()
		}
	}
}

func (s *Scheduler) refreshSchedules() {
	var schedules []models.SyncSchedule
	if err := s.db.Find(&schedules).Error; err != nil {
		log.Printf("Failed to refresh schedules: %v", err)
		return
	}

	currentIDs := make(map[uint]bool)
	for _, schedule := range schedules {
		currentIDs[schedule.ID] = true
		if schedule.Enabled {
			s.mu.RLock()
			_, exists := s.jobs[schedule.ID]
			s.mu.RUnlock()
			if !exists {
				if err := s.addJob(schedule); err != nil {
					log.Printf("Failed to add schedule %d: %v", schedule.ID, err)
				}
			}
		} else {
			s.removeJob(schedule.ID)
		}
	}

	s.mu.Lock()
	for id := range s.jobs {
		if !currentIDs[id] {
			s.cron.Remove(s.jobs[id])
			delete(s.jobs, id)
		}
	}
	s.mu.Unlock()
}

// TriggerSync manually triggers a sync for a schedule.
func (s *Scheduler) TriggerSync(scheduleID uint) error {
	var schedule models.SyncSchedule
	if err := s.db.First(&schedule, scheduleID).Error; err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}
	go s.runSync(scheduleID)
	return nil
}
