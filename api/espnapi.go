package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Game struct {
	HomeTeam   string    `json:"home_team"`
	AwayTeam   string    `json:"away_team"`
	StartTime  time.Time `json:"start_time"`
	League     string    `json:"league"`
	Status     string    `json:"status"`
	HomeScore  int       `json:"home_score"`
	AwayScore  int       `json:"away_score"`
	HomeRecord string    `json:"home_record"`
	AwayRecord string    `json:"away_record"`
}

type ESPNResponse struct {
	Events []struct {
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
		Date      string `json:"date"`
		Status    struct {
			Type struct {
				Description string `json:"description"`
			} `json:"type"`
		} `json:"status"`
		Competitions []struct {
			Notes []struct {
				Headline string `json:"headline"`
			} `json:"notes"`
			Competitors []struct {
				Team struct {
					DisplayName  string `json:"displayName"`
					Abbreviation string `json:"abbreviation"`
					ID           string `json:"id"`
				} `json:"team"`
				HomeAway string `json:"homeAway"`
				Score    string `json:"score"`
				Records  []struct {
					Name    string `json:"name"`
					Summary string `json:"summary"`
					Type    string `json:"type"`
				} `json:"records"`
			} `json:"competitors"`
		} `json:"competitions"`
	} `json:"events"`
}

// Team's win-loss record
type TeamRecord struct {
	TeamName string
	Record   string
}

// Fetches games for the specified league and date
func GetGames(league string, date time.Time) ([]Game, error) {
	var games []Game
	leagues := []string{"nfl", "nba", "nhl", "mlb"}

	if league != "all" {
		leagues = []string{strings.ToLower(league)}
	}

	for _, l := range leagues {
		leagueGames, err := fetchGamesForLeague(l, date)
		if err != nil {
			fmt.Printf("Warning: Could not fetch games for %s: %v\n", l, err)
			continue
		}
		games = append(games, leagueGames...)
	}

	return games, nil
}

// Fetches games for a specific league
func fetchGamesForLeague(league string, date time.Time) ([]Game, error) {
	dateStr := date.Format("20060102")

	// ESPN API endpoint for different leagues
	var url string
	switch league {
	case "nfl":
		url = fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard?dates=%s", dateStr)
	case "nba":
		url = fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard?dates=%s", dateStr)
	case "nhl":
		url = fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/hockey/nhl/scoreboard?dates=%s", dateStr)
	case "mlb":
		url = fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/baseball/mlb/scoreboard?dates=%s", dateStr)
	default:
		return nil, fmt.Errorf("unsupported league: %s", league)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var espnResp ESPNResponse
	if err := json.Unmarshal(body, &espnResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var games []Game
	for _, event := range espnResp.Events {
		// Fixing the game start time
		startTime, err := time.Parse(time.RFC3339, event.Date)
		if err != nil {
			startTime, err = time.Parse("2006-01-02T15:04Z", event.Date)
			if err != nil {
				fmt.Printf("Warning: Could not parse date '%s' for game %s: %v\n", event.Date, event.Name, err)
				startTime = time.Now()
			}
		}

		// Process team sports
		if len(event.Competitions) == 0 || len(event.Competitions[0].Competitors) < 2 {
			continue
		}

		comp := event.Competitions[0]
		var homeTeam, awayTeam string
		var homeScore, awayScore int
		var homeRecord, awayRecord string

		for _, competitor := range comp.Competitors {
			// Extract record from the competitor data
			record := extractRecord(competitor.Records, league)

			if competitor.HomeAway == "home" {
				homeTeam = competitor.Team.DisplayName
				homeRecord = record
				if competitor.Score != "" {
					fmt.Sscanf(competitor.Score, "%d", &homeScore)
				}
			} else {
				awayTeam = competitor.Team.DisplayName
				awayRecord = record
				if competitor.Score != "" {
					fmt.Sscanf(competitor.Score, "%d", &awayScore)
				}
			}
		}

		game := Game{
			HomeTeam:   homeTeam,
			AwayTeam:   awayTeam,
			StartTime:  startTime,
			League:     strings.ToUpper(league),
			Status:     event.Status.Type.Description,
			HomeScore:  homeScore,
			AwayScore:  awayScore,
			HomeRecord: homeRecord,
			AwayRecord: awayRecord,
		}

		games = append(games, game)
	}

	return games, nil
}

// Extracts the appropriate record from the records array
func extractRecord(records []struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Type    string `json:"type"`
}, league string) string {

	if len(records) == 0 {
		return ""
	}

	// For different leagues, look for different record types
	switch league {
	case "nfl":
		for _, record := range records {
			if record.Name == "overall" || record.Type == "total" {
				return record.Summary
			}
		}
	case "nba", "nhl":
		for _, record := range records {
			if record.Name == "overall" || record.Type == "total" {
				return record.Summary
			}
		}
	case "mlb":
		for _, record := range records {
			if record.Name == "overall" || record.Type == "total" {
				return record.Summary
			}
		}
	}

	// If no specific record found, return the first one
	if len(records) > 0 {
		return records[0].Summary
	}

	return ""
}
