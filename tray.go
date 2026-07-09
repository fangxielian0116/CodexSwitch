//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"log"
	"runtime"
	"sort"
	"sync"

	"codexswitch/internal/codexswitch"
	"github.com/alttpo/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var trayIconICO []byte

type trayCmdKind int

const (
	trayCmdRefresh trayCmdKind = iota
	trayCmdShow
	trayCmdQuit
	trayCmdSwitch
)

type trayCmd struct {
	kind    trayCmdKind
	payload string
}

type trayManager struct {
	app *App

	cmdCh     chan trayCmd
	switchCh  chan string
	closeOnce sync.Once

	mCurrent *systray.MenuItem
	mOpen    *systray.MenuItem
	mSwitch  *systray.MenuItem
	mQuit    *systray.MenuItem

	quickItems map[string]*systray.MenuItem
}

var globalTray *trayManager

func startTray(app *App) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	globalTray = &trayManager{
		app:        app,
		cmdCh:      make(chan trayCmd, 16),
		switchCh:   make(chan string, 16),
		quickItems: make(map[string]*systray.MenuItem),
	}
	systray.Run(globalTray.onReady, globalTray.onExit)
}

func stopTray() {
	if globalTray == nil {
		return
	}
	globalTray.closeOnce.Do(func() {
		systray.Quit()
	})
}

func refreshTrayMenu() {
	if globalTray == nil {
		return
	}
	select {
	case globalTray.cmdCh <- trayCmd{kind: trayCmdRefresh}:
	default:
	}
}

func (t *trayManager) enqueueShow() {
	go t.showMainWindow()
}

func (t *trayManager) enqueueQuit() {
	select {
	case t.cmdCh <- trayCmd{kind: trayCmdQuit}:
	default:
	}
}

func (t *trayManager) enqueueSwitch(id string) {
	select {
	case t.cmdCh <- trayCmd{kind: trayCmdSwitch, payload: id}:
	default:
	}
}

func (t *trayManager) onReady() {
	systray.SetIcon(trayIconICO)
	systray.SetTitle("CodexSwitch")
	systray.SetTooltip("CodexSwitch - 账号与 API 配置管理")

	t.mCurrent = systray.AddMenuItem("当前配置：加载中…", "当前激活的配置")
	t.mCurrent.Disable()
	systray.AddSeparator()

	t.mOpen = systray.AddMenuItem("打开主窗口", "显示 CodexSwitch 主窗口")
	systray.AddSeparator()

	t.mSwitch = systray.AddMenuItem("快速切换", "切换到其他配置")
	systray.AddSeparator()

	t.mQuit = systray.AddMenuItem("退出", "完全退出 CodexSwitch")

	// 左键：直接恢复主窗口；右键：默认显示菜单
	systray.SetOnTapped(t.enqueueShow)

	go t.handleOpenClick()
	go t.handleQuitClick()
	go t.processCommands()

	t.doRefresh()
}

func (t *trayManager) onExit() {
	// systray 已退出，无需清理
}

func (t *trayManager) handleOpenClick() {
	for range t.mOpen.ClickedCh {
		t.enqueueShow()
	}
}

func (t *trayManager) handleQuitClick() {
	for range t.mQuit.ClickedCh {
		t.enqueueQuit()
	}
}

func (t *trayManager) processCommands() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("tray: processCommands panic: %v", r)
		}
	}()
	for cmd := range t.cmdCh {
		t.handleCommand(cmd)
	}
}

func (t *trayManager) handleCommand(cmd trayCmd) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("tray: handler panic: %v", r)
		}
	}()
	switch cmd.kind {
	case trayCmdRefresh:
		t.doRefresh()
	case trayCmdShow:
		t.showMainWindow()
	case trayCmdQuit:
		t.quitApp()
	case trayCmdSwitch:
		t.switchProfile(cmd.payload)
	}
}

func (t *trayManager) showMainWindow() {
	if showMainWindowNative() {
		return
	}
	ctx := t.app.runtimeContext()
	if ctx == nil {
		log.Println("tray: showMainWindow skipped, ctx not ready")
		return
	}
	log.Println("tray: showMainWindow -> WindowShow")
	wailsruntime.WindowShow(ctx)
}

func (t *trayManager) quitApp() {
	ctx := t.app.runtimeContext()
	if ctx == nil {
		return
	}
	t.app.allowNextClose.Store(true)
	wailsruntime.Quit(ctx)
}

func (t *trayManager) switchProfile(profileID string) {
	state, err := t.app.SwitchProfile(profileID)
	if err != nil {
		log.Printf("tray switch failed: %v", err)
	} else if state.Settings.RestartCodexAfterSwitch {
		if err := t.app.RestartCodex(); err != nil {
			log.Printf("tray restart codex failed: %v", err)
		}
	}
	t.doRefresh()
}

func (t *trayManager) doRefresh() {
	state, err := t.app.GetAppState()
	if err != nil {
		return
	}

	currentTitle := "当前配置：未检测到配置"
	switch {
	case state.Current.DisplayName != "":
		currentTitle = fmt.Sprintf("当前配置：%s", state.Current.DisplayName)
	case state.Current.Available:
		currentTitle = "当前配置：未托管"
	}
	t.mCurrent.SetTitle(currentTitle)

	for id, item := range t.quickItems {
		item.Hide()
		delete(t.quickItems, id)
	}

	profiles := append([]codexswitch.ProfileMeta{}, state.Profiles...)
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].DisplayName < profiles[j].DisplayName
	})

	hasItems := false
	for _, p := range profiles {
		if p.Disabled || p.IsActive {
			continue
		}
		title := p.DisplayName
		switch p.Type {
		case codexswitch.ProfileTypeOfficial:
			title = fmt.Sprintf("[官方] %s", p.DisplayName)
		case codexswitch.ProfileTypeAPI:
			title = fmt.Sprintf("[API] %s", p.DisplayName)
		}
		item := t.mSwitch.AddSubMenuItem(title, fmt.Sprintf("切换到 %s", p.DisplayName))
		t.quickItems[p.ID] = item

		id := p.ID
		go func(item *systray.MenuItem) {
			for range item.ClickedCh {
				t.enqueueSwitch(id)
			}
		}(item)
		hasItems = true
	}

	if !hasItems {
		empty := t.mSwitch.AddSubMenuItem("(无可切换配置)", "")
		empty.Disable()
	}
}
