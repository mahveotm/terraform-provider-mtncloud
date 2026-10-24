package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type RouterRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type Router struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Enabled       bool      `json:"enabled"`
	Status        string    `json:"status"`
	ExternalIP    string    `json:"externalIp"`
	Type          RouterRef `json:"type"`
	Site          RouterRef `json:"site"`
	Zone          RouterRef `json:"zone"`
	NetworkServer RouterRef `json:"networkServer"`
}

type RouterType struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	Code             string            `json:"code"`
	Creatable        bool              `json:"creatable"`
	HasNetworkServer bool              `json:"hasNetworkServer"`
	OptionTypes      []json.RawMessage `json:"optionTypes"`
}

func (c *Client) ListRouterTypes(ctx context.Context) ([]RouterType, error) {
	return listObjects[RouterType](c, ctx, "/network-router-types", "networkRouterTypes")
}

func (c *Client) GetRouterTypeByName(ctx context.Context, name string) (*RouterType, error) {
	types, err := c.ListRouterTypes(ctx)
	if err != nil {
		return nil, err
	}
	var match *RouterType
	for i := range types {
		if types[i].Name == name || types[i].Code == name {
			if match != nil {
				return nil, fmt.Errorf("router type %q is ambiguous; use its unique code", name)
			}
			match = &types[i]
		}
	}
	if match == nil {
		return nil, fmt.Errorf("router type %q not found", name)
	}
	return c.GetRouterType(ctx, match.ID)
}

func (c *Client) GetRouterType(ctx context.Context, id int64) (*RouterType, error) {
	typ, err := getByID[RouterType](c, ctx, fmt.Sprintf("/network-router-types/%d", id), "networkRouterType")
	if err != nil {
		return nil, err
	}
	if typ.ID != id {
		return nil, fmt.Errorf("router type response did not contain requested ID %d", id)
	}
	return typ, nil
}

// RouterOptions decodes additional fields inside networkRouter. Core fields
// cannot be overridden. Raw JSON preserves large IDs in type-specific options.
func RouterOptions(raw string) (map[string]any, error) {
	options := map[string]any{}
	if raw != "" {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &fields); err != nil || fields == nil {
			return nil, fmt.Errorf("options_json must be a JSON object")
		}
		for key, value := range fields {
			options[key] = value
		}
	}
	for _, key := range []string{"id", "name", "type", "site", "zone", "networkServer", "enabled"} {
		if _, exists := options[key]; exists {
			return nil, fmt.Errorf("options_json cannot override %q", key)
		}
	}
	return options, nil
}

func (c *Client) CreateRouter(ctx context.Context, fields map[string]any) (int64, error) {
	var response struct {
		ID      int64 `json:"id"`
		Success bool  `json:"success"`
	}
	if err := c.post(ctx, "/networks/routers", map[string]any{"networkRouter": fields}, &response); err != nil {
		return 0, err
	}
	if !response.Success || response.ID <= 0 {
		return 0, fmt.Errorf("router creation did not return success and a valid ID")
	}
	return response.ID, nil
}

func (c *Client) GetRouter(ctx context.Context, id int64) (*Router, error) {
	router, err := getByID[Router](c, ctx, fmt.Sprintf("/networks/routers/%d", id), "networkRouter")
	if err != nil {
		return nil, err
	}
	if router.ID != id {
		return nil, fmt.Errorf("router response did not contain requested ID %d", id)
	}
	return router, nil
}

func (c *Client) UpdateRouter(ctx context.Context, id int64, name string, enabled bool) error {
	var response struct {
		Success bool `json:"success"`
	}
	err := c.put(ctx, fmt.Sprintf("/networks/routers/%d", id), map[string]any{"networkRouter": map[string]any{"name": name, "enabled": enabled}}, &response)
	if err != nil {
		return err
	}
	if !response.Success {
		return fmt.Errorf("router update did not report success")
	}
	return nil
}

func (c *Client) DeleteRouter(ctx context.Context, id int64) error {
	var response struct {
		Success bool `json:"success"`
	}
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/networks/routers/%d", id), nil, nil, &response); err != nil {
		return err
	}
	if !response.Success {
		return fmt.Errorf("router deletion did not report success")
	}
	return nil
}
