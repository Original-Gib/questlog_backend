package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Original_Gib/questlog/clients"
	"github.com/Original_Gib/questlog/services"
	"github.com/gin-gonic/gin"
)

// IGDBServiceInterface allows the handler to be tested without a real service.
type IGDBServiceInterface interface {
	SearchGames(ctx context.Context, query string) ([]services.GameSummary, error)
	GetGameByID(ctx context.Context, id int) (*services.GameDetail, error)
	GetPopularGames(ctx context.Context) ([]services.GameSummary, error)
	GetUpcomingGames(ctx context.Context) ([]services.GameSummary, error)
}

// GamesHandler handles all game-related HTTP routes.
type GamesHandler struct {
	service IGDBServiceInterface
}

// NewGamesHandler creates a new GamesHandler backed by the given service.
func NewGamesHandler(service IGDBServiceInterface) *GamesHandler {
	return &GamesHandler{service: service}
}

// RegisterRoutes attaches the game routes to the given router group.
// Static routes (/popular) are registered before parameterized ones (/:id).
func (h *GamesHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/games/search", h.SearchGames)
	rg.GET("/games/popular", h.GetPopularGames)
	rg.GET("/games/upcoming", h.GetUpcomingGames)
	rg.GET("/games/:id", h.GetGameByID)
}

// SearchGames handles POST /api/v1/games/search
// Body: {"query": "zelda"}
func (h *GamesHandler) SearchGames(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	games, err := h.service.SearchGames(c.Request.Context(), req.Query)
	if err != nil {
		httpError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"games": games})
}

// GetGameByID handles GET /api/v1/games/:id
func (h *GamesHandler) GetGameByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be an integer"})
		return
	}

	game, err := h.service.GetGameByID(c.Request.Context(), id)
	if err != nil {
		httpError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"game": game})
}

// GetPopularGames handles GET /api/v1/games/popular
func (h *GamesHandler) GetPopularGames(c *gin.Context) {
	games, err := h.service.GetPopularGames(c.Request.Context())
	if err != nil {
		httpError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"games": games})
}

// GetUpcomingGames handles GET /api/v1/games/upcoming
func (h *GamesHandler) GetUpcomingGames(c *gin.Context) {
	games, err := h.service.GetUpcomingGames(c.Request.Context())
	if err != nil {
		httpError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"games": games})
}

// httpError maps errors to appropriate HTTP status codes and writes a JSON error response.
func httpError(c *gin.Context, err error) {
	var igdbErr *clients.IGDBError
	if errors.As(err, &igdbErr) {
		switch igdbErr.StatusCode {
		case http.StatusNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		case http.StatusUnauthorized, http.StatusForbidden:
			// Don't leak auth details upstream
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream authentication error"})
		case http.StatusTooManyRequests:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		default:
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream error"})
		}
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
