package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/leaanthony/go-ansi-parser"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	screenWidth  = 800
	screenHeight = 600
	frameDelay   = 16 // ~60 FPS (1000ms / 60)
)

type Cursor struct {
	X              int32
	Y              int32
	currentFgColor sdl.Color
	currentBgColor sdl.Color
}

var (
	outputBuffer []*ansi.StyledText
	charWidth    int32  = 8
	charHeight   int32  = 16
	cursor       Cursor = Cursor{
		X:              0,
		Y:              0,
		currentFgColor: sdl.Color{R: 255, G: 255, B: 255, A: 255},
	}
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
	os.Setenv("TERM", "ansi")
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
			handleAnsi(buf[:n])
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

		y := cursor.Y
		x := cursor.X // Track the X position for rendering characters within a line

		for _, segment := range outputBuffer {
			color := sdl.Color{R: 255, G: 255, B: 255, A: 255}
			if segment.FgCol != nil {
				color = sdl.Color{
					R: segment.FgCol.Rgb.R,
					G: segment.FgCol.Rgb.G,
					B: segment.FgCol.Rgb.B,
					A: 255,
				}
			}

			// Render each character in the segment
			for _, char := range segment.Label {
				if char == '\n' {
					x = 0           // Reset X position to the start of the line
					y += charHeight // Move to the next line
					continue
				}

				charSurface, err := font.RenderUTF8Solid(string(char), color)
				if err != nil {
					log.Printf("Error rendering character: %v", err)
					continue
				}
				defer charSurface.Free()

				charTexture, err := renderer.CreateTextureFromSurface(charSurface)
				if err != nil {
					log.Printf("Error creating texture: %v", err)
					continue
				}
				defer charTexture.Destroy()

				// Render the character at the current position
				renderer.Copy(charTexture, nil, &sdl.Rect{X: x, Y: y, W: charSurface.W, H: charSurface.H})

				// Advance the X position for the next character
				x += charWidth
			}
		}

		renderer.Present()
		sdl.Delay(frameDelay)
	}
}

func handleTextInputEvent(e *sdl.TextInputEvent, p *os.File) {
	text := e.GetText()
	p.Write([]byte(text))
}

func handleSpecialKeys(e *sdl.KeyboardEvent, p *os.File) {
    if e.Type == sdl.KEYDOWN {
        switch e.Keysym.Sym {
        case sdl.K_RETURN:
            p.Write([]byte{'\n'}) // Send the newline to the PTY

            // Add a newline to the outputBuffer to render it immediately
            outputBuffer = append(outputBuffer, &ansi.StyledText{
                Label: "\n",
                FgCol: &ansi.Col{
                    Rgb: ansi.Rgb{R: 255, G: 255, B: 255},
                },
            })
        case sdl.K_BACKSPACE:
            p.Write([]byte{'\x7f'}) // Send DEL character to PTY
        }
    }
}

func handleAnsi(buf []byte) {
	if len(buf) == 0 {
		return
	}
	parsed, err := ansi.Parse(string(buf))
	if err != nil {
		log.Printf("Error parsing ANSI: %v\n  String was: %s", err, string(buf))
		return
	}

	for _, segment := range parsed {
		outputBuffer = append(outputBuffer, segment)
	}
}
