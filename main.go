package main    

import (
	"context"
	"fmt"
	"time"
	"os"
	"os/signal"
	"syscall"

	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/mcbk51/scores_dash/config"
)

func main (){
	app := tview.NewApplication()

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = tcell.ColorDefault
	
	scoreview := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quitChan := make(chan bool, 1)
	quit := func() {
		select {
		case quitChan <- true:
		default:
		}
		cancel()

		go func() {
			time.Sleep(time.Millisecond * 30)
			fmt.Println("\nQuitting...")
			os.Exit(0)
		}()
		app.Stop()
	}

	// Scrolling
	scroller := config.NewScroller(app, scoreview)
	scroller.Start(ctx, quitChan)

	display := config.NewDisplay(app, scoreview, scroller, ctx, quitChan)

	// Handle signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		quit()
	}()

	// Input Handling
	scoreview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
	})


	// Initial Load
	go display.UpdateScores()

	// Refresh ticker
	display.StartTicker(time.Second * 30)

	if err := app.SetRoot(scoreview, true).Run(); err != nil {
		os.Exit(0)
	}
}
