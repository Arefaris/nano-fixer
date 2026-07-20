package hotkey

import (
	"context"
	"log"
	"strings"

	"golang.design/x/hotkey"
)

func parseModifier(modStr string) []hotkey.Modifier {
	mods := []hotkey.Modifier{}
	parts := strings.Split(strings.ToLower(modStr), "+")
	for _, p := range parts {
		switch p {
		case "ctrl":
			mods = append(mods, hotkey.ModCtrl)
		case "alt":
			mods = append(mods, hotkey.ModAlt)
		case "shift":
			mods = append(mods, hotkey.ModShift)
		case "win":
			mods = append(mods, hotkey.ModWin)
		}
	}
	return mods
}

func parseKey(keyStr string) hotkey.Key {
	// Simple mapping for single character letters A-Z
	if len(keyStr) == 1 {
		char := strings.ToUpper(keyStr)[0]
		if char >= 'A' && char <= 'Z' {
			// In golang.design/x/hotkey, keys are mapped to virtual key codes.
			// hotkey.KeyC is usually the ASCII value for letters
			return hotkey.Key(char)
		}
	}
	// Add more complex mappings if needed (e.g. F1-F12, Space, etc.)
	// Default to C if parsing fails
	return hotkey.Key('C')
}

func Listen(ctx context.Context, modStr, keyStr string, callback func()) error {
	mods := parseModifier(modStr)
	key := parseKey(keyStr)

	hk := hotkey.New(mods, key)
	err := hk.Register()
	if err != nil {
		return err
	}
	defer hk.Unregister()

	log.Printf("Listening for hotkey: %s+%s\n", modStr, keyStr)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-hk.Keydown():
			// Wait for keyup to avoid triggering while keys are still held
			<-hk.Keyup()
			log.Println("Hotkey triggered!")
			callback()
		}
	}
}
