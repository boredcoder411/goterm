package main

import (
	"log"
	"os"
	"os/exec"
	"strings"

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
	charWidth    int32 = 8
	charHeight   int32 = 16
)

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Fatalf("Could not initialize SDL: %v", err)
	}
	defer sdl.Quit()

	// Initialize TTF
	if err := ttf.Init(); err != nil {
		log.Fatalf("Could not initialize TTF: %v", err)
	}
	defer ttf.Quit()

	window, err := sdl.CreateWindow("SDL2 Text Input Example", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		screenWidth, screenHeight, sdl.WINDOW_SHOWN)
	if err != nil {
		log.Fatalf("Could not create window: %v", err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatalf("Could not create renderer: %v", err)
	}
	defer renderer.Destroy()

	// Load font
	fontPath := "nerd_font.ttf"
	font, err := ttf.OpenFont(fontPath, 14)
	if err != nil {
		log.Fatalf("Could not load font: %v", err)
	}
	defer font.Close()

	// Start text input
	sdl.StartTextInput()
	defer sdl.StopTextInput()

	// Initialize PTY
	os.Setenv("TERM", "dumb")
	c := exec.Command("/bin/bash")
	p, err := pty.Start(c)
	if err != nil {
		log.Fatalf("Could not start pty: %v", err)
	}

	// Goroutine to handle PTY output
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

	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.TextInputEvent:
				handleTextInputEvent(e, p)
			case *sdl.KeyboardEvent:
				handleSpecialKeys(e, p)
			}
		}

		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()

		y := int32(0)
		joinedOutput := strings.Join(outputBuffer, "")
		for _, line := range strings.Split(joinedOutput, "\n") {
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

func handleTextInputEvent(e *sdl.TextInputEvent, p *os.File) {
	text := e.GetText()
	log.Printf("Text input: %s", text)
	p.Write([]byte(text))
}

// handle backspace and enter cause its bullshit
func handleSpecialKeys(e *sdl.KeyboardEvent, p *os.File) {
	if e.Type == sdl.KEYDOWN {
		switch e.Keysym.Sym {
		case sdl.K_RETURN:
			p.Write([]byte{'\n'})
			outputBuffer = append(outputBuffer, "")
		case sdl.K_BACKSPACE:
			if len(outputBuffer) > 0 {
				currentLine := outputBuffer[len(outputBuffer)-1]
				if len(currentLine) > 0 {
					outputBuffer[len(outputBuffer)-1] = currentLine[:len(currentLine)-1]
				}
			}
		}
	}
}

