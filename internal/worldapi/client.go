// Package worldapi reads the parts of WorldAPI's REST API that the exporter turns into metrics.
package worldapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Client calls one WorldAPI, like http://localhost:8110/api/v1.
type Client struct {
	base      string
	token     string
	http      *http.Client
	userAgent string
}

func NewClient(base, token, userAgent string, httpClient *http.Client) *Client {
	return &Client{base: strings.TrimRight(base, "/"), token: token, http: httpClient, userAgent: userAgent}
}

// Error is WorldAPI's answer to a request it refused, like 403 forbidden.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("HTTP %d", e.Status)
	}
	return fmt.Sprintf("HTTP %d %s: %s", e.Status, e.Code, e.Message)
}

// Get reads path with the query into out.
func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) error {
	target := c.base + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var answer struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &answer)
		return &Error{Status: resp.StatusCode, Code: answer.Error.Code, Message: answer.Error.Message}
	}
	return json.Unmarshal(body, out)
}

// GetServer reads GET /server.
func (c *Client) GetServer(ctx context.Context) (*Server, error) {
	var out Server
	return &out, c.Get(ctx, "/server", nil, &out)
}

// GetPerformance reads GET /server/performance.
func (c *Client) GetPerformance(ctx context.Context) (*Performance, error) {
	var out Performance
	return &out, c.Get(ctx, "/server/performance", nil, &out)
}

// GetWorlds reads GET /worlds, then each world's details.
func (c *Client) GetWorlds(ctx context.Context) ([]World, error) {
	var list []struct {
		Name string `json:"name"`
	}
	if err := c.Get(ctx, "/worlds", nil, &list); err != nil {
		return nil, err
	}
	worlds := make([]World, 0, len(list))
	for _, item := range list {
		var world World
		if err := c.Get(ctx, "/worlds/"+url.PathEscape(item.Name), nil, &world); err != nil {
			return nil, err
		}
		worlds = append(worlds, world)
	}
	return worlds, nil
}

// GetOnlinePlayers reads GET /players/online.
func (c *Client) GetOnlinePlayers(ctx context.Context) (*OnlinePlayers, error) {
	var out OnlinePlayers
	return &out, c.Get(ctx, "/players/online", nil, &out)
}

// GetPlayers reads every page of GET /players.
func (c *Client) GetPlayers(ctx context.Context) ([]PlayerSummary, error) {
	var players []PlayerSummary
	cursor := ""
	for range 1000 {
		query := url.Values{"limit": {"100"}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		var page struct {
			Items      []PlayerSummary `json:"items"`
			NextCursor *string         `json:"nextCursor"`
		}
		if err := c.Get(ctx, "/players", query, &page); err != nil {
			return nil, err
		}
		players = append(players, page.Items...)
		if page.NextCursor == nil || *page.NextCursor == "" || len(page.Items) == 0 {
			break
		}
		cursor = *page.NextCursor
	}
	return players, nil
}

// GetLeaderboard reads GET /leaderboards/{statistic}: a preset or a statistic like custom:jump.
func (c *Client) GetLeaderboard(ctx context.Context, statistic string) (*Leaderboard, error) {
	var out Leaderboard
	return &out, c.Get(ctx, "/leaderboards/"+url.PathEscape(statistic), url.Values{"limit": {"500"}}, &out)
}

// GetSystem reads GET /server/system: the host (system:read).
func (c *Client) GetSystem(ctx context.Context) (*System, error) {
	var out System
	return &out, c.Get(ctx, "/server/system", nil, &out)
}

// GetLogs reads the newest console lines of GET /server/logs, oldest first (logs:read).
func (c *Client) GetLogs(ctx context.Context, limit int) ([]LogLine, error) {
	var out struct {
		Lines []LogLine `json:"lines"`
	}
	return out.Lines, c.Get(ctx, "/server/logs", url.Values{"limit": {strconv.Itoa(limit)}}, &out)
}

// GetPlugins reads GET /server/plugins.
func (c *Client) GetPlugins(ctx context.Context) ([]Plugin, error) {
	var out []Plugin
	return out, c.Get(ctx, "/server/plugins", nil, &out)
}

// GetModeration reads the bans, operators, and whitelist (players:moderate).
func (c *Client) GetModeration(ctx context.Context) (*Moderation, error) {
	var (
		out       Moderation
		bans      struct{ Players, Ips []json.RawMessage }
		ops       struct{ Players []json.RawMessage }
		whitelist struct{ Players []json.RawMessage }
	)
	if err := c.Get(ctx, "/server/bans", nil, &bans); err != nil {
		return nil, err
	}
	if err := c.Get(ctx, "/server/ops", nil, &ops); err != nil {
		return nil, err
	}
	if err := c.Get(ctx, "/server/whitelist", nil, &whitelist); err != nil {
		return nil, err
	}
	out.BannedPlayers, out.BannedAddresses = len(bans.Players), len(bans.Ips)
	out.Operators, out.Whitelisted = len(ops.Players), len(whitelist.Players)
	return &out, nil
}

// GetEntitySummary reads GET /worlds/{world}/entities/summary: what crowds a world.
func (c *Client) GetEntitySummary(ctx context.Context, world string) (*EntitySummary, error) {
	var out EntitySummary
	return &out, c.Get(ctx, "/worlds/"+url.PathEscape(world)+"/entities/summary", nil, &out)
}

// GetBossStats reads Mo’Bosses' GET /extensions/mobosses/stats.
func (c *Client) GetBossStats(ctx context.Context) (*BossStats, error) {
	var out BossStats
	return &out, c.Get(ctx, "/extensions/mobosses/stats", nil, &out)
}

// GetExtensions reads GET /extensions: the plugins that share data through WorldAPI.
func (c *Client) GetExtensions(ctx context.Context) ([]Extension, error) {
	var out []Extension
	return out, c.Get(ctx, "/extensions", nil, &out)
}

// GetEvents reads one page of GET /events.
func (c *Client) GetEvents(ctx context.Context, query url.Values) (*EventPage, error) {
	var out EventPage
	return &out, c.Get(ctx, "/events", query, &out)
}
