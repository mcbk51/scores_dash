package main    

import (
	"context"
	"fmt"
	"time"
	"sort"
	"os"
	"os/signal"
	"syscall"

	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/mcbk51/scores_dash/api"
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

	quit := func() {
		cancel()
		app.Stop()
		os.Exit(0)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		quit()
	}()

	updateScores := func() {
		games, err := api.GetGames("all", time.Now())
		if err != nil {
			fmt.Fprintf(scoreview, "[red]Error fetching scores: %v[-]\n", err)
			app.Draw()
			return
		}

		scoreview.Clear()
		fmt.Fprintf(scoreview, "[yellow]=== LIVE SPORTS SCORES ===[-]\n\n")


		//  Group by league
		gamesByLeague := make(map[string][]api.Game)
		for _, game := range games {
			if config.IsLive(game.Status) || config.IsUpcoming(game.StartTime, 30*time.Minute) {
				gamesByLeague[game.League] = append(gamesByLeague[game.League], game)
			}
		}

		leagueOrder := []string{"NFL", "NBA", "NHL", "MLB"}
		leagueColors := map[string]string{
			"NFL": "red",
			"NBA": "blue",
			"NHL": "orange",
			"MLB": "green",
		}

		for _, league := range leagueOrder {
			games, exists := gamesByLeague[league]
			if !exists || len(games) == 0 {
				nextGame := config.FindNextGame(league)
				fmt.Fprintf(scoreview, "[%s]▼ %s[-]\n", leagueColors[league], league)
				fmt.Fprintf(scoreview, "  [gray]No games currently[-]\n")
				if !nextGame.IsZero() {
					localTime := nextGame.Local()
					fmt.Fprintf(scoreview, "  [gray]Next game: %s[-]\n\n", localTime.Format("Mon, Jan 2 at 3:04 PM"))
				} else {
					fmt.Fprintf(scoreview, "\n")
				}
				continue
			}

			sort.Slice(games, func(i, j int) bool {
				statusI := config.IsLive(games[i].Status)
				statusJ := config.IsLive(games[j].Status)
				if statusI != statusJ {
					return statusI
				}
				return games[i].StartTime.Before(games[j].StartTime)
			})

			liveCount := config.CountLiveGames(games)
			if liveCount > 0 {
				fmt.Fprintf(scoreview, "[%s]▼ %s[-] [green]● %d LIVE[-]\n", leagueColors[league], league, liveCount)
			} else {
				fmt.Fprintf(scoreview, "[%s]▼ %s[-]\n", leagueColors[league], league)
			}

			for _, game := range games {
				statusColor := "white"
				statusText := game.Status

				if config.IsLive(game.Status) {
					statusColor = "green"
					statusText = "LIVE"
				} else if config.IsUpcoming(game.StartTime, 30*time.Minute) {
					statusColor = "yellow"
					localTime := game.StartTime.Local()
					minutesUntil := int(time.Until(game.StartTime).Minutes())
					statusText = fmt.Sprintf("Starts in %dm (%s)", minutesUntil, localTime.Format("3:04 PM"))
				} else {
					continue
				}
					
				fmt.Fprintf(scoreview, "  %s (%s) [white]%d[-]  @  %s (%s) [white]%d[-]  [%s][%s][-]\n",
					game.AwayTeam,
					game.AwayRecord,
					game.AwayScore,
					game.HomeTeam,
					game.HomeRecord,
					game.HomeScore,
					statusColor,
					statusText)
			}
			fmt.Fprintf(scoreview, "\n")
			
		}

		app.Draw()
	}

	// Input capture
	scoreview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC || event.Rune() == 'q' || event.Rune() == 'Q' {
			quit()
			return nil
		}
		return event
	})

	// Initial Load
	go updateScores()

	// Update footer every second
	go func() {
		footerTicker := time.NewTicker(time.Second)
		defer footerTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return

			case <-footerTicker.C:
				app.QueueUpdateDraw(func() {
				})
			}
		}
	}()

	//Set up a ticker to update scores every 30 seconds
	go func() {
		ticker := time.NewTicker(time.Second * 30)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				app.QueueUpdateDraw(func() {
					updateScores()
				})
			}
		}
	}()

	if err := app.SetRoot(scoreview, true).Run(); err != nil {
		os.Exit(0)
	}
}
