package main

import "math"

type Color struct{ R, G, B uint8 }

type Raster struct {
	pix        []byte
	camX, camY float64
	shX, shY   float64
}

func (r *Raster) to(wx, wy float64) (float64, float64) {
	return wx - r.camX + float64(W)/2 + r.shX, wy - r.camY + float64(H)/2 + r.shY
}

func (r *Raster) blend(x, y int, c Color, a float64) {
	if x < 0 || y < 0 || x >= W || y >= H || a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	i := (y*W + x) * 4
	inv := 1 - a
	r.pix[i] = uint8(float64(r.pix[i])*inv + float64(c.R)*a)
	r.pix[i+1] = uint8(float64(r.pix[i+1])*inv + float64(c.G)*a)
	r.pix[i+2] = uint8(float64(r.pix[i+2])*inv + float64(c.B)*a)
	r.pix[i+3] = 255
}

func (r *Raster) hline(y int, c Color) {
	if y < 0 || y >= H {
		return
	}
	o := y * W * 4
	for x := 0; x < W; x++ {
		r.pix[o] = c.R
		r.pix[o+1] = c.G
		r.pix[o+2] = c.B
		r.pix[o+3] = 255
		o += 4
	}
}

func lerpC(a, b Color, t float64) Color {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return Color{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
	}
}

func (r *Raster) circle(cx, cy, rad float64, c Color, a float64) {
	if rad <= 0 || a <= 0 {
		return
	}
	minx := int(math.Floor(cx - rad))
	maxx := int(math.Ceil(cx + rad))
	miny := int(math.Floor(cy - rad))
	maxy := int(math.Ceil(cy + rad))
	if maxx < 0 || maxy < 0 || minx >= W || miny >= H {
		return
	}
	if minx < 0 {
		minx = 0
	}
	if miny < 0 {
		miny = 0
	}
	if maxx >= W {
		maxx = W - 1
	}
	if maxy >= H {
		maxy = H - 1
	}
	rr := rad * rad
	for y := miny; y <= maxy; y++ {
		dy := float64(y) + 0.5 - cy
		for x := minx; x <= maxx; x++ {
			dx := float64(x) + 0.5 - cx
			if dx*dx+dy*dy <= rr {
				r.blend(x, y, c, a)
			}
		}
	}
}

func edge(a, b, p [2]float64) float64 {
	return (p[0]-a[0])*(b[1]-a[1]) - (p[1]-a[1])*(b[0]-a[0])
}

func (r *Raster) tri(a, b, c [2]float64, col Color, alpha float64) {
	if edge(a, b, c) < 0 {
		b, c = c, b
	}
	minx := int(math.Floor(math.Min(a[0], math.Min(b[0], c[0]))))
	maxx := int(math.Ceil(math.Max(a[0], math.Max(b[0], c[0]))))
	miny := int(math.Floor(math.Min(a[1], math.Min(b[1], c[1]))))
	maxy := int(math.Ceil(math.Max(a[1], math.Max(b[1], c[1]))))
	if maxx < 0 || maxy < 0 || minx >= W || miny >= H {
		return
	}
	if minx < 0 {
		minx = 0
	}
	if miny < 0 {
		miny = 0
	}
	if maxx >= W {
		maxx = W - 1
	}
	if maxy >= H {
		maxy = H - 1
	}
	for y := miny; y <= maxy; y++ {
		for x := minx; x <= maxx; x++ {
			p := [2]float64{float64(x) + 0.5, float64(y) + 0.5}
			if edge(a, b, p) >= 0 && edge(b, c, p) >= 0 && edge(c, a, p) >= 0 {
				r.blend(x, y, col, alpha)
			}
		}
	}
}

func (r *Raster) poly(pts [][2]float64, c Color, a float64) {
	if len(pts) < 3 || a <= 0 {
		return
	}
	for i := 1; i < len(pts)-1; i++ {
		r.tri(pts[0], pts[i], pts[i+1], c, a)
	}
}

func (r *Raster) line(x0, y0, x1, y1, thick float64, c Color, a float64) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l < 0.001 {
		r.circle(x0, y0, thick*0.5, c, a)
		return
	}
	nx, ny := -dy/l*thick*0.5, dx/l*thick*0.5
	r.poly([][2]float64{
		{x0 + nx, y0 + ny},
		{x1 + nx, y1 + ny},
		{x1 - nx, y1 - ny},
		{x0 - nx, y0 - ny},
	}, c, a)
}

func (r *Raster) wcircle(wx, wy, rad float64, c Color, a float64) {
	sx, sy := r.to(wx, wy)
	if sx < -rad-2 || sy < -rad-2 || sx > float64(W)+rad || sy > float64(H)+rad {
		return
	}
	r.circle(sx, sy, rad, c, a)
}

func (r *Raster) wline(x0, y0, x1, y1, thick float64, c Color, a float64) {
	a0x, a0y := r.to(x0, y0)
	a1x, a1y := r.to(x1, y1)
	r.line(a0x, a0y, a1x, a1y, thick, c, a)
}

func (r *Raster) wpoly(pts [][2]float64, c Color, a float64) {
	s := make([][2]float64, len(pts))
	for i, p := range pts {
		s[i][0], s[i][1] = r.to(p[0], p[1])
	}
	r.poly(s, c, a)
}

func (r *Raster) fill(x, y, w, h int, c Color, a float64) {
	if a <= 0 {
		return
	}
	x2, y2 := x+w, y+h
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x2 > W {
		x2 = W
	}
	if y2 > H {
		y2 = H
	}
	for yy := y; yy < y2; yy++ {
		for xx := x; xx < x2; xx++ {
			r.blend(xx, yy, c, a)
		}
	}
}

func (r *Raster) borderVignette(frost float64) {
	const band = 42
	for y := 0; y < H; y++ {
		dy := 0
		if y < band {
			dy = band - y
		} else if y > H-1-band {
			dy = y - (H - 1 - band)
		}
		for x := 0; x < W; x++ {
			dx := 0
			if x < band {
				dx = band - x
			} else if x > W-1-band {
				dx = x - (W - 1 - band)
			}
			d := dx
			if dy > d {
				d = dy
			}
			if d == 0 && frost <= 0 {
				continue
			}
			if d > 0 {
				a := float64(d) / float64(band)
				r.blend(x, y, Color{2, 5, 12}, a*0.78)
			}
			if frost > 0 {
				edge := math.Max(math.Abs(float64(x)/float64(W)-0.5), math.Abs(float64(y)/float64(H)-0.5)) * 2
				if edge > 1-frost*0.65 {
					r.blend(x, y, Color{214, 232, 242}, frost*0.28)
				}
			}
		}
	}
}
