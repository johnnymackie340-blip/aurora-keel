//go:build js

package main

import (
	"encoding/json"
	"syscall/js"
)

var (
	audioCtx js.Value
	master   js.Value
	thrustG  js.Value
	keep     []js.Func
)

func readSave() [5]int {
	var out [5]int
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return out
	}
	raw := ls.Call("getItem", "aurora-keel-v1")
	if raw.IsNull() || raw.IsUndefined() || raw.String() == "" {
		return out
	}
	var disk struct {
		Best [5]int `json:"best"`
	}
	if json.Unmarshal([]byte(raw.String()), &disk) == nil {
		return disk.Best
	}
	return out
}

func writeSave(best [5]int) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return
	}
	b, _ := json.Marshal(struct {
		Best [5]int `json:"best"`
	}{best})
	ls.Call("setItem", "aurora-keel-v1", string(b))
}

func resumeAudio() {
	if audioCtx.IsUndefined() || audioCtx.IsNull() {
		ctor := js.Global().Get("AudioContext")
		if ctor.IsUndefined() {
			ctor = js.Global().Get("webkitAudioContext")
		}
		if ctor.IsUndefined() {
			return
		}
		audioCtx = ctor.New()
		master = audioCtx.Call("createGain")
		master.Get("gain").Set("value", 0.35)
		master.Call("connect", audioCtx.Get("destination"))
		// cold drone
		for _, freq := range []float64{55, 82.5} {
			o := audioCtx.Call("createOscillator")
			g := audioCtx.Call("createGain")
			o.Set("type", "sine")
			o.Get("frequency").Set("value", freq)
			g.Get("gain").Set("value", 0.03)
			o.Call("connect", g)
			g.Call("connect", master)
			o.Call("start")
		}
		// thrust hiss
		n := 22050
		buf := audioCtx.Call("createBuffer", 1, n, 22050)
		data := buf.Call("getChannelData", 0)
		for i := 0; i < n; i++ {
			v := float64((i*1103515245+12345)&0xffff)/32768.0 - 1
			data.SetIndex(i, v*0.25)
		}
		src := audioCtx.Call("createBufferSource")
		src.Set("buffer", buf)
		src.Set("loop", true)
		filt := audioCtx.Call("createBiquadFilter")
		filt.Set("type", "lowpass")
		filt.Get("frequency").Set("value", 400)
		thrustG = audioCtx.Call("createGain")
		thrustG.Get("gain").Set("value", 0)
		src.Call("connect", filt)
		filt.Call("connect", thrustG)
		thrustG.Call("connect", master)
		src.Call("start")
	}
	audioCtx.Call("resume")
}

func tone(freq, dur, gain float64, kind string) {
	if audioCtx.IsUndefined() || audioCtx.IsNull() {
		return
	}
	o := audioCtx.Call("createOscillator")
	g := audioCtx.Call("createGain")
	o.Set("type", kind)
	o.Get("frequency").Set("value", freq)
	now := audioCtx.Get("currentTime").Float()
	g.Get("gain").Call("setValueAtTime", gain, now)
	g.Get("gain").Call("exponentialRampToValueAtTime", 0.0001, now+dur)
	o.Call("connect", g)
	g.Call("connect", master)
	o.Call("start", now)
	o.Call("stop", now+dur+0.02)
}

func playSFX(name string) {
	switch name {
	case "chime":
		tone(880, 0.18, 0.08, "sine")
		tone(1320, 0.28, 0.05, "sine")
	case "hit":
		tone(90, 0.22, 0.12, "triangle")
		tone(180, 0.12, 0.05, "square")
	case "hop":
		tone(420, 0.12, 0.05, "sine")
		tone(720, 0.16, 0.04, "sine")
	case "heal":
		tone(660, 0.1, 0.05, "sine")
		tone(990, 0.16, 0.04, "sine")
	case "win":
		tone(523, 0.16, 0.07, "sine")
		tone(659, 0.2, 0.06, "sine")
		tone(784, 0.34, 0.05, "sine")
	case "lose":
		tone(220, 0.3, 0.07, "triangle")
		tone(164, 0.4, 0.06, "sine")
	case "pause":
		tone(330, 0.08, 0.04, "sine")
	case "start":
		tone(392, 0.12, 0.05, "sine")
		tone(523, 0.2, 0.04, "sine")
	}
}

func (g *Game) publish() {
	h := js.Global().Get("Object").New()
	h.Set("mode", modeName(g.Mode))
	h.Set("chart", g.Chart)
	h.Set("chartName", g.C().Name)
	h.Set("hull", g.Hull)
	h.Set("time", g.TimeLeft)
	h.Set("score", g.Score)
	h.Set("buoys", g.buoysRung())
	h.Set("buoysNeed", len(g.buoys))
	h.Set("best", g.displayBest())
	arr := js.Global().Get("Array").New()
	for i := 0; i < 5; i++ {
		arr.SetIndex(i, bestScores[i])
	}
	h.Set("bests", arr)
	ready := 1.0
	if g.HopCD > 0 {
		ready = 1 - g.HopCD/1.2
		if ready < 0 {
			ready = 0
		}
	}
	h.Set("hop", ready)
	if g.MsgT > 0 {
		h.Set("msg", g.Msg)
	} else {
		h.Set("msg", "")
	}
	h.Set("endless", g.C().Endless)
	js.Global().Set("__keelHUD", h)
	if !thrustG.IsUndefined() && !thrustG.IsNull() {
		v := 0.0
		if g.Mode == ModePlay && g.Speed > 20 {
			v = 0.04 * (g.Speed / 148)
		}
		thrustG.Get("gain").Set("value", v)
	}
}

func main() {
	g := newGame()
	g.externalUI = true
	doc := js.Global().Get("document")
	win := js.Global()

	add := func(ev string, fn func(js.Value)) {
		cb := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				fn(args[0])
			}
			return nil
		})
		keep = append(keep, cb)
		win.Call("addEventListener", ev, cb)
	}
	add("keydown", func(e js.Value) {
		code := e.Get("code").String()
		if code == "Space" || code == "ArrowUp" || code == "ArrowDown" || code == "ArrowLeft" || code == "ArrowRight" {
			e.Call("preventDefault")
		}
		g.Keys[code] = true
	})
	add("keyup", func(e js.Value) {
		g.Keys[e.Get("code").String()] = false
	})
	add("blur", func(js.Value) {
		g.Keys = map[string]bool{}
	})

	reg := func(name string, fn func(this js.Value, args []js.Value) any) {
		cb := js.FuncOf(fn)
		keep = append(keep, cb)
		win.Set(name, cb)
	}
	reg("keelStart", func(this js.Value, args []js.Value) any {
		idx := 0
		if len(args) > 0 && args[0].Type() == js.TypeNumber {
			idx = args[0].Int()
		}
		resumeAudio()
		g.Start(idx)
		return nil
	})
	reg("keelAction", func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			resumeAudio()
			g.Action(args[0].String())
		}
		return nil
	})
	reg("keelTouch", func(this js.Value, args []js.Value) any {
		if len(args) >= 3 {
			g.touchThrust = args[0].Float()
			g.touchSteer = args[1].Float()
			g.touchHop = args[2].Bool()
		}
		return nil
	})
	reg("keelBlock", func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			g.blockMenu = args[0].Bool()
		}
		return nil
	})

	probe := win.Get("Object").New()
	regProbe := func(name string, fn func() any) {
		cb := js.FuncOf(func(this js.Value, args []js.Value) any { return fn() })
		keep = append(keep, cb)
		probe.Set(name, cb)
	}
	regProbe("getYaw", func() any { return g.Yaw })
	regProbe("getSpeed", func() any { return g.Speed })
	regProbe("getHull", func() any { return g.Hull })
	regProbe("getMode", func() any { return modeName(g.Mode) })
	regProbe("getX", func() any { return g.X })
	regProbe("getY", func() any { return g.Y })
	setKeys := js.FuncOf(func(this js.Value, args []js.Value) any {
		g.qa = map[string]bool{}
		g.qaOn = true
		if len(args) == 0 || args[0].IsNull() || args[0].IsUndefined() || args[0].Length() == 0 {
			g.qaOn = false
			return nil
		}
		arr := args[0]
		for i := 0; i < arr.Length(); i++ {
			g.qa[arr.Index(i).String()] = true
		}
		return nil
	})
	keep = append(keep, setKeys)
	probe.Set("setKeys", setKeys)
	win.Set("__controlsTest", probe)

	win.Set("keelReady", true)
	pend := win.Get("__keelPending")
	if pend.Type() == js.TypeNumber {
		resumeAudio()
		g.Start(pend.Int())
		win.Set("__keelPending", js.Null())
	}

	var last float64
	var raf js.Func
	raf = js.FuncOf(func(this js.Value, args []js.Value) any {
		t := 0.0
		if len(args) > 0 {
			t = args[0].Float()
		}
		dt := 1.0 / 60
		if last > 0 {
			dt = (t - last) / 1000
		}
		last = t
		g.Frame(dt)
		g.Render()
		canvas := doc.Call("getElementById", "keel")
		if !canvas.IsNull() && !canvas.IsUndefined() {
			ctx := canvas.Call("getContext", "2d", map[string]any{"alpha": false})
			img := ctx.Call("createImageData", W, H)
			js.CopyBytesToJS(img.Get("data"), g.pix)
			ctx.Call("putImageData", img, 0, 0)
		}
		g.publish()
		win.Call("requestAnimationFrame", raf)
		return nil
	})
	keep = append(keep, raf)
	win.Call("requestAnimationFrame", raf)
	g.publish()
	select {}
}
