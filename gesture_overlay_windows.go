//go:build windows

package main

import (
	"math"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	gestureOverlayWSPopup          = 0x80000000
	gestureOverlayWSExTopmost      = 0x00000008
	gestureOverlayWSExTransparent  = 0x00000020
	gestureOverlayWSExToolWindow   = 0x00000080
	gestureOverlayWSExLayered      = 0x00080000
	gestureOverlayWSExNoActivate   = 0x08000000
	gestureOverlayWMEraseBkgnd     = 0x0014
	gestureOverlayWMPaint          = 0x000F
	gestureOverlayWMNCHitTest      = 0x0084
	gestureOverlayWMMouseActivate  = 0x0021
	gestureOverlayHTTransparent    = ^uintptr(0)
	gestureOverlayMANoActivate     = 3
	gestureOverlayLWAColorKey      = 0x00000001
	gestureOverlaySWShowNA         = 8
	gestureOverlayTransparent      = 1
	gestureOverlayDTCalcRect       = 0x00000400
	gestureOverlayDTCenter         = 0x00000001
	gestureOverlayDTVCenter        = 0x00000004
	gestureOverlayDTSingleLine     = 0x00000020
	gestureOverlayDTNoPrefix       = 0x00000800
	gestureOverlayDTEndEllipsis    = 0x00008000
	gestureOverlayPSGeometric      = 0x00010000
	gestureOverlayBSSolid          = 0
	gestureOverlaySRCCopy          = 0x00CC0020
	gestureOverlayFontWeight       = 500
	gestureOverlayDefaultCharset   = 1
	gestureOverlayClearTypeQuality = 4

	gestureOverlayColorKey    = 0x00FF00FF
	gestureOverlayPathBorder  = 0x00947724 // RGB(36, 119, 148)，主题色深色外沿
	gestureOverlayPathMain    = 0x00DEA934 // RGB(52, 169, 222)，#34A9DE
	gestureOverlayLabelBg     = 0x00272422 // RGB(34, 36, 39)
	gestureOverlayLabelBorder = 0x005C5246 // RGB(70, 82, 92)
	gestureOverlayLabelText   = 0x00FFFFFF
)

var (
	gestureOverlayGDI32                          = windows.NewLazySystemDLL("gdi32.dll")
	gestureOverlayGDIPlus                        = windows.NewLazySystemDLL("gdiplus.dll")
	procGestureOverlayBeginPaint                 = user32.NewProc("BeginPaint")
	procGestureOverlayEndPaint                   = user32.NewProc("EndPaint")
	procGestureOverlayFillRect                   = user32.NewProc("FillRect")
	procGestureOverlayDrawTextW                  = user32.NewProc("DrawTextW")
	procGestureOverlaySetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procGestureOverlayGetDC                      = user32.NewProc("GetDC")
	procGestureOverlayReleaseDC                  = user32.NewProc("ReleaseDC")
	procGestureOverlayCreateSolidBrush           = gestureOverlayGDI32.NewProc("CreateSolidBrush")
	procGestureOverlayCreatePen                  = gestureOverlayGDI32.NewProc("CreatePen")
	procGestureOverlayExtCreatePen               = gestureOverlayGDI32.NewProc("ExtCreatePen")
	procGestureOverlayCreateCompatibleDC         = gestureOverlayGDI32.NewProc("CreateCompatibleDC")
	procGestureOverlayDeleteDC                   = gestureOverlayGDI32.NewProc("DeleteDC")
	procGestureOverlayCreateCompatibleBitmap     = gestureOverlayGDI32.NewProc("CreateCompatibleBitmap")
	procGestureOverlayBitBlt                     = gestureOverlayGDI32.NewProc("BitBlt")
	procGestureOverlaySelectObject               = gestureOverlayGDI32.NewProc("SelectObject")
	procGestureOverlayDeleteObject               = gestureOverlayGDI32.NewProc("DeleteObject")
	procGestureOverlayMoveToEx                   = gestureOverlayGDI32.NewProc("MoveToEx")
	procGestureOverlayLineTo                     = gestureOverlayGDI32.NewProc("LineTo")
	procGestureOverlaySetBkMode                  = gestureOverlayGDI32.NewProc("SetBkMode")
	procGestureOverlaySetTextColor               = gestureOverlayGDI32.NewProc("SetTextColor")
	procGestureOverlayRoundRect                  = gestureOverlayGDI32.NewProc("RoundRect")
	procGestureOverlayCreateFontW                = gestureOverlayGDI32.NewProc("CreateFontW")
	procGestureOverlayGDIFlush                   = gestureOverlayGDI32.NewProc("GdiFlush")
	procGestureOverlayGDIPlusStartup             = gestureOverlayGDIPlus.NewProc("GdiplusStartup")
	procGestureOverlayCreateGraphicsFromHDC      = gestureOverlayGDIPlus.NewProc("GdipCreateFromHDC")
	procGestureOverlayDeleteGraphics             = gestureOverlayGDIPlus.NewProc("GdipDeleteGraphics")
	procGestureOverlaySetSmoothingMode           = gestureOverlayGDIPlus.NewProc("GdipSetSmoothingMode")
	procGestureOverlaySetPixelOffsetMode         = gestureOverlayGDIPlus.NewProc("GdipSetPixelOffsetMode")
	procGestureOverlayCreatePen1                 = gestureOverlayGDIPlus.NewProc("GdipCreatePen1")
	procGestureOverlaySetPenStartCap             = gestureOverlayGDIPlus.NewProc("GdipSetPenStartCap")
	procGestureOverlaySetPenEndCap               = gestureOverlayGDIPlus.NewProc("GdipSetPenEndCap")
	procGestureOverlaySetPenLineJoin             = gestureOverlayGDIPlus.NewProc("GdipSetPenLineJoin")
	procGestureOverlayDrawLines                  = gestureOverlayGDIPlus.NewProc("GdipDrawLines")
	procGestureOverlayFlushGraphics              = gestureOverlayGDIPlus.NewProc("GdipFlush")
	procGestureOverlayDeletePen                  = gestureOverlayGDIPlus.NewProc("GdipDeletePen")
	gestureOverlayWndProcPtr                     = syscall.NewCallback(gestureOverlayWndProc)

	activeGestureOverlay       *nativeGestureOverlay
	gestureOverlayGDIPlusOnce  sync.Once
	gestureOverlayGDIPlusToken uintptr
)

type gestureOverlayPaintStruct struct {
	HDC       windows.Handle
	Erase     int32
	Paint     rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}

type gestureOverlayLogBrush struct {
	Style uint32
	Color uint32
	Hatch uintptr
}

type gestureOverlayGDIPlusStartupInput struct {
	Version                  uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}

type gestureOverlayPointF struct {
	X float32
	Y float32
}

type nativeGestureOverlay struct {
	hwnd windows.Handle
	mu   sync.Mutex

	bounds        rect
	visible       bool
	points        []GesturePoint
	label         string
	pathRect      rect
	labelRect     rect
	backDC        uintptr
	backBitmap    uintptr
	backOldBitmap uintptr
	bufferWidth   int32
	bufferHeight  int32
	dpiScale      float64
}

func createNativeGestureOverlay() *nativeGestureOverlay {
	className, err := windows.UTF16PtrFromString("KeyRunnerGestureOverlayWindow")
	if err != nil {
		return nil
	}
	instance, _, _ := procGetModuleHandleW.Call(0)
	wndClass := wndClassW{
		WndProc:   gestureOverlayWndProcPtr,
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	procRegisterClassW.Call(uintptr(unsafe.Pointer(&wndClass)))

	bounds := virtualScreenRect()
	hwnd, _, _ := procCreateWindowExW.Call(
		gestureOverlayWSExTopmost|
			gestureOverlayWSExTransparent|
			gestureOverlayWSExToolWindow|
			gestureOverlayWSExLayered|
			gestureOverlayWSExNoActivate,
		uintptr(unsafe.Pointer(className)),
		0,
		gestureOverlayWSPopup,
		uintptr(bounds.Left),
		uintptr(bounds.Top),
		uintptr(bounds.Right-bounds.Left),
		uintptr(bounds.Bottom-bounds.Top),
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		return nil
	}

	overlay := &nativeGestureOverlay{
		hwnd:   windows.Handle(hwnd),
		bounds: bounds,
	}
	activeGestureOverlay = overlay

	procGestureOverlaySetLayeredWindowAttributes.Call(
		hwnd,
		gestureOverlayColorKey,
		255,
		gestureOverlayLWAColorKey,
	)
	procSetWindowPos.Call(
		hwnd,
		^uintptr(0),
		uintptr(bounds.Left),
		uintptr(bounds.Top),
		uintptr(bounds.Right-bounds.Left),
		uintptr(bounds.Bottom-bounds.Top),
		swpNoActivate,
	)
	procShowWindow.Call(hwnd, gestureOverlaySWShowNA)

	overlay.mu.Lock()
	if overlay.ensureBackBufferLocked() {
		full := rect{Right: overlay.bufferWidth, Bottom: overlay.bufferHeight}
		overlay.clearRectLocked(full)
		overlay.presentRectLocked(full)
	}
	overlay.mu.Unlock()

	return overlay
}

func (o *nativeGestureOverlay) Destroy() {
	if o == nil || o.hwnd == 0 {
		return
	}

	o.mu.Lock()
	o.destroyBackBufferLocked()
	o.mu.Unlock()

	procDestroyWindow.Call(uintptr(o.hwnd))
	if activeGestureOverlay == o {
		activeGestureOverlay = nil
	}
	o.hwnd = 0
}

func (o *nativeGestureOverlay) Update(event GestureOverlayEvent) {
	if o == nil || o.hwnd == 0 {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	bounds := virtualScreenRect()
	boundsChanged := bounds != o.bounds
	if boundsChanged {
		o.bounds = bounds
		procSetWindowPos.Call(
			uintptr(o.hwnd),
			^uintptr(0),
			uintptr(bounds.Left),
			uintptr(bounds.Top),
			uintptr(bounds.Right-bounds.Left),
			uintptr(bounds.Bottom-bounds.Top),
			swpNoActivate,
		)
		o.destroyBackBufferLocked()
	}

	if !o.ensureBackBufferLocked() {
		return
	}

	client := rect{Right: o.bufferWidth, Bottom: o.bufferHeight}
	if boundsChanged {
		o.clearRectLocked(client)
		o.presentRectLocked(client)
		o.pathRect = rect{}
		o.labelRect = rect{}
		o.points = nil
		o.visible = false
	}

	if !event.Visible {
		if !o.visible {
			return
		}
		dirty := unionOverlayRect(o.pathRect, o.labelRect)
		if !overlayRectEmpty(dirty) {
			o.clearRectLocked(dirty)
			o.presentRectLocked(dirty)
		}
		o.visible = false
		o.points = o.points[:0]
		o.label = ""
		o.pathRect = rect{}
		o.labelRect = rect{}
		return
	}

	if len(event.Points) > 0 {
		last := event.Points[len(event.Points)-1]
		scale := dpiScaleAtPoint(point{X: int32(last.X) + bounds.Left, Y: int32(last.Y) + bounds.Top})
		if o.visible && !event.Reset && scale != o.dpiScale {
			// 跨缩放不同的显示器时重绘轨迹及文字，坐标保持贴合光标。
			points := append([]GesturePoint(nil), o.points...)
			event.Points = append(points, event.Points...)
			event.Reset = true
		}
		o.dpiScale = scale
	}
	if o.dpiScale <= 0 {
		o.dpiScale = 1
	}

	if event.Reset || !o.visible {
		// 置顶窗口可能后来进入 topmost Z-order 并盖住常驻 Overlay。
		// 手势开始时只提升一次，不在每个鼠标移动事件上调整 Z-order。
		procSetWindowPos.Call(
			uintptr(o.hwnd),
			^uintptr(0),
			0,
			0,
			0,
			0,
			swpNoMove|swpNoSize|swpNoActivate,
		)

		dirty := unionOverlayRect(o.pathRect, o.labelRect)
		if !overlayRectEmpty(dirty) {
			o.clearRectLocked(dirty)
		}
		o.points = append(o.points[:0], event.Points...)
		o.label = event.Label
		o.visible = true
		o.pathRect = rect{}
		o.labelRect = rect{}

		if len(o.points) >= 2 {
			drawGestureOverlayPath(o.backDC, o.points, o.dpiScale)
			o.pathRect = gestureOverlayPointsRect(o.points, scaleGestureOverlay(9, o.dpiScale))
			dirty = unionOverlayRect(dirty, o.pathRect)
		}
		if o.label != "" {
			o.labelRect = drawGestureOverlayLabel(
				o.backDC,
				o.labelAreaLocked(client),
				o.label,
				o.dpiScale,
			)
			dirty = unionOverlayRect(dirty, o.labelRect)
		}
		if !overlayRectEmpty(dirty) {
			o.presentRectLocked(dirty)
		}
		return
	}

	dirty := rect{}
	var newSegment []GesturePoint
	if len(event.Points) > 0 {
		if len(o.points) > 0 {
			newSegment = make([]GesturePoint, 0, len(event.Points)+1)
			newSegment = append(newSegment, o.points[len(o.points)-1])
			newSegment = append(newSegment, event.Points...)
		} else {
			newSegment = append(newSegment, event.Points...)
		}
		o.points = append(o.points, event.Points...)
		if len(o.points) > 1024 {
			o.points = append([]GesturePoint(nil), o.points[len(o.points)-1024:]...)
		}
	}

	if len(newSegment) >= 2 {
		// 增量绘制时，粗外沿的新端帽会覆盖上一小段主线。
		// 先画新增外沿，再把最近几段主线重新压回去，保证内层连续。
		drawGestureOverlayPathBorder(o.backDC, newSegment, o.dpiScale)

		tailStart := max(0, len(o.points)-12)
		mainTail := o.points[tailStart:]
		drawGestureOverlayPathMain(o.backDC, mainTail, o.dpiScale)

		segmentRect := gestureOverlayPointsRect(
			newSegment,
			scaleGestureOverlay(9, o.dpiScale),
		)
		tailRect := gestureOverlayPointsRect(
			mainTail,
			scaleGestureOverlay(6, o.dpiScale),
		)
		o.pathRect = unionOverlayRect(o.pathRect, segmentRect)
		dirty = unionOverlayRect(dirty, unionOverlayRect(segmentRect, tailRect))
	}

	labelChanged := event.Label != o.label
	if labelChanged {
		oldLabelRect := o.labelRect
		o.label = event.Label
		o.labelRect = rect{}
		newLabelRect := rect{}
		if o.label != "" {
			newLabelRect = gestureOverlayMeasureLabelRect(
				o.backDC,
				o.labelAreaLocked(client),
				o.label,
				o.dpiScale,
			)
		}
		labelDirty := unionOverlayRect(oldLabelRect, newLabelRect)
		if !overlayRectEmpty(labelDirty) {
			o.clearRectLocked(labelDirty)
			if len(o.points) >= 2 {
				drawGestureOverlayPath(o.backDC, o.points, o.dpiScale)
			}
			if o.label != "" {
				o.labelRect = drawGestureOverlayLabel(
					o.backDC,
					o.labelAreaLocked(client),
					o.label,
					o.dpiScale,
				)
			}
			dirty = unionOverlayRect(dirty, labelDirty)
		}
	} else if !overlayRectEmpty(o.labelRect) && overlayRectsIntersect(dirty, o.labelRect) {
		drawGestureOverlayLabel(
			o.backDC,
			o.labelAreaLocked(client),
			o.label,
			o.dpiScale,
		)
		dirty = unionOverlayRect(dirty, o.labelRect)
	}

	if !overlayRectEmpty(dirty) {
		o.presentRectLocked(dirty)
	}
}

func (o *nativeGestureOverlay) labelAreaLocked(client rect) rect {
	if len(o.points) == 0 {
		return client
	}

	global := point{
		X: int32(o.points[0].X) + o.bounds.Left,
		Y: int32(o.points[0].Y) + o.bounds.Top,
	}
	monitor, ok := monitorRectAtPoint(global)
	if !ok {
		return client
	}

	area := rect{
		Left:   monitor.Left - o.bounds.Left,
		Top:    monitor.Top - o.bounds.Top,
		Right:  monitor.Right - o.bounds.Left,
		Bottom: monitor.Bottom - o.bounds.Top,
	}
	return intersectOverlayRect(area, client)
}

func (o *nativeGestureOverlay) ensureBackBufferLocked() bool {
	width := o.bounds.Right - o.bounds.Left
	height := o.bounds.Bottom - o.bounds.Top
	if width <= 0 || height <= 0 {
		return false
	}
	if o.backDC != 0 &&
		o.backBitmap != 0 &&
		o.bufferWidth == width &&
		o.bufferHeight == height {
		return true
	}

	o.destroyBackBufferLocked()

	windowDC, _, _ := procGestureOverlayGetDC.Call(uintptr(o.hwnd))
	if windowDC == 0 {
		return false
	}
	defer procGestureOverlayReleaseDC.Call(uintptr(o.hwnd), windowDC)

	backDC, _, _ := procGestureOverlayCreateCompatibleDC.Call(windowDC)
	if backDC == 0 {
		return false
	}
	bitmap, _, _ := procGestureOverlayCreateCompatibleBitmap.Call(
		windowDC,
		uintptr(width),
		uintptr(height),
	)
	if bitmap == 0 {
		procGestureOverlayDeleteDC.Call(backDC)
		return false
	}

	oldBitmap, _, _ := procGestureOverlaySelectObject.Call(backDC, bitmap)
	o.backDC = backDC
	o.backBitmap = bitmap
	o.backOldBitmap = oldBitmap
	o.bufferWidth = width
	o.bufferHeight = height

	o.clearRectLocked(rect{Right: width, Bottom: height})
	return true
}

func (o *nativeGestureOverlay) destroyBackBufferLocked() {
	if o.backDC != 0 && o.backOldBitmap != 0 {
		procGestureOverlaySelectObject.Call(o.backDC, o.backOldBitmap)
	}
	if o.backBitmap != 0 {
		procGestureOverlayDeleteObject.Call(o.backBitmap)
	}
	if o.backDC != 0 {
		procGestureOverlayDeleteDC.Call(o.backDC)
	}
	o.backDC = 0
	o.backBitmap = 0
	o.backOldBitmap = 0
	o.bufferWidth = 0
	o.bufferHeight = 0
}

func (o *nativeGestureOverlay) clearRectLocked(value rect) {
	value = clampOverlayRect(value, rect{Right: o.bufferWidth, Bottom: o.bufferHeight})
	if overlayRectEmpty(value) {
		return
	}
	brush, _, _ := procGestureOverlayCreateSolidBrush.Call(gestureOverlayColorKey)
	if brush == 0 {
		return
	}
	procGestureOverlayFillRect.Call(
		o.backDC,
		uintptr(unsafe.Pointer(&value)),
		brush,
	)
	procGestureOverlayDeleteObject.Call(brush)
}

func (o *nativeGestureOverlay) presentRectLocked(value rect) {
	value = clampOverlayRect(value, rect{Right: o.bufferWidth, Bottom: o.bufferHeight})
	if overlayRectEmpty(value) {
		return
	}

	windowDC, _, _ := procGestureOverlayGetDC.Call(uintptr(o.hwnd))
	if windowDC == 0 {
		return
	}
	procGestureOverlayBitBlt.Call(
		windowDC,
		uintptr(value.Left),
		uintptr(value.Top),
		uintptr(value.Right-value.Left),
		uintptr(value.Bottom-value.Top),
		o.backDC,
		uintptr(value.Left),
		uintptr(value.Top),
		gestureOverlaySRCCopy,
	)
	procGestureOverlayGDIFlush.Call()
	procGestureOverlayReleaseDC.Call(uintptr(o.hwnd), windowDC)
}

func gestureOverlayWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case gestureOverlayWMEraseBkgnd:
		return 1
	case gestureOverlayWMNCHitTest:
		return gestureOverlayHTTransparent
	case gestureOverlayWMMouseActivate:
		return gestureOverlayMANoActivate
	case gestureOverlayWMPaint:
		paintNativeGestureOverlay(hwnd)
		return 0
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func paintNativeGestureOverlay(hwnd uintptr) {
	var paint gestureOverlayPaintStruct
	hdc, _, _ := procGestureOverlayBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
	if hdc == 0 {
		return
	}
	defer procGestureOverlayEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))

	overlay := activeGestureOverlay
	if overlay == nil {
		return
	}

	overlay.mu.Lock()
	defer overlay.mu.Unlock()
	if overlay.backDC == 0 {
		return
	}

	dirty := clampOverlayRect(
		paint.Paint,
		rect{Right: overlay.bufferWidth, Bottom: overlay.bufferHeight},
	)
	if overlayRectEmpty(dirty) {
		return
	}
	procGestureOverlayBitBlt.Call(
		hdc,
		uintptr(dirty.Left),
		uintptr(dirty.Top),
		uintptr(dirty.Right-dirty.Left),
		uintptr(dirty.Bottom-dirty.Top),
		overlay.backDC,
		uintptr(dirty.Left),
		uintptr(dirty.Top),
		gestureOverlaySRCCopy,
	)
}

func drawGestureOverlayPath(
	hdc uintptr,
	points []GesturePoint,
	dpiScale float64,
) {
	if len(points) < 2 {
		return
	}

	drawGestureOverlayPathBorder(hdc, points, dpiScale)
	drawGestureOverlayPathMain(hdc, points, dpiScale)
}

func drawGestureOverlayPathBorder(
	hdc uintptr,
	points []GesturePoint,
	dpiScale float64,
) {
	drawGestureOverlayLine(
		hdc,
		points,
		int(scaleGestureOverlay(6, dpiScale)),
		gestureOverlayPathBorder,
	)
}

func drawGestureOverlayPathMain(
	hdc uintptr,
	points []GesturePoint,
	dpiScale float64,
) {
	width := float32(scaleGestureOverlay(3, dpiScale))
	if drawGestureOverlayLineAA(hdc, points, width, 0xFF34A9DE) {
		return
	}
	drawGestureOverlayLine(
		hdc,
		points,
		int(width),
		gestureOverlayPathMain,
	)
}

func drawGestureOverlayLine(hdc uintptr, points []GesturePoint, width int, colour uintptr) {
	brush := gestureOverlayLogBrush{
		Style: gestureOverlayBSSolid,
		Color: uint32(colour),
	}
	pen, _, _ := procGestureOverlayExtCreatePen.Call(
		gestureOverlayPSGeometric,
		uintptr(width),
		uintptr(unsafe.Pointer(&brush)),
		0,
		0,
	)
	if pen == 0 {
		return
	}
	oldPen, _, _ := procGestureOverlaySelectObject.Call(hdc, pen)
	defer func() {
		procGestureOverlaySelectObject.Call(hdc, oldPen)
		procGestureOverlayDeleteObject.Call(pen)
	}()

	procGestureOverlayMoveToEx.Call(
		hdc,
		uintptr(int32(points[0].X)),
		uintptr(int32(points[0].Y)),
		0,
	)
	for _, value := range points[1:] {
		procGestureOverlayLineTo.Call(
			hdc,
			uintptr(int32(value.X)),
			uintptr(int32(value.Y)),
		)
	}
}

func drawGestureOverlayLineAA(
	hdc uintptr,
	points []GesturePoint,
	width float32,
	argb uint32,
) bool {
	if hdc == 0 || len(points) < 2 || width <= 0 || !ensureGestureOverlayGDIPlus() {
		return false
	}

	var graphics uintptr
	status, _, _ := procGestureOverlayCreateGraphicsFromHDC.Call(
		hdc,
		uintptr(unsafe.Pointer(&graphics)),
	)
	if status != 0 || graphics == 0 {
		return false
	}
	defer procGestureOverlayDeleteGraphics.Call(graphics)

	procGestureOverlaySetSmoothingMode.Call(graphics, 4)
	procGestureOverlaySetPixelOffsetMode.Call(graphics, 4)

	var pen uintptr
	status, _, _ = procGestureOverlayCreatePen1.Call(
		uintptr(argb),
		uintptr(math.Float32bits(width)),
		2,
		uintptr(unsafe.Pointer(&pen)),
	)
	if status != 0 || pen == 0 {
		return false
	}
	defer procGestureOverlayDeletePen.Call(pen)

	procGestureOverlaySetPenStartCap.Call(pen, 2)
	procGestureOverlaySetPenEndCap.Call(pen, 2)
	procGestureOverlaySetPenLineJoin.Call(pen, 2)

	gdiPoints := make([]gestureOverlayPointF, len(points))
	for i, value := range points {
		gdiPoints[i] = gestureOverlayPointF{
			X: float32(value.X),
			Y: float32(value.Y),
		}
	}

	status, _, _ = procGestureOverlayDrawLines.Call(
		graphics,
		pen,
		uintptr(unsafe.Pointer(&gdiPoints[0])),
		uintptr(len(gdiPoints)),
	)
	if status != 0 {
		return false
	}
	procGestureOverlayFlushGraphics.Call(graphics, 1)
	return true
}

func ensureGestureOverlayGDIPlus() bool {
	gestureOverlayGDIPlusOnce.Do(func() {
		input := gestureOverlayGDIPlusStartupInput{Version: 1}
		status, _, _ := procGestureOverlayGDIPlusStartup.Call(
			uintptr(unsafe.Pointer(&gestureOverlayGDIPlusToken)),
			uintptr(unsafe.Pointer(&input)),
			0,
		)
		if status != 0 {
			gestureOverlayGDIPlusToken = 0
		}
	})
	return gestureOverlayGDIPlusToken != 0
}

func gestureOverlayMeasureLabelRect(
	_ uintptr,
	area rect,
	_ string,
	dpiScale float64,
) rect {
	if overlayRectEmpty(area) {
		return rect{}
	}

	// 固定提示区域：动作名称变化时只更新文本，不再跟随文本长度改变尺寸。
	width := scaleGestureOverlay(168, dpiScale)
	height := scaleGestureOverlay(52, dpiScale)
	maxWidth := area.Right - area.Left - scaleGestureOverlay(24, dpiScale)
	if width > maxWidth {
		width = maxWidth
	}

	x := area.Left + (area.Right-area.Left-width)/2
	bottomOffset := scaleGestureOverlay(72, dpiScale)
	y := area.Bottom - bottomOffset - height
	topMargin := scaleGestureOverlay(16, dpiScale)
	if y < area.Top+topMargin {
		y = area.Top + topMargin
	}
	return rect{
		Left:   x,
		Top:    y,
		Right:  x + width,
		Bottom: y + height,
	}
}

func drawGestureOverlayLabel(
	hdc uintptr,
	area rect,
	label string,
	dpiScale float64,
) rect {
	box := gestureOverlayMeasureLabelRect(hdc, area, label, dpiScale)
	if overlayRectEmpty(box) {
		return rect{}
	}

	text, err := windows.UTF16FromString(label)
	if err != nil || len(text) <= 1 {
		return rect{}
	}

	brush, _, _ := procGestureOverlayCreateSolidBrush.Call(gestureOverlayLabelBg)
	pen, _, _ := procGestureOverlayCreatePen.Call(0, 1, gestureOverlayLabelBorder)
	oldBrush, _, _ := procGestureOverlaySelectObject.Call(hdc, brush)
	oldPen, _, _ := procGestureOverlaySelectObject.Call(hdc, pen)
	procGestureOverlayRoundRect.Call(
		hdc,
		uintptr(box.Left),
		uintptr(box.Top),
		uintptr(box.Right),
		uintptr(box.Bottom),
		uintptr(scaleGestureOverlay(10, dpiScale)),
		uintptr(scaleGestureOverlay(10, dpiScale)),
	)
	procGestureOverlaySelectObject.Call(hdc, oldBrush)
	procGestureOverlaySelectObject.Call(hdc, oldPen)
	if brush != 0 {
		procGestureOverlayDeleteObject.Call(brush)
	}
	if pen != 0 {
		procGestureOverlayDeleteObject.Call(pen)
	}

	font := createGestureOverlayFont(dpiScale)
	if font == 0 {
		return box
	}
	oldFont, _, _ := procGestureOverlaySelectObject.Call(hdc, font)
	defer func() {
		procGestureOverlaySelectObject.Call(hdc, oldFont)
		procGestureOverlayDeleteObject.Call(font)
	}()

	procGestureOverlaySetBkMode.Call(hdc, gestureOverlayTransparent)
	procGestureOverlaySetTextColor.Call(hdc, gestureOverlayLabelText)
	horizontalPadding := scaleGestureOverlay(10, dpiScale)
	verticalPadding := scaleGestureOverlay(4, dpiScale)
	textBox := rect{
		Left:   box.Left + horizontalPadding,
		Top:    box.Top + verticalPadding,
		Right:  box.Right - horizontalPadding,
		Bottom: box.Bottom - verticalPadding,
	}
	procGestureOverlayDrawTextW.Call(
		hdc,
		uintptr(unsafe.Pointer(&text[0])),
		uintptr(len(text)-1),
		uintptr(unsafe.Pointer(&textBox)),
		gestureOverlayDTCenter|
			gestureOverlayDTVCenter|
			gestureOverlayDTSingleLine|
			gestureOverlayDTNoPrefix|
			gestureOverlayDTEndEllipsis,
	)
	return box
}

func createGestureOverlayFont(dpiScale float64) uintptr {
	face, err := windows.UTF16PtrFromString("Segoe UI")
	if err != nil {
		return 0
	}
	fontHeight := -scaleGestureOverlay(20, dpiScale)
	font, _, _ := procGestureOverlayCreateFontW.Call(
		uintptr(uint32(fontHeight)),
		0,
		0,
		0,
		gestureOverlayFontWeight,
		0,
		0,
		0,
		gestureOverlayDefaultCharset,
		0,
		0,
		gestureOverlayClearTypeQuality,
		0,
		uintptr(unsafe.Pointer(face)),
	)
	return font
}

func gestureOverlayPointsRect(points []GesturePoint, padding int32) rect {
	if len(points) == 0 {
		return rect{}
	}

	left := int32(points[0].X)
	right := left
	top := int32(points[0].Y)
	bottom := top
	for _, value := range points[1:] {
		x := int32(value.X)
		y := int32(value.Y)
		if x < left {
			left = x
		}
		if x > right {
			right = x
		}
		if y < top {
			top = y
		}
		if y > bottom {
			bottom = y
		}
	}
	return rect{
		Left:   left - padding,
		Top:    top - padding,
		Right:  right + padding + 1,
		Bottom: bottom + padding + 1,
	}
}

func overlayRectEmpty(value rect) bool {
	return value.Right <= value.Left || value.Bottom <= value.Top
}

func unionOverlayRect(a, b rect) rect {
	if overlayRectEmpty(a) {
		return b
	}
	if overlayRectEmpty(b) {
		return a
	}
	if b.Left < a.Left {
		a.Left = b.Left
	}
	if b.Top < a.Top {
		a.Top = b.Top
	}
	if b.Right > a.Right {
		a.Right = b.Right
	}
	if b.Bottom > a.Bottom {
		a.Bottom = b.Bottom
	}
	return a
}

func intersectOverlayRect(a, b rect) rect {
	if b.Left > a.Left {
		a.Left = b.Left
	}
	if b.Top > a.Top {
		a.Top = b.Top
	}
	if b.Right < a.Right {
		a.Right = b.Right
	}
	if b.Bottom < a.Bottom {
		a.Bottom = b.Bottom
	}
	if overlayRectEmpty(a) {
		return rect{}
	}
	return a
}

func clampOverlayRect(value, bounds rect) rect {
	return intersectOverlayRect(value, bounds)
}

func overlayRectsIntersect(a, b rect) bool {
	if overlayRectEmpty(a) || overlayRectEmpty(b) {
		return false
	}
	return a.Left < b.Right &&
		a.Right > b.Left &&
		a.Top < b.Bottom &&
		a.Bottom > b.Top
}

func scaleGestureOverlay(value float64, dpiScale float64) int32 {
	return physicalPixels(value, dpiScale)
}
