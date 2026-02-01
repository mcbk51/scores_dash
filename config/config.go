package config

import (
	"fmt"
	"sort"	
	"time"
	"github.com/mcbk51/scores_dash/api"
	"github.com/rivo/tview"
)


func FindNextGame(league string) (time.Time, string, string, string, string, string) {
	games, err := api.GetGames(league, time.Now())
	if err == nil && len(games) > 0 {
		now := time.Now()
		for _, game := range games {
			if game.StartTime.After(now) {
				awayOdds := FormatOdds(game.AwaySpread, game.AwayOdds)
				homeOdds := FormatOdds(game.HomeSpread, game.HomeOdds)
				dateStr  := formatGameDate(game.StartTime)
				return game.StartTime, game.AwayTeam, game.HomeTeam,  dateStr, awayOdds, homeOdds
			}
		}
	}

	for i := 1 ; i < 7 ; i++ {
		futureDate := time.Now().AddDate(0, 0, i)
		games, err := api.GetGames(league, futureDate)
		if err != nil {
			continue
		}
		if len(games) > 0 {
			sort.Slice(games, func(i, j int) bool {
				return games[i].StartTime.Before(games[j].StartTime)
			})
			dateStr  := formatGameDate(games[0].StartTime)
			awayOdds := FormatOdds(games[0].AwaySpread, games[0].AwayOdds)
			homeOdds := FormatOdds(games[0].HomeSpread, games[0].HomeOdds)
			return games[0].StartTime,games[0].AwayTeam, games[0].HomeTeam,  dateStr, awayOdds,homeOdds
		}
	}
	return time.Time{}, "", "", "", "", ""
}

func FormatOdds(spread string, moneyline string) string {
	if spread != "" && moneyline != "" {
		return fmt.Sprintf("[%s | %s]", spread, moneyline)
	} else if spread != "" {
		return fmt.Sprintf("[%s]", spread)
	} else if moneyline != "" {
		return fmt.Sprintf("[%s]", moneyline)
	}
	return ""
}

func formatGameDate(t time.Time) string {
	now := time.Now()
	gameDate := t.Local()

	if gameDate.Year() == now.Year() && gameDate.Year() == now.YearDay() {
		return "Today"
	}

	tomorrow := now.AddDate(0, 0, 1)
	if gameDate.Year() == tomorrow.Year() && gameDate.Month() == tomorrow.Month() && gameDate.Day() == tomorrow.Day() {
		return "Tomorrow"
	}

	return gameDate.Format("Mon, Jan 2")
}

func AllGameFinishedforToday(games []api.Game) bool {
	if len(games) == 0 {
		return false
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)
	hasGamesToday := false

	for _, game := range games {
		if game.StartTime.After(todayStart) && game.StartTime.Before(todayEnd) {
			hasGamesToday = true

			if game.StartTime.After(now) || IsLive(game.Status) {
				return false
			}
		}
	}
	
	return hasGamesToday
}

func GetFinishedGamesToday(games []api.Game) []api.Game {
	var finishedGames []api.Game
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)

	for _, game := range games {
		if game.StartTime.After(todayStart) && game.StartTime.Before(todayEnd) {
			if IsFinished(game.Status) {
				finishedGames = append(finishedGames, game)
			}
		}
}

	sort.Slice(finishedGames, func(i, j int) bool {
		return finishedGames[i].StartTime.Before(finishedGames[j].StartTime)
	})
	
	return finishedGames
}

func IsFinished(status string) bool {
	return status == "Final" || status == "STATUS_FINAL" ||
		status == "Final/OT" || status == "STATUS_FINAL_OT" ||
		status == "Final/2OT" || status == "Final/3OT" ||
		status == "Postponed" || status == "STATUS_POSTPONED" ||
		status == "Canceled" || status == "STATUS_CANCELED"
}
	
func IsUpcoming(startTime time.Time, duration time.Duration) bool {
	now := time.Now()
	return startTime.After(now) && startTime.Before(now.Add(duration))
}

func IsLive(status string) bool {
	return status == "STATUS_IN_PROGRESS" || status == "In Progress" || status == "STATUS_HALFTIME" || status == "Halftime"
}

func CountLiveGames(games []api.Game) int {
	count := 0
	for _, game := range games {
		if IsLive(game.Status) {
			count++
		}
	}
	return count
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

	awayOdds := ""
	if game.AwaySpread != "" {
		awayOdds = fmt.Sprintf(" %s", game.AwaySpread)
	}

	homeOdds := ""
	if game.HomeSpread != "" {
		homeOdds = fmt.Sprintf(" %s", game.HomeSpread)
	}
	
	oddsInfo := ""
	if game.OverUnder != "" {
		oddsInfo = fmt.Sprintf(" [blue]%s[-]", game.OverUnder)
	}

	fmt.Fprintf(scoreview, "  [%s]%s (%s)%s %d[-]  @ [%s]%d %s (%s)%s [-]  [gray]FINAL[-]%s\n", 
		awayStyle, game.AwayTeam, game.AwayRecord, awayOdds, game.AwayScore, homeStyle, game.HomeScore,  game.HomeTeam, game.HomeRecord, homeOdds,  oddsInfo)
}

