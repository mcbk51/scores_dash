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
			if game.StartTime.Before(now) && IsLive(game.Status) {
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
	return status == "STATUS_FINAL" || status == "Final" || status == "STATUS_FINAL_OT" || status == "Final/OT" || status == "STATUS_POSTPONED" || status == "Postponed"
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

	oddsInfo := ""
	if game.OverUnder != "" {
		oddsInfo = fmt.Sprintf(" [blue]%s[-]", game.OverUnder)
	}

	spreadInfo := ""
	if game.AwaySpread != "" {
		spreadInfo = fmt.Sprintf(" [blue]%s[-]", game.AwaySpread)
	}

	fmt.Fprintf(scoreview, "  [%s]%s%s %d[-]  @  [%s]%s %d[-]  [gray]FINAL[-]%s\n", 
		awayStyle, game.AwayTeam, spreadInfo, game.AwayScore, homeStyle, game.HomeTeam, game.HomeScore, oddsInfo)
}

