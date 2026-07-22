package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/getlantern/systray"

	"nano-fixer/action"
	"nano-fixer/ai"
	"nano-fixer/config"
	"nano-fixer/gui"
	"nano-fixer/hotkey"
)

//go:embed settings_ui/*
var settingsUI embed.FS

//go:embed icon.ico
var iconBytes []byte

var (
	currentConfig *config.Config
	configMutex   sync.RWMutex
	aiClient      *ai.Client

	listenerCancel context.CancelFunc
	listenerDone   chan struct{}
	listenerMutex  sync.Mutex
)

func main() {
	// Initialize logger to file in executable directory
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	logFilePath := filepath.Join(filepath.Dir(exePath), "app.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Starting Nano Fixer...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	currentConfig = cfg

	// Initialize AI Client
	aiClient = ai.NewClient(cfg.APIKey, cfg.APIBaseURL, cfg.ModelName)

	// Start Hotkey Listener
	restartHotkeyListener()

	// Start Systray
	systray.Run(onReady, onExit)
}

func getConfig() *config.Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return &config.Config{
		APIKey:         currentConfig.APIKey,
		APIBaseURL:     currentConfig.APIBaseURL,
		ModelName:      currentConfig.ModelName,
		HotkeyMod:      currentConfig.HotkeyMod,
		HotkeyKey:      currentConfig.HotkeyKey,
		TargetLanguage: currentConfig.TargetLanguage,
		Autostart:      currentConfig.Autostart,
	}
}

func restartHotkeyListener() {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()

	if listenerCancel != nil {
		listenerCancel()
		<-listenerDone // Wait until it actually exits and unregisters!
		listenerCancel = nil
	}

	var ctx context.Context
	ctx, listenerCancel = context.WithCancel(context.Background())
	listenerDone = make(chan struct{})

	configMutex.RLock()
	mod := currentConfig.HotkeyMod
	key := currentConfig.HotkeyKey
	configMutex.RUnlock()

	go func() {
		defer close(listenerDone)
		err := hotkey.Listen(ctx, mod, key, func() {
			go action.RunCorrection(aiClient, getConfig)
		})
		if err != nil && ctx.Err() == nil {
			log.Printf("Hotkey listener failed: %v", err)
		}
	}()
}

func openSettings() {
	gui.ShowWebViewSettings(settingsUI, getConfig, func(newCfg config.Config) {
		configMutex.Lock()
		hotkeyChanged := currentConfig.HotkeyMod != newCfg.HotkeyMod || currentConfig.HotkeyKey != newCfg.HotkeyKey

		currentConfig.APIKey = newCfg.APIKey
		currentConfig.APIBaseURL = newCfg.APIBaseURL
		currentConfig.ModelName = newCfg.ModelName
		currentConfig.HotkeyMod = newCfg.HotkeyMod
		currentConfig.HotkeyKey = newCfg.HotkeyKey
		currentConfig.TargetLanguage = newCfg.TargetLanguage
		currentConfig.Autostart = newCfg.Autostart

		err := config.Save(currentConfig)
		configMutex.Unlock()

		if err != nil {
			log.Printf("Failed to save config: %v", err)
			return
		}

		// Update AI client configuration
		aiClient.UpdateConfig(newCfg.APIKey, newCfg.APIBaseURL, newCfg.ModelName)

		// Restart hotkey listener if hotkey changed
		if hotkeyChanged {
			log.Println("Hotkey settings changed. Restarting listener...")
			restartHotkeyListener()
		}
	})
}

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTitle("Nano Fixer")
	systray.SetTooltip("Nano Fixer - Background Grammar Corrector")

	mSettings := systray.AddMenuItem("Settings", "Configure API and Hotkeys")
	systray.AddSeparator()
	mExit := systray.AddMenuItem("Exit", "Quit application")

	go func() {
		for {
			select {
			case <-mSettings.ClickedCh:
				openSettings()
			case <-mExit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	log.Println("Exiting Nano Fixer...")
	listenerMutex.Lock()
	if listenerCancel != nil {
		listenerCancel()
	}
	listenerMutex.Unlock()
}
