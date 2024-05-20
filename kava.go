package kava

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

//go:embed RedditMono-ExtraBold.ttf
var fontData []byte

const fontFileName = "RedditMono-ExtraBold.ttf"

type Generator struct {
	font           *truetype.Font
	cache          *LRUCache
	queryParamName string
}

type GenerateOpts struct {
	Dest      io.Writer
	Text      string
	Width     int
	Height    int
	FontSize  int
	TextColor color.Color
	BgColor   color.Color
	OffsetY   int
}

type GeneratorOpts struct {
	TtfFile            string
	QueryParamName     string
	FlushCacheEverySec int64
	CacheSize          int64
}

func New(opts GeneratorOpts) (*Generator, error) {
	ig := &Generator{}
	var ff string
	if opts.TtfFile != "" {
		ff = opts.TtfFile
	} else {
		ff = fontFileName
	}
	var fnt *truetype.Font
	var err error
	if opts.TtfFile != "" {
		var fd []byte
		fd, err = os.ReadFile(ff)
		if err != nil {
			fmt.Println("Error reading font file:", err)
			return nil, err
		}
		fnt, err = freetype.ParseFont(fd)
	} else {
		fnt, err = freetype.ParseFont(fontData)
	}
	if err != nil {
		fmt.Println("Error parsing font:", err)
		return nil, err
	}
	ig.font = fnt
	if opts.CacheSize == 0 {
		opts.CacheSize = 20
	}
	ig.queryParamName = opts.QueryParamName
	ig.cache = NewLRUCache(opts.CacheSize, opts.FlushCacheEverySec)
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

func (ig *Generator) Generate(opts GenerateOpts) error {
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

	// Check the cache first
	cacheKey := opts.Text
	if cachedImage, exists := ig.cache.Get(cacheKey); exists {
		opts.Dest.Write(cachedImage) // Serve the image from cache
		return nil
	}

	// Generate the image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)
	drawText(img, opts.Text, ig.font, fonts, textColor, opts.OffsetY)

	// Encode the image to PNG
	var imgBuffer bytes.Buffer
	err := png.Encode(&imgBuffer, img)
	if err != nil {
		return err
	}

	// Cache the generated image
	ig.cache.Put(cacheKey, imgBuffer.Bytes())

	// Write the image to the destination writer
	opts.Dest.Write(imgBuffer.Bytes())
	return nil
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

// LRUCache represents an LRU cache limited by memory size
type LRUCache struct {
	memLimit int64
	currSize int64
	cache    map[string][]byte
	mutex    sync.RWMutex
}

// NewLRUCache creates a new LRUCache with the given memory limit in megabytes
func NewLRUCache(memLimitMB int64, flushEverySec ...int64) *LRUCache {
	memLimitBytes := memLimitMB * 1024 * 1024 // Convert megabytes to bytes

	cache := &LRUCache{
		memLimit: memLimitBytes,
		cache:    make(map[string][]byte),
	}
	if len(flushEverySec) > 0 && flushEverySec[0] > 0 {
		go cache.periodicFlush(flushEverySec[0])
	}
	return cache
}

func (c *LRUCache) periodicFlush(intervalSec int64) {
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		c.Flush()
	}
}

// Put inserts a value into the cache
func (c *LRUCache) Put(key string, value []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if adding this item exceeds the memory limit
	itemSize := int64(len(value))
	if c.currSize+itemSize > c.memLimit {
		// Evict older items until memory usage falls below the limit
		for c.currSize+itemSize > c.memLimit {
			// Evict the oldest item
			for k, v := range c.cache {
				delete(c.cache, k)
				c.currSize -= int64(len(v))
				break
			}
		}
	}

	// Add the new item
	c.cache[key] = value
	c.currSize += itemSize
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) ([]byte, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if val, exists := c.cache[key]; exists {
		return val, true
	}
	return nil, false
}

// Flush clears the cache
func (c *LRUCache) Flush() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string][]byte)
	c.currSize = 0
}
