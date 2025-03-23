package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"

	"github.com/bjvanbemmel/go-templ/assets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Stage int
type Pipe struct {
	Height         int
	RenderedHeight float64
	Mutex          *sync.Mutex
	X              float64
	Y              float64
	Image          *ebiten.Image
}

type Pipes struct {
	Top    *Pipe
	Bottom *Pipe
}

const (
	STAGE_GAME Stage = iota
	STAGE_OVER
	PLAYER_SPRITE_SIZE           int = 16
	PIPE_PART_SPRITE_SIZE_HEIGHT int = 16
	PIPE_PART_SPITE_SIZE_WIDTH   int = 32
)

var (
	pixelifySansFaceSource *text.GoTextFaceSource
)

type Player struct {
	Sprite       *ebiten.Image
	CurrentFrame int
	X            float64
	Y            float64
	Velocity     float64
	IsJumping    bool
}

type Game struct {
	Stage            Stage
	Pipes            []*Pipes
	Player           Player
	Ticks            uint64
	Time             int
	Night            bool
	NightBackgrounds []*ebiten.Image
	DuskBackgrounds  []*ebiten.Image
	TrueBoundingBox  image.Rectangle
	Verbose          bool
}

// TODO: Make the Top pipe work
func (g *Game) AddPipes(amount int) error {
	for range amount {
		// side := rand.Intn(2)
		pipe := Pipes{
			Bottom: &Pipe{
				Mutex: &sync.Mutex{},
			},
		}
		var err error

		pipe.Bottom.Image, _, err = ebitenutil.NewImageFromFileSystem(assets.FS, "images/pipes/pipes.png")
		if err != nil {
			return err
		}
		pipe.Bottom.Height = rand.Intn(g.TrueBoundingBox.Dy()/2) + PIPE_PART_SPRITE_SIZE_HEIGHT

		g.Pipes = append(g.Pipes, &pipe)
	}

	return nil
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		os.Exit(0)
		return nil
	}

	if g.Stage != STAGE_GAME {
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		g.Verbose = !g.Verbose
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.Player.Velocity = -10
	}

	if int(g.Ticks)%g.TrueBoundingBox.Dx() == 0 {
		backgroundImage, _, err := ebitenutil.NewImageFromFileSystem(assets.FS, "images/backgrounds/night.png")
		if err != nil {
			return err
		}

		duskBackgroundImage, _, err := ebitenutil.NewImageFromFileSystem(assets.FS, "images/backgrounds/dusk.png")
		if err != nil {
			return err
		}

		g.NightBackgrounds = append(g.NightBackgrounds, backgroundImage)
		g.DuskBackgrounds = append(g.DuskBackgrounds, duskBackgroundImage)
	}

	if g.Player.Velocity > 20 {
		g.Player.Velocity = 20
	}

	g.Player.Velocity += 1
	g.Player.Y += 0.3 * g.Player.Velocity

	maxY := float64(g.TrueBoundingBox.Dx())
	if g.Player.Y > maxY || g.Player.Y < float64(0-PLAYER_SPRITE_SIZE) {
		g.Stage = STAGE_OVER
	}

	if len(g.Pipes) == 0 || g.Pipes[len(g.Pipes)-1].Bottom != nil && g.Pipes[len(g.Pipes)-1].Bottom.X < float64(g.TrueBoundingBox.Dx()) {
		g.AddPipes(1)
	}

	for _, pipe := range g.Pipes {
		// TODO: Enable top pipes and check their coords
		if pipe.Bottom == nil {
			continue
		}
		pipe.Bottom.Mutex.Lock()

		if g.Player.X+float64(PLAYER_SPRITE_SIZE) < pipe.Bottom.X || g.Player.X+float64(PLAYER_SPRITE_SIZE) > pipe.Bottom.X+float64(PIPE_PART_SPITE_SIZE_WIDTH) {
			pipe.Bottom.Mutex.Unlock()
			continue
		}

		if g.Player.Y < pipe.Bottom.Y-pipe.Bottom.RenderedHeight || g.Player.Y > pipe.Bottom.Y {
			pipe.Bottom.Mutex.Unlock()
			continue
		}

		g.Stage = STAGE_OVER
		pipe.Bottom.Mutex.Unlock()
	}

	if g.Ticks == ^uint64(0) {
		g.Ticks = 0
	}
	g.Ticks += 1

	if g.Ticks%50 == 0 {
		if g.Night {
			g.Time -= 1
		} else {
			g.Time += 1
		}
	}

	if g.Time == 0 {
		g.Night = false
	} else if g.Time == 100 {
		g.Night = true
	}

	if g.Ticks%10 != 0 {
		return nil
	}

	if g.Player.CurrentFrame < 3 {
		g.Player.CurrentFrame += 1
	} else {
		g.Player.CurrentFrame = 0
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for i, img := range g.DuskBackgrounds {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(i*img.Bounds().Dx()-int(g.Ticks/2)), 0)

		screen.DrawImage(img, opts)
	}

	for i, img := range g.NightBackgrounds {
		opts := &ebiten.DrawImageOptions{}
		opts.ColorScale.ScaleAlpha(float32(g.Time) / 100)
		opts.GeoM.Translate(float64(i*img.Bounds().Dx()-int(g.Ticks/2)), 0)

		screen.DrawImage(img, opts)
	}

	for i, pipe := range g.Pipes {
		if pipe.Bottom.Height > 0 {
			// We use a Mutex here, because we need the RenderedHeight property when checking for collisions within the Update() method.
			// The Draw() and Update() method are being executed within different goroutines, so in order to preserve data integrity (checking only after completely
			// setting the RenderedHeight), we use a Mutex Lock here.
			pipe.Bottom.Mutex.Lock()
			pipe.Bottom.RenderedHeight = 0
			pipe.Bottom.X = float64(g.TrueBoundingBox.Dx()) + float64(i*PIPE_PART_SPITE_SIZE_WIDTH*3) - float64(g.Ticks)
			pipe.Bottom.Y = float64(g.TrueBoundingBox.Dy() - PIPE_PART_SPRITE_SIZE_HEIGHT)

			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(pipe.Bottom.X, pipe.Bottom.Y)
			screen.DrawImage(
				pipe.Bottom.Image.SubImage(
					image.Rect(0, 0, PIPE_PART_SPITE_SIZE_WIDTH, PIPE_PART_SPRITE_SIZE_HEIGHT),
				).(*ebiten.Image),
				opts,
			)

			pipe.Bottom.RenderedHeight += float64(PIPE_PART_SPRITE_SIZE_HEIGHT)

			height := 1
			for height <= pipe.Bottom.Height/PIPE_PART_SPRITE_SIZE_HEIGHT {
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(pipe.Bottom.X, pipe.Bottom.Y-float64(height)*float64(PIPE_PART_SPRITE_SIZE_HEIGHT))

				screen.DrawImage(
					pipe.Bottom.Image.SubImage(
						image.Rect(0, PIPE_PART_SPRITE_SIZE_HEIGHT, PIPE_PART_SPITE_SIZE_WIDTH, PIPE_PART_SPRITE_SIZE_HEIGHT*2),
					).(*ebiten.Image),
					opts,
				)

				height += 1
				pipe.Bottom.RenderedHeight += float64(PIPE_PART_SPRITE_SIZE_HEIGHT)
			}

			headerOpts := &ebiten.DrawImageOptions{}
			headerOpts.GeoM.Translate(pipe.Bottom.X, pipe.Bottom.Y-float64(height)*float64(PIPE_PART_SPRITE_SIZE_HEIGHT))
			screen.DrawImage(
				pipe.Bottom.Image.SubImage(
					image.Rect(0, 0, PIPE_PART_SPITE_SIZE_WIDTH, PIPE_PART_SPRITE_SIZE_HEIGHT),
				).(*ebiten.Image),
				headerOpts,
			)
			pipe.Bottom.RenderedHeight += float64(PIPE_PART_SPRITE_SIZE_HEIGHT)
			pipe.Bottom.Mutex.Unlock()
		}
	}

	opts := ebiten.DrawImageOptions{}
	// We want to rotate from the center of surface, so we create some sort of anchor point by moving half-width and half-height before rotating.
	opts.GeoM.Translate(-float64(PLAYER_SPRITE_SIZE)/2, -(float64(PLAYER_SPRITE_SIZE) / 2))
	opts.GeoM.Rotate(g.Player.Velocity * 6 / 96.0 * math.Pi / 6)
	opts.Filter = ebiten.FilterNearest
	opts.GeoM.Translate(float64(PLAYER_SPRITE_SIZE)/2, float64(PLAYER_SPRITE_SIZE)/2)

	// Move bird according to the game logic.
	opts.GeoM.Translate(g.Player.X, g.Player.Y)

	x0, y0, x1, y1 := 0, 0, PLAYER_SPRITE_SIZE, PLAYER_SPRITE_SIZE

	if g.Player.CurrentFrame > 0 {
		x0 = PLAYER_SPRITE_SIZE * g.Player.CurrentFrame
		x1 = PLAYER_SPRITE_SIZE * (g.Player.CurrentFrame + 1)
	}

	screen.DrawImage(
		g.Player.Sprite.SubImage(
			image.Rect(x0, y0, x1, y1),
		).(*ebiten.Image),
		&opts,
	)

	if g.Verbose {
		ebitenutil.DebugPrint(screen, fmt.Sprintf(
			"FPS: %.0f\nVelocity: %v\nX: %.2f, Y: %.2f\nTicks: %v\nBackgrounds: %v\nTime: %v\nStage: %v\nPipes: %v",
			ebiten.ActualFPS(), g.Player.Velocity, g.Player.X, g.Player.Y, g.Ticks, len(g.NightBackgrounds), g.Time, g.Stage, len(g.Pipes),
		))
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.TrueBoundingBox.Dx(), g.TrueBoundingBox.Dy()
}

func main() {
	raw, err := assets.FS.ReadFile("fonts/pixelify-sans.ttf")
	if err != nil {
		log.Fatal(err)
		return
	}

	f, err := text.NewGoTextFaceSource(bytes.NewReader(raw))
	if err != nil {
		log.Fatal(err)
		return
	}

	pixelifySansFaceSource = f

	playerImage, _, err := ebitenutil.NewImageFromFileSystem(assets.FS, "images/player.png")
	if err != nil {
		log.Fatal(err)
		return
	}

	backgroundDuskImage, _, err := ebitenutil.NewImageFromFileSystem(assets.FS, "images/backgrounds/dusk.png")
	if err != nil {
		log.Fatal(err)
		return
	}

	backgroundImage, _, err := ebitenutil.NewImageFromFileSystem(assets.FS, "images/backgrounds/night.png")
	if err != nil {
		log.Fatal(err)
		return
	}

	ebiten.SetWindowSize(
		backgroundImage.Bounds().Dx()*2,
		backgroundImage.Bounds().Dy()*2,
	)
	ebiten.SetWindowTitle("Flappy Bird")

	game := Game{
		Stage: STAGE_GAME,
		Player: Player{
			Sprite:   playerImage,
			X:        float64(backgroundImage.Bounds().Dx()) / 3,
			Y:        float64(backgroundImage.Bounds().Dy()/2) - float64(PLAYER_SPRITE_SIZE),
			Velocity: 2,
		},
		Pipes:            []*Pipes{},
		DuskBackgrounds:  []*ebiten.Image{backgroundDuskImage},
		NightBackgrounds: []*ebiten.Image{backgroundImage},
		TrueBoundingBox:  backgroundImage.Bounds(),
	}

	if err := ebiten.RunGame(&game); err != nil {
		log.Fatal(err)
	}
}
