package config

import (
	"context"
	"fmt"
	"sort"
	"time"
	"strconv"

	"github.com/mcbk51/scores_dash/api"
	"github.com/rivo/tview"
)

var leagueColors = map[string]string{
	"NFL": "red",
	"NBA": "blue",
	"NHL": "orange",
	"MLB": "green",
}

var leagueOrder = []string{"NFL", "NBA", "NHL", "MLB"}

type Display struct {
	app      *tview.Application
	view     *tview.TextView
	scroller *Scroller
	ctx      context.Context
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

func (d *Display) cancelled() bool {
	select {
	case <-d.quitChan:
		return true
	case <-d.ctx.Done():
		return true
	default:
		return false
	}
}

func (d *Display) MainOutput() {
	if d.cancelled() {
		return
	}

	games, err := api.GetGames("all", time.Now())
	if d.cancelled() {
		return
	}
	if err != nil {
		d.view.Clear()
		fmt.Fprintf(d.view, "[red]Error fetching scores: %v[-]\n", err)
		d.app.Draw()
		return
	}

	activeByLeague, allByLeague := groupGamesByLeague(games)
	sortedLeagues := sortLeaguesByActivity(allByLeague)

	d.view.Clear()
	fmt.Fprintf(d.view, "[yellow]=== Scores Dash ===[-] [grey]Updated: %s| %s[-]\n", time.Now().Format("3:04 PM"), d.scroller.FormatStatus())

	for _, league := range sortedLeagues {
		activeGames := activeByLeague[league]
		allGames := allByLeague[league]

		finishedGames := getFinishedGamesToday(allGames)
		color := leagueColors[league]

		// No Active Games
		if len(activeGames) == 0 {
			d.renderNoLiveGames(league, color, finishedGames)
			continue
		}
		sortGamesByStatus(activeGames)
		d.renderLiveGames(league, color, activeGames)
		d.renderFinishedGames(finishedGames)
		fmt.Fprintf(d.view, "\n")
	}
}

func (d *Display) renderNoLiveGames(league, color string, finishedGames []api.Game){
	fmt.Fprintf(d.view, "[%s]▼ %s[-][gray] No games currently[-]\n", color, league)

	nextGameTime, awayTeam, homeTeam, dateStr, awayOdds, homeOdds := findNextGame(league)
	if !nextGameTime.IsZero() {
		localTime := nextGameTime.Local()
		// Output for next game
		fmt.Fprintf(d.view, "  [gray]Next game: %s%s @ %s%s - %s at %s[-]\n", awayTeam, awayOdds,  homeOdds, homeTeam, dateStr, localTime.Format("3:04 PM"))
	}
	d.renderFinishedGames(finishedGames)
}

func (d *Display) renderLiveGames(league,color string, games []api.Game) {
	liveCount := countLiveGames(games)

	if liveCount > 0 {
		fmt.Fprintf(d.view, "[%s]▼ %s[-] [green]● %d LIVE[-]\n", color, league, liveCount)
	}

	for _, game := range games {
		statusColor, statusText := formatGameStatus(game)
		if statusColor == "" {
			continue
		}
		awayOdds := formatOdds(game.AwaySpread, game.AwayOdds)
		homeOdds := formatOdds(game.HomeSpread, game.HomeOdds)

		awayInfo := fmt.Sprintf("%s (%s)", game.AwayTeam, game.AwayRecord)
		if game.AwaySpread != "" {
			awayInfo += fmt.Sprintf("[blue]%s[-]", awayOdds)
		}

		homeInfo := ""
		if game.HomeSpread != "" {
			homeInfo = fmt.Sprintf("[blue]%s[-] ", homeOdds)
		}
		homeInfo += fmt.Sprintf("%s (%s)", game.HomeTeam, game.HomeRecord)

		fmt.Fprintf(d.view, " [-][blue]%s [white]%s [-][purple]%d  [white]@  [purple]%d [-]%s  [%s]{%s}[-]\n",
			game.OverUnder,
			awayInfo,
			game.AwayScore,
			game.HomeScore,
			homeInfo,
			statusColor,
			statusText)
	}
}

func (d *Display) renderFinishedGames(games []api.Game) {
	if len(games) == 0 {
		return
	}
	fmt.Fprintf(d.view, "[orange]── Finished Games Results ──[-]\n")
	for _, game := range games {
		printFinishedGames(d.view, game)
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

// helper functions
func groupGamesByLeague(games []api.Game) (map[string][]api.Game, map[string][]api.Game) {
	active := make(map[string][]api.Game)
	all := make(map[string][]api.Game)
	for _, game := range games {
		all[game.League] = append(all[game.League], game)
		if isLive(game.Status) || isUpcoming(game.StartTime, 30*time.Minute) {
			active[game.League] = append(active[game.League], game)
		}
	}
	return active, all
}

func sortLeaguesByActivity(allByLeague map[string][]api.Game) []string {
	withGames := make([]string, 0, len(leagueOrder))
	withoutGames := make([]string, 0, len(leagueOrder))
	for _, league := range leagueOrder {
		if len(allByLeague[league]) > 0 {
			withGames = append(withGames, league)
		} else {
			withoutGames = append(withoutGames, league)
		}
	}
	return append(withGames, withoutGames...)
}

func sortGamesByStatus(games []api.Game) []api.Game {
	sort.Slice(games, func(i, j int) bool {
		liveI := isLive(games[i].Status)
		liveJ := isLive(games[j].Status)
		if liveI != liveJ {
			return liveI
		}
		return games[i].StartTime.Before(games[j].StartTime)
	})
	return games
}

func formatGameStatus(game api.Game) (color, text string) {
	switch {
	case isLive(game.Status):
		text = "LIVE"
		if game.Clock != "" && game.Period != "" {
			text = fmt.Sprintf("%s - %s", game.Clock, game.Period)
		}
		return "green", text

	case isUpcoming(game.StartTime, 45*time.Minute):
		localTime := game.StartTime.Local()
		minutesUntil := int(time.Until(game.StartTime).Minutes())
		text = fmt.Sprintf("Starts in %dm (%s)", minutesUntil, localTime.Format("3:04 PM"))
		return "yellow", text
	default:
		return "", ""
	}
}


func spreadResult(spread string, scoreDiff int, teamWon bool) string {
	if spread == "" || !teamWon {
		return ""
	}

	spreadValue, err := strconv.ParseFloat(spread, 64)
	if err != nil {
		return ""
	}

	if spreadValue > 0 {
		return "[green]✓[-]"
	}

	scoreDiffFloat := float64(scoreDiff)
	needed := -spreadValue
	switch {
	case scoreDiffFloat > needed:
		return "[green]✓[-]"
	case scoreDiffFloat == needed:
		return "[yellow]P[-]"
	default:
		return "[red]✗[-]"
	}
}


func checkSpreadIfHomeWin(game api.Game) string {
	return spreadResult(game.HomeSpread, game.HomeScore - game.AwayScore, game.HomeScore > game.AwayScore)
}


func checkSpreadIfAwayWin(game api.Game) string {
	return spreadResult(game.AwaySpread, game.AwayScore - game.HomeScore, game.AwayScore > game.HomeScore)
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


func printFinishedGames(scoreview *tview.TextView, game api.Game) {
	awayStyle, homeStyle := "white", "white"
	switch {
	case game.AwayScore > game.HomeScore:
		awayStyle, homeStyle = "green", "gray"
	case game.HomeScore > game.AwayScore:
		awayStyle, homeStyle = "gray", "green"
	}

	awayOdds := formatOdds(game.AwaySpread, game.AwayOdds)
	awaySpreadResult := checkSpreadIfAwayWin(game)
	homeOdds := formatOdds(game.HomeSpread, game.HomeOdds)
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
		awayStyle, game.AwayTeam, game.AwayRecord,  awaySpreadResult, awayStyle, awayOdds, game.AwayScore, 
		homeStyle, game.HomeScore, homeOdds, homeSpreadResult, homeStyle, game.HomeTeam, game.HomeRecord, oddsInfo)
}
