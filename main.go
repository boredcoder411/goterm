package main

import (
	"log"
	"os"
	"os/exec"
	"time"
  "fmt"

	"github.com/creack/pty"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	screenWidth  = 800
	screenHeight = 600
	frameDelay   = 16 // ~60 FPS (1000ms / 60)
)

var (
	outputBuffer []string
	cursorX      int32
	cursorY      int32
	charWidth    int32 = 8  // Adjust based on your font
	charHeight   int32 = 16 // Adjust based on your font
)

// Initialize the buffer with an empty first line

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
      fmt.Println([]byte(buf[:n]))
			outputBuffer = append(outputBuffer, output)
		}
	}()

	// Game loop
	running := true

	// Game loop rendering output
	for running {
		select {
		case <-quit:
			running = false
			break
		default:
			renderer.SetDrawColor(0, 0, 0, 255) // Clear with black
			renderer.Clear()

			// Render PTY output line by line
			y := int32(0)
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

				renderer.Copy(textTexture, nil, &sdl.Rect{X: 0, Y: y, W: textSurface.W, H: textSurface.H})
				y += charHeight
			}

			renderer.Present()
			sdl.Delay(frameDelay)
		}
	}
}

// Handle keyboard events
func handleKeyboardEvent(e *sdl.KeyboardEvent, p *os.File) {
	if e.Type == sdl.KEYDOWN {
		char := e.Keysym.Sym

		// Handle special keys like Enter and Backspace
		switch char {
		case sdl.K_RETURN:
			outputBuffer = append(outputBuffer, "")
			cursorX = 0
			cursorY += charHeight
		case sdl.K_BACKSPACE:
			if len(outputBuffer) > 0 && cursorX > 0 {
				currentLine := outputBuffer[len(outputBuffer)-1]
				if len(currentLine) > 0 {
					outputBuffer[len(outputBuffer)-1] = currentLine[:len(currentLine)-1]
					cursorX -= charWidth
				}
			}
		default:
			// Append printable characters
			if len(outputBuffer) == 0 {
				outputBuffer = append(outputBuffer, "")
			}
			lastLine := outputBuffer[len(outputBuffer)-1]
			outputBuffer[len(outputBuffer)-1] = lastLine + string(char)
			cursorX += charWidth

			// Wrap to a new line if needed
			if cursorX >= screenWidth {
				outputBuffer = append(outputBuffer, "")
				cursorX = 0
				cursorY += charHeight
			}
		}

		// Write the character to the PTY
		p.Write([]byte{byte(char)})
	}
}
