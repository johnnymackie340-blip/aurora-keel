package main

func vec(x, y float64) Vec { return Vec{x, y} }

func floe(x, y, r, amp, phase float64, axis int, gate bool) FloeDef {
	return FloeDef{X: x, Y: y, R: r, Amp: amp, Phase: phase, Axis: axis, Gate: gate}
}

var charts = []Chart{
	{
		Name:  "Glassmouth",
		Blurb: "A wide cove. Three buoys. Learn the keel.",
		W:     960, H: 2300,
		StartX: 480, StartY: 2080,
		Dock:  Rect{X: 400, Y: 48, W: 160, H: 110},
		Time:  100,
		Drift: -12,
		Buoys: []Vec{vec(480, 1680), vec(480, 1120), vec(480, 540)},
		Vials: []Vec{vec(480, 1400), vec(530, 820), vec(430, 280)},
		Floes: []FloeDef{
			floe(170, 1900, 78, 0, 0, 0, false),
			floe(800, 1860, 84, 0, 1, 0, false),
			floe(150, 1500, 70, 0, 0, 0, false),
			floe(820, 1460, 76, 0, 2, 0, false),
			floe(180, 1080, 80, 0, 0, 0, false),
			floe(790, 1040, 72, 0, 1, 0, false),
			floe(160, 680, 68, 0, 0, 0, false),
			floe(810, 640, 82, 0, 0.4, 0, false),
			floe(200, 300, 60, 0, 0, 0, false),
			floe(760, 280, 66, 18, 0.2, 1, false),
		},
	},
	{
		Name:  "Heron Narrows",
		Blurb: "Tight ice, diving herons, clockwork krill.",
		W:     860, H: 2680,
		StartX: 430, StartY: 2480,
		Dock:    Rect{X: 340, Y: 46, W: 180, H: 100},
		Time:    120,
		Ordered: true,
		Drift:   -6,
		Buoys: []Vec{
			vec(430, 2140), vec(190, 1760), vec(670, 1380), vec(250, 980), vec(430, 520),
		},
		Vials:  []Vec{vec(190, 1960), vec(670, 1180), vec(430, 300)},
		Herons: []Vec{vec(300, 1560), vec(560, 860)},
		Krill:  []Circle{{X: 470, Y: 1180, R: 58}, {X: 250, Y: 720, R: 48}},
		Bands: []Band{
			{Y0: 1600, Y1: 2000, VX: 42},
			{Y0: 900, Y1: 1280, VX: -48},
		},
		Floes: []FloeDef{
			floe(640, 2020, 48, 0, 0, 0, false),
			floe(160, 1880, 44, 0, 0, 0, false),
			floe(120, 1680, 40, 0, 0, 0, false),
			floe(360, 1560, 50, 0, 1, 0, false),
			floe(520, 1500, 36, 26, 0.5, 0, false),
			floe(80, 1280, 48, 0, 0, 0, false),
			floe(430, 1240, 58, 0, 0, 0, false),
			floe(760, 1100, 44, 0, 0, 0, false),
			floe(520, 860, 52, 0, 0, 0, false),
			floe(120, 780, 42, 0, 0, 0, false),
			floe(300, 640, 40, 22, 1.2, 1, false),
			floe(680, 460, 50, 0, 0, 0, false),
			floe(180, 360, 46, 0, 0, 0, false),
			floe(620, 240, 42, 0, 0, 0, false),
		},
	},
	{
		Name:  "Leviathan Road",
		Blurb: "Shock rings under the ice. Hop, or be thrown.",
		W:     1200, H: 2500,
		StartX: 600, StartY: 2280,
		Dock:    Rect{X: 520, Y: 48, W: 180, H: 110},
		Time:    115,
		Ordered: true,
		Rings:   true,
		Drift:   -8,
		Buoys: []Vec{
			vec(600, 1900), vec(280, 1450), vec(920, 1000), vec(600, 520),
		},
		Vials: []Vec{vec(760, 1680), vec(280, 1100), vec(900, 700)},
		Bands: []Band{
			{Y0: 1600, Y1: 2050, VX: 58},
			{Y0: 850, Y1: 1300, VX: -64},
			{Y0: 300, Y1: 700, VX: 36},
		},
		Floes: []FloeDef{
			floe(180, 2000, 70, 0, 0, 0, false),
			floe(1000, 1880, 80, 0, 0, 0, false),
			floe(240, 1200, 60, 30, 0.4, 0, false),
			floe(980, 760, 66, 0, 0, 0, false),
			floe(200, 420, 54, 0, 0, 0, false),
			floe(980, 360, 58, 20, 1, 1, false),
		},
	},
	{
		Name:  "Mirror Whale",
		Blurb: "Ring six buoys. The ice gate opens after the last song.",
		W:     1000, H: 1000,
		StartX: 500, StartY: 860,
		Dock:    Rect{X: 420, Y: 40, W: 160, H: 78},
		Time:    160,
		Ordered: true,
		Whale:   true,
		Buoys: []Vec{
			vec(500, 760), vec(730, 640), vec(730, 400), vec(500, 340), vec(270, 400), vec(270, 640),
		},
		Vials: []Vec{vec(500, 520), vec(860, 700), vec(140, 700)},
		Floes: []FloeDef{
			floe(170, 168, 108, 0, 0, 0, true),
			floe(390, 162, 108, 0, 0, 0, true),
			floe(610, 162, 108, 0, 0, 0, true),
			floe(830, 168, 108, 0, 0, 0, true),
			floe(70, 900, 40, 0, 0, 0, false),
			floe(930, 900, 40, 0, 0, 0, false),
		},
	},
	{
		Name:  "Uncharted Night",
		Blurb: "No dock. Sing buoys to hold the freeze back.",
		W:     4000, H: 4000,
		StartX: 0, StartY: 0,
		Time:    46,
		Endless: true,
	},
}

var decorFloes = []FloeDef{
	floe(260, 520, 54, 12, 0.2, 0, false),
	floe(700, 560, 62, 0, 0, 0, false),
	floe(340, 820, 40, 16, 1, 1, false),
	floe(690, 800, 48, 0, 0, 0, false),
	floe(500, 430, 28, 10, 0.6, 0, false),
}

var decorBuoys = []Vec{vec(420, 610), vec(610, 740)}
