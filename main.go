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

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		quit()
	}()

	updateScores := func() {
		select {
		case <-quitChan:
			return
		case <-ctx.Done():
			return
		default:
		}

		games, err := api.GetGames("all", time.Now())
		select {
		case <-quitChan:
			return
		case <-ctx.Done():
			return
		default:
		}


		if err != nil {
			scoreview.Clear()
			fmt.Fprintf(scoreview, "[red]Error fetching scores: %v[-]\n", err)
			app.Draw()
			return
		}

		scoreview.Clear()
		fmt.Fprintf(scoreview, "[yellow]=== LIVE SPORTS SCORES ===[-]\n\n")
		fmt.Fprintf(scoreview, "[grey]Updated: %s | Press 'q' to quit[-]\n\n", time.Now().Format("3:04 PM"))


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
				nextGameTime, awayTeam, homeTeam, dateStr := config.FindNextGame(league)
				fmt.Fprintf(scoreview, "[%s]▼ %s[-]\n", leagueColors[league], league)
				fmt.Fprintf(scoreview, "  [gray]No games currently[-]\n")
				if !nextGameTime.IsZero() {
					localTime := nextGameTime.Local()
					fmt.Fprintf(scoreview, "  [gray]Next game: %s @ %s - %s at %s[-]\n\n", awayTeam, homeTeam, dateStr, localTime.Format("3:04 PM"))
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
					if game.Clock != "" && game.Period != "" {
						statusText = fmt.Sprintf("%s - %s", game.Clock, game.Period)
					} else {
						statusText = "LIVE"
					}
				} else if config.IsUpcoming(game.StartTime, 30*time.Minute) {
					statusColor = "yellow"
					localTime := game.StartTime.Local()
					minutesUntil := int(time.Until(game.StartTime).Minutes())
					statusText = fmt.Sprintf("Starts in %dm (%s)", minutesUntil, localTime.Format("3:04 PM"))
				} else {
					continue
				}

				awayInfo := fmt.Sprintf("%s (%s)", game.AwayTeam, game.AwayRecord)
				if game.AwaySpread != "" {
					awayInfo += fmt.Sprintf("[cyan][%s][-]", game.AwaySpread)
				}

				homeInfo := fmt.Sprintf("%s (%s)", game.HomeTeam, game.HomeRecord)
				if game.HomeSpread != "" {
					homeInfo += fmt.Sprintf("[cyan][%s][-]", game.HomeSpread)
				}

				fmt.Fprintf(scoreview, " [cyan][-]%s %s [purple][-]%d  @  %s [purple][-]%d  [%s]{%s}[-]\n",
					game.OverUnder,
					awayInfo,
					game.AwayScore,
					homeInfo,
					game.HomeScore,
					statusColor,
					statusText)

			}
			fmt.Fprintf(scoreview, "\n")
			
		}

		app.Draw()
	}

	scoreview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC || event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			quit()
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
		for {
			select {
			case <-ctx.Done():
				return
			case <-quitChan:
				return
			case <-ticker.C:
			  	go updateScores()
			}
		}
	}()

	if err := app.SetRoot(scoreview, true).Run(); err != nil {
		os.Exit(0)
	}
}
