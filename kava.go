package kava

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math/rand"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

//go:embed RedditMono-ExtraBold.ttf
var fontData []byte

const fontFileName = "RedditMono-ExtraBold.ttf"

type Generator struct {
	font *truetype.Font
}

var _ = embed.FS{}

type GenOpts struct {
	Dest      io.Writer
	Text      string
	Width     int
	Height    int
	FontSize  int
	TextColor color.Color
	BgColor   color.Color
	OffsetY   int
}

func New(ttfFile ...string) (*Generator, error) {
	ig := &Generator{}
	var ff string
	if len(ttfFile) > 0 {
		ff = ttfFile[0]
	} else {
		ff = fontFileName
	}
	fd, err := os.ReadFile(ff)
	if err != nil {
		fmt.Println("Error reading font file:", err)
		return nil, err
	}
	var fnt *truetype.Font
	if len(ttfFile) > 0 {
		fnt, err = freetype.ParseFont(fd)
	} else {
		fnt, err = freetype.ParseFont(fontData)
	}

	if err != nil {
		fmt.Println("Error parsing font:", err)
		return nil, err
	}
	ig.font = fnt
	return ig, nil
}

func randomColor() color.Color {
	return &color.RGBA{
		R: uint8(rand.Intn(255)),
		G: uint8(rand.Intn(255)),
		B: uint8(rand.Intn(255)),
		A: 255,
	}
}

func (ig *Generator) GetFont() *truetype.Font {
	return ig.font
}

func (ig *Generator) Generate(opts GenOpts) error {
	var (
		fonts, width, height int
		textColor, bgColor   color.Color
	)
	if opts.FontSize > 0 {
		fonts = opts.FontSize
	} else {
		fonts = 100
	}
	if opts.Width > 0 {
		width = opts.Width
	} else {
		width = 300
	}
	if opts.Height > 0 {
		height = opts.Height
	} else {
		height = 300
	}
	if opts.TextColor != nil {
		textColor = opts.TextColor
	} else {
		textColor = randomColor()
	}
	if opts.BgColor != nil {
		bgColor = opts.BgColor
	} else {
		bgColor = randomColor()
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)
	drawText(img, opts.Text, ig.font, fonts, textColor, opts.OffsetY)
	return png.Encode(opts.Dest, img)
}

// drawText draws the initials onto the image using the specified font and size
func drawText(img *image.RGBA, text string, fff *truetype.Font, fontSize int, c color.Color, offsetY int) {
	// Initialize the drawer
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: truetype.NewFace(fff, &truetype.Options{Size: float64(fontSize)}),
	}

	// Calculate the position for the initials to be centered
	width := d.MeasureString(text)
	x := (img.Bounds().Max.X - img.Bounds().Min.X - int(width.Round())) / 2
	y := (img.Bounds().Max.Y - img.Bounds().Min.Y - fontSize - offsetY) / 2

	// Draw the initials onto the image
	d.Dot = fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y + fontSize),
	}
	d.DrawString(text)
}
