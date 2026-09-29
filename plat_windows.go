//go:build windows

package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")

	pRegisterClassExW   = user32.NewProc("RegisterClassExW")
	pCreateWindowExW    = user32.NewProc("CreateWindowExW")
	pDefWindowProcW     = user32.NewProc("DefWindowProcW")
	pPeekMessageW       = user32.NewProc("PeekMessageW")
	pTranslateMessage   = user32.NewProc("TranslateMessage")
	pDispatchMessageW   = user32.NewProc("DispatchMessageW")
	pPostQuitMessage    = user32.NewProc("PostQuitMessage")
	pDestroyWindow      = user32.NewProc("DestroyWindow")
	pLoadCursorW        = user32.NewProc("LoadCursorW")
	pGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	pSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	pGetDC              = user32.NewProc("GetDC")
	pGetClientRect      = user32.NewProc("GetClientRect")
	pFillRect           = user32.NewProc("FillRect")
	pBeginPaint         = user32.NewProc("BeginPaint")
	pEndPaint           = user32.NewProc("EndPaint")
	pShowWindow         = user32.NewProc("ShowWindow")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pStretchBlt         = gdi32.NewProc("StretchBlt")
	pDeleteDC           = gdi32.NewProc("DeleteDC")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pCreateSolidBrush   = gdi32.NewProc("CreateSolidBrush")
	pSetStretchBltMode  = gdi32.NewProc("SetStretchBltMode")

	pTimeBegin = winmm.NewProc("timeBeginPeriod")
	pWaveOpen  = winmm.NewProc("waveOutOpen")
	pWavePrep  = winmm.NewProc("waveOutPrepareHeader")
	pWaveWrite = winmm.NewProc("waveOutWrite")
	pWaveReset = winmm.NewProc("waveOutReset")
	pWaveClose = winmm.NewProc("waveOutClose")
)

func logLine(s string) {
	exe, err := os.Executable()
	dir := "."
	if err == nil {
		dir = filepath.Dir(exe)
	}
	f, err := os.OpenFile(filepath.Join(dir, "aurorakeel-log.txt"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(s + "\n")
	_ = f.Close()
}

func savePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "AuroraKeel", "scores.json")
}

func readSave() [5]int {
	var out [5]int
	b, err := os.ReadFile(savePath())
	if err != nil {
		return out
	}
	var disk struct {
		Best [5]int `json:"best"`
	}
	if json.Unmarshal(b, &disk) == nil {
		return disk.Best
	}
	return out
}

func writeSave(best [5]int) {
	p := savePath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	b, _ := json.Marshal(struct {
		Best [5]int `json:"best"`
	}{best})
	_ = os.WriteFile(p, b, 0o644)
}

const waveSamples = 1024

var (
	samplesA [waveSamples]int16
	samplesB [waveSamples]int16
	hwo      uintptr
	musicT   float64
	sfxN     [8]int
	noise    uint32 = 0x1234567
	haveWave bool
)

type wavefmt struct {
	tag      uint16
	channels uint16
	rate     uint32
	avg      uint32
	align    uint16
	bits     uint16
	size     uint16
}

type wavehdr struct {
	data     uintptr
	length   uint32
	recorded uint32
	user     uintptr
	flags    uint32
	loops    uint32
	next     uintptr
	reserved uintptr
}

var hdrA, hdrB wavehdr
var pinned runtime.Pinner

func mix(dst []int16) {
	for i := range dst {
		t := musicT
		musicT += 1.0 / 22050
		s := math.Sin(2*math.Pi*55*t)*0.05 + math.Sin(2*math.Pi*82.5*t)*0.028
		// soft pentatonic pulse, not a hymn
		step := int(t*1.5) % 5
		freqs := [...]float64{220, 261.6, 293.7, 329.6, 392}
		env := 1 - math.Mod(t*1.5, 1)
		s += math.Sin(2*math.Pi*freqs[step]*t) * env * env * 0.03
		if sfxN[0] > 0 { // chime
			u := float64(sfxN[0]) / 5000
			s += math.Sin(2*math.Pi*880*t) * u * 0.12
			s += math.Sin(2*math.Pi*1320*t) * u * 0.06
			sfxN[0]--
		}
		if sfxN[1] > 0 { // hit
			u := float64(sfxN[1]) / 4000
			noise = noise*1664525 + 1013904223
			n := float64(int32(noise)) / float64(1<<31)
			s += math.Sin(2*math.Pi*90*t)*u*0.18 + n*u*0.08
			sfxN[1]--
		}
		if sfxN[2] > 0 { // hop
			u := float64(sfxN[2]) / 3000
			s += math.Sin(2*math.Pi*(400+u*400)*t) * u * 0.08
			sfxN[2]--
		}
		if sfxN[3] > 0 { // heal / win-ish glass
			u := float64(sfxN[3]) / 5000
			s += math.Sin(2*math.Pi*660*t) * u * 0.08
			sfxN[3]--
		}
		if sfxN[4] > 0 {
			u := float64(sfxN[4]) / 8000
			s += math.Sin(2*math.Pi*523*t)*u*0.07 + math.Sin(2*math.Pi*784*t)*u*0.05
			sfxN[4]--
		}
		if sfxN[5] > 0 {
			u := float64(sfxN[5]) / 8000
			s += math.Sin(2*math.Pi*164*t) * u * 0.1
			sfxN[5]--
		}
		if s > 1 {
			s = 1
		}
		if s < -1 {
			s = -1
		}
		dst[i] = int16(s * 26000)
	}
}

func playSFX(name string) {
	switch name {
	case "chime":
		sfxN[0] = 5000
	case "hit":
		sfxN[1] = 4000
	case "hop":
		sfxN[2] = 2800
	case "heal", "pause", "start":
		sfxN[3] = 3600
	case "win":
		sfxN[4] = 8000
	case "lose":
		sfxN[5] = 8000
	}
}

func pumpAudio() {
	if !haveWave {
		return
	}
	const done = 0x00000001
	if hdrA.flags&done != 0 {
		mix(samplesA[:])
		hdrA.flags &^= done
		_, _, _ = pWaveWrite.Call(hwo, uintptr(unsafe.Pointer(&hdrA)), uintptr(unsafe.Sizeof(hdrA)))
	}
	if hdrB.flags&done != 0 {
		mix(samplesB[:])
		hdrB.flags &^= done
		_, _, _ = pWaveWrite.Call(hwo, uintptr(unsafe.Pointer(&hdrB)), uintptr(unsafe.Sizeof(hdrB)))
	}
}

func openAudio() {
	_, _, _ = pTimeBegin.Call(1)
	var fmt wavefmt
	fmt.tag = 1
	fmt.channels = 1
	fmt.rate = 22050
	fmt.bits = 16
	fmt.align = 2
	fmt.avg = 44100
	r, _, _ := pWaveOpen.Call(uintptr(unsafe.Pointer(&hwo)), uintptr(0xFFFFFFFF), uintptr(unsafe.Pointer(&fmt)), 0, 0, 0)
	if r != 0 {
		return
	}
	pinned.Pin(&samplesA)
	pinned.Pin(&samplesB)
	hdrA.data = uintptr(unsafe.Pointer(&samplesA[0]))
	hdrA.length = waveSamples * 2
	hdrB.data = uintptr(unsafe.Pointer(&samplesB[0]))
	hdrB.length = waveSamples * 2
	_, _, _ = pWavePrep.Call(hwo, uintptr(unsafe.Pointer(&hdrA)), uintptr(unsafe.Sizeof(hdrA)))
	_, _, _ = pWavePrep.Call(hwo, uintptr(unsafe.Pointer(&hdrB)), uintptr(unsafe.Sizeof(hdrB)))
	mix(samplesA[:])
	mix(samplesB[:])
	_, _, _ = pWaveWrite.Call(hwo, uintptr(unsafe.Pointer(&hdrA)), uintptr(unsafe.Sizeof(hdrA)))
	_, _, _ = pWaveWrite.Call(hwo, uintptr(unsafe.Pointer(&hdrB)), uintptr(unsafe.Sizeof(hdrB)))
	haveWave = true
}

type wndclass struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type winmsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type winrect struct{ left, top, right, bottom int32 }

type paint struct {
	hdc       uintptr
	erase     int32
	rc        winrect
	restore   int32
	incUpdate int32
	rgb       [32]byte
}

var (
	hwnd     uintptr
	keysDown [256]bool
	quitMsg  bool
	cbWnd    uintptr
)

func wndProc(h uintptr, msg uint32, wp, lp uintptr) uintptr {
	const (
		wmDestroy    = 0x0002
		wmClose      = 0x0010
		wmKeyDown    = 0x0100
		wmKeyUp      = 0x0101
		wmSysKeyDown = 0x0104
		wmSysKeyUp   = 0x0105
		wmKillFocus  = 0x0008
		wmErase      = 0x0014
		wmPaint      = 0x000F
	)
	switch msg {
	case wmKeyDown, wmSysKeyDown:
		if wp < 256 {
			keysDown[wp] = true
		}
		return 0
	case wmKeyUp, wmSysKeyUp:
		if wp < 256 {
			keysDown[wp] = false
		}
		return 0
	case wmKillFocus:
		for i := range keysDown {
			keysDown[i] = false
		}
		return 0
	case wmErase:
		return 1
	case wmPaint:
		var ps paint
		hdc, _, _ := pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		_, _, _ = pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		_ = hdc
		return 0
	case wmClose:
		_, _, _ = pDestroyWindow.Call(h)
		return 0
	case wmDestroy:
		quitMsg = true
		_, _, _ = pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, uintptr(msg), wp, lp)
	return r
}

func syncKeys(g *Game) {
	if g.qaOn {
		return
	}
	g.Keys = map[string]bool{}
	set := func(code string, vk int) {
		if keysDown[vk] {
			g.Keys[code] = true
		}
	}
	set("KeyW", 0x57)
	set("KeyA", 0x41)
	set("KeyS", 0x53)
	set("KeyD", 0x44)
	set("ArrowLeft", 0x25)
	set("ArrowUp", 0x26)
	set("ArrowRight", 0x27)
	set("ArrowDown", 0x28)
	set("Space", 0x20)
	set("Enter", 0x0D)
	set("Escape", 0x1B)
}

func present(pix []byte, memDC, winDC, brush uintptr) {
	// BGRA
	raw := unsafe.Slice((*byte)(bitsPtr), W*H*4)
	for i := 0; i < W*H; i++ {
		raw[i*4+0] = pix[i*4+2]
		raw[i*4+1] = pix[i*4+1]
		raw[i*4+2] = pix[i*4+0]
		raw[i*4+3] = 255
	}
	var rc winrect
	_, _, _ = pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	_, _, _ = pFillRect.Call(winDC, uintptr(unsafe.Pointer(&rc)), brush)
	cw := int(rc.right - rc.left)
	ch := int(rc.bottom - rc.top)
	if cw < 2 || ch < 2 {
		return
	}
	scale := float64(cw) / float64(W)
	if float64(ch)/float64(H) < scale {
		scale = float64(ch) / float64(H)
	}
	dw := int(float64(W) * scale)
	dh := int(float64(H) * scale)
	dx := (cw - dw) / 2
	dy := (ch - dh) / 2
	_, _, _ = pStretchBlt.Call(winDC, uintptr(dx), uintptr(dy), uintptr(dw), uintptr(dh), memDC, 0, 0, uintptr(W), uintptr(H), 0x00CC0020)
}

var bitsPtr unsafe.Pointer

func main() {
	runtime.LockOSThread()
	_, _, _ = pSetProcessDPIAware.Call()
	openAudio()

	g := newGame()
	hinst, _, _ := pGetModuleHandleW.Call(0)
	cls, _ := syscall.UTF16PtrFromString("AuroraKeelWindow")
	title, _ := syscall.UTF16PtrFromString("Aurora Keel")
	cursor, _, _ := pLoadCursorW.Call(0, 32512)
	cbWnd = syscall.NewCallback(wndProc)
	wc := wndclass{
		cbSize:        uint32(unsafe.Sizeof(wndclass{})),
		style:         3,
		lpfnWndProc:   cbWnd,
		hInstance:     hinst,
		hCursor:       cursor,
		lpszClassName: cls,
	}
	atom, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		logLine("RegisterClass failed: " + err.Error())
		return
	}
	hwnd, _, err = pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(title)),
		0x00CF0000|0x10000000,
		80, 40, 1100, 680,
		0, 0, hinst, 0,
	)
	if hwnd == 0 {
		logLine("CreateWindow failed: " + err.Error())
		return
	}
	_, _, _ = pShowWindow.Call(hwnd, 1)
	winDC, _, _ := pGetDC.Call(hwnd)
	memDC, _, _ := pCreateCompatibleDC.Call(winDC)
	_, _, _ = pSetStretchBltMode.Call(winDC, 3)
	type header struct {
		biSize          uint32
		biWidth         int32
		biHeight        int32
		biPlanes        uint16
		biBitCount      uint16
		biCompression   uint32
		biSizeImage     uint32
		biXPelsPerMeter int32
		biYPelsPerMeter int32
		biClrUsed       uint32
		biClrImportant  uint32
	}
	dib := header{biSize: 40, biWidth: W, biHeight: -H, biPlanes: 1, biBitCount: 32}
	hbmp, _, err := pCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&dib)), 0, uintptr(unsafe.Pointer(&bitsPtr)), 0, 0)
	if hbmp == 0 || bitsPtr == nil {
		logLine("DIB failed: " + err.Error())
		return
	}
	_, _, _ = pSelectObject.Call(memDC, hbmp)
	brush, _, _ := pCreateSolidBrush.Call(0x00160B07)

	last := time.Now()
	for !quitMsg && !g.quit {
		var m winmsg
		for {
			r, _, _ := pPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1)
			if r == 0 {
				break
			}
			if m.message == 0x0012 {
				quitMsg = true
				break
			}
			_, _, _ = pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			_, _, _ = pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
		now := time.Now()
		dt := now.Sub(last).Seconds()
		last = now
		syncKeys(g)
		g.Frame(dt)
		present(g.Render(), memDC, winDC, brush)
		pumpAudio()
		elapsed := time.Since(now)
		if elapsed < 15*time.Millisecond {
			time.Sleep(15*time.Millisecond - elapsed)
		}
	}
	if haveWave {
		_, _, _ = pWaveReset.Call(hwo)
		_, _, _ = pWaveClose.Call(hwo)
	}
	_, _, _ = pDeleteObject.Call(brush)
	_, _, _ = pDeleteObject.Call(hbmp)
	_, _, _ = pDeleteDC.Call(memDC)
}
