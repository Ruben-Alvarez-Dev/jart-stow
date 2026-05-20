package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// MonitorService orchestrates the daemon: watches workspace roots, auto-excludes artifacts,
// and runs periodic junk scans.
type MonitorService struct {
	projectRepo      ports.ProjectRepository
	exclusionRepo    ports.ExclusionRepository
	eventRepo        ports.EventRepository
	scanJobRepo      ports.ScanJobRepository
	watchRootRepo    ports.WatchRootRepository
	junkCategoryRepo ports.JunkCategoryRepository
	junkItemRepo     ports.JunkItemRepository
	backups          []ports.BackupProvider
	watcher          ports.FileSystemWatcher
	excludeService   *ExcludeService
	junkService      *JunkScanService

	debounceDelay time.Duration
	junkInterval  time.Duration

	rootsMu sync.RWMutex

	scanQueue     chan string
	workerCount   int
	detectionDepth int
	wg            sync.WaitGroup
}

// MonitorConfig holds configuration for the monitor service.
type MonitorConfig struct {
	DebounceDelay time.Duration
	JunkInterval  time.Duration
	WorkerCount   int
	DetectionDepth int
}

// DefaultMonitorConfig returns sensible defaults for the monitor service.
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		DebounceDelay: 2 * time.Second,
		JunkInterval:  24 * time.Hour,
		WorkerCount:   4,
		DetectionDepth: 2,
	}
}

// NewMonitorService creates a new MonitorService with constructor injection.
func NewMonitorService(
	projectRepo ports.ProjectRepository,
	exclusionRepo ports.ExclusionRepository,
	eventRepo ports.EventRepository,
	scanJobRepo ports.ScanJobRepository,
	watchRootRepo ports.WatchRootRepository,
	junkCategoryRepo ports.JunkCategoryRepository,
	junkItemRepo ports.JunkItemRepository,
	watcher ports.FileSystemWatcher,
	excludeService *ExcludeService,
	junkService *JunkScanService,
	cfg MonitorConfig,
	backups ...ports.BackupProvider,
) *MonitorService {
	return &MonitorService{
		projectRepo:      projectRepo,
		exclusionRepo:    exclusionRepo,
		eventRepo:        eventRepo,
		scanJobRepo:      scanJobRepo,
		watchRootRepo:    watchRootRepo,
		junkCategoryRepo: junkCategoryRepo,
		junkItemRepo:     junkItemRepo,
		backups:          backups,
		watcher:          watcher,
		excludeService:   excludeService,
		junkService:      junkService,
		debounceDelay:    cfg.DebounceDelay,
		junkInterval:     cfg.JunkInterval,
		scanQueue:        make(chan string, 100),
		workerCount:      cfg.WorkerCount,
		detectionDepth:   cfg.DetectionDepth,
	}
}

// Run starts the daemon main loop. It watches workspace roots, handles file system events,
// and runs periodic junk scans. Returns when a shutdown signal is received.
func (m *MonitorService) Run(ctx context.Context) error {
	log.Println("event=daemon_started")

	// Wire shutdown signals
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start watching configured roots
	if err := m.refreshRoots(ctx); err != nil {
		return fmt.Errorf("loading watch roots: %w", err)
	}

	m.rootsMu.RLock()
	for _, root := range m.roots {
		if !root.Enabled {
			continue
		}
	}
	m.rootsMu.RUnlock()

	// Start workers
	m.startWorkers(ctx)

	// Create channels for event handling
	junkTicker := time.NewTicker(m.junkInterval)
	defer junkTicker.Stop()

	// Process events
	for {
		select {
		case <-ctx.Done():
			log.Println("event=daemon_stopping")
			return m.shutdown()

		case event := <-m.watcher.Events():
			m.handleFileEvent(ctx, event)

		case err := <-m.watcher.Errors():
			log.Printf("event=watcher_error error=%v", err)

		case <-junkTicker.C:
			m.runPeriodicJunkScan(ctx)
		}
	}
}

// watch adds a watch root to the file system watcher.
func (m *MonitorService) watch(_ context.Context, root domain.WatchRoot) error {
	log.Printf("event=watching root=%s", root.Path)
	return m.watcher.Watch(context.Background(), root.Path)
}

// handleFileEvent processes a file system event from the watcher.
func (m *MonitorService) handleFileEvent(ctx context.Context, event ports.FileSystemEvent) {
	if event.Op != ports.OpCreate {
		return
	}

	info, err := os.Stat(event.Path)
	if err != nil {
		return
	}
	if !info.IsDir() {
		return
	}

	// Check if under a watch root
	m.rootsMu.RLock()
	defer m.rootsMu.RUnlock()

	for _, root := range m.roots {
		if !root.Enabled {
			continue
		}
		rel, err := filepath.Rel(root.Path, event.Path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		
		// Check depth
		parts := strings.Split(filepath.Clean(rel), string(os.PathSeparator))
		if len(parts) > m.detectionDepth {
			continue
		}
		
		// Debounce
		go func(path string) {
			time.Sleep(m.debounceDelay)
			select {
			case m.scanQueue <- path:
			case <-ctx.Done():
			}
		}(event.Path)
		return
	}
}

// startWorkers launches the scan worker pool.
func (m *MonitorService) startWorkers(ctx context.Context) {
	for i := 0; i < m.workerCount; i++ {
		m.wg.Add(1)
		go func(workerID int) {
			defer m.wg.Done()
			log.Printf("event=worker_started id=%d", workerID)
			for {
				select {
				case <-ctx.Done():
					log.Printf("event=worker_stopped id=%d", workerID)
					return
				case path, ok := <-m.scanQueue:
					if !ok {
						return
					}
					m.handleNewDirectory(ctx, path)
				}
			}
		}(i)
	}
}

// refreshRoots loads all watch roots from the repository and updates the cache.
func (m *MonitorService) refreshRoots(ctx context.Context) error {
	roots, err := m.watchRootRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	m.rootsMu.Lock()
	m.roots = roots
	m.rootsMu.Unlock()
	return nil
}

// handleNewDirectory processes a newly detected project directory.
func (m *MonitorService) handleNewDirectory(ctx context.Context, path string) {
	// Check if already known
	project, err := m.projectRepo.FindByPath(ctx, path)
	if err != nil && err != domain.ErrProjectNotFound {
		log.Printf("event=project_lookup_error path=%s error=%v", path, err)
		return
	}
	if project != nil && project.Status == domain.ProjectStatusActive {
		return
	}

	// Upsert project
	project, err = m.projectRepo.Upsert(ctx, &domain.Project{
		Path:   path,
		Name:   filepath.Base(path),
		Status: domain.ProjectStatusActive,
	})
	if err != nil {
		log.Printf("event=project_upsert_error path=%s error=%v", path, err)
		return
	}

	log.Printf("event=project_detected path=%s", path)

	// Log event
	_ = m.eventRepo.Log(ctx, &domain.DaemonEvent{
		EventType: domain.EventTypeProjectDetected,
		ProjectID: &project.ID,
		Details:   fmt.Sprintf(`{"path":"%s"}`, path),
	})

	// Scan and exclude
	count, size, err := m.excludeService.ExcludeProject(ctx, path)
	if err != nil {
		log.Printf("event=exclusion_error path=%s error=%v", path, err)
		_ = m.eventRepo.Log(ctx, &domain.DaemonEvent{
			EventType: domain.EventTypeError,
			ProjectID: &project.ID,
			Details:   fmt.Sprintf(`{"path":"%s","error":"%s"}`, path, err),
		})
		return
	}

	log.Printf("event=scan_completed project=%s folders=%d size=%d", filepath.Base(path), count, size)

	// Log completion
	_ = m.eventRepo.Log(ctx, &domain.DaemonEvent{
		EventType:  domain.EventTypeScanCompleted,
		ProjectID:  &project.ID,
		FolderPath: path,
		Details:    fmt.Sprintf(`{"artifacts":%d,"total_size":%d}`, count, size),
	})
}

// runPeriodicJunkScan scans all enabled junk categories and populates junk_items.
func (m *MonitorService) runPeriodicJunkScan(ctx context.Context) {
	log.Println("event=junk_scan_started")
	categories, err := m.junkCategoryRepo.FindAll(ctx)
	if err != nil {
		log.Printf("event=junk_scan_error error=%v", err)
		return
	}

	var enabled int
	for _, cat := range categories {
		if !cat.Enabled {
			continue
		}
		enabled++

		items, err := m.junkService.ScanCategory(ctx, cat)
		if err != nil {
			log.Printf("event=junk_scan_category_error category=%s error=%v", cat.Name, err)
			continue
		}
		for _, item := range items {
			if _, err := m.junkItemRepo.Save(ctx, &item); err != nil {
				log.Printf("event=junk_item_save_error category=%s path=%s error=%v", cat.Name, item.Path, err)
			}
		}
	}
	log.Printf("event=junk_scan_completed categories=%d", enabled)
}

// shutdown performs a graceful shutdown: closes the watcher and logs the event.
func (m *MonitorService) shutdown() error {
	close(m.scanQueue)
	m.wg.Wait()
	if err := m.watcher.Close(); err != nil {
		return fmt.Errorf("closing watcher: %w", err)
	}
	log.Println("event=daemon_stopped")
	return nil
}
