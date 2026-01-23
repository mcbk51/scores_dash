package api

import (
	"encoding/json"
	"strconv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	OddsProviderCaesar     = 38
	OddsProviderBet365     = 2000
	OddsProviderDraftKings = 41
)

var sportMap = map[string]string{
	"nfl": "football",
	"nba": "basketball",
	"nhl": "hockey",
	"mlb": "baseball",
}

type Game struct {
	EventID       string    `json:"event_id"`
	CompetitionID string    `json:"competition_id"`
	HomeTeam      string    `json:"home_team"`
	AwayTeam      string    `json:"away_team"`
	StartTime     time.Time `json:"start_time"`
	League        string    `json:"league"`
	Status        string    `json:"status"`
	HomeScore     int       `json:"home_score"`
	AwayScore     int       `json:"away_score"`
	HomeRecord    string    `json:"home_record"`
	AwayRecord    string    `json:"away_record"`
	Clock         string    `json:"clock"`
	Period        string    `json:"period"`
	HomeOdds      string    `json:"home_odds"`
	AwayOdds      string    `json:"away_odds"`
	AwaySpread    string    `json:"away_spread"`
	HomeSpread    string    `json:"home_spread"`
	OverUnder     string    `json:"over_under"`
}

type ESPNResponse struct {
	Events []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
		Date      string `json:"date"`
		Status    struct {
			Type struct {
				Description string `json:"description"`
			} `json:"type"`
			DisplayClock string `json:"displayClock"`
			Period       int    `json:"period"`
		} `json:"status"`
		Competitions []struct {
			ID    string `json:"id"`
			Notes []struct {
				Headline string `json:"headline"`
			} `json:"notes"`
			Odds []struct {
				Provider struct {
					Name string `json:"name"`
				} `json:"provider"`
				Details      string  `json:"details"`
				OverUnder    float64 `json:"overUnder"`
				Spread       float64 `json:"spread"`
				OverOdds     int     `json:"overOdds"`
				UnderOdds    int     `json:"underOdds"`
				AwayTeamOdds struct {
					Favorite   bool `json:"favorite"`
					Underdog   bool `json:"underdog"`
					Moneyline  int  `json:"moneyline"`
					SpreadOdds int  `json:"spreadOdds"`
				} `json:"awayTeamOdds"`
				HomeTeamOdds struct {
					Favorite   bool `json:"favorite"`
					Underdog   bool `json:"underdog"`
					Moneyline  int  `json:"moneyline"`
					SpreadOdds int  `json:"spreadOdds"`
				} `json:"homeTeamOdds"`
			} `json:"odds"`
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

	fecthAllOdds(games)

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
		clock := event.Status.DisplayClock
		period := formatPeriod(event.Status.Period, league)
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

		eventID := event.ID
		compID := comp.ID
		if compID == "" {
			compID = eventID
		}

		game := Game{
			EventID:       eventID,
			CompetitionID: compID,
			HomeTeam:      homeTeam,
			AwayTeam:      awayTeam,
			StartTime:     startTime,
			League:        strings.ToUpper(league),
			Status:        event.Status.Type.Description,
			HomeScore:     homeScore,
			AwayScore:     awayScore,
			HomeRecord:    homeRecord,
			AwayRecord:    awayRecord,
			Clock:         clock,
			Period:        period,
		}

		games = append(games, game)
	}

	return games, nil
}

func formatPeriod(period int, league string) string {
	switch league {
	case "nfl":
		switch period {
		case 1:
			return "1st Qtr"
		case 2:
			return "2nd Qtr"
		case 3:
			return "3rd Qtr"
		case 4:
			return "4th Qtr"
		case 5:
			return "OT"
		default:
			return ""
		}

	case "nba":
		switch period {
		case 1:
			return "1st Qtr"
		case 2:
			return "2nd Qtr"
		case 3:
			return "3rd Qtr"
		case 4:
			return "4th Qtr"
		case 5:
			return "OT"
		default:
			return ""
		}

	case "nhl":
		switch period {
		case 1:
			return "1st Per"
		case 2:
			return "2nd Per"
		case 3:
			return "3rd Per"
		case 4:
			return "OT"
		default:
			return ""
		}

	case "mlb":
		switch period {
		case 1:
			return "1st Inn"
		case 2:
			return "2nd Inn"
		case 3:
			return "3rd Inn"
		case 4:
			return "4th Inn"
		case 5:
			return "5th Inn"
		case 6:
			return "6th Inn"
		case 7:
			return "7th Inn"
		case 8:
			return "8th Inn"
		case 9:
			return "9th Inn"
		case 10:
			return "Extra Inn"
		default:
			return fmt.Sprintf("%dth", period)
		}
	default:
		return ""
	}
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

func fecthAllOdds(games []Game) {
	var wg sync.WaitGroup
	for i := range games {
		if games[i].OverUnder == "" && games[i].HomeSpread == ""{
			wg.Add(1)
			go func(game *Game) {
				defer wg.Done()
				fetchOddsForGame(game, OddsProviderDraftKings)
			}(&games[i])
		}
	}
	wg.Wait()
}

type OddsResponse struct {
	Items []OddsItem `json:"items"`
} 

type OddsItem struct {
	Provider Provider `json:"provider"`
	Spread   float64  `json:"spread"`
	OverUnder float64 `json:"overUnder"`
	HomeTeamOdds TeamOdds `json:"homeTeamOdds"`
	AwayTeamOdds TeamOdds `json:"awayTeamOdds"`
}

type Provider struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

type TeamOdds struct {
	Favorite   bool `json:"favorite"`
	MoneyLine  int `json:"moneyLine"`
}


func fetchOddsForGame(game *Game, providerID int) {
	sport, ok := sportMap[strings.ToLower(game.League)]
	if !ok {
		return
	}
	league := strings.ToLower(game.League)
	url := fmt.Sprintf("https://sports.core.api.espn.com/v2/sports/%s/leagues/%s/events/%s/competitions/%s/odds?lang=en&region=us", sport, league, game.EventID, game.CompetitionID)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var odds OddsResponse
	if err := json.Unmarshal(body, &odds); err != nil {
		return
	}

	// Multiple odds providers
	for _, item := range odds.Items {
		if item.Provider.ID == strconv.Itoa(providerID) {
			continue
		}

		applyOddsToGame(game, item)
		return
	}
}

func applyOddsToGame(game *Game, odds OddsItem) {
	if odds.Spread != 0 {
		if odds.HomeTeamOdds.Favorite {
			game.HomeSpread = fmt.Sprintf("%.1f", odds.Spread)
			game.AwaySpread = fmt.Sprintf("%.1f", -odds.Spread)
		}else if odds.AwayTeamOdds.Favorite {
			game.HomeSpread = fmt.Sprintf("%.1f", odds.Spread)
			game.AwaySpread = fmt.Sprintf("%.1f", -odds.Spread)
		} else {
			if odds.Spread > 0 {
				game.HomeSpread = fmt.Sprintf("%.1f", odds.Spread)
				game.AwaySpread = fmt.Sprintf("%.1f", odds.Spread)
			} else {
				game.AwaySpread = fmt.Sprintf("%.1f", odds.Spread)
				game.HomeSpread = fmt.Sprintf("%.1f", odds.Spread)
			}
		}
	}

	if odds.OverUnder != 0 { 
		game.OverUnder = fmt.Sprintf("O/U %.1f", odds.OverUnder)
	}

	// Moneyline odds
	if odds.HomeTeamOdds.MoneyLine != 0 {
		if odds.HomeTeamOdds.MoneyLine > 0 {
			game.HomeOdds = fmt.Sprintf("+%d", odds.HomeTeamOdds.MoneyLine)
		} else {
			game.HomeOdds = fmt.Sprintf("%d", odds.HomeTeamOdds.MoneyLine)
		}
	}
	if odds.AwayTeamOdds.MoneyLine != 0 {
		if odds.AwayTeamOdds.MoneyLine > 0 {
			game.AwayOdds = fmt.Sprintf("+%d", odds.AwayTeamOdds.MoneyLine)
		} else {
			game.AwayOdds = fmt.Sprintf("%d", odds.AwayTeamOdds.MoneyLine)
		}
	}
}
