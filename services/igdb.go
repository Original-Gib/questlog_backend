package services

import (
	"context"
	"strings"

	"github.com/Original_Gib/questlog/clients"
)

// IGDBClientInterface allows the service to be tested without real HTTP calls.
type IGDBClientInterface interface {
	SearchGames(ctx context.Context, query string) ([]clients.IGDBGame, error)
	GetGameByID(ctx context.Context, id int) (*clients.IGDBGame, error)
	GetPopularGames(ctx context.Context) ([]clients.IGDBGame, error)
}

// GameSummary is the frontend-facing representation of a game in list views.
type GameSummary struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	CoverURL         string   `json:"cover_url"`
	Platforms        []string `json:"platforms"`
	FirstReleaseDate *int64   `json:"first_release_date"` // null if not set
	Rating           *float64 `json:"rating"`             // null if not rated
}

// GameDetail is the frontend-facing representation of a single game's full details.
type GameDetail struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	CoverURL         string   `json:"cover_url"`
	Platforms        []string `json:"platforms"`
	Genres           []string `json:"genres"`
	FirstReleaseDate *int64   `json:"first_release_date"`
	Rating           *float64 `json:"rating"`
	Summary          string   `json:"summary"`
	Storyline        string   `json:"storyline"`
}

// IGDBService transforms raw IGDB data into clean API response types.
type IGDBService struct {
	client IGDBClientInterface
}

// NewIGDBService creates a new IGDBService backed by the given client.
func NewIGDBService(client IGDBClientInterface) *IGDBService {
	return &IGDBService{client: client}
}

func (s *IGDBService) SearchGames(ctx context.Context, query string) ([]GameSummary, error) {
	games, err := s.client.SearchGames(ctx, query)
	if err != nil {
		return nil, err
	}
	results := make([]GameSummary, len(games))
	for i, g := range games {
		results[i] = toGameSummary(g)
	}
	return results, nil
}

func (s *IGDBService) GetGameByID(ctx context.Context, id int) (*GameDetail, error) {
	game, err := s.client.GetGameByID(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := toGameDetail(*game)
	return &detail, nil
}

func (s *IGDBService) GetPopularGames(ctx context.Context) ([]GameSummary, error) {
	games, err := s.client.GetPopularGames(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]GameSummary, len(games))
	for i, g := range games {
		results[i] = toGameSummary(g)
	}
	return results, nil
}

// --- Transform helpers ---

func toGameSummary(g clients.IGDBGame) GameSummary {
	return GameSummary{
		ID:               g.ID,
		Name:             g.Name,
		CoverURL:         coverURL(g.Cover),
		Platforms:        platformNames(g.Platforms),
		FirstReleaseDate: nullableTimestamp(g.FirstReleaseDate),
		Rating:           nullableRating(g.Rating),
	}
}

func toGameDetail(g clients.IGDBGame) GameDetail {
	return GameDetail{
		ID:               g.ID,
		Name:             g.Name,
		CoverURL:         coverURL(g.Cover),
		Platforms:        platformNames(g.Platforms),
		Genres:           genreNames(g.Genres),
		FirstReleaseDate: nullableTimestamp(g.FirstReleaseDate),
		Rating:           nullableRating(g.Rating),
		Summary:          g.Summary,
		Storyline:        g.Storyline,
	}
}

// coverURL converts a protocol-relative IGDB cover URL to a full HTTPS URL.
// IGDB returns URLs like //images.igdb.com/igdb/image/upload/t_thumb/...
// We upgrade to t_cover_big for better quality.
func coverURL(cover *clients.IGDBCover) string {
	if cover == nil || cover.URL == "" {
		return ""
	}
	url := cover.URL
	if strings.HasPrefix(url, "//") {
		url = "https:" + url
	}
	return strings.Replace(url, "t_thumb", "t_cover_big", 1)
}

func platformNames(platforms []clients.IGDBPlatform) []string {
	names := make([]string, len(platforms))
	for i, p := range platforms {
		names[i] = p.Name
	}
	return names
}

func genreNames(genres []clients.IGDBGenre) []string {
	names := make([]string, len(genres))
	for i, g := range genres {
		names[i] = g.Name
	}
	return names
}

func nullableRating(r float64) *float64 {
	if r == 0 {
		return nil
	}
	return &r
}

func nullableTimestamp(ts int64) *int64 {
	if ts == 0 {
		return nil
	}
	return &ts
}
