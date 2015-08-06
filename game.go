package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

func tbprint(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
}

func tbrect(x, y, w, h int, fg, bg termbox.Attribute, border bool) {
	end := " " + strings.Repeat("_", w)
	if border {
		tbprint(x, y-1, fg, bg, end)
	}

	s := strings.Repeat(" ", w)
	if border {
		s = fmt.Sprintf("%c%s%c", '|', s, '|')
	}

	for i := 0; i < h; i++ {
		tbprint(x, y, fg, bg, s)
		y++
	}

	if border {
		tbprint(x, y, fg, bg, end)
	}
}

// print a multi-line sprite
func tbprintsprite(x, y int, fg, bg termbox.Attribute, sprite string) {
	lines := strings.Split(sprite, "\n")
	for _, l := range lines {
		tbprint(x, y, fg, bg, l)
		y++
	}
}

const (
	highscoreFilename = "hs"
	fgDefault         = termbox.ColorRed
	bgDefault         = termbox.ColorYellow
	fps               = 30
)

// GameState is used as an enum
type GameState uint8

const (
	MenuState GameState = iota
	HowtoState
	PlayState
)

type Game struct {
	highscores map[string]int

	state GameState
	evq   chan termbox.Event
	timer <-chan time.Time

	// frame counter
	fc uint8

	// highlighted menu item
	hmi int
	w   int
	h   int

	// fg and bg colors used when termbox.Clear() is called
	cfg termbox.Attribute
	cbg termbox.Attribute
}

func NewGame() *Game {
	return &Game{
		highscores: make(map[string]int),
		evq:        make(chan termbox.Event),
		timer:      time.Tick(time.Duration(1000/fps) * time.Millisecond),
		fc:         1,
	}
}

// Tick allows us to rate limit the FPS
func (g *Game) Tick() {
	<-g.timer
	g.fc++
	if g.fc > fps {
		g.fc = 1
	}
}

func (g *Game) Listen() {
	go func() {
		for {
			g.evq <- termbox.PollEvent()
		}
	}()
}

func (g *Game) HandleKey(k termbox.Key) {
	switch g.state {
	case MenuState:
		g.HandleKeyMenu(k)
	case HowtoState:
		g.HandleKeyHowto(k)
	case PlayState:
		g.HandleKeyPlay(k)
	}
}

func (g *Game) FitScreen() {
	termbox.Clear(g.cfg, g.cbg)
	g.w, g.h = termbox.Size()
	g.Draw()
}

func (g *Game) Draw() {
	termbox.Clear(g.cfg, g.cbg)

	switch g.state {
	case MenuState:
		g.DrawMenu()
	case HowtoState:
		g.DrawHowto()
	case PlayState:
		g.DrawPlay()
	}

	termbox.Flush()
}

func (g *Game) Update() {
	g.Tick()

	switch g.state {
	case MenuState:
		g.UpdateMenu()
	case HowtoState:
		g.UpdateHowto()
	case PlayState:
		g.UpdatePlay()
	}

	return
}

func (g *Game) loadHighscores() {
	data, err := ioutil.ReadFile(highscoreFilename)
	if err != nil {
		log.Fatalln(err)
	}
	lines := strings.Split(string(data), "\n")
	for _, l := range lines {
		parts := strings.Split(l, ":")
		if i, err := strconv.Atoi(parts[1]); err == nil {
			g.highscores[parts[0]] = i
		} else {
			log.Fatalln(err)
		}
	}
}

func main() {
	if err := termbox.Init(); err != nil {
		log.Fatalln(err)
	}
	termbox.SetOutputMode(termbox.Output256)
	defer termbox.Close()

	f, err := os.Create("diwe.log")
	if err != nil {
		log.Fatalln(err)
	}
	log.SetOutput(f)

	g := NewGame()

	if _, err := os.Stat(highscoreFilename); err == nil {
		g.loadHighscores()
	}

	g.Listen()
	g.GoMenu()
	g.FitScreen()

main:
	for {
		select {
		case ev := <-g.evq:
			switch ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case 0:
					if ev.Ch == 'q' {
						break main
					}
				default:
					g.HandleKey(ev.Key)
				}
			case termbox.EventResize:
				g.FitScreen()
			}
		default:
		}

		g.Update()
		g.Draw()
	}
}
