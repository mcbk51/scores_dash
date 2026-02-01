package config

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mcbk51/scores_dash/api"
	"github.com/rivo/tview"
)

type Display struct {
	app  	 *tview.Application
	view 	 *tview.TextView
    scroller *Scroller
	ctx  	 context.Context
	quitChan chan bool
}

func NewDisplay(app *tview.Application, view *tview.TextView, scroller *Scroller, ctx context.Context, quitChan chan bool) *Display {
	return &Display{
		app: app,
		view: view,
		scroller: scroller,
		ctx: ctx,
		quitChan: quitChan,
	}
}

func (d *Display) UpdateScores() {
	select {
	case <-d.quitChan:
		return
	case <-d.ctx.Done():
		return
	default:
	}

	games, err := api.GetGames("all", time.Now())
	select {
	case <-d.quitChan:
		return
	case <-d.ctx.Done():
		return
	default:
	}


	if err != nil {
		d.view.Clear()
		fmt.Fprintf(d.view, "[red]Error fetching scores: %v[-]\n", err)
		d.app.Draw()
		return
	}

	d.view.Clear()
	fmt.Fprintf(d.view, "[yellow]=== LIVE SPORTS SCORES ===[-]\n\n")
	fmt.Fprintf(d.view, "[grey]Updated: %s| %s[-]\n", time.Now().Format("3:04 PM"), d.scroller.FormatStatus())
	fmt.Fprintf(d.view, "[grey]'q' quit | 's' scroll | '+/-' speed | 'r' reverse | 'j/k' manual[-]\n\n")


	//  Group by league
	activeByLeague := make(map[string][]api.Game)
	allByLeague := make(map[string][]api.Game)

	for _, game := range games {
		allByLeague[game.League] = append(allByLeague[game.League], game)
		if IsLive(game.Status) || IsUpcoming(game.StartTime, 30*time.Minute) {
			activeByLeague[game.League] = append(activeByLeague[game.League], game)
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
		activeGames := activeByLeague[league]
		AllLeagueGames := allByLeague[league]
		finishedGames := GetFinishedGamesToday(AllLeagueGames)

		// No Active Games
		if len(activeGames) == 0 {
			nextGameTime, awayTeam, homeTeam, dateStr, awayOdds, homeOdds := FindNextGame(league)
			fmt.Fprintf(d.view, "[%s]▼ %s[-]\n", leagueColors[league], league)
			fmt.Fprintf(d.view, "  [gray]No games currently[-]\n")
			if !nextGameTime.IsZero() {
				localTime := nextGameTime.Local()
				// Output for next game
				fmt.Fprintf(d.view, "  [gray]Next game: %s%s @ %s %s - %s at %s[-]\n", awayTeam, awayOdds,  homeOdds, homeTeam, dateStr, localTime.Format("3:04 PM"))
			} 			

			if len(finishedGames) > 0 {
				fmt.Fprintf(d.view, "[orange]── Finished Games Results ──[-]\n")
				for _, game := range finishedGames {
					PrintFinishedGames(d.view, game)
				}
			}
			fmt.Fprintf(d.view, "\n")
			continue
		}

		sort.Slice(activeGames, func(i, j int) bool {
			statusI := IsLive(activeGames[i].Status)
			statusJ := IsLive(activeGames[j].Status)
			if statusI != statusJ {
				return statusI
			}
			return activeGames[i].StartTime.Before(activeGames[j].StartTime)
		})

		liveCount := CountLiveGames(activeGames)
		if liveCount > 0 {
			fmt.Fprintf(d.view, "[%s]▼ %s[-] [green]● %d LIVE[-]\n", leagueColors[league], league, liveCount)
		} else {
			fmt.Fprintf(d.view, "[%s]▼ %s[-]\n", leagueColors[league], league)
		}

		for _, game := range activeGames {
			statusColor := "white"
			statusText := game.Status
			awayOdds := FormatOdds(game.AwaySpread, game.AwayOdds)
			homeOdds := FormatOdds(game.HomeSpread, game.HomeOdds)

			if IsLive(game.Status) {
				statusColor = "green"
				if game.Clock != "" && game.Period != "" {
					statusText = fmt.Sprintf("%s - %s", game.Clock, game.Period)
				} else {
					statusText = "LIVE"
				}
			} else if IsUpcoming(game.StartTime, 30*time.Minute) {
				statusColor = "yellow"
				localTime := game.StartTime.Local()
				minutesUntil := int(time.Until(game.StartTime).Minutes())
				statusText = fmt.Sprintf("Starts in %dm (%s)", minutesUntil, localTime.Format("3:04 PM"))
			} else {
				continue
			}

			awayInfo := fmt.Sprintf("%s (%s)", game.AwayTeam, game.AwayRecord)
			if game.AwaySpread != "" {
				awayInfo += fmt.Sprintf("[blue]%s[-]", awayOdds)
			}

			homeInfo := fmt.Sprintf("%s (%s)", game.HomeTeam, game.HomeRecord)
			if game.HomeSpread != "" {
				homeInfo += fmt.Sprintf("[blue]%s[-]", homeOdds)
			}

			// Mian output for live games
			fmt.Fprintf(d.view, " [-][blue]%s [white]%s [-][purple]%d  [white]@  [purple]%d [-]%s  [%s]{%s}[-]\n",
				game.OverUnder,
				awayInfo,
				game.AwayScore,
				game.HomeScore,
				homeInfo,
				statusColor,
				statusText)

		}

		if len(finishedGames) > 0 {
			fmt.Fprintf(d.view, "[orange]── Finished Games Results ──[-]\n")
			for _, game := range finishedGames {
				PrintFinishedGames(d.view, game)
			}
		}

		fmt.Fprintf(d.view, "\n")
	}
}

func (d *Display) StartTicker(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-d.quitChan:
				return
			case <-ticker.C:
			  	go d.UpdateScores()
			}
		}
	}()
}
