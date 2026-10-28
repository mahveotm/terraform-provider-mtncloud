package client

import (
	"context"
	"fmt"
	"strconv"
)

// ExternalNetwork is a router gateway option. Its value is a Morpheus network
// ID, not an OpenStack UUID, and need not be readable through /networks/{id}.
type ExternalNetwork struct {
	ID   int64  `json:"value"`
	Name string `json:"name"`
}

func (c *Client) ListExternalNetworks(ctx context.Context, cloudID, groupID, routerTypeID int64, poolCode string) ([]ExternalNetwork, error) {
	var response struct {
		Success *bool             `json:"success"`
		Data    []ExternalNetwork `json:"data"`
	}
	query := map[string]string{
		"router.zone.id": strconv.FormatInt(cloudID, 10),
		"router.site.id": strconv.FormatInt(groupID, 10),
		"router.poolId":  poolCode,
		"routerType.id":  strconv.FormatInt(routerTypeID, 10),
	}
	if err := c.get(ctx, "/options/openstackExternalNetworks", query, &response); err != nil {
		return nil, err
	}
	if response.Success != nil && !*response.Success {
		return nil, fmt.Errorf("external-network options request was unsuccessful")
	}
	for _, network := range response.Data {
		if network.ID <= 0 || network.Name == "" {
			return nil, fmt.Errorf("external-network options returned an invalid name or ID")
		}
	}
	return response.Data, nil
}
