package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const MTNRouterTypeCode = "neutronRouter"

type RouterRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type Router struct {
	ID              int64             `json:"id"`
	Name            string            `json:"name"`
	Enabled         bool              `json:"enabled"`
	Status          string            `json:"status"`
	ExternalIP      string            `json:"externalIp"`
	Type            RouterRef         `json:"type"`
	Site            RouterRef         `json:"site"`
	Zone            RouterRef         `json:"zone"`
	NetworkServer   RouterRef         `json:"networkServer"`
	ExternalNetwork RouterRef         `json:"externalNetwork"`
	Interfaces      []RouterInterface `json:"interfaces"`
}

type RouterInterface struct {
	CIDR    string    `json:"cidr"`
	Network RouterRef `json:"network"`
}

type RouterNetwork struct {
	ZonePool RouterRef `json:"zonePool"`
	Subnets  []struct {
		ID   int64  `json:"id"`
		CIDR string `json:"cidr"`
	} `json:"subnets"`
}

func (c *Client) GetRouterNetwork(ctx context.Context, id int64) (*RouterNetwork, error) {
	return getByID[RouterNetwork](c, ctx, fmt.Sprintf("/networks/%d", id), "network")
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

// WaitForRouter checks backend completion rather than trusting the create/delete acknowledgement.
func (c *Client) WaitForRouter(ctx context.Context, id int64, deleted bool) (*Router, error) {
	for {
		router, err := c.GetRouter(ctx, id)
		if deleted && IsNotFound(err) {
			return nil, nil
		}
		if err != nil && (!IsNotFound(err) || deleted) {
			return nil, err
		}
		if err == nil && !deleted {
			switch strings.ToLower(router.Status) {
			case "ok", "available", "active":
				return router, nil
			case "failed", "error":
				return nil, fmt.Errorf("MTN router %d provisioning status is %q", id, router.Status)
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
