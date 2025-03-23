package main

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Stage int

const (
	STAGE_GAME Stage = iota
	STAGE_OVER
	PLAYER_SPRITE_SIZE int = 16
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
	Ticks            uint64
	Player           Player
	Time             int
	Night            bool
	NightBackgrounds []*ebiten.Image
	DuskBackgrounds  []*ebiten.Image
	TrueBoundingBox  image.Rectangle
	Verbose          bool
}

func init() {
	raw, err := os.ReadFile("assets/fonts/pixelify-sans.ttf")
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
		backgroundImage, _, err := ebitenutil.NewImageFromFile("assets/images/backgrounds/night.png")
		if err != nil {
			return err
		}

		duskBackgroundImage, _, err := ebitenutil.NewImageFromFile("assets/images/backgrounds/dusk.png")
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
		opts.GeoM.Translate(float64(i*img.Bounds().Dx()-int(g.Ticks)), 0)

		screen.DrawImage(img, opts)
	}

	for i, img := range g.NightBackgrounds {
		opts := &ebiten.DrawImageOptions{}
		opts.ColorScale.ScaleAlpha(float32(g.Time) / 100)
		opts.GeoM.Translate(float64(i*img.Bounds().Dx()-int(g.Ticks)), 0)

		screen.DrawImage(img, opts)
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
			"FPS: %.0f\nVelocity: %v\nX: %.2f, Y: %.2f\nTicks: %v\nBackgrounds: %v\nTime: %v\nStage: %v",
			ebiten.ActualFPS(), g.Player.Velocity, g.Player.X, g.Player.Y, g.Ticks, len(g.NightBackgrounds), g.Time, g.Stage,
		))
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.TrueBoundingBox.Dx(), g.TrueBoundingBox.Dy()
}

func main() {
	playerImage, _, err := ebitenutil.NewImageFromFile("assets/images/player.png")
	if err != nil {
		log.Fatal(err)
		return
	}

	backgroundDuskImage, _, err := ebitenutil.NewImageFromFile("assets/images/backgrounds/dusk.png")
	if err != nil {
		log.Fatal(err)
		return
	}

	backgroundImage, _, err := ebitenutil.NewImageFromFile("assets/images/backgrounds/night.png")
	if err != nil {
		log.Fatal(err)
		return
	}

	ebiten.SetWindowSize(
		backgroundImage.Bounds().Dx()*2,
		backgroundImage.Bounds().Dy()*2,
	)
	ebiten.SetWindowTitle("Flappy Bird")
	if err := ebiten.RunGame(&Game{
		Stage: STAGE_GAME,
		Player: Player{
			Sprite:   playerImage,
			X:        float64(backgroundImage.Bounds().Dx()) / 3,
			Y:        float64(backgroundImage.Bounds().Dy()/2) - float64(PLAYER_SPRITE_SIZE),
			Velocity: 2,
		},
		DuskBackgrounds:  []*ebiten.Image{backgroundDuskImage},
		NightBackgrounds: []*ebiten.Image{backgroundImage},
		TrueBoundingBox:  backgroundImage.Bounds(),
	}); err != nil {
		log.Fatal(err)
	}
}
