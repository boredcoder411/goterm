package main

import (
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	screenWidth  = 800
	screenHeight = 600
	frameDelay   = 16 // ~60 FPS (1000ms / 60)
)

type Cursor struct {
	X, Y int
}

func main() {
	// Initialize SDL
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Fatalf("Could not initialize SDL: %v", err)
	}
	defer sdl.Quit()

	// Initialize TTF
	if err := ttf.Init(); err != nil {
		log.Fatalf("Could not initialize TTF: %v", err)
	}
	defer ttf.Quit()

	// Create window
	window, err := sdl.CreateWindow("SDL2 Frame-Based Text Rendering", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		screenWidth, screenHeight, sdl.WINDOW_SHOWN)
	if err != nil {
		log.Fatalf("Could not create window: %v", err)
	}
	defer window.Destroy()

	// Create renderer
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatalf("Could not create renderer: %v", err)
	}
	defer renderer.Destroy()

	// Load font
	fontPath := "/home/dev/nerd_font.ttf"   // Replace with your font file path
	font, err := ttf.OpenFont(fontPath, 14) // 14 is the font size
	if err != nil {
		log.Fatalf("Could not load font: %v", err)
	}
	defer font.Close()

	os.Setenv("TERM", "dumb")
	c := exec.Command("/bin/bash")
	p, err := pty.Start(c)

	if err != nil {
		log.Fatalf("Could not start pty: %v", err)
		os.Exit(1)
	}

	// Goroutine to handle event processing
	quit := make(chan bool) // Channel to signal quitting the program
	go func() {
		for {
			event := sdl.PollEvent()
			if event != nil {
				switch e := event.(type) {
				case *sdl.QuitEvent:
					quit <- true
					return
				case *sdl.KeyboardEvent:
					handleKeyboardEvent(e, p)
				}
			}
			time.Sleep(1 * time.Millisecond) // Prevent CPU overuse
		}
	}()

	var outputBuffer []string

	// Goroutine to read from PTY
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := p.Read(buf)
			if err != nil {
				log.Fatalf("Error reading from PTY: %v", err)
			}
			output := string(buf[:n])
			outputBuffer = append(outputBuffer, output)
		}
	}()

	// Game loop
	running := true
	for running {
		select {
		case <-quit:
			running = false
			break
		default:
			renderer.SetDrawColor(0, 0, 0, 255) // Clear with black
			renderer.Clear()

			// Render PTY output
			y := 0
			for _, line := range outputBuffer {
				textSurface, err := font.RenderUTF8Solid(line, sdl.Color{R: 255, G: 255, B: 255, A: 255})
				if err != nil {
					log.Printf("Error rendering text: %v", err)
					continue
				}
				defer textSurface.Free()

				textTexture, err := renderer.CreateTextureFromSurface(textSurface)
				if err != nil {
					log.Printf("Error creating texture: %v", err)
					continue
				}
				defer textTexture.Destroy()

				renderer.Copy(textTexture, nil, &sdl.Rect{X: 0, Y: int32(y), W: textSurface.W, H: textSurface.H})
			}

			renderer.Present()
			sdl.Delay(frameDelay)
		}
	}
}

func handleKeyboardEvent(e *sdl.KeyboardEvent, p *os.File) {
	if e.Type == sdl.KEYDOWN {
		char := e.Keysym.Sym
		p.Write([]byte{byte(char)})
	}
}
