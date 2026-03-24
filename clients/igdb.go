package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	igdbBaseURL  = "https://api.igdb.com/v4"
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
)

// tokenCache holds the current Twitch OAuth token and its expiry.
type tokenCache struct {
	accessToken string
	expiresAt   time.Time
	mu          sync.Mutex
}

// IGDBClient is a raw HTTP client for the IGDB API.
// It handles Twitch OAuth token fetching and caching transparently.
type IGDBClient struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	token        tokenCache
}

// NewIGDBClient creates a new IGDBClient. The token is fetched lazily on first use.
func NewIGDBClient(clientID, clientSecret string) *IGDBClient {
	return &IGDBClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

// twitchTokenResponse is the shape of the Twitch OAuth token endpoint response.
type twitchTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// getToken returns a valid access token, refreshing it if expired.
// The mutex is held for the duration of any refresh to prevent concurrent fetches.
func (c *IGDBClient) getToken(ctx context.Context) (string, error) {
	c.token.mu.Lock()
	defer c.token.mu.Unlock()

	if c.token.accessToken != "" && time.Now().Before(c.token.expiresAt) {
		return c.token.accessToken, nil
	}

	url := fmt.Sprintf("%s?client_id=%s&client_secret=%s&grant_type=client_credentials",
		twitchTokenURL, c.clientID, c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("building token request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching twitch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("twitch token endpoint returned %d", resp.StatusCode)
	}

	var tok twitchTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	c.token.accessToken = tok.AccessToken
	c.token.expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	return c.token.accessToken, nil
}

// IGDBError represents an error response from the IGDB API.
type IGDBError struct {
	StatusCode int
	Message    string
}

func (e *IGDBError) Error() string {
	return fmt.Sprintf("IGDB API error %d: %s", e.StatusCode, e.Message)
}

// Query sends an Apicalypse query to the given IGDB endpoint and returns the raw response bytes.
// endpoint is the path suffix (e.g. "games"). body is the raw Apicalypse query string.
func (c *IGDBClient) Query(ctx context.Context, endpoint, body string) ([]byte, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s", igdbBaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building IGDB request: %w", err)
	}

	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing IGDB request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading IGDB response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &IGDBError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	return respBody, nil
}

// --- Raw IGDB response types ---

type IGDBCover struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type IGDBPlatform struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type IGDBGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type IGDBGame struct {
	ID               int            `json:"id"`
	Name             string         `json:"name"`
	Cover            *IGDBCover     `json:"cover"`
	Platforms        []IGDBPlatform `json:"platforms"`
	Genres           []IGDBGenre    `json:"genres"`
	FirstReleaseDate int64          `json:"first_release_date"`
	Rating           float64        `json:"rating"`
	RatingCount      int            `json:"rating_count"`
	Summary          string         `json:"summary"`
	Storyline        string         `json:"storyline"`
}

// SearchGames searches IGDB for games matching the given query string.
func (c *IGDBClient) SearchGames(ctx context.Context, query string) ([]IGDBGame, error) {
	body := fmt.Sprintf(`search "%s"; fields name,cover.url,platforms.name,first_release_date,rating,summary; limit 20;`, query)
	raw, err := c.Query(ctx, "games", body)
	if err != nil {
		return nil, err
	}
	var games []IGDBGame
	if err := json.Unmarshal(raw, &games); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}
	return games, nil
}

// GetGameByID fetches a single game by its IGDB ID.
// Returns an IGDBError with StatusCode 404 if the game is not found.
func (c *IGDBClient) GetGameByID(ctx context.Context, id int) (*IGDBGame, error) {
	body := fmt.Sprintf(`where id = %d; fields name,cover.url,platforms.name,first_release_date,rating,summary,genres.name,storyline; limit 1;`, id)
	raw, err := c.Query(ctx, "games", body)
	if err != nil {
		return nil, err
	}
	var games []IGDBGame
	if err := json.Unmarshal(raw, &games); err != nil {
		return nil, fmt.Errorf("parsing game response: %w", err)
	}
	if len(games) == 0 {
		return nil, &IGDBError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("game %d not found", id)}
	}
	return &games[0], nil
}

// GetPopularGames fetches highly-rated main games (excludes DLC, bundles, expansions).
func (c *IGDBClient) GetPopularGames(ctx context.Context) ([]IGDBGame, error) {
	// parent_game = null excludes DLC/expansions; version_parent = null excludes GOTY/Complete editions.
	// category filter alone is unreliable as IGDB leaves it null for many games.
	body := `sort rating desc; where rating_count > 100 & parent_game = null & version_parent = null; fields name,cover.url,platforms.name,first_release_date,rating; limit 20;`
	raw, err := c.Query(ctx, "games", body)
	if err != nil {
		return nil, err
	}
	var games []IGDBGame
	if err := json.Unmarshal(raw, &games); err != nil {
		return nil, fmt.Errorf("parsing popular games response: %w", err)
	}
	return games, nil
}
