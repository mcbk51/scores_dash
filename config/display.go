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

func (d *Display) MainOutput() {
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
	fmt.Fprintf(d.view, "[yellow]=== Scores Dash ===[-] [grey]Updated: %s| %s[-]\n", time.Now().Format("3:04 PM"), d.scroller.FormatStatus())

	//  Group by league
	activeByLeague := make(map[string][]api.Game)
	allByLeague := make(map[string][]api.Game)

	for _, game := range games {
		allByLeague[game.League] = append(allByLeague[game.League], game)
		if IsLive(game.Status) || IsUpcoming(game.StartTime, 30*time.Minute) {
			activeByLeague[game.League] = append(activeByLeague[game.League], game)
		}
	}

	// Sort leagues 
	baseOrder := []string{"NFL", "NBA", "NHL", "MLB"}
	leagueOrder := make([]string, 0, len(baseOrder))

	leaguesWithGames := make([]string, 0)
	leaguesWithoutGames := make([]string, 0)

	for _, league := range baseOrder {
		if len(allByLeague[league]) > 0 {
			leaguesWithGames = append(leaguesWithGames, league)
		} else {
			leaguesWithoutGames = append(leaguesWithoutGames, league)
		}
	}

	// Combine: leagues with games first, then leagues without games
	leagueOrder = append(leagueOrder, leaguesWithGames...)
	leagueOrder = append(leagueOrder, leaguesWithoutGames...)

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
			fmt.Fprintf(d.view, "[%s]▼ %s[-][gray] No games currently[-]\n", leagueColors[league], league)
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

			homeInfo := ""
			if game.HomeSpread != "" {
				homeInfo = fmt.Sprintf("[blue]%s[-] ", homeOdds)
			}
			homeInfo += fmt.Sprintf("%s (%s)", game.HomeTeam, game.HomeRecord)

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
			  	go d.MainOutput()
			}
		}
	}()
}

func checkSpreadIfHomeWin(game api.Game) string {
	if game.HomeSpread == "" {
		return ""
	}

	var spreadValue float64
	fmt.Sscanf(game.HomeSpread, "%f", &spreadValue)
	gameSpread := game.HomeScore - game.AwayScore

	if game.HomeScore > game.AwayScore {
		// Home team won
		if spreadValue < 0 {
			// Home was favored (negative spread), they need to win by more than the spread
			if gameSpread > int(-spreadValue) {
				return "[green]✓[-]"
			} else if gameSpread == int(-spreadValue) {
				return "[yellow]P[-]"
			} else {
				return "[red]✗[-]"
			}
		} else {
			// Home was underdog (positive spread), they just need to win
			return "[green]✓[-]"
		}
	}
	return ""
}

func checkSpreadIfAwayWin(game api.Game) string {
	if game.AwaySpread == "" {
		return ""
	}

	var spreadValue float64
	fmt.Sscanf(game.AwaySpread, "%f", &spreadValue)
	gameSpread := game.AwayScore - game.HomeScore

	if game.AwayScore > game.HomeScore {
		// Away team won
		if spreadValue < 0 {
			// Away was favored (negative spread), they need to win by more than the spread
			if gameSpread > int(-spreadValue) {
				return "[green]✓[-]"
			} else if gameSpread == int(-spreadValue) {
				return "[yellow]P[-]"
			} else {
				return "[red]✗[-]"
			}
		} else {
			// Away was underdog (positive spread), they just need to win
			return "[green]✓[-]"
		}
	}
	return ""
}


func checkOverUnderResult(game api.Game) string {
	if game.OverUnder == "" {
		return ""
	}

	// Parse the over/under value from the format "O/U 45.5"
	var ouValue float64
	_, err := fmt.Sscanf(game.OverUnder, "O/U %f", &ouValue)
	if err != nil {
		return ""
	}

	totalScore := float64(game.HomeScore + game.AwayScore)

	if totalScore > ouValue {
		return "[green]↑[-]"
	} else if totalScore < ouValue {
		return "[green]↓[-]"
	}

	// Push (exact match)
	return "[yellow]P[-]"
}


func PrintFinishedGames(scoreview *tview.TextView, game api.Game) {
	var awayStyle, homeStyle string
	if game.AwayScore > game.HomeScore {
		awayStyle = "green"
		homeStyle = "gray"
	} else if game.HomeScore > game.AwayScore {
		awayStyle = "gray"
		homeStyle = "green"
	} else {
		awayStyle = "white"
		homeStyle = "white"
	}

	awayOdds := FormatOdds(game.AwaySpread, game.AwayOdds)
	awaySpreadResult := checkSpreadIfAwayWin(game)
	homeOdds := FormatOdds(game.HomeSpread, game.HomeOdds)
	homeSpreadResult := checkSpreadIfHomeWin(game)

	if game.AwaySpread != "" {
		awayOdds = fmt.Sprintf("%s", awayOdds)
	}

	if game.HomeSpread != "" {
		homeOdds = fmt.Sprintf("%s", homeOdds)
	}
	
	oddsInfo := ""
	overUnderResult := checkOverUnderResult(game)
	if game.OverUnder != "" {
		oddsInfo = fmt.Sprintf(" [blue]%s %s[-]", game.OverUnder, overUnderResult)
	}

	fmt.Fprintf(scoreview, "  [%s]%s(%s) %s [%s]%s %d[-]  @ [%s]%d %s %s [%s]%s(%s) [-]%s\n", 
		awayStyle, game.AwayTeam, game.AwayRecord,  awaySpreadResult, awayStyle, awayOdds, game.AwayScore, homeStyle, game.HomeScore, homeOdds, homeSpreadResult, homeStyle, game.HomeTeam, game.HomeRecord, oddsInfo)
}
