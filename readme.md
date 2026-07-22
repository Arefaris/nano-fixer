# Nano Fixer

Nano Fixer is a background Windows utility that uses the OpenAI API to automatically correct grammar, fix typos, and improve text phrasing in any application.

## Features

- **Global Hotkey Integration**: Highlight text in any application (browser, text editor, word processor, etc.) and press your configured hotkey combination. The application automatically replaces the selected text with the corrected version.
- **Language Selection & Auto-Detection**: Can be configured to translate/correct to a specific language, or automatically detect and correct the text in its original language.
- **Background Execution**: Runs silently in the Windows system tray with a menu to access settings, view logs, or exit.
- **Windows Autostart**: Option to launch automatically when logging into Windows.
- **Credential Protection**: Uses Windows DPAPI (Data Protection API) to encrypt the OpenAI API key locally on disk. The key is decrypted only in memory during application startup.

## How It Works

1. When the configured hotkey is pressed, the application backup-copies the current clipboard content.
2. It simulates `Ctrl+C` to copy the currently highlighted text.
3. The selected text is sent to the OpenAI API with a system prompt optimized for grammar and style correction.
4. The application copies the corrected response to the clipboard and simulates `Ctrl+V` to replace the selected text.
5. The application restores the original clipboard content.

## Setup & Build

### Prerequisites
- [Go](https://go.dev/doc/install) (1.20+)
- Windows operating system

### Build Instructions

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yourusername/nano-fixer.git
   cd nano-fixer
   ```

2. **Compile resources:**
   Generate the Windows application manifest and compile the icon resource by running:
   ```bash
   go run scripts/generate_resources.go
   ```

3. **Build the executable:**
   To build the application as a background GUI utility (without opening a command prompt window), run:
   ```bash
   go build -ldflags "-H windowsgui" -o nano-fixer.exe
   ```

## Usage

1. Start `nano-fixer.exe`.
2. Right-click the Nano Fixer icon in the system tray and select **Settings**.
3. Enter your OpenAI API key and configure your preferences (API Base URL, Model Name, Hotkey, and Target Language).
4. Save the settings.
5. Select text in any application and press your hotkey to correct it.

## Security & Privacy

- **Native Windows GUI**: The settings interface opens directly as a native Windows dialog without running an HTTP web server or opening external browser tabs.
- **API Key Security**: The API key is stored encrypted in `config.json` via Windows DPAPI, protecting it from being read by unauthorized users or other machines.
- **No Third-Party Routing**: Text is sent directly to the OpenAI API endpoint specified in your settings. No third-party servers are involved.
- **Privacy-Safe Logs**: Application logs (`app.log`) do not store the text being corrected or your key details.
