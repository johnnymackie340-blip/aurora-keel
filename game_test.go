package main

import (
	"math"
	"testing"
)

func TestControlsSigns(t *testing.T) {
	g := newGame()
	g.Start(0)
	g.qaOn = true
	g.qa = map[string]bool{"KeyW": true}
	for i := 0; i < 36; i++ {
		g.Frame(1.0 / 60)
	}
	if g.Speed <= 0.1 {
		t.Fatalf("W speed %v", g.Speed)
	}
	y0 := g.Yaw
	g.qa["KeyA"] = true
	for i := 0; i < 20; i++ {
		g.Frame(1.0 / 60)
	}
	if wrapAng(g.Yaw-y0) < 0.05 {
		t.Fatalf("A yaw %v -> %v", y0, g.Yaw)
	}

	g2 := newGame()
	g2.Start(0)
	g2.qaOn = true
	g2.qa = map[string]bool{"KeyW": true}
	for i := 0; i < 36; i++ {
		g2.Frame(1.0 / 60)
	}
	y0 = g2.Yaw
	g2.qa["KeyD"] = true
	for i := 0; i < 20; i++ {
		g2.Frame(1.0 / 60)
	}
	if wrapAng(g2.Yaw-y0) > -0.05 {
		t.Fatalf("D yaw %v -> %v", y0, g2.Yaw)
	}
}

func hazard(g *Game) bool {
	if g.whale.Active && g.whale.State >= 1 && g.whale.State <= 2 {
		if math.Hypot(g.X-g.whale.X, g.Y-g.whale.Y) < g.whale.R+6 {
			return true
		}
	}
	for _, ring := range g.rings {
		d := math.Hypot(g.X-ring.X, g.Y-ring.Y)
		if ring.Telegraph > 0 && ring.Telegraph < 0.4 && d < 26 {
			return true
		}
		if ring.Telegraph <= 0 && math.Abs(d-ring.R) < 18 {
			return true
		}
	}
	for _, h := range g.herons {
		if h.State == 1 && h.T < 0.28 && math.Hypot(g.X-h.TX, g.Y-h.TY) < 22 {
			return true
		}
		if h.State == 2 && math.Hypot(g.X-h.X, g.Y-h.Y) < 22 {
			return true
		}
	}
	return false
}

func drive(g *Game, tx, ty float64) {
	fx, fy, _, _ := heading(g.Yaw)
	ax, ay := tx, ty
	for _, f := range g.floes {
		x, y := floePos(f.FloeDef, g.T)
		dx, dy := x-g.X, y-g.Y
		d := math.Hypot(dx, dy)
		if d < 0.001 {
			continue
		}
		if d < f.R+34 {
			ax -= dx / d * 260
			ay -= dy / d * 260
		} else if d < f.R+90 && dx*fx+dy*fy > 0 {
			ax += -dy / d * 120
			ay += dx / d * 120
		}
	}
	dx, dy := ax-g.X, ay-g.Y
	want := math.Atan2(-dx, -dy)
	diff := wrapAng(want - g.Yaw)
	g.qa = map[string]bool{"KeyW": true}
	if diff > 0.12 {
		g.qa["KeyA"] = true
	} else if diff < -0.12 {
		g.qa["KeyD"] = true
	}
	if math.Hypot(dx, dy) < 36 && math.Abs(diff) > 0.8 {
		delete(g.qa, "KeyW")
	}
	if hazard(g) && g.HopCD <= 0 && g.HopT <= 0 && !g.prevHop {
		g.qa["Space"] = true
	}
}

func goal(g *Game) (float64, float64) {
	if !g.buoysDone() {
		if g.C().Ordered {
			n := g.nextOrder()
			for _, b := range g.buoys {
				if !b.On && b.Order == n {
					return b.X, b.Y
				}
			}
		}
		best := 1e18
		x, y := g.X, g.Y
		for _, b := range g.buoys {
			if b.On {
				continue
			}
			d := math.Hypot(b.X-g.X, b.Y-g.Y)
			if d < best {
				best = d
				x, y = b.X, b.Y
			}
		}
		return x, y
	}
	d := g.C().Dock
	return d.X + d.W/2, d.Y + d.H/2
}

func finishChart(t *testing.T, idx int, limit int) {
	t.Helper()
	g := newGame()
	g.Start(idx)
	g.qaOn = true
	for i := 0; i < limit; i++ {
		if g.Mode == ModeWin {
			t.Logf("chart %d won score %d hull %.0f t %.1f", idx, g.Score, g.Hull, g.C().Time-g.TimeLeft)
			return
		}
		if g.Mode == ModeLose {
			t.Fatalf("chart %d lost: %s at %.0f,%.0f hull %.0f buoys %d/%d", idx, g.Msg, g.X, g.Y, g.Hull, g.buoysRung(), len(g.buoys))
		}
		tx, ty := goal(g)
		drive(g, tx, ty)
		g.Frame(1.0 / 60)
	}
	t.Fatalf("chart %d timeout at %.0f,%.0f buoys %d/%d hull %.0f", idx, g.X, g.Y, g.buoysRung(), len(g.buoys), g.Hull)
}

func TestChartsReachable(t *testing.T) {
	for i := 0; i < 4; i++ {
		finishChart(t, i, 60*150)
	}
}

func TestEndlessSurvives(t *testing.T) {
	g := newGame()
	g.Start(4)
	g.qaOn = true
	g.qa = map[string]bool{"KeyW": true}
	for i := 0; i < 60*8; i++ {
		g.Frame(1.0 / 60)
		if g.Mode == ModeLose {
			t.Fatal(g.Msg, g.Hull)
		}
	}
	if g.Mode != ModePlay {
		t.Fatal(modeName(g.Mode))
	}
}

func TestFontRenders(t *testing.T) {
	g := newGame()
	for i := 0; i < 5; i++ {
		g.Frame(1.0 / 60)
	}
	pix := g.Render()
	if len(pix) != W*H*4 {
		t.Fatal(len(pix))
	}
	lit := 0
	for i := 0; i < len(pix); i += 4 {
		if pix[i] > 20 || pix[i+1] > 30 || pix[i+2] > 40 {
			lit++
		}
	}
	if lit < 1000 {
		t.Fatalf("frame looks empty: %d", lit)
	}
}
