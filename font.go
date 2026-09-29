package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"image/png"
	"math"
)

//go:embed font.png
var fontPNG []byte

//go:embed font.json
var fontJSON []byte

type glyph struct {
	X, Y, W, H, A, Oy int
}

var (
	fontAlpha  []byte
	fontW      int
	fontH      int
	fontGlyphs [95]glyph
	fontLine   int
	fontAscent int
)

func init() {
	im, err := png.Decode(bytes.NewReader(fontPNG))
	if err != nil {
		panic(err)
	}
	b := im.Bounds()
	fontW, fontH = b.Dx(), b.Dy()
	fontAlpha = make([]byte, fontW*fontH)
	for y := 0; y < fontH; y++ {
		for x := 0; x < fontW; x++ {
			_, _, _, a := im.At(b.Min.X+x, b.Min.Y+y).RGBA()
			fontAlpha[y*fontW+x] = byte(a >> 8)
		}
	}
	var meta struct {
		Line   int `json:"line"`
		Ascent int `json:"ascent"`
		Glyphs []struct {
			C  string `json:"c"`
			X  int    `json:"x"`
			Y  int    `json:"y"`
			W  int    `json:"w"`
			H  int    `json:"h"`
			A  int    `json:"a"`
			Oy int    `json:"oy"`
		} `json:"glyphs"`
	}
	if err := json.Unmarshal(fontJSON, &meta); err != nil {
		panic(err)
	}
	fontLine = meta.Line
	fontAscent = meta.Ascent
	for _, g := range meta.Glyphs {
		if g.C == "" {
			continue
		}
		ch := g.C[0]
		if ch < 32 || ch > 126 {
			continue
		}
		fontGlyphs[ch-32] = glyph{g.X, g.Y, g.W, g.H, g.A, g.Oy}
	}
}

func (r *Raster) blitGlyph(g glyph, dx, dy, scale float64, col Color, alpha float64) {
	if g.W == 0 || g.H == 0 || scale <= 0 {
		return
	}
	dw := int(math.Ceil(float64(g.W) * scale))
	dh := int(math.Ceil(float64(g.H) * scale))
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	ox := int(math.Floor(dx))
	oy := int(math.Floor(dy))
	for y := 0; y < dh; y++ {
		sy := g.Y + int(float64(y)/scale)
		if sy < g.Y {
			sy = g.Y
		}
		if sy >= g.Y+g.H {
			sy = g.Y + g.H - 1
		}
		py := oy + y
		if py < 0 || py >= H || sy < 0 || sy >= fontH {
			continue
		}
		row := sy * fontW
		for x := 0; x < dw; x++ {
			sx := g.X + int(float64(x)/scale)
			if sx < g.X {
				sx = g.X
			}
			if sx >= g.X+g.W {
				sx = g.X + g.W - 1
			}
			px := ox + x
			if px < 0 || px >= W || sx < 0 || sx >= fontW {
				continue
			}
			a := float64(fontAlpha[row+sx]) / 255
			if a < 0.05 {
				continue
			}
			r.blend(px, py, col, a*alpha)
		}
	}
}

func (r *Raster) textWidth(s string, scale float64) float64 {
	w := 0.0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < 32 || ch > 126 {
			ch = '?'
		}
		w += float64(fontGlyphs[ch-32].A) * scale
	}
	return w
}

func (r *Raster) text(s string, x, y, scale float64, col Color, alpha float64) {
	pen := x
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < 32 || ch > 126 {
			ch = '?'
		}
		g := fontGlyphs[ch-32]
		r.blitGlyph(g, pen, y+float64(g.Oy)*scale, scale, col, alpha)
		pen += float64(g.A) * scale
	}
}

func (r *Raster) textShadow(s string, x, y, scale float64, col Color, a float64) {
	r.text(s, x+1, y+1, scale, Color{2, 6, 14}, a*0.9)
	r.text(s, x, y, scale, col, a)
}

func (r *Raster) textCenter(s string, y, scale float64, col Color, a float64) float64 {
	w := r.textWidth(s, scale)
	r.textShadow(s, (float64(W)-w)/2, y, scale, col, a)
	return w
}

func (r *Raster) textCenterFit(s string, y, maxW, scale float64, col Color, a float64) {
	w := r.textWidth(s, scale)
	if w > maxW && w > 1 {
		scale *= maxW / w
	}
	r.textCenter(s, y, scale, col, a)
}
