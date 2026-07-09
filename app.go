package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"codexswitch/internal/codexswitch"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	ctxMu   sync.RWMutex
	service *codexswitch.Service
	mu      sync.Mutex

	// allowNextClose 让托盘「退出」可以绕过最小化到托盘的拦截
	allowNextClose atomic.Bool
}

func NewApp() *App {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	service, err := codexswitch.NewService(codexswitch.ServiceOptions{
		Logger: logger,
	})
	if err != nil {
		panic(err)
	}

	return &App{service: service}
}

func (a *App) startup(ctx context.Context) {
	a.ctxMu.Lock()
	a.ctx = ctx
	a.ctxMu.Unlock()
	installCloseToTrayHandler(a)
}

func (a *App) runtimeContext() context.Context {
	a.ctxMu.RLock()
	defer a.ctxMu.RUnlock()
	return a.ctx
}

func (a *App) shutdown(context.Context) {
	if a.service != nil {
		_ = a.service.Close()
	}
}

// beforeClose 由 Wails 在窗口关闭前调用，根据设置决定是否最小化到托盘。
func (a *App) beforeClose(ctx context.Context) bool {
	if a.allowNextClose.Swap(false) {
		return false
	}
	if !shouldHandleCloseToTrayInBeforeClose() {
		return false
	}
	if a.service == nil {
		return false
	}
	settings, err := a.service.GetSettings()
	if err != nil {
		return false
	}
	if !settings.MinimizeToTrayOnClose {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

func (a *App) GetAppState() (codexswitch.AppState, error) {
	return withAppLock(&a.mu, a.service.GetAppState)
}

func (a *App) ImportCurrentProfile() (codexswitch.AppState, error) {
	return withAppLock(&a.mu, a.service.ImportCurrentProfile)
}

func (a *App) ImportOfficialProfileFile() (codexswitch.AppState, error) {
	ctx := a.runtimeContext()

	if ctx == nil {
		return codexswitch.AppState{}, errors.New("Wails runtime 未就绪")
	}

	selectedFile, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "选择官方账号文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON Files (*.json)",
				Pattern:     "*.json",
			},
		},
	})
	if err != nil {
		return codexswitch.AppState{}, err
	}
	if strings.TrimSpace(selectedFile) == "" {
		return codexswitch.AppState{}, errors.New("已取消文件选择")
	}

	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.ImportOfficialProfileFile(selectedFile)
	})
}

func (a *App) CreateApiProfile(input codexswitch.APIProfileInput) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.CreateAPIProfile(input)
	})
}

func (a *App) UpdateApiProfile(id string, input codexswitch.APIProfileInput) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.UpdateAPIProfile(id, input)
	})
}

func (a *App) GetApiProfileInput(id string) (codexswitch.APIProfileInput, error) {
	return withAppLock(&a.mu, func() (codexswitch.APIProfileInput, error) {
		return a.service.GetAPIProfileInput(id)
	})
}

func (a *App) SwitchProfile(id string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.SwitchProfile(id)
	})
}

func (a *App) RestartCodex() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.service.RestartCodex()
}

func (a *App) RepairNetwork() (codexswitch.NetworkRepairResult, error) {
	return withAppLock(&a.mu, a.service.RepairNetwork)
}

func (a *App) DeleteProfile(id string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.DeleteProfile(id)
	})
}

func (a *App) SetProfileDisabled(id string, disabled bool) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.SetProfileDisabled(id, disabled)
	})
}

func (a *App) RefreshRateLimits(ids []string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.RefreshRateLimits(ids)
	})
}

func (a *App) RefreshApiLatencyTests(ids []string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.RefreshAPILatencyTests(ids)
	})
}

func (a *App) AutoRefreshApiLatencyTests(ids []string) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.AutoRefreshAPILatencyTests(ids)
	})
}

func (a *App) UpdateSettings(input codexswitch.UpdateSettingsInput) (codexswitch.AppState, error) {
	return withAppLock(&a.mu, func() (codexswitch.AppState, error) {
		return a.service.UpdateSettings(input)
	})
}

func (a *App) GetAppVersion() string {
	return appVersion
}

func withAppLock[T any](mu *sync.Mutex, fn func() (T, error)) (T, error) {
	mu.Lock()
	defer mu.Unlock()
	return fn()
}
