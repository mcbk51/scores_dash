package config

import (
	"fmt"
	"sort"	
	"time"
	"github.com/mcbk51/scores_dash/api"
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
				return game.StartTime, game.HomeTeam, game.AwayTeam, dateStr, awayOdds, homeOdds
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
			return games[0].StartTime, games[0].HomeTeam, games[0].AwayTeam, dateStr, homeOdds, awayOdds
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



