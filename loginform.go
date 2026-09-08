package main

// Custom BerwinCode login window (pure Win32, no dependencies).
// Shows: instruction label, 6-digit code box, Send Code button,
// Login button, status label. Resend has a 60s cooldown, codes die
// after 5 minutes, nothing is ever saved: every launch needs a code.

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	idcSend  = 101
	idcLogin = 102

	wmCommand   = 0x0111
	wmClose     = 0x0010
	wmDestroy   = 0x0002
	wmTimer     = 0x0113
	wmSetFont   = 0x0030
	wmAppResult = 0x8001

	bnClicked = 0
	idOK      = 1
	idCancel  = 2

	wsChild       = 0x40000000
	wsVisible     = 0x10000000
	wsBorder      = 0x00800000
	wsTabStop     = 0x00010000
	esAutoHScroll = 0x0080
	esNumber      = 0x2000
	esCenter      = 0x0001
	emSetLimit    = 0x00C5
	emSetSel      = 0x00B1

	codeCooldown = 60 * time.Second
	codeExpiry   = 5 * time.Minute
)

var (
	modU32           = syscall.NewLazyDLL("user32.dll")
	modK32           = syscall.NewLazyDLL("kernel32.dll")
	modGdi           = syscall.NewLazyDLL("gdi32.dll")
	pRegisterClassEx = modU32.NewProc("RegisterClassExW")
	pCreateWindowEx  = modU32.NewProc("CreateWindowExW")
	pDefWindowProc   = modU32.NewProc("DefWindowProcW")
	pShowWindow      = modU32.NewProc("ShowWindow")
	pUpdateWindow    = modU32.NewProc("UpdateWindow")
	pDestroyWindow   = modU32.NewProc("DestroyWindow")
	pPostQuit        = modU32.NewProc("PostQuitMessage")
	pGetMessage      = modU32.NewProc("GetMessageW")
	pTranslateMsg    = modU32.NewProc("TranslateMessage")
	pDispatchMsg     = modU32.NewProc("DispatchMessageW")
	pIsDialogMsg     = modU32.NewProc("IsDialogMessageW")
	pSetTimer        = modU32.NewProc("SetTimer")
	pKillTimer       = modU32.NewProc("KillTimer")
	pSetText         = modU32.NewProc("SetWindowTextW")
	pGetText         = modU32.NewProc("GetWindowTextW")
	pSendMsg         = modU32.NewProc("SendMessageW")
	pPostMsg         = modU32.NewProc("PostMessageW")
	pEnableWindow    = modU32.NewProc("EnableWindow")
	pGetStockObj     = modGdi.NewProc("GetStockObject")
	pGetSysMetrics   = modU32.NewProc("GetSystemMetrics")
	pGetModuleHandle = modK32.NewProc("GetModuleHandleW")
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type winMsg struct {
	Hwnd    uintptr
	Message uint32
	_       uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	PtX     int32
	PtY     int32
	Private uint32
}

type verifyState struct {
	sync.Mutex
	code      string
	sentAt    time.Time
	sending   bool
	everSent  bool
	coolUntil time.Time
}

var (
	vstate     = &verifyState{}
	vwebhook   string
	formHwnd   uintptr
	editHwnd   uintptr
	sendHwnd   uintptr
	loginHwnd  uintptr
	statusHwnd uintptr
	footHwnd   uintptr
	formExit   = 1
	wndProcCb  = syscall.NewCallback(wndProc)
)

var keepAlive [][]uint16

func wstr(s string) *uint16 {
	b, _ := syscall.UTF16FromString(s)
	keepAlive = append(keepAlive, b)
	return &b[0]
}

func setText(hwnd uintptr, s string) {
	pSetText.Call(hwnd, uintptr(unsafe.Pointer(wstr(s))))
}

func enableCtl(hwnd uintptr, on bool) {
	v := uintptr(0)
	if on {
		v = 1
	}
	pEnableWindow.Call(hwnd, v)
}

func makeCtl(className, text string, style uintptr, x, y, w, h int32, parent uintptr, id uintptr, hInst uintptr) uintptr {
	r, _, _ := pCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(wstr(className))),
		uintptr(unsafe.Pointer(wstr(text))),
		style, uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, id, hInst, 0)
	return r
}

func flog(m string) {
	if p := os.Getenv("BERWINCODE_FORMLOG"); p != "" {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			fmt.Fprintln(f, m)
			f.Close()
		}
	}
}

// showLoginForm runs the modal login window. 0 = verified, 1 = give up.
func showLoginForm(webhook string) (code int) {
	code = 1
	flog("enter")
	defer func() {
		if r := recover(); r != nil {
			flog(fmt.Sprintf("PANIC: %v", r))
			code = 1
		}
	}()
	vwebhook = webhook
	vstate = &verifyState{}
	formExit = 1

	hInst, _, _ := pGetModuleHandle.Call(0)
	className := wstr("BerwinCodeLogin")
	wc := wndClassEx{Style: 3}
	wc.Size = uint32(unsafe.Sizeof(wc))
	wc.WndProc = wndProcCb
	wc.Instance = hInst
	cur, _, _ := modU32.NewProc("LoadCursorW").Call(0, 32512)
	wc.Cursor = cur
	wc.Background = 16
	wc.ClassName = className
	atom, _, _ := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	flog(fmt.Sprintf("class-atom=%d", atom))

	sw, _, _ := pGetSysMetrics.Call(0)
	sh, _, _ := pGetSysMetrics.Call(1)
	ww, wh := int32(380), int32(250)
	fx, fy := (int32(sw)-ww)/2, (int32(sh)-wh)/2

	formHwnd, _, _ = pCreateWindowEx.Call(0x00000001,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(wstr("BerwinCode Login"))),
		0x00CA0000, uintptr(fx), uintptr(fy), uintptr(ww), uintptr(wh),
		0, 0, hInst, 0)
	if formHwnd == 0 {
		le, _, _ := modK32.NewProc("GetLastError").Call()
		flog(fmt.Sprintf("window-failed lasterror=%d", le))
		return 1
	}
	flog(fmt.Sprintf("window=%d edit=%d send=%d login=%d status=%d", formHwnd, editHwnd, sendHwnd, loginHwnd, statusHwnd))

	font, _, _ := pGetStockObj.Call(17)
	place := func(hwnd uintptr) {
		pSendMsg.Call(hwnd, wmSetFont, font, 1)
	}
	base := uintptr(wsChild | wsVisible)
	infoH := makeCtl("STATIC", "Press Send Code, then type the 6-digit code from your Discord channel:", base, 16, 14, 332, 34, formHwnd, 0, hInst)
	editHwnd = makeCtl("EDIT", "", base|wsBorder|wsTabStop|esAutoHScroll|esNumber|esCenter, 16, 54, 332, 28, formHwnd, 0, hInst)
	sendHwnd = makeCtl("BUTTON", "Send Code", base|wsTabStop, 16, 94, 150, 30, formHwnd, idcSend, hInst)
	loginHwnd = makeCtl("BUTTON", "Login", base|wsTabStop|0x00000001, 198, 94, 150, 30, formHwnd, idcLogin, hInst)
	statusHwnd = makeCtl("STATIC", "", base, 16, 134, 332, 44, formHwnd, 0, hInst)
	note := updateNote
	if note == "" {
		note = "starting"
	}
	footHwnd = makeCtl("STATIC", "BerwinCode v"+berwinVersion+" - "+note, base, 16, 186, 332, 16, formHwnd, 0, hInst)
	le, _, _ := modK32.NewProc("GetLastError").Call()
	flog(fmt.Sprintf("ctl edit=%d send=%d login=%d status=%d lasterr=%d", editHwnd, sendHwnd, loginHwnd, statusHwnd, le))
	if editHwnd == 0 || sendHwnd == 0 {
		return 1
	}
	place(infoH)
	place(editHwnd)
	place(sendHwnd)
	place(loginHwnd)
	place(statusHwnd)
	place(footHwnd)
	flog("fonts-ok")
	pSendMsg.Call(editHwnd, emSetLimit, 6, 0)
	flog("limit-ok")

	pSetTimer.Call(formHwnd, 1, 1000, 0)
	flog("timer-ok")
	pShowWindow.Call(formHwnd, 5)
	flog("shown-ok")
	pUpdateWindow.Call(formHwnd)
	flog("updated-ok")
	visProc := modU32.NewProc("IsWindowVisible")
	vv, _, _ := visProc.Call(formHwnd)
	flog(fmt.Sprintf("visible=%d", vv))

	flog("loop-start")
	var m winMsg
	for {
		r1, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r1) == 0 {
			break
		}
		if int32(r1) == -1 {
			formExit = 1
			break
		}
		dr, _, _ := pIsDialogMsg.Call(formHwnd, uintptr(unsafe.Pointer(&m)))
		if dr != 0 {
			continue
		}
		pTranslateMsg.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMsg.Call(uintptr(unsafe.Pointer(&m)))
	}
	pKillTimer.Call(formHwnd, 1)
	return formExit
}

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case wmCommand:
		id := wParam & 0xFFFF
		notify := (wParam >> 16) & 0xFFFF
		if (notify == bnClicked && (id == idcSend || id == idcLogin)) || id == idOK {
			if id == idcSend {
				onSendCode()
			} else {
				onVerify()
			}
			return 0
		}
		if id == idCancel {
			endForm(1)
			return 0
		}
	case wmTimer:
		onTick()
		return 0
	case wmAppResult:
		onSendResult(wParam == 1)
		return 0
	case wmClose:
		endForm(1)
		return 0
	case wmDestroy:
		pPostQuit.Call(uintptr(formExit))
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wParam, lParam)
	return r
}

func endForm(code int) {
	formExit = code
	pDestroyWindow.Call(formHwnd)
}

func onSendCode() {
	vstate.Lock()
	if vstate.sending {
		vstate.Unlock()
		return
	}
	if time.Now().Before(vstate.coolUntil) {
		vstate.Unlock()
		return
	}
	vstate.sending = true
	vstate.Unlock()
	setText(statusHwnd, "Sending code to Discord...")
	enableCtl(sendHwnd, false)
	go doSendCode()
}

func genCode() string {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%900000+100000)
	}
	n := binary.BigEndian.Uint32(buf[:])%900000 + 100000
	return fmt.Sprintf("%06d", n)
}

func doSendCode() {
	code := genCode()
	msg := map[string]string{
		"content": "BerwinCode verification code: " + code + "\nIt expires in 5 minutes. If you did not ask for this, ignore it.",
	}
	data, _ := json.Marshal(msg)
	ok := false
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(vwebhook, "application/json", strings.NewReader(string(data)))
	if err == nil {
		resp.Body.Close()
		ok = resp.StatusCode >= 200 && resp.StatusCode < 300
	}
	vstate.Lock()
	vstate.sending = false
	if ok {
		vstate.code = code
		vstate.sentAt = time.Now()
		vstate.coolUntil = time.Now().Add(codeCooldown)
		vstate.everSent = true
	}
	vstate.Unlock()
	b := uintptr(0)
	if ok {
		b = 1
	}
	pPostMsg.Call(formHwnd, wmAppResult, b, 0)
}

func onSendResult(ok bool) {
	if ok {
		setText(statusHwnd, "Code sent! Check Discord. It expires in 5 minutes.")
	} else {
		setText(statusHwnd, "Send failed. Check internet and webhook, then retry.")
		enableCtl(sendHwnd, true)
	}
}

func onTick() {
	vstate.Lock()
	sending := vstate.sending
	ever := vstate.everSent
	remain := time.Until(vstate.coolUntil)
	code := vstate.code
	sentAt := vstate.sentAt
	vstate.Unlock()
	if sending {
		return
	}
	if code != "" && time.Since(sentAt) > codeExpiry {
		setText(statusHwnd, "Code expired. Press Resend Code for a new one.")
	}
	if remain > 0 {
		secs := int(remain.Seconds()) + 1
		setText(sendHwnd, fmt.Sprintf("Resend (%ds)", secs))
		enableCtl(sendHwnd, false)
		return
	}
	if ever {
		setText(sendHwnd, "Resend Code")
	} else {
		setText(sendHwnd, "Send Code")
	}
	enableCtl(sendHwnd, true)
}

func onVerify() {
	vstate.Lock()
	sending := vstate.sending
	code := vstate.code
	sentAt := vstate.sentAt
	vstate.Unlock()
	if sending {
		setText(statusHwnd, "Still sending, wait a moment.")
		return
	}
	if code == "" {
		setText(statusHwnd, "Press Send Code first.")
		return
	}
	if time.Since(sentAt) > codeExpiry {
		setText(statusHwnd, "Code expired. Press Resend Code.")
		return
	}
	var buf [16]uint16
	pGetText.Call(editHwnd, uintptr(unsafe.Pointer(&buf[0])), 16)
	typed := strings.TrimSpace(syscall.UTF16ToString(buf[:]))
	if typed == code {
		endForm(0)
		return
	}
	setText(statusHwnd, "Wrong code. Try again.")
	pSendMsg.Call(editHwnd, emSetSel, 0, 0xFFFFFFFF)
}
