package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"


	"github.com/getlantern/systray"

	"nano-fixer/action"
	"nano-fixer/ai"
	"nano-fixer/config"
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

	serverPort  int
	serverToken string
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

	// Start local settings web server
	startSettingsServer()

	// Start Systray
	systray.Run(onReady, onExit)
}

func getConfig() *config.Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	// Return a copy to avoid data races
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

func generateToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "secret"
	}
	return hex.EncodeToString(bytes)
}

func startSettingsServer() {
	serverToken = generateToken()

	subFS, err := fs.Sub(settingsUI, "settings_ui")
	if err != nil {
		log.Fatalf("Failed to load embedded UI files: %v", err)
	}

	// Token validation middleware
	validateToken := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("token")
			if token != serverToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}

	// File Server with token validation (only on HTML page) and MIME type overrides for Windows Registry issues
	fileServer := http.FileServer(http.FS(subFS))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only validate token for the HTML page requests (root or index.html)
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			token := r.URL.Query().Get("token")
			if token != serverToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Explicitly set Content-Type header on Windows to bypass potential registry corruption
		ext := filepath.Ext(r.URL.Path)
		switch ext {
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		}

		fileServer.ServeHTTP(w, r)
	})

	// API Handlers
	http.HandleFunc("/api/config", validateToken(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			configMutex.RLock()
			// Return a copy with masked API Key so it's not exposed
			cfgCopy := *currentConfig
			if cfgCopy.APIKey != "" {
				cfgCopy.APIKey = "••••••••••••"
			}
			json.NewEncoder(w).Encode(cfgCopy)
			configMutex.RUnlock()
		} else if r.Method == http.MethodPost {
			var newCfg config.Config
			err := json.NewDecoder(r.Body).Decode(&newCfg)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Validate and Save
			configMutex.Lock()
			hotkeyChanged := currentConfig.HotkeyMod != newCfg.HotkeyMod || currentConfig.HotkeyKey != newCfg.HotkeyKey
			
			// If key is masked, keep the original key
			if newCfg.APIKey == "••••••••••••" {
				newCfg.APIKey = currentConfig.APIKey
			}

			currentConfig.APIKey = newCfg.APIKey
			currentConfig.APIBaseURL = newCfg.APIBaseURL
			currentConfig.ModelName = newCfg.ModelName
			currentConfig.HotkeyMod = newCfg.HotkeyMod
			currentConfig.HotkeyKey = newCfg.HotkeyKey
			currentConfig.TargetLanguage = newCfg.TargetLanguage
			currentConfig.Autostart = newCfg.Autostart

			err = config.Save(currentConfig)
			configMutex.Unlock()

			if err != nil {
				log.Printf("Failed to save config: %v", err)
				http.Error(w, "Failed to save config", http.StatusInternalServerError)
				return
			}

			// Update AI client configuration
			aiClient.UpdateConfig(newCfg.APIKey, newCfg.APIBaseURL, newCfg.ModelName)

			// Restart hotkey listener if hotkey changed
			if hotkeyChanged {
				log.Println("Hotkey settings changed. Restarting listener...")
				restartHotkeyListener()
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}
	}))

	http.HandleFunc("/api/logs", validateToken(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			exePath, err := os.Executable()
			if err == nil {
				logFilePath := filepath.Join(filepath.Dir(exePath), "app.log")
				go exec.Command("notepad.exe", logFilePath).Start()
			}
			w.WriteHeader(http.StatusOK)
		}
	}))

	// Bind to a random port on localhost only
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}

	serverPort = listener.Addr().(*net.TCPAddr).Port
	log.Printf("Settings server listening on http://127.0.0.1:%d\n", serverPort)

	go func() {
		if err := http.Serve(listener, nil); err != nil {
			log.Printf("HTTP Server error: %v", err)
		}
	}()
}

func openSettings() {
	url := fmt.Sprintf("http://127.0.0.1:%d/?token=%s", serverPort, serverToken)
	log.Println("Opening settings page...")
	go exec.Command("cmd", "/c", "start", url).Start()
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
