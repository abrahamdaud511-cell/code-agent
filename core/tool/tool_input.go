//go:build windows

package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"time"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procSetCursorPos = user32.NewProc("SetCursorPos")
	procMouseEvent   = user32.NewProc("mouse_event")
	procKeybdEvent   = user32.NewProc("keybd_event")
	procVkKeyScan    = user32.NewProc("VkKeyScanA")
)

const (
	MOUSEEVENTF_LEFTDOWN  = 0x0002
	MOUSEEVENTF_LEFTUP    = 0x0004
	MOUSEEVENTF_RIGHTDOWN = 0x0008
	MOUSEEVENTF_RIGHTUP   = 0x0010
	MOUSEEVENTF_MIDDLEDOWN = 0x0020
	MOUSEEVENTF_MIDDLEUP  = 0x0040
	MOUSEEVENTF_WHEEL     = 0x0800

	KEYEVENTF_KEYUP = 0x0002

	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_MENU    = 0x12
	VK_RETURN  = 0x0D
	VK_TAB     = 0x09
	VK_SPACE   = 0x20
	VK_BACK    = 0x08
	VK_ESCAPE  = 0x1B
	VK_CAPITAL = 0x14
	VK_DELETE  = 0x2E
	VK_UP      = 0x26
	VK_DOWN    = 0x28
	VK_LEFT    = 0x25
	VK_RIGHT   = 0x27
	VK_HOME    = 0x24
	VK_END     = 0x23
	VK_PRIOR   = 0x21
	VK_NEXT    = 0x22
	VK_SNAPSHOT = 0x2C
	VK_INSERT  = 0x2D
	VK_LWIN    = 0x5B
	VK_F1      = 0x70
	VK_F2      = 0x71
	VK_F3      = 0x72
	VK_F4      = 0x73
	VK_F5      = 0x74
	VK_F6      = 0x75
	VK_F7      = 0x76
	VK_F8      = 0x77
	VK_F9      = 0x78
	VK_F10     = 0x79
	VK_F11     = 0x7A
	VK_F12     = 0x7B
)

type MouseMoveArgs struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MouseMoveTool struct{}

func NewMouseMoveTool() *MouseMoveTool { return &MouseMoveTool{} }

func (t *MouseMoveTool) Name() string { return "mouse_move" }

func (t *MouseMoveTool) Description() string {
	return "Move the mouse cursor to absolute screen coordinates (x, y). 0,0 is top-left corner."
}

func (t *MouseMoveTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"x": map[string]interface{}{"type": "integer", "description": "X coordinate (0 = left edge)"},
			"y": map[string]interface{}{"type": "integer", "description": "Y coordinate (0 = top edge)"},
		},
		"required": []string{"x", "y"},
	}
}

func (t *MouseMoveTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args MouseMoveArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	ret, _, _ := procSetCursorPos.Call(uintptr(args.X), uintptr(args.Y))
	if ret == 0 {
		return "", fmt.Errorf("SetCursorPos failed")
	}
	return fmt.Sprintf("Moved mouse to (%d, %d)", args.X, args.Y), nil
}

type MouseClickArgs struct {
	Button string `json:"button"`
	Double bool   `json:"double,omitempty"`
}

type MouseClickTool struct{}

func NewMouseClickTool() *MouseClickTool { return &MouseClickTool{} }

func (t *MouseClickTool) Name() string { return "mouse_click" }

func (t *MouseClickTool) Description() string {
	return "Click a mouse button at current position. Supports left, right, middle and double-click."
}

func (t *MouseClickTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"button": map[string]interface{}{
				"type":        "string",
				"description": "Mouse button to click",
				"enum":        []string{"left", "right", "middle"},
			},
			"double": map[string]interface{}{
				"type":        "boolean",
				"description": "Double-click (default: false)",
			},
		},
		"required": []string{"button"},
	}
}

func (t *MouseClickTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args MouseClickArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	var downFlag, upFlag uint32
	switch args.Button {
	case "left":
		downFlag, upFlag = MOUSEEVENTF_LEFTDOWN, MOUSEEVENTF_LEFTUP
	case "right":
		downFlag, upFlag = MOUSEEVENTF_RIGHTDOWN, MOUSEEVENTF_RIGHTUP
	case "middle":
		downFlag, upFlag = MOUSEEVENTF_MIDDLEDOWN, MOUSEEVENTF_MIDDLEUP
	default:
		return "", fmt.Errorf("unsupported button: %s (use left, right, or middle)", args.Button)
	}

	clicks := 1
	if args.Double {
		clicks = 2
	}

	for i := 0; i < clicks; i++ {
		procMouseEvent.Call(uintptr(downFlag), 0, 0, 0, 0)
		procMouseEvent.Call(uintptr(upFlag), 0, 0, 0, 0)
		if i == 0 && clicks == 2 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	label := "Clicked"
	if args.Double {
		label = "Double-clicked"
	}
	return fmt.Sprintf("%s %s button at current position", label, args.Button), nil
}

type MouseScrollArgs struct {
	Amount int `json:"amount"`
}

type MouseScrollTool struct{}

func NewMouseScrollTool() *MouseScrollTool { return &MouseScrollTool{} }

func (t *MouseScrollTool) Name() string { return "mouse_scroll" }

func (t *MouseScrollTool) Description() string {
	return "Scroll the mouse wheel. Positive amount scrolls up, negative scrolls down."
}

func (t *MouseScrollTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"amount": map[string]interface{}{
				"type":        "integer",
				"description": "Scroll amount (positive=up, negative=down). Each unit is ~1 notch.",
			},
		},
		"required": []string{"amount"},
	}
}

func (t *MouseScrollTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args MouseScrollArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	wheelData := uintptr(args.Amount * 120)
	procMouseEvent.Call(uintptr(MOUSEEVENTF_WHEEL), 0, 0, wheelData, 0)

	direction := "up"
	amount := args.Amount
	if amount < 0 {
		direction = "down"
		amount = -amount
	}
	return fmt.Sprintf("Scrolled %s by %d notches", direction, amount), nil
}

type KeyboardTypeArgs struct {
	Text  string `json:"text"`
	Delay int    `json:"delay,omitempty"`
}

type KeyboardTypeTool struct{}

func NewKeyboardTypeTool() *KeyboardTypeTool { return &KeyboardTypeTool{} }

func (t *KeyboardTypeTool) Name() string { return "keyboard_type" }

func (t *KeyboardTypeTool) Description() string {
	return "Type a string of text. Supports ASCII characters and common symbols."
}

func (t *KeyboardTypeTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"text": map[string]interface{}{"type": "string", "description": "Text to type"},
			"delay": map[string]interface{}{
				"type":        "integer",
				"description": "Delay between keystrokes in milliseconds (default: 10)",
			},
		},
		"required": []string{"text"},
	}
}

func (t *KeyboardTypeTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args KeyboardTypeArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Text == "" {
		return "", fmt.Errorf("text is required")
	}
	delay := args.Delay
	if delay <= 0 {
		delay = 10
	}

	for _, ch := range args.Text {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if err := typeChar(ch); err != nil {
			return "", fmt.Errorf("failed to type character '%c': %w", ch, err)
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	return fmt.Sprintf("Typed %d characters", len(args.Text)), nil
}

func typeChar(ch rune) error {
	ret, _, _ := procVkKeyScan.Call(uintptr(ch))
	if int32(ret) == -1 {
		return nil
	}

	vk := byte(ret & 0xFF)
	shiftState := byte((ret >> 8) & 0xFF)
	needsShift := shiftState&1 != 0

	if needsShift {
		procKeybdEvent.Call(VK_SHIFT, 0, 0, 0)
	}
	procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vk), 0, KEYEVENTF_KEYUP, 0)
	if needsShift {
		procKeybdEvent.Call(VK_SHIFT, 0, KEYEVENTF_KEYUP, 0)
	}

	return nil
}

type KeyboardPressArgs struct {
	Keys []string `json:"keys"`
}

type KeyboardPressTool struct{}

func NewKeyboardPressTool() *KeyboardPressTool { return &KeyboardPressTool{} }

func (t *KeyboardPressTool) Name() string { return "keyboard_press" }

func (t *KeyboardPressTool) Description() string {
	return "Press key combinations. Examples: [\"ctrl\",\"c\"] for copy, [\"alt\",\"tab\"] for window switch, [\"ctrl\",\"alt\",\"del\"]."
}

func (t *KeyboardPressTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"keys": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Keys to press together (e.g. [\"ctrl\",\"c\"])",
			},
		},
		"required": []string{"keys"},
	}
}

func (t *KeyboardPressTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args KeyboardPressArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if len(args.Keys) == 0 {
		return "", fmt.Errorf("keys is required")
	}

	vkCodes := make([]byte, 0, len(args.Keys))
	for _, key := range args.Keys {
		vk, ok := keyNameToVK(key)
		if !ok {
			return "", fmt.Errorf("unknown key: %s", key)
		}
		vkCodes = append(vkCodes, vk)
	}

	for _, vk := range vkCodes {
		procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
	}

	time.Sleep(30 * time.Millisecond)

	for i := len(vkCodes) - 1; i >= 0; i-- {
		procKeybdEvent.Call(uintptr(vkCodes[i]), 0, KEYEVENTF_KEYUP, 0)
	}

	return fmt.Sprintf("Pressed: %s", strings.Join(args.Keys, " + ")), nil
}

func keyNameToVK(name string) (byte, bool) {
	switch strings.ToLower(name) {
	case "enter", "return":
		return VK_RETURN, true
	case "tab":
		return VK_TAB, true
	case "space":
		return VK_SPACE, true
	case "backspace", "back":
		return VK_BACK, true
	case "escape", "esc":
		return VK_ESCAPE, true
	case "shift":
		return VK_SHIFT, true
	case "ctrl", "control":
		return VK_CONTROL, true
	case "alt":
		return VK_MENU, true
	case "delete", "del":
		return VK_DELETE, true
	case "up":
		return VK_UP, true
	case "down":
		return VK_DOWN, true
	case "left":
		return VK_LEFT, true
	case "right":
		return VK_RIGHT, true
	case "home":
		return VK_HOME, true
	case "end":
		return VK_END, true
	case "pageup", "pgup":
		return VK_PRIOR, true
	case "pagedown", "pgdn":
		return VK_NEXT, true
	case "printscreen", "prtsc":
		return VK_SNAPSHOT, true
	case "insert", "ins":
		return VK_INSERT, true
	case "win", "windows":
		return VK_LWIN, true
	case "capslock":
		return VK_CAPITAL, true
	case "f1":
		return VK_F1, true
	case "f2":
		return VK_F2, true
	case "f3":
		return VK_F3, true
	case "f4":
		return VK_F4, true
	case "f5":
		return VK_F5, true
	case "f6":
		return VK_F6, true
	case "f7":
		return VK_F7, true
	case "f8":
		return VK_F8, true
	case "f9":
		return VK_F9, true
	case "f10":
		return VK_F10, true
	case "f11":
		return VK_F11, true
	case "f12":
		return VK_F12, true
	default:
		if len(name) == 1 {
			ch := name[0]
			if ch >= 'a' && ch <= 'z' {
				return ch - 32, true
			}
			if ch >= 'A' && ch <= 'Z' {
				return ch, true
			}
			if ch >= '0' && ch <= '9' {
				return ch, true
			}
		}
		return 0, false
	}
}

var _ Tool = (*MouseMoveTool)(nil)
var _ Tool = (*MouseClickTool)(nil)
var _ Tool = (*MouseScrollTool)(nil)
var _ Tool = (*KeyboardTypeTool)(nil)
var _ Tool = (*KeyboardPressTool)(nil)
