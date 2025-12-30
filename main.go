package main    

import (
	"fmt"
	"time"
	"sort"

	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/mcbk51/scores_dash/api"
	"github.com/mcbk51/scores_dash/config"
)

func main (){
	app := tview.NewApplication()
	
	scoreview := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	updateScores := func() {
		games, err := api.GetGames("all", time.Now())
		if err != nil {
			fmt.Printf("Error fetching scores: %v\n", err)
			app.Draw()
			return
		}

		scoreview.Clear()
		fmt.Fprintf(scoreview, "[yellow]===LIVE SPORTS SCORES===[-]\n\n")

		//  Group by league
		gamesByLeague := make(map[string][]api.Game)
		for _, game := range games {
			gamesByLeague[game.League] = append(gamesByLeague[game.League], game)
		}

		leagueOrder := []string{"NFL", "NBA", "NHL", "MLB"}
		leagueColors := map[string]string{
			"NFL": "red",
			"NBA": "blue",
			"NHL": "cyan",
			"MLB": "green",
		}

		for _, league := range leagueOrder {
			games, exists := gamesByLeague[league]
			if !exists || len(games) == 0 {
				nextGame := config.FindNextGame(league)
				fmt.Fprintf(scoreview, "[%s]▼ %s[-]\n", leagueColors[league], league)
				fmt.Fprintf(scoreview, "  [gray]No games currently[-]\n")
				if !nextGame.IsZero() {
					fmt.Fprintf(scoreview, "  [gray]Next game: %s[-]\n\n", nextGame.Format("Mon, Jan 2 at 3:04 PM"))
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
			if liveCount == 0 {
				fmt.Fprintf(scoreview, "[%s]▼ %s[-] [green]● %d LIVE[-]\n", leagueColors[league], league, liveCount)
			} else {
				fmt.Fprintf(scoreview, "[%s]▼ %s[-]\n", leagueColors[league], league)
			}

			for _, game := range games {
				statusColor := "white"
				statusText := game.Status

				if config.isLive(game.Status) {
					statusColor = "green"
					statusText = "LIVE"
				} else if game.Status == "Final" {
					statusColor = "gray"
					statusText = "FINAL"
				} else {
					statusColor = "yellow"
					statusText = game.StartTime.Format("3:04 PM")
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

		fmt.Fprintf(scoreview, "\n [gray]Last updated: %s | Press 'q' or Ctrl+C to quit[-]\n", time.Now().Format("3:04 PM"))
		app.Draw()
	}

	// Input capture
	scoreview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC || event.Key() == 'q' {
			app.Stop()
			return nil
		}
		return event
	})


	// Initial Load
	go updateScores()

	//Set up a ticker to update scores every 30 seconds
	go func() {
		ticker := time.NewTicker(time.Second * 30)
		defer ticker.Stop()
		for range ticker.C {
			app.QueueUpdateDraw(func() {
				updateScores()
			})
		}
	}()

	if err := app.SetRoot(scoreview, true).Run(); err != nil {
		panic(err)
	}
}
