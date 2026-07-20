package keyboard

import (
	"unsafe"
	"golang.org/x/sys/windows"
)

// Win32 constants
const (
	INPUT_KEYBOARD = 1
	KEYEVENTF_KEYUP = 0x0002
	VK_SHIFT = 0x10
	VK_CONTROL = 0x11
	VK_MENU = 0x12 // Alt
	VK_C = 0x43
	VK_V = 0x56
	VK_LWIN = 0x5B
	VK_RWIN = 0x5C
)

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type INPUT struct {
	Type uint32
	Ki   KEYBDINPUT
	// padding to make the struct size match the C union size
	_ uint64
}

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	procSendInput = user32.NewProc("SendInput")
)

func sendInput(inputs []INPUT) error {
	if len(inputs) == 0 {
		return nil
	}
	cbSize := unsafe.Sizeof(INPUT{})
	ret, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(cbSize),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func SimulateCopy() error {
	// Release any modifiers the user might be holding down
	inputs := []INPUT{
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_SHIFT, DwFlags: KEYEVENTF_KEYUP}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_MENU, DwFlags: KEYEVENTF_KEYUP}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_LWIN, DwFlags: KEYEVENTF_KEYUP}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_RWIN, DwFlags: KEYEVENTF_KEYUP}},
		// Ctrl Down, C Down, C Up, Ctrl Up
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_CONTROL}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_C}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_C, DwFlags: KEYEVENTF_KEYUP}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_CONTROL, DwFlags: KEYEVENTF_KEYUP}},
	}
	return sendInput(inputs)
}

func SimulatePaste() error {
	inputs := []INPUT{
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_SHIFT, DwFlags: KEYEVENTF_KEYUP}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_MENU, DwFlags: KEYEVENTF_KEYUP}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_LWIN, DwFlags: KEYEVENTF_KEYUP}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_RWIN, DwFlags: KEYEVENTF_KEYUP}},
		// Ctrl Down, V Down, V Up, Ctrl Up
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_CONTROL}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_V}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_V, DwFlags: KEYEVENTF_KEYUP}},
		{Type: INPUT_KEYBOARD, Ki: KEYBDINPUT{WVk: VK_CONTROL, DwFlags: KEYEVENTF_KEYUP}},
	}
	return sendInput(inputs)
}
