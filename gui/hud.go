package gui

import (
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procUpdateWindow               = user32.NewProc("UpdateWindow")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procSendMessageW               = user32.NewProc("SendMessageW")
	procDestroyWindow              = user32.NewProc("DestroyWindow")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procPostQuitMessage            = user32.NewProc("PostQuitMessage")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procGetGUIThreadInfo           = user32.NewProc("GetGUIThreadInfo")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procClientToScreen             = user32.NewProc("ClientToScreen")
	procFillRect                   = user32.NewProc("FillRect")
	procCreateSolidBrush           = gdi32.NewProc("CreateSolidBrush")
	procSetTextColor               = gdi32.NewProc("SetTextColor")
	procSetBkMode                  = gdi32.NewProc("SetBkMode")
	procDrawTextW                  = user32.NewProc("DrawTextW")
	procBeginPaint                 = user32.NewProc("BeginPaint")
	procEndPaint                   = user32.NewProc("EndPaint")
	procGetStockObject             = gdi32.NewProc("GetStockObject")
	procSelectObject               = gdi32.NewProc("SelectObject")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	PfnWndProc    uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type MSG struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type GUITHREADINFO struct {
	CbSize        uint32
	Flags         uint32
	HwndActive    windows.Handle
	HwndFocus     windows.Handle
	HwndCapture   windows.Handle
	HwndMenuOwner windows.Handle
	HwndMoveSize  windows.Handle
	HwndCaret     windows.Handle
	RcCaret       RECT
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type POINT struct {
	X, Y int32
}

type PAINTSTRUCT struct {
	Hdc         windows.Handle
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

var hudHwnd windows.Handle

func getModuleHandle() windows.Handle {
	h, _, _ := procGetModuleHandleW.Call(0)
	return windows.Handle(h)
}

func ShowHUD(text string) {
	if hudHwnd != 0 {
		return
	}

	x, y := getHUDPosition()

	go func() {
		runtime.LockOSThread()

		hInstance := getModuleHandle()
		className, _ := syscall.UTF16PtrFromString("NanoFixerHUDClass")

		var wndClass WNDCLASSEXW
		wndClass.CbSize = uint32(unsafe.Sizeof(wndClass))
		wndClass.PfnWndProc = syscall.NewCallback(hudWndProc)
		wndClass.HInstance = hInstance
		wndClass.LpszClassName = className

		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))

		width, height := 160, 36
		hwnd, _, _ := procCreateWindowExW.Call(
			uintptr(0x00000008|0x00080000), // WS_EX_TOPMOST | WS_EX_LAYERED
			uintptr(unsafe.Pointer(className)),
			uintptr(0),
			uintptr(0x80000000|0x00800000), // WS_POPUP | WS_BORDER
			uintptr(x), uintptr(y-int32(height)-8), uintptr(width), uintptr(height),
			0, 0, uintptr(hInstance), 0,
		)

		if hwnd == 0 {
			return
		}

		// 220 is opacity (0-255)
		procSetLayeredWindowAttributes.Call(hwnd, 0, 220, 2) // LWA_ALPHA = 2

		hudHwnd = windows.Handle(hwnd)

		procShowWindow.Call(hwnd, 4) // SW_SHOWNOACTIVATE
		procUpdateWindow.Call(hwnd)

		var msg MSG
		for {
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if ret == 0 || ret == ^uintptr(0) {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}()
}

func HideHUD() {
	if hudHwnd != 0 {
		procSendMessageW.Call(uintptr(hudHwnd), 0x0010, 0, 0) // WM_CLOSE
		hudHwnd = 0
	}
}

func getHUDPosition() (int32, int32) {
	// Attempt GetGUIThreadInfo for caret position
	hwndFg, _, _ := procGetForegroundWindow.Call()
	if hwndFg != 0 {
		threadId, _, _ := procGetWindowThreadProcessId.Call(hwndFg, 0)
		var gti GUITHREADINFO
		gti.CbSize = uint32(unsafe.Sizeof(gti))
		ret, _, _ := procGetGUIThreadInfo.Call(threadId, uintptr(unsafe.Pointer(&gti)))
		if ret != 0 && gti.HwndCaret != 0 {
			pt := POINT{X: gti.RcCaret.Left, Y: gti.RcCaret.Top}
			procClientToScreen.Call(uintptr(gti.HwndCaret), uintptr(unsafe.Pointer(&pt)))
			if pt.X != 0 || pt.Y != 0 {
				return pt.X, pt.Y
			}
		}
	}

	// Fallback to Mouse Cursor
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt.X, pt.Y
}

func hudWndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case 0x000F: // WM_PAINT
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		// Background: #141720 (BGR: 0x201714)
		hbr, _, _ := procCreateSolidBrush.Call(uintptr(0x201714))
		var rect RECT
		rect.Right = 160
		rect.Bottom = 36
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rect)), hbr)

		procSetBkMode.Call(hdc, 1)                      // TRANSPARENT
		procSetTextColor.Call(hdc, uintptr(0xaaFF00))   // BGR for #00FFaa

		text, _ := syscall.UTF16PtrFromString("✨ AI is fixing...")
		fontRes, _, _ := procGetStockObject.Call(uintptr(17)) // DEFAULT_GUI_FONT
		procSelectObject.Call(hdc, fontRes)

		rect.Left = 14
		rect.Top = 10
		procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(text)), uintptr(18), uintptr(unsafe.Pointer(&rect)), 0)

		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		return 0
	case 0x0010: // WM_CLOSE
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case 0x0002: // WM_DESTROY
		procPostQuitMessage.Call(0)
		return 0
	}
	res, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return res
}
