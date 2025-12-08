package utils

import (
	"bytes"
	"image/color"
	"math/rand"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

type Captcha struct {
	Code  string
	Image []byte
}

const (
	width  = 300
	height = 120
)

func GenerateCaptcha() (*Captcha, error) {
	dc := gg.NewContext(width, height)
	rand.Seed(time.Now().UnixNano())

	// Background gradient
	grad := gg.NewLinearGradient(0, 0, width, height)
	grad.AddColorStop(0, color.RGBA{240, 240, 240, 255})
	grad.AddColorStop(0.5, color.White)
	grad.AddColorStop(1, color.RGBA{232, 232, 232, 255})
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, width, height)
	dc.Fill()

	// Add noise (dots)
	for i := 0; i < 100; i++ {
		dc.SetColor(randomColor(100, 200))
		dc.DrawRectangle(rand.Float64()*width, rand.Float64()*height, 2, 2)
		dc.Fill()
	}

	// Add distortion lines
	for i := 0; i < 5; i++ {
		dc.SetColor(randomColor(150, 220))
		dc.SetLineWidth(rand.Float64()*2 + 1)
		dc.MoveTo(rand.Float64()*width, rand.Float64()*height)
		dc.CubicTo(
			rand.Float64()*width, rand.Float64()*height,
			rand.Float64()*width, rand.Float64()*height,
			rand.Float64()*width, rand.Float64()*height,
		)
		dc.Stroke()
	}

	// Generate code
	code := generateCode(6)

	// Load font
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	face := truetype.NewFace(font, &truetype.Options{Size: 48})
	dc.SetFontFace(face)

	// Draw text
	charSpacing := float64(width) / float64(len(code)+1)
	for i, char := range code {
		x := charSpacing * float64(i+1)
		y := float64(height) / 2

		dc.Push()
		dc.RotateAbout(gg.Radians((rand.Float64()-0.5)*20), x, y)

		// Shadow
		dc.SetColor(color.RGBA{0, 0, 0, 50})
		dc.DrawStringAnchored(string(char), x+2, y+2, 0.5, 0.5)

		// Main text
		dc.SetColor(randomColor(0, 100))
		dc.DrawStringAnchored(string(char), x, y, 0.5, 0.5)

		dc.Pop()
	}

	// More noise on top
	for i := 0; i < 50; i++ {
		dc.SetColor(randomColor(200, 255))
		dc.DrawRectangle(rand.Float64()*width, rand.Float64()*height, 1, 1)
		dc.Fill()
	}

	buf := new(bytes.Buffer)
	if err := dc.EncodePNG(buf); err != nil {
		return nil, err
	}

	return &Captcha{
		Code:  code,
		Image: buf.Bytes(),
	}, nil
}

func generateCode(length int) string {
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func randomColor(min, max int) color.Color {
	r := rand.Intn(max-min) + min
	g := rand.Intn(max-min) + min
	b := rand.Intn(max-min) + min
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}
