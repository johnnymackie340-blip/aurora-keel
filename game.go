package main

import "math"

const (
	W = 480
	H = 270

	boatR = 12.0
)

const (
	ModeAttract = iota
	ModeCharts
	ModeHelp
	ModePlay
	ModePause
	ModeWin
	ModeLose
)

type Vec struct{ X, Y float64 }
type Rect struct{ X, Y, W, H float64 }
type Circle struct{ X, Y, R float64 }
type Band struct{ Y0, Y1, VX, VY float64 }

type FloeDef struct {
	X, Y, R, Amp, Phase float64
	Axis                int
	Gate                bool
}

type Chart struct {
	Name, Blurb                    string
	W, H                           float64
	StartX, StartY                 float64
	StartYaw                       float64
	Dock                           Rect
	Buoys, Vials, Herons           []Vec
	Ordered, Rings, Whale, Endless bool
	Floes                          []FloeDef
	Krill                          []Circle
	Bands                          []Band
	Time, Drift                    float64
}

type ck struct{ X, Y int }

type FloeRT struct {
	FloeDef
	Chunk ck
	Perm  bool
}

type liveBuoy struct {
	X, Y  float64
	On    bool
	Order int
	Chunk ck
	Perm  bool
}

type liveVial struct {
	X, Y  float64
	Chunk ck
	Perm  bool
}

type Krill struct {
	X, Y, R float64
	Chunk   ck
	Perm    bool
}

type Heron struct {
	X, Y, HomeX, HomeY, Phase float64
	State                     int
	T, TX, TY                 float64
	Hit                       bool
	Chunk                     ck
	Perm                      bool
}

type Ring struct {
	X, Y, R, MaxR, Speed, Telegraph float64
	Hit                             bool
}

type Whale struct {
	Active     bool
	X, Y, R, T float64
	State      int
	Hit        bool
}

type Part struct {
	X, Y, VX, VY, Life, Max, Sz float64
	Kind                        int
}

type Game struct {
	Mode, Chart               int
	X, Y, Yaw, Speed          float64
	Hull, Invuln, HopT, HopCD float64
	TimeLeft, ScoreF          float64
	Score, Mult               int
	T, clock, acc             float64
	Shake, Flash, MsgT, Tutor float64
	HintCD, hold              float64
	Msg                       string
	CamX, CamY                float64
	Keys, qa                  map[string]bool
	qaOn                      bool
	prevHop, prevEsc, prevEnt bool
	prevUp, prevDown          bool
	touchThrust, touchSteer   float64
	touchHop                  bool
	externalUI, blockMenu     bool
	menuSel                   int
	gateOpen                  bool
	floes                     []FloeRT
	buoys                     []liveBuoy
	vials                     []liveVial
	krill                     []Krill
	herons                    []Heron
	rings                     []Ring
	whale                     Whale
	parts                     []Part
	chunkHas                  map[ck]bool
	ringAcc, pruneAcc         float64
	rng                       uint32
	pix                       []byte
	quit                      bool
}

var bestScores [5]int

func newGame() *Game {
	g := &Game{
		Mode:     ModeAttract,
		Keys:     map[string]bool{},
		qa:       map[string]bool{},
		chunkHas: map[ck]bool{},
		rng:      1,
		Hull:     100,
		Mult:     1,
		CamX:     480,
		CamY:     640,
	}
	g.bestLoad()
	return g
}

func (g *Game) bestLoad() {
	bestScores = readSave()
}

func (g *Game) C() *Chart { return &charts[g.Chart] }

func (g *Game) rnd() float64 {
	g.rng = g.rng*1664525 + 1013904223
	return float64(g.rng>>8) / float64(1<<24)
}

func hashU(n int) uint32 {
	x := uint32(n) * 2246822519
	x ^= x >> 13
	x *= 3266489917
	x ^= x >> 16
	return x
}

func hash01(n int) float64 { return float64(hashU(n)&0xffffff) / float64(0xffffff) }

func heading(yaw float64) (fx, fy, rx, ry float64) {
	s, c := math.Sin(yaw), math.Cos(yaw)
	return -s, -c, c, -s
}

func wrapAng(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func floePos(f FloeDef, t float64) (float64, float64) {
	d := math.Sin(t*0.7+f.Phase) * f.Amp
	if f.Axis == 1 {
		return f.X, f.Y + d
	}
	return f.X + d, f.Y
}

func (rc Rect) has(x, y, pad float64) bool {
	return x >= rc.X-pad && y >= rc.Y-pad && x <= rc.X+rc.W+pad && y <= rc.Y+rc.H+pad
}

func (g *Game) down(code string) bool {
	if g.qaOn {
		return g.qa[code]
	}
	return g.Keys[code]
}

func (g *Game) say(s string) { g.Msg, g.MsgT = s, 1.7 }

func (g *Game) Start(idx int) {
	if idx < 0 || idx >= len(charts) {
		idx = 0
	}
	c := &charts[idx]
	g.Mode = ModePlay
	g.Chart = idx
	g.X, g.Y, g.Yaw = c.StartX, c.StartY, c.StartYaw
	g.Speed = 0
	g.Hull = 100
	g.Invuln, g.HopT, g.HopCD = 0, 0, 0
	g.TimeLeft = c.Time
	g.ScoreF = 0
	g.Score = 0
	g.Mult = 1
	g.T, g.acc = 0, 0
	g.Shake, g.Flash = 0, 0
	g.MsgT = 2.4
	g.Msg = c.Name
	g.Tutor = 0
	if idx == 0 {
		g.Tutor = 7
	}
	g.HintCD = 0
	g.prevHop = true
	g.gateOpen = !c.Whale
	g.floes = g.floes[:0]
	g.buoys = g.buoys[:0]
	g.vials = g.vials[:0]
	g.krill = g.krill[:0]
	g.herons = g.herons[:0]
	g.rings = g.rings[:0]
	g.parts = g.parts[:0]
	g.chunkHas = map[ck]bool{}
	g.ringAcc, g.pruneAcc = 0, 0
	g.whale = Whale{}
	fx, fy, _, _ := heading(g.Yaw)
	g.CamX, g.CamY = g.X+fx*20, g.Y+fy*20
	if c.Endless {
		g.ensureChunks()
		g.clearRadius(140)
		playSFX("start")
		return
	}
	for _, f := range c.Floes {
		g.floes = append(g.floes, FloeRT{FloeDef: f, Perm: true})
	}
	for i, b := range c.Buoys {
		order := -1
		if c.Ordered {
			order = i
		}
		g.buoys = append(g.buoys, liveBuoy{X: b.X, Y: b.Y, Order: order, Perm: true})
	}
	for _, v := range c.Vials {
		g.vials = append(g.vials, liveVial{X: v.X, Y: v.Y, Perm: true})
	}
	for _, k := range c.Krill {
		g.krill = append(g.krill, Krill{X: k.X, Y: k.Y, R: k.R, Perm: true})
	}
	for _, h := range c.Herons {
		g.herons = append(g.herons, Heron{X: h.X, Y: h.Y, HomeX: h.X, HomeY: h.Y, Perm: true, T: 1.2})
	}
	if c.Whale {
		g.whale = Whale{Active: true, State: 0, T: 2.2, R: 58, X: 500, Y: 500}
	}
	playSFX("start")
}

func (g *Game) clearRadius(rad float64) {
	dst := g.floes[:0]
	for _, f := range g.floes {
		if math.Hypot(f.X-g.X, f.Y-g.Y) > rad+f.R {
			dst = append(dst, f)
		}
	}
	g.floes = dst
}

func (g *Game) menuish() bool {
	return g.Mode == ModeAttract || g.Mode == ModeCharts || g.Mode == ModeHelp
}

func (g *Game) Frame(dt float64) {
	if dt > 0.1 {
		dt = 0.1
	}
	if dt < 0 {
		dt = 0
	}
	g.pollEdges()
	switch g.Mode {
	case ModeAttract, ModeCharts, ModeHelp:
		g.attract(dt)
	case ModePlay:
		g.clock += dt
		if g.HitStop() {
			return
		}
		g.acc += dt
		if g.acc > 0.25 {
			g.acc = 0.25
		}
		const step = 1.0 / 60
		for g.acc >= step {
			g.step(step)
			g.acc -= step
			if g.Mode != ModePlay {
				break
			}
		}
	default:
		g.clock += dt
		if g.Shake > 0 {
			g.Shake *= math.Exp(-3 * dt)
		}
		if g.Flash > 0 {
			g.Flash -= dt
		}
	}
}

func (g *Game) HitStop() bool {
	if g.Flash > 0 {
		g.Flash -= 1.0 / 120
	}
	if g.hold > 0 {
		g.hold -= 1.0 / 60
		g.clock += 1.0 / 60
		return true
	}
	return false
}

func (g *Game) attract(dt float64) {
	g.clock += dt
	g.T += dt
	g.X = 480 + math.Sin(g.T*0.35)*100
	g.Y = 690 + math.Sin(g.T*0.22)*24
	g.Yaw = math.Sin(g.T*0.45) * 0.55
	g.Speed = 48
	tx := g.X
	ty := g.Y + float64(H)/2 - 92
	k := 1 - math.Exp(-2.4*dt)
	g.CamX += (tx - g.CamX) * k
	g.CamY += (ty - g.CamY) * k
}

func (g *Game) pollEdges() {
	esc := g.down("Escape")
	if esc && !g.prevEsc {
		switch g.Mode {
		case ModePlay:
			g.Mode = ModePause
			playSFX("pause")
		case ModePause:
			g.Mode = ModePlay
		case ModeCharts, ModeHelp:
			if !g.externalUI {
				g.Mode = ModeAttract
				g.menuSel = 0
			}
		case ModeWin, ModeLose:
			if !g.externalUI {
				g.Mode = ModeAttract
				g.menuSel = 0
			}
		}
	}
	g.prevEsc = esc

	if g.externalUI || g.Mode == ModePlay || g.Mode == ModePause {
		ent := g.down("Enter")
		if ent && !g.prevEnt && !g.blockMenu && g.Mode == ModeAttract && !g.externalUI {
			g.activate(g.menuSel)
		} else if ent && !g.prevEnt && !g.externalUI && g.Mode != ModePlay && g.Mode != ModePause && g.Mode != ModeAttract {
			g.activate(g.menuSel)
		} else if ent && !g.prevEnt && g.externalUI && g.Mode == ModeAttract && !g.blockMenu {
			g.Start(0)
		}
		g.prevEnt = ent
		return
	}

	up := g.down("ArrowUp") || g.down("KeyW")
	down := g.down("ArrowDown") || g.down("KeyS")
	items := g.menuItems()
	if up && !g.prevUp {
		g.menuSel--
		if g.menuSel < 0 {
			g.menuSel = len(items) - 1
		}
	}
	if down && !g.prevDown {
		g.menuSel++
		if len(items) > 0 && g.menuSel >= len(items) {
			g.menuSel = 0
		}
	}
	g.prevUp, g.prevDown = up, down
	ent := g.down("Enter") || g.down("Space")
	if ent && !g.prevEnt {
		g.activate(g.menuSel)
	}
	g.prevEnt = ent
}

func (g *Game) menuItems() []string {
	switch g.Mode {
	case ModeAttract:
		return []string{"Start voyage", "Charts", "How to sail", "Quit"}
	case ModeCharts:
		out := make([]string, 0, len(charts)+1)
		for _, c := range charts {
			out = append(out, c.Name)
		}
		return append(out, "Back")
	case ModePause:
		return []string{"Resume", "Retry", "Title"}
	case ModeWin:
		if g.Chart < len(charts)-1 {
			return []string{"Next chart", "Retry", "Title"}
		}
		return []string{"Retry", "Title"}
	case ModeLose:
		return []string{"Retry", "Title"}
	default:
		return []string{"Back"}
	}
}

func (g *Game) activate(sel int) {
	items := g.menuItems()
	if sel < 0 || sel >= len(items) {
		return
	}
	label := items[sel]
	switch g.Mode {
	case ModeAttract:
		switch label {
		case "Start voyage":
			g.Start(0)
		case "Charts":
			g.Mode = ModeCharts
			g.menuSel = 0
		case "How to sail":
			g.Mode = ModeHelp
			g.menuSel = 0
		case "Quit":
			g.quit = true
		}
	case ModeCharts:
		if label == "Back" {
			g.Mode = ModeAttract
			g.menuSel = 0
			return
		}
		g.Start(sel)
	case ModeHelp:
		g.Mode = ModeAttract
		g.menuSel = 0
	case ModePause:
		switch label {
		case "Resume":
			g.Mode = ModePlay
		case "Retry":
			g.Start(g.Chart)
		case "Title":
			g.Mode = ModeAttract
			g.menuSel = 0
		}
	case ModeWin:
		switch label {
		case "Next chart":
			g.Start(g.Chart + 1)
		case "Retry":
			g.Start(g.Chart)
		case "Title":
			g.Mode = ModeAttract
			g.menuSel = 0
		}
	case ModeLose:
		switch label {
		case "Retry":
			g.Start(g.Chart)
		case "Title":
			g.Mode = ModeAttract
			g.menuSel = 0
		}
	}
}

func (g *Game) Action(name string) {
	switch name {
	case "resume":
		if g.Mode == ModePause {
			g.Mode = ModePlay
		}
	case "pause":
		if g.Mode == ModePlay {
			g.Mode = ModePause
			playSFX("pause")
		}
	case "retry":
		if g.Mode == ModePlay || g.Mode == ModePause || g.Mode == ModeWin || g.Mode == ModeLose {
			g.Start(g.Chart)
		}
	case "title":
		g.Mode = ModeAttract
		g.menuSel = 0
	}
}

func (g *Game) readMove() (thrust, steer float64, hop bool) {
	if g.down("KeyW") || g.down("ArrowUp") {
		thrust += 1
	}
	if g.down("KeyS") || g.down("ArrowDown") {
		thrust -= 1
	}
	if g.down("KeyA") || g.down("ArrowLeft") {
		steer += 1
	}
	if g.down("KeyD") || g.down("ArrowRight") {
		steer -= 1
	}
	if !g.qaOn {
		thrust += g.touchThrust
		steer += g.touchSteer
	}
	thrust = clamp(thrust, -1, 1)
	steer = clamp(steer, -1, 1)
	space := g.down("Space") || (!g.qaOn && g.touchHop)
	hop = space && !g.prevHop
	g.prevHop = space
	return
}

func (g *Game) step(dt float64) {
	g.T += dt
	if g.Tutor > 0 {
		g.Tutor -= dt
	}
	if g.HintCD > 0 {
		g.HintCD -= dt
	}
	if g.MsgT > 0 {
		g.MsgT -= dt
	}
	if g.Invuln > 0 {
		g.Invuln -= dt
	}
	if g.Flash > 0 {
		g.Flash -= dt
	}
	if g.Shake > 0 {
		g.Shake *= math.Exp(-3.2 * dt)
	}

	thrust, steer, hopEdge := g.readMove()
	justLanded := false
	if hopEdge && g.HopCD <= 0 && g.HopT <= 0 {
		g.HopT = 0.42
		g.HopCD = 1.2
		playSFX("hop")
		g.puff(g.X, g.Y, 10, 0, 40)
	}
	if g.HopT > 0 {
		g.HopT -= dt
		if g.HopT <= 0 {
			g.HopT = 0
			justLanded = true
		}
	}
	if g.HopCD > 0 {
		g.HopCD -= dt
	}

	sp := math.Abs(g.Speed)
	sf := 0.62 + 0.38*math.Min(1, sp/90)
	rev := 1.0
	if g.Speed < 0 {
		rev = -1
	}
	g.Yaw += steer * 2.75 * sf * rev * dt

	if thrust != 0 {
		g.Speed += thrust * 240 * dt
	}
	g.Speed *= math.Pow(0.5, dt)
	g.Speed = clamp(g.Speed, -60, 148)

	fx, fy, _, _ := heading(g.Yaw)
	cx, cy := g.currentAt(g.X, g.Y)
	kmul := 1.0
	if g.inKrill() {
		kmul = 0.45
	}
	g.X += (fx*g.Speed*kmul + cx) * dt
	g.Y += (fy*g.Speed*kmul + cy) * dt

	if !g.C().Endless {
		g.clampShore()
	} else {
		g.ScoreF += math.Hypot(fx*g.Speed*kmul, fy*g.Speed*kmul) * dt * 0.45
	}

	if g.HopT <= 0 {
		g.collideFloes(justLanded)
	}
	g.touchBuoys()
	g.touchVials()
	for i := range g.herons {
		g.updateHeron(&g.herons[i], dt)
	}
	g.updateRings(dt)
	g.updateWhale(dt)
	if g.C().Rings {
		g.ringAcc += dt
		if g.ringAcc > 3.15 {
			g.ringAcc = 0
			j := (g.rnd() - 0.5) * 50
			g.rings = append(g.rings, Ring{
				X: g.X + fx*90 + j, Y: g.Y + fy*90, Telegraph: 0.78, MaxR: 150, Speed: 105,
			})
		}
	}
	if g.C().Endless {
		g.ensureChunks()
		g.pruneAcc += dt
		if g.pruneAcc > 0.6 {
			g.pruneAcc = 0
			g.pruneChunks()
		}
	}
	g.updateParts(dt)
	g.wake()

	if g.Mode != ModePlay {
		g.syncScore()
		return
	}
	if !g.C().Endless && g.dockOpen() && g.buoysDone() && g.C().Dock.has(g.X, g.Y, 6) {
		g.win()
		g.syncScore()
		return
	}
	if !g.C().Endless && g.dockOpen() && !g.buoysDone() && g.C().Dock.has(g.X, g.Y, 6) && g.HintCD <= 0 {
		g.say("Sing the buoys first")
		g.HintCD = 2.2
	}
	g.TimeLeft -= dt
	if g.TimeLeft <= 0 {
		g.TimeLeft = 0
		g.fail("The lane sealed")
	}
	g.aimCam(dt)
	g.syncScore()
}

func (g *Game) syncScore() { g.Score = int(g.ScoreF) }

func (g *Game) aimCam(dt float64) {
	fx, fy, _, _ := heading(g.Yaw)
	tx := g.X + fx*34
	ty := g.Y + fy*34
	k := 1 - math.Exp(-3.6*dt)
	g.CamX += (tx - g.CamX) * k
	g.CamY += (ty - g.CamY) * k
}

func (g *Game) clampShore() {
	c := g.C()
	pad := 32.0
	if g.X < pad {
		g.X = pad
		g.Speed *= 0.45
	}
	if g.Y < pad {
		g.Y = pad
		g.Speed *= 0.45
	}
	if g.X > c.W-pad {
		g.X = c.W - pad
		g.Speed *= 0.45
	}
	if g.Y > c.H-pad {
		g.Y = c.H - pad
		g.Speed *= 0.45
	}
}

func (g *Game) currentAt(x, y float64) (float64, float64) {
	if g.C().Endless {
		return math.Sin(y*0.004) * 28, math.Cos(x*0.003) * 18
	}
	var vx, vy float64
	for _, b := range g.C().Bands {
		if y >= b.Y0 && y <= b.Y1 {
			vx += b.VX
			vy += b.VY
		}
	}
	vy += g.C().Drift
	_ = x
	return vx, vy
}

func (g *Game) inKrill() bool {
	for _, k := range g.krill {
		if math.Hypot(g.X-k.X, g.Y-k.Y) < k.R {
			return true
		}
	}
	return false
}

func (g *Game) collideFloes(justLanded bool) {
	fx, fy, _, _ := heading(g.Yaw)
	for _, f := range g.floes {
		x, y := floePos(f.FloeDef, g.T)
		dx, dy := g.X-x, g.Y-y
		d := math.Hypot(dx, dy)
		min := f.R + boatR
		if d >= min || d < 0.001 {
			continue
		}
		nx, ny := dx/d, dy/d
		pen := min - d
		g.X += nx * (pen + 0.4)
		g.Y += ny * (pen + 0.4)
		vx := fx * g.Speed
		vy := fy * g.Speed
		vn := vx*nx + vy*ny
		approach := -vn
		if vn < 0 {
			vx -= vn * nx
			vy -= vn * ny
			g.Speed = vx*fx + vy*fy
		}
		if !justLanded && approach > 52 {
			dmg := 4 + (approach-52)*0.05
			if dmg > 15 {
				dmg = 15
			}
			g.hurt(dmg, "Ice")
		}
	}
}

func (g *Game) hurt(dmg float64, msg string) {
	if g.Mode != ModePlay || g.Invuln > 0 || g.HopT > 0 || dmg <= 0 {
		return
	}
	g.Hull -= dmg
	g.Invuln = 0.65
	g.Shake = math.Max(g.Shake, 4.2)
	g.hold = 0.04
	g.Flash = 0.16
	g.Mult = 1
	g.Speed *= 0.5
	playSFX("hit")
	g.say(msg)
	g.puff(g.X, g.Y, 12, 1, 70)
	if g.Hull <= 0 {
		g.Hull = 0
		g.fail("The keel shattered")
	}
}

// hold is hit-stop time. Field declared below via embedding in struct...
// added as method receiver field — declared on Game as hold.

func (g *Game) touchBuoys() {
	for i := range g.buoys {
		b := &g.buoys[i]
		if b.On || math.Hypot(g.X-b.X, g.Y-b.Y) > 28 {
			continue
		}
		if g.C().Ordered && b.Order != g.nextOrder() {
			continue
		}
		b.On = true
		g.ScoreF += float64(220 * g.Mult)
		if g.Mult < 8 {
			g.Mult++
		}
		g.TimeLeft += 4.5
		if g.C().Endless {
			if g.TimeLeft > 72 {
				g.TimeLeft = 72
			}
		} else if g.TimeLeft > g.C().Time {
			g.TimeLeft = g.C().Time
		}
		playSFX("chime")
		g.puff(b.X, b.Y, 14, 2, 50)
		g.say("Buoy sung")
		if g.C().Whale && g.buoysDone() && !g.gateOpen {
			g.gateOpen = true
			dst := g.floes[:0]
			for _, f := range g.floes {
				if !f.Gate {
					dst = append(dst, f)
				}
			}
			g.floes = dst
			g.say("The ice gate opens")
			playSFX("win")
		}
	}
}

func (g *Game) nextOrder() int {
	n := -1
	for _, b := range g.buoys {
		if !b.On && (n < 0 || b.Order < n) {
			n = b.Order
		}
	}
	return n
}

func (g *Game) buoysDone() bool {
	if len(g.buoys) == 0 {
		return !g.C().Endless
	}
	for _, b := range g.buoys {
		if !b.On {
			return false
		}
	}
	return true
}

func (g *Game) buoysRung() int {
	n := 0
	for _, b := range g.buoys {
		if b.On {
			n++
		}
	}
	return n
}

func (g *Game) dockOpen() bool {
	if g.C().Endless {
		return false
	}
	if g.C().Whale {
		return g.gateOpen
	}
	return true
}

func (g *Game) touchVials() {
	dst := g.vials[:0]
	for _, v := range g.vials {
		if math.Hypot(g.X-v.X, g.Y-v.Y) < 16 {
			g.Hull = math.Min(100, g.Hull+10)
			g.ScoreF += 100
			if g.C().Endless {
				g.TimeLeft += 3
				if g.TimeLeft > 72 {
					g.TimeLeft = 72
				}
			}
			playSFX("heal")
			g.puff(v.X, v.Y, 8, 2, 30)
			g.say("Moon-glass")
			continue
		}
		dst = append(dst, v)
	}
	g.vials = dst
}

func (g *Game) updateHeron(h *Heron, dt float64) {
	switch h.State {
	case 0:
		h.Phase += dt
		h.X = h.HomeX + math.Cos(h.Phase*0.85)*64
		h.Y = h.HomeY + math.Sin(h.Phase*0.85)*42
		h.T += dt
		if h.T > 2.6 && math.Hypot(g.X-h.X, g.Y-h.Y) < 230 {
			h.State = 1
			h.T = 0.72
			h.TX, h.TY = g.X, g.Y
			h.Hit = false
		}
	case 1:
		h.T -= dt
		if h.T <= 0 {
			h.State = 2
			h.T = 0.24
		}
	case 2:
		h.T -= dt
		h.X += (h.TX - h.X) * 0.35
		h.Y += (h.TY - h.Y) * 0.35
		if !h.Hit && math.Hypot(g.X-h.X, g.Y-h.Y) < 16 {
			h.Hit = true
			g.hurt(18, "Heron")
		}
		if h.T <= 0 {
			h.State = 3
			h.T = 0.85
		}
	default:
		h.T -= dt
		h.X += (h.HomeX - h.X) * 0.08
		h.Y += (h.HomeY - h.Y) * 0.08
		if h.T <= 0 {
			h.State = 0
			h.T = 0
		}
	}
}

func (g *Game) updateRings(dt float64) {
	dst := g.rings[:0]
	for _, ring := range g.rings {
		if ring.Telegraph > 0 {
			ring.Telegraph -= dt
			dst = append(dst, ring)
			continue
		}
		ring.R += ring.Speed * dt
		if !ring.Hit {
			d := math.Hypot(g.X-ring.X, g.Y-ring.Y)
			if math.Abs(d-ring.R) < 12 && ring.R > 8 {
				ring.Hit = true
				g.hurt(11, "Shock")
			}
		}
		if ring.R < ring.MaxR {
			dst = append(dst, ring)
		}
	}
	g.rings = dst
}

func (g *Game) updateWhale(dt float64) {
	w := &g.whale
	if !w.Active {
		return
	}
	switch w.State {
	case 0:
		w.T -= dt
		if w.T <= 0 {
			w.State = 1
			w.T = 0.9
			w.X, w.Y = g.X, g.Y
			w.Hit = false
		}
	case 1:
		w.T -= dt
		w.X += (g.X - w.X) * 0.035
		w.Y += (g.Y - w.Y) * 0.035
		if w.T <= 0 {
			w.State = 2
			w.T = 0.48
		}
	case 2:
		w.T -= dt
		if w.T <= 0 {
			if math.Hypot(g.X-w.X, g.Y-w.Y) < w.R {
				g.hurt(16, "Breach")
			}
			w.State = 3
			w.T = 0.18
		}
	default:
		w.T -= dt
		if w.T <= 0 {
			g.rings = append(g.rings,
				Ring{X: w.X, Y: w.Y, Telegraph: 0.12, MaxR: 170, Speed: 120},
				Ring{X: w.X, Y: w.Y, Telegraph: 0.42, MaxR: 250, Speed: 95},
			)
			w.State = 0
			w.T = 3.6
		}
	}
}

func (g *Game) ensureChunks() {
	const cs = 520.0
	pcx := int(math.Floor(g.X / cs))
	pcy := int(math.Floor(g.Y / cs))
	for cy := pcy - 1; cy <= pcy+1; cy++ {
		for cx := pcx - 1; cx <= pcx+1; cx++ {
			id := ck{cx, cy}
			if g.chunkHas[id] {
				continue
			}
			g.chunkHas[id] = true
			if cx == 0 && cy == 0 {
				continue
			}
			g.spawnChunk(cx, cy)
		}
	}
}

func (g *Game) spawnChunk(cx, cy int) {
	const cs = 520.0
	baseX := float64(cx) * cs
	baseY := float64(cy) * cs
	h := func(i int) float64 {
		return hash01(cx*92821 + cy*68917 + i*374761 + 99)
	}
	n := 2 + int(h(1)*3)
	id := ck{cx, cy}
	for i := 0; i < n; i++ {
		g.floes = append(g.floes, FloeRT{FloeDef: FloeDef{
			X:   baseX + 50 + h(10+i)*420,
			Y:   baseY + 50 + h(30+i)*420,
			R:   26 + h(50+i)*46,
			Amp: h(70+i) * 22, Phase: h(90+i) * 6, Axis: int(h(110+i) * 2),
		}, Chunk: id})
	}
	if h(2) < 0.6 {
		g.buoys = append(g.buoys, liveBuoy{
			X: baseX + 90 + h(3)*340, Y: baseY + 90 + h(4)*340, Order: -1, Chunk: id,
		})
	}
	if h(5) < 0.75 {
		g.vials = append(g.vials, liveVial{X: baseX + 70 + h(6)*380, Y: baseY + 70 + h(7)*380, Chunk: id})
	}
	if h(8) < 0.22 {
		hx, hy := baseX+260, baseY+260
		g.herons = append(g.herons, Heron{X: hx, Y: hy, HomeX: hx, HomeY: hy, Chunk: id, T: 2})
	}
	if h(9) < 0.18 {
		g.krill = append(g.krill, Krill{X: baseX + 200, Y: baseY + 200, R: 46 + h(12)*20, Chunk: id})
	}
}

func (g *Game) pruneChunks() {
	const cs = 520.0
	pcx := int(math.Floor(g.X / cs))
	pcy := int(math.Floor(g.Y / cs))
	far := func(c ck) bool {
		if c.X == 0 && c.Y == 0 && !g.chunkHas[c] {
			return false
		}
		dx, dy := c.X-pcx, c.Y-pcy
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		return dx > 2 || dy > 2
	}
	for id := range g.chunkHas {
		if far(id) {
			delete(g.chunkHas, id)
		}
	}
	ff := g.floes[:0]
	for _, f := range g.floes {
		if f.Perm || !far(f.Chunk) {
			ff = append(ff, f)
		}
	}
	g.floes = ff
	bb := g.buoys[:0]
	for _, b := range g.buoys {
		if b.Perm || !far(b.Chunk) {
			bb = append(bb, b)
		}
	}
	g.buoys = bb
	vv := g.vials[:0]
	for _, v := range g.vials {
		if v.Perm || !far(v.Chunk) {
			vv = append(vv, v)
		}
	}
	g.vials = vv
	hh := g.herons[:0]
	for _, h := range g.herons {
		if h.Perm || !far(h.Chunk) {
			hh = append(hh, h)
		}
	}
	g.herons = hh
	kk := g.krill[:0]
	for _, k := range g.krill {
		if k.Perm || !far(k.Chunk) {
			kk = append(kk, k)
		}
	}
	g.krill = kk
}

func (g *Game) puff(x, y float64, n, kind int, spd float64) {
	for i := 0; i < n; i++ {
		a := (g.rnd()*2 - 1) * math.Pi
		life := 0.28 + g.rnd()*0.35
		g.parts = append(g.parts, Part{
			X: x, Y: y,
			VX:   math.Cos(a) * spd * (0.35 + g.rnd()),
			VY:   math.Sin(a) * spd * (0.35 + g.rnd()),
			Life: life, Max: life, Sz: 1.3 + g.rnd()*1.6, Kind: kind,
		})
	}
	if len(g.parts) > 150 {
		g.parts = append([]Part(nil), g.parts[len(g.parts)-150:]...)
	}
}

func (g *Game) wake() {
	if g.Speed < 28 || g.Mode != ModePlay {
		return
	}
	if int(g.T*60)%3 != 0 {
		return
	}
	fx, fy, _, _ := heading(g.Yaw)
	g.parts = append(g.parts, Part{
		X: g.X - fx*12, Y: g.Y - fy*12,
		VX: -fx * 8, VY: -fy * 8,
		Life: 0.45, Max: 0.45, Sz: 2.2, Kind: 0,
	})
}

func (g *Game) updateParts(dt float64) {
	n := 0
	for _, p := range g.parts {
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Life -= dt
		if p.Life > 0 {
			g.parts[n] = p
			n++
		}
	}
	g.parts = g.parts[:n]
}

func (g *Game) win() {
	if g.Mode != ModePlay {
		return
	}
	g.ScoreF += 1000 + g.TimeLeft*8 + g.Hull*4
	g.syncScore()
	g.Mode = ModeWin
	g.Msg = "Delivered"
	g.MsgT = 3
	g.noteScore()
	playSFX("win")
}

func (g *Game) fail(reason string) {
	if g.Mode != ModePlay {
		return
	}
	g.syncScore()
	g.Mode = ModeLose
	g.Msg = reason
	g.MsgT = 3
	g.noteScore()
	playSFX("lose")
}

func (g *Game) noteScore() {
	if g.Chart < 0 || g.Chart >= len(bestScores) {
		return
	}
	if g.Score > bestScores[g.Chart] {
		bestScores[g.Chart] = g.Score
		writeSave(bestScores)
	}
}

func (g *Game) displayBest() int {
	b := 0
	if g.Chart >= 0 && g.Chart < len(bestScores) {
		b = bestScores[g.Chart]
	}
	if g.Score > b {
		return g.Score
	}
	return b
}

func modeName(m int) string {
	switch m {
	case ModeAttract:
		return "attract"
	case ModeCharts:
		return "charts"
	case ModeHelp:
		return "help"
	case ModePlay:
		return "play"
	case ModePause:
		return "pause"
	case ModeWin:
		return "win"
	case ModeLose:
		return "lose"
	default:
		return "attract"
	}
}
