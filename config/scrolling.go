package config

import (
	"context"
	"sync"
	"time"

	"github.com/rivo/tview"
)

type Scroller struct {
	mu        sync.Mutex
	enabled   bool
	speed     time.Duration
	direction int
	view      *tview.TextView
	app       *tview.Application
}

func NewScroller(app *tview.Application, view *tview.TextView) *Scroller {
	return &Scroller{
		enabled:   false,
		speed:     time.Millisecond * 4000,
		direction: 1,
		view:      view,
		app:       app,
	}
}

func (s *Scroller) Toggle() {
	s.mu.Lock()
	s.enabled = !s.enabled
	s.mu.Unlock()
}

func (s *Scroller) IsEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled
}

func (s *Scroller) SpeedUp() {
	s.mu.Lock()
	if s.speed > 100*time.Millisecond {
		s.speed -= time.Millisecond * 100
	}
	s.mu.Unlock()
}

func (s *Scroller) SlowDown() {
	s.mu.Lock()
	if s.speed < 2000*time.Millisecond {
		s.speed += time.Millisecond * 100
	}
	s.mu.Unlock()
}

func (s *Scroller) Reverse() {
	s.mu.Lock()
	s.direction *= -1
	s.mu.Unlock()
}

func (s *Scroller) GetSpeed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.speed
}

func (s *Scroller) ScrollUp() {
	row, col := s.view.GetScrollOffset()
	if row > 0 {
		s.view.ScrollTo(row-1, col)
	}
}

func (s *Scroller) ScrollDown() {
	row, col := s.view.GetScrollOffset()
	s.view.ScrollTo(row+1, col)
}

func (s *Scroller) Start(ctx context.Context, quitChan chan bool) {
	go func() {
		resetTicker := time.NewTicker(time.Second * 100)
		defer resetTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-quitChan:
				return
			case <-resetTicker.C:
				s.app.QueueUpdateDraw(func() {
					s.view.ScrollTo(0, 0)
				})
			default:
				s.mu.Lock()
				enabled := s.enabled
				speed := s.speed
				dir := s.direction
				s.mu.Unlock()

				if !enabled {
					s.app.QueueUpdateDraw(func() {
						row, col := s.view.GetScrollOffset()
						newRow := row + dir
						if newRow >= 0 {
							s.view.ScrollTo(newRow, col)
						}
					})
				}
				time.Sleep(speed)
			}
		}
	}()
}

func (s *Scroller) StatusString() string {
	s.mu.Lock()
 	defer s.mu.Unlock()

	if s.enabled {
		return "[green]scroll: on (%dms)[-]"
	}
	return "[gray]scroll: off[-]"
}

func (s *Scroller) FormatStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.enabled {
		speedMs := s.speed.Milliseconds()
		dir := "↓"
		if s.direction < 0 {
			dir = "↑"
		}
		return "[green]scroll: on " + dir + " (" + itoa(int(speedMs)) + "ms)[-]"
	}
	return "[gray]scroll: off[-]"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte

	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
