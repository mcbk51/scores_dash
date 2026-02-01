package config

import (
	"github.com/gdamore/tcell/v2"
)


func NewInputHandler(scroller *Scroller, display *Display, quit func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC, tcell.KeyEscape:
			quit()
			return nil
		}

		switch event.Rune() {
		case 'q':
			quit()
			return nil
		case 's', 'S':
			scroller.Toggle()
			go display.UpdateScores()
			return nil
		case '+', '=':
			scroller.SpeedUp()
			go display.UpdateScores()
			return nil
		case '-', '_':
			scroller.SlowDown()
			go display.UpdateScores()
			return nil
		case 'r', 'R':
			scroller.Reverse()
			go display.UpdateScores()
			return nil
		case 'j':
			scroller.ScrollDown()
			return nil
		case 'k':
			scroller.ScrollUp()
			return nil
		}
		return event
	}
}

