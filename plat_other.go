//go:build !js && !windows

package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

func readSave() [5]int { return [5]int{} }

func writeSave([5]int) {}

func playSFX(string) {}

func writePNG(path string, pix []byte) error {
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	copy(img.Pix, pix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-dump" {
		g := newGame()
		for i := 0; i < 40; i++ {
			g.Frame(1.0 / 60)
		}
		if err := writePNG("/workspace/screenshots/keel-title.png", g.Render()); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		g.Start(0)
		g.qaOn = true
		g.qa = map[string]bool{"KeyW": true}
		for i := 0; i < 150; i++ {
			g.Frame(1.0 / 60)
		}
		if err := writePNG("/workspace/screenshots/keel-sail.png", g.Render()); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		g.Mode = ModeCharts
		g.menuSel = 1
		if err := writePNG("/workspace/screenshots/keel-charts.png", g.Render()); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		g.Mode = ModeHelp
		if err := writePNG("/workspace/screenshots/keel-help.png", g.Render()); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("dumped")
		return
	}
	fmt.Println("Aurora Keel — build the Windows game with:")
	fmt.Println("  GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags \"-s -w -H windowsgui\" -o AuroraKeel.exe .")
}
