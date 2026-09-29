package main

import "math"

var (
	colIce    = Color{231, 244, 255}
	colMint   = Color{94, 224, 181}
	colViolet = Color{168, 156, 255}
	colBrass  = Color{224, 177, 90}
	colRose   = Color{255, 120, 164}
)

func (g *Game) Render() []byte {
	if cap(g.pix) < W*H*4 {
		g.pix = make([]byte, W*H*4)
	}
	g.pix = g.pix[:W*H*4]
	r := &Raster{pix: g.pix, camX: g.CamX, camY: g.CamY}
	r.shX = math.Sin(g.clock*46) * g.Shake
	r.shY = math.Cos(g.clock*41) * g.Shake
	g.draw(r)
	return g.pix
}

func (g *Game) draw(r *Raster) {
	g.drawWater(r)
	g.drawStars(r)
	g.drawAurora(r)
	if g.menuish() {
		for _, f := range decorFloes {
			g.drawFloe(r, FloeRT{FloeDef: f}, g.T)
		}
		for _, b := range decorBuoys {
			g.drawBuoy(r, b.X, b.Y, true, false)
		}
		g.drawBoat(r, g.X, g.Y, g.Yaw, 0, math.Sin(g.T*0.45))
	} else {
		g.drawWorld(r)
	}
	frost := 0.0
	if g.Mode == ModePlay || g.Mode == ModePause || g.Mode == ModeLose {
		frost = g.frost()
	}
	r.borderVignette(frost)
	if g.Flash > 0 {
		r.fill(0, 0, W, H, colRose, g.Flash*1.4)
	}
	if g.Mode == ModePause || g.Mode == ModeWin || g.Mode == ModeLose {
		r.fill(0, 0, W, H, Color{3, 6, 14}, 0.45)
	}
	g.drawChrome(r)
}

func (g *Game) frost() float64 {
	lim := g.C().Time
	if lim <= 0 {
		return 0
	}
	u := 1 - g.TimeLeft/lim
	if u < 0.42 {
		return 0
	}
	return (u - 0.42) / 0.58
}

func (g *Game) drawWater(r *Raster) {
	for y := 0; y < H; y++ {
		u := float64(y) / float64(H-1)
		wave := math.Sin(float64(y)*0.08+g.clock*1.25) * 0.035
		c := lerpC(Color{6, 16, 32}, Color{12, 46, 72}, clamp(u+0.08+wave, 0, 1))
		b1 := math.Sin(float64(y)*0.05 + g.clock*0.75)
		if b1 > 0.55 {
			c = lerpC(c, Color{28, 96, 98}, (b1-0.55)*0.55)
		}
		b2 := math.Sin(float64(y)*0.033 - g.clock*0.4 + 1.7)
		if b2 > 0.62 {
			c = lerpC(c, Color{62, 52, 120}, (b2-0.62)*0.7)
		}
		r.hline(y, c)
	}
	// drifting foam, world anchored
	for i := 0; i < 18; i++ {
		wx := math.Floor((g.CamX-80)/90)*90 + float64(i%6)*90 + 20
		wy := math.Floor((g.CamY-60)/70)*70 + float64(i/6)*70
		wy += math.Mod(g.clock*12+float64(i)*9, 70)
		r.wline(wx, wy, wx+18, wy+3, 1.2, colIce, 0.13)
	}
}

func (g *Game) drawStars(r *Raster) {
	for i := 0; i < 32; i++ {
		sx := int(hash01(i*17+3) * float64(W-4))
		sy := int(hash01(i*29+5) * float64(H) * 0.62)
		tw := 0.25 + 0.75*(0.5+0.5*math.Sin(g.clock*1.6+float64(i)))
		r.blend(sx, sy, colIce, tw*0.75)
		if i%4 == 0 {
			r.blend(sx+1, sy, colIce, tw*0.35)
		}
	}
}

func (g *Game) drawAurora(r *Raster) {
	cols := []Color{colMint, colViolet, {190, 255, 226}}
	for i, col := range cols {
		var prev [2]float64
		for x := 0; x <= W; x += 8 {
			xf := float64(x)
			y := 22 + float64(i)*14 + math.Sin(xf*0.018+g.clock*0.55+float64(i))*12 + math.Sin(xf*0.007-g.clock*0.22)*8
			p := [2]float64{xf, y}
			if x > 0 {
				r.line(prev[0], prev[1], p[0], p[1], 8-float64(i)*1.6, col, 0.22)
			}
			prev = p
		}
	}
}

func (g *Game) drawWorld(r *Raster) {
	if g.C().Whale && g.whale.State > 0 {
		r.wcircle(g.whale.X, g.whale.Y, 80, Color{8, 20, 36}, 0.35)
	}
	if g.C().Rings {
		r.wcircle(g.X*0.15+g.CamX*0.1, g.Y, 90, Color{6, 16, 30}, 0.22)
	}
	for _, k := range g.krill {
		for i := 0; i < 7; i++ {
			a := g.clock*1.6 + float64(i)
			r.wcircle(k.X+math.Cos(a)*k.R*0.55, k.Y+math.Sin(a*1.3)*k.R*0.55, 1.7, Color{150, 206, 255}, 0.75)
		}
		r.wcircle(k.X, k.Y, k.R, Color{80, 140, 180}, 0.06)
	}
	if g.dockOpen() || g.C().Whale {
		g.drawDock(r)
	}
	for _, f := range g.floes {
		g.drawFloe(r, f, g.T)
	}
	for _, v := range g.vials {
		g.drawVial(r, v.X, v.Y)
	}
	next := g.nextOrder()
	for _, b := range g.buoys {
		hot := !b.On && (!g.C().Ordered || b.Order == next)
		g.drawBuoy(r, b.X, b.Y, hot, b.On)
	}
	for i := range g.rings {
		g.drawRing(r, &g.rings[i])
	}
	if g.whale.Active && g.whale.State > 0 {
		g.drawWhaleMark(r)
	}
	for i := range g.herons {
		g.drawHeron(r, &g.herons[i])
	}
	for _, p := range g.parts {
		a := p.Life / p.Max
		col := colIce
		if p.Kind == 2 {
			col = colMint
		} else if p.Kind == 1 {
			col = colRose
		}
		r.wcircle(p.X, p.Y, p.Sz, col, a*0.85)
	}
	steer := 0.0
	if g.down("KeyA") || g.down("ArrowLeft") || (!g.qaOn && g.touchSteer > 0.2) {
		steer += 1
	}
	if g.down("KeyD") || g.down("ArrowRight") || (!g.qaOn && g.touchSteer < -0.2) {
		steer -= 1
	}
	g.drawBoat(r, g.X, g.Y, g.Yaw, g.HopT, steer)
}

func (g *Game) drawFloe(r *Raster, f FloeRT, t float64) {
	x, y := floePos(f.FloeDef, t)
	pts := make([][2]float64, 7)
	for i := 0; i < 7; i++ {
		a := float64(i)/7*math.Pi*2 - 0.4
		jag := 0.8 + 0.2*hash01(int(f.X)*5+int(f.Y)*3+i*13)
		pts[i] = [2]float64{x + math.Cos(a)*f.R*jag, y + math.Sin(a)*f.R*jag}
	}
	r.wpoly(pts, Color{176, 208, 224}, 1)
	in := make([][2]float64, 7)
	for i := 0; i < 7; i++ {
		in[i] = [2]float64{x + (pts[i][0]-x)*0.62, y + (pts[i][1]-y)*0.62}
	}
	r.wpoly(in, Color{214, 236, 246}, 1)
	r.wline(x-f.R*0.28, y+f.R*0.05, x+f.R*0.32, y-f.R*0.08, 1.1, Color{120, 156, 176}, 0.8)
	if f.Gate {
		r.wline(x-f.R*0.4, y, x+f.R*0.4, y, 1.4, colViolet, 0.7)
	}
}

func (g *Game) drawBuoy(r *Raster, x, y float64, hot, done bool) {
	r.wcircle(x+1, y+4, 4.5, Color{0, 0, 0}, 0.28)
	r.wline(x, y+7, x, y-8, 2.1, Color{122, 84, 36}, 1)
	col := Color{70, 130, 142}
	a := 0.55
	pulse := 0.5 + 0.5*math.Sin(g.clock*4)
	if hot {
		col = colMint
		a = 0.4 + 0.55*pulse
		r.wcircle(x, y-11, 9+pulse*2, colMint, 0.14)
	}
	if done {
		col = colBrass
		a = 0.95
	}
	r.wcircle(x, y-11, 5.2, col, a)
	r.wcircle(x, y-11, 2.1, colIce, a*0.9)
}

func (g *Game) drawVial(r *Raster, x, y float64) {
	a := g.clock*2 + x*0.01
	c, s := math.Cos(a), math.Sin(a)
	r.wpoly([][2]float64{
		{x + c*5, y + s*5},
		{x - s*3.2, y + c*3.2},
		{x - c*5, y - s*5},
		{x + s*3.2, y - c*3.2},
	}, colMint, 0.95)
}

func (g *Game) drawDock(r *Raster) {
	d := g.C().Dock
	open := g.dockOpen()
	wood := Color{42, 58, 72}
	if open {
		wood = Color{58, 72, 78}
	}
	r.wpoly([][2]float64{
		{d.X, d.Y}, {d.X + d.W, d.Y}, {d.X + d.W, d.Y + d.H}, {d.X, d.Y + d.H},
	}, wood, 1)
	r.wline(d.X, d.Y, d.X+d.W, d.Y, 2, colBrass, 0.9)
	for _, px := range []float64{d.X + 14, d.X + d.W - 14} {
		r.wline(px, d.Y+d.H, px, d.Y-14, 2.2, Color{122, 84, 36}, 1)
		r.wcircle(px, d.Y-18, 4.2, colMint, 0.35+0.25*math.Sin(g.clock*3))
		r.wcircle(px, d.Y-18, 2, colIce, 0.9)
	}
}

func (g *Game) drawRing(r *Raster, ring *Ring) {
	if ring.Telegraph > 0 {
		a := 0.25 + 0.35*math.Sin(g.clock*18)
		g.strokeCircle(r, ring.X, ring.Y, 18+(0.78-ring.Telegraph)*10, 1.6, colViolet, a)
		return
	}
	g.strokeCircle(r, ring.X, ring.Y, ring.R, 2.4, colIce, 0.55)
	g.strokeCircle(r, ring.X, ring.Y, ring.R-5, 1.2, colMint, 0.25)
}

func (g *Game) drawWhaleMark(r *Raster) {
	w := g.whale
	a := 0.35
	if w.State == 2 {
		a = 0.55 + 0.4*math.Sin(g.clock*22)
	}
	g.strokeCircle(r, w.X, w.Y, w.R, 2.2, colIce, a)
	r.wcircle(w.X, w.Y, 8, colIce, 0.25)
}

func (g *Game) strokeCircle(r *Raster, x, y, rad, thick float64, c Color, a float64) {
	if rad < 2 {
		return
	}
	segs := int(rad*0.45) + 14
	if segs > 64 {
		segs = 64
	}
	for i := 0; i < segs; i++ {
		a0 := float64(i) / float64(segs) * math.Pi * 2
		a1 := float64(i+1) / float64(segs) * math.Pi * 2
		r.wline(x+math.Cos(a0)*rad, y+math.Sin(a0)*rad, x+math.Cos(a1)*rad, y+math.Sin(a1)*rad, thick, c, a)
	}
}

func (g *Game) drawHeron(r *Raster, h *Heron) {
	sx, sy := h.X, h.Y
	if h.State == 1 {
		sx, sy = h.TX, h.TY
	}
	r.wcircle(sx, sy, 5.5, Color{0, 0, 0}, 0.32)
	lift := 0.0
	if h.State == 1 {
		lift = (0.72 - h.T) * 16
	}
	if h.State == 2 {
		lift = h.T * 22
	}
	bx, by := h.X, h.Y-lift
	flap := math.Sin(g.clock*9) * 5
	if h.State == 2 {
		flap = 1
	}
	r.wpoly([][2]float64{{bx, by}, {bx - 12, by - flap}, {bx - 3, by + 1}}, Color{186, 206, 220}, 1)
	r.wpoly([][2]float64{{bx, by}, {bx + 12, by - flap}, {bx + 3, by + 1}}, Color{160, 186, 206}, 1)
	r.wcircle(bx, by, 3.1, Color{214, 226, 234}, 1)
	r.wline(bx, by, bx+4, by-3, 1.2, colBrass, 1)
}

func (g *Game) drawBoat(r *Raster, x, y, yaw, hop, steer float64) {
	fx, fy, rx, ry := heading(yaw)
	x += rx * steer * 1.4
	y += ry * steer * 1.4
	sh := 4.5
	if hop > 0 {
		sh = 11
	}
	r.wcircle(x+1.5, y+sh, 7.5, Color{0, 0, 0}, 0.38)
	if g.Speed > 22 || g.menuish() {
		for i := 1; i <= 4; i++ {
			d := float64(i) * 6.5
			r.wcircle(x-fx*d, y-fy*d, 2.2+float64(i)*0.35, colIce, 0.22*(1-float64(i)/5))
		}
	}
	nose := [2]float64{x + fx*16, y + fy*16}
	port := [2]float64{x - fx*11 - rx*6.4, y - fy*11 - ry*6.4}
	star := [2]float64{x - fx*11 + rx*6.4, y - fy*11 + ry*6.4}
	r.wpoly([][2]float64{nose, star, port}, Color{18, 64, 82}, 1)
	r.wline(nose[0], nose[1], port[0], port[1], 1.3, colBrass, 0.85)
	r.wline(nose[0], nose[1], star[0], star[1], 1.3, colBrass, 0.85)
	deck := [2]float64{x + fx*3.5, y + fy*3.5}
	r.wpoly([][2]float64{
		deck,
		{x - fx*3 + rx*3.1, y - fy*3 + ry*3.1},
		{x - fx*3 - rx*3.1, y - fy*3 - ry*3.1},
	}, colBrass, 1)
	r.wline(x-fx*7, y-fy*7, x+fx*13, y+fy*13, 1.5, colMint, 0.95)
	r.wcircle(x, y, 2.1, colIce, 0.95)
	ready := g.HopCD <= 0 && g.HopT <= 0
	if g.Mode == ModePlay {
		col := Color{60, 90, 100}
		a := 0.35
		if ready {
			col = colMint
			a = 0.55
		}
		if hop > 0 {
			col = colIce
			a = 0.7
		}
		g.strokeCircle(r, x, y, 18, 1.2, col, a)
	}
}

func (g *Game) drawChrome(r *Raster) {
	if g.externalUI {
		if g.Mode == ModePlay && g.Tutor > 0 {
			a := 1.0
			if g.Tutor < 1 {
				a = g.Tutor
			}
			r.textCenterFit("W thrust    A left    D right    Space hop", float64(H)-28, float64(W)-24, 0.38, colIce, a)
		}
		if g.MsgT > 0 && g.Msg != "" && !g.menuish() {
			r.textCenterFit(g.Msg, 78, float64(W)-30, 0.55, colMint, math.Min(1, g.MsgT))
		}
		return
	}
	switch g.Mode {
	case ModeAttract:
		r.fill(36, 8, W-72, 62, Color{4, 10, 18}, 0.55)
		r.textCenterFit("AURORA KEEL", 12, float64(W)-40, 0.78, colIce, 1)
		r.textCenterFit("Night courier of the black glass", 50, float64(W)-36, 0.36, colMint, 0.95)
		g.drawMenu(r, 156)
	case ModeCharts:
		r.fill(20, 8, W-40, 28, Color{4, 10, 18}, 0.5)
		r.textCenterFit("Charts", 10, 200, 0.55, colBrass, 1)
		g.drawMenu(r, 46)
	case ModeHelp:
		r.fill(16, 12, W-32, H-24, Color{4, 10, 18}, 0.72)
		lines := []string{
			"HOW TO SAIL",
			"W or Up — thrust ahead",
			"S or Down — reverse",
			"A or Left — steer left",
			"D or Right — steer right",
			"Space — keel-hop over ice and shocks",
			"Sing the glass buoys, then reach the moon-dock",
			"before the freeze seals the lane.",
			"Moon-glass mends the hull.",
			"Enter confirms.  Esc steps back.",
		}
		for i, ln := range lines {
			col := colIce
			sc := 0.34
			if i == 0 {
				col = colBrass
				sc = 0.46
			}
			r.textCenterFit(ln, 18+float64(i)*22, float64(W)-28, sc, col, 1)
		}
	case ModePlay, ModePause:
		g.drawHUD(r)
		if g.Mode == ModePause {
			r.textCenterFit("PAUSED", 78, 300, 0.7, colIce, 1)
			g.drawMenu(r, 118)
		}
	case ModeWin:
		r.textCenterFit("DELIVERED", 48, float64(W)-20, 0.72, colMint, 1)
		r.textCenterFit("Score "+itoa(g.Score), 92, 300, 0.48, colIce, 1)
		r.textCenterFit("Best "+itoa(g.displayBest()), 116, 300, 0.4, colBrass, 1)
		g.drawMenu(r, 150)
	case ModeLose:
		r.textCenterFit(g.Msg, 48, float64(W)-24, 0.62, colRose, 1)
		r.textCenterFit("Score "+itoa(g.Score), 92, 300, 0.48, colIce, 1)
		r.textCenterFit("Best "+itoa(g.displayBest()), 116, 300, 0.4, colBrass, 1)
		g.drawMenu(r, 150)
	}
}

func (g *Game) drawHUD(r *Raster) {
	name := g.C().Name
	r.textShadow(name, 10, 8, 0.4, colIce, 1)
	// hull bar
	r.fill(10, 30, 92, 7, Color{8, 16, 28}, 0.8)
	hw := int(90 * clamp(g.Hull, 0, 100) / 100)
	hc := colMint
	if g.Hull < 35 {
		hc = colRose
	}
	if hw > 0 {
		r.fill(11, 31, hw, 5, hc, 1)
	}
	r.textShadow("HULL", 106, 26, 0.32, colIce, 0.8)
	sec := int(math.Ceil(g.TimeLeft))
	if sec < 0 {
		sec = 0
	}
	tc := colIce
	if g.TimeLeft < 12 {
		tc = colRose
	}
	r.textShadow(itoa(sec)+"s", float64(W)-78, 8, 0.48, tc, 1)
	r.textShadow(itoa(g.Score), 10, float64(H)-26, 0.42, colBrass, 1)
	// buoys
	if !g.C().Endless {
		n := len(g.buoys)
		gap := 12.0
		width := float64(n) * gap
		x0 := (float64(W) - width) / 2
		for i, b := range g.buoys {
			col := Color{50, 80, 90}
			if b.On {
				col = colBrass
			} else if !g.C().Ordered || b.Order == g.nextOrder() {
				col = colMint
			}
			r.circle(x0+float64(i)*gap, float64(H)-16, 3.2, col, 1)
		}
	}
	if g.Tutor > 0 {
		a := 1.0
		if g.Tutor < 1 {
			a = g.Tutor
		}
		r.textCenterFit("W thrust   A left   D right   Space hop", 48, float64(W)-20, 0.36, colIce, a)
	}
	if g.MsgT > 0 && g.Msg != "" {
		r.textCenterFit(g.Msg, 70, float64(W)-30, 0.5, colMint, math.Min(1, g.MsgT))
	}
	hop := "HOP"
	col := Color{70, 100, 110}
	if g.HopCD <= 0 {
		col = colMint
	}
	r.textShadow(hop, float64(W)-52, float64(H)-26, 0.4, col, 1)
}

func (g *Game) drawMenu(r *Raster, y float64) {
	const step = 26.0
	items := g.menuItems()
	r.fill(48, int(y)-8, W-96, int(step)*len(items)+12, Color{4, 10, 18}, 0.72)
	for i, label := range items {
		col := colIce
		if i == g.menuSel {
			col = colBrass
			label = "> " + label
		}
		r.textCenterFit(label, y+float64(i)*step, float64(W)-70, 0.4, col, 1)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
