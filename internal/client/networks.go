package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type NetworkSubnet struct {
	ID int64 `json:"id"`
}

type Network struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Code         string          `json:"code"`
	DisplayName  string          `json:"displayName"`
	Description  string          `json:"description"`
	CIDR         string          `json:"cidr"`
	Gateway      string          `json:"gateway"`
	DNSPrimary   string          `json:"dnsPrimary"`
	DNSSecondary string          `json:"dnsSecondary"`
	VlanID       *int64          `json:"vlanId"`
	Status       string          `json:"status"`
	Active       *bool           `json:"active"`
	Visibility   string          `json:"visibility"`
	Type         map[string]any  `json:"type"`
	Zone         map[string]any  `json:"zone"`
	Site         map[string]any  `json:"site"`
	ZonePool     map[string]any  `json:"zonePool"`
	Subnets      []NetworkSubnet `json:"subnets"`
}

type NetworkType struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Category string `json:"category"`
}

// NetworkInput carries the fields used to create or update a network. The
// reference fields (GroupID/CloudID/TypeID/ResourcePoolID) are only sent on
// create; networks cannot be moved between zones, groups, types, or pools.
type NetworkInput struct {
	Name                string
	GroupID             int64
	CloudID             int64
	TypeID              *int64
	ResourcePoolID      *int64
	Description         string
	Labels              []string
	CIDR                string
	Gateway             string
	DNSPrimary          string
	DNSSecondary        string
	VlanID              *int64
	DHCPServer          *bool
	AssignPublicIP      *bool
	AllowStaticOverride *bool
	Active              *bool
	Visibility          string
}

func networkBody(input NetworkInput, includeRefs bool) map[string]any {
	network := map[string]any{"name": input.Name}
	if includeRefs {
		network["site"] = map[string]any{"id": input.GroupID}
		network["zone"] = map[string]any{"id": input.CloudID}
		if input.TypeID != nil {
			network["type"] = map[string]any{"id": *input.TypeID}
		}
		if input.ResourcePoolID != nil {
			network["zonePool"] = map[string]any{"id": *input.ResourcePoolID}
		}
	}
	if input.Description != "" {
		network["description"] = input.Description
	}
	if input.Labels != nil {
		network["labels"] = input.Labels
	}
	if input.CIDR != "" {
		network["cidr"] = input.CIDR
	}
	if input.Gateway != "" {
		network["gateway"] = input.Gateway
	}
	if input.DNSPrimary != "" {
		network["dnsPrimary"] = input.DNSPrimary
	}
	if input.DNSSecondary != "" {
		network["dnsSecondary"] = input.DNSSecondary
	}
	if input.VlanID != nil {
		network["vlanId"] = *input.VlanID
	}
	if input.DHCPServer != nil {
		network["dhcpServer"] = *input.DHCPServer
	}
	if input.AssignPublicIP != nil {
		network["assignPublicIp"] = *input.AssignPublicIP
	}
	if input.AllowStaticOverride != nil {
		network["allowStaticOverride"] = *input.AllowStaticOverride
	}
	if input.Active != nil {
		network["active"] = *input.Active
	}
	if input.Visibility != "" {
		network["visibility"] = input.Visibility
	}
	return map[string]any{"network": network}
}

func (c *Client) CreateNetwork(ctx context.Context, input NetworkInput) (*Network, error) {
	var response struct {
		Network Network `json:"network"`
	}
	if err := c.post(ctx, "/networks", networkBody(input, true), &response); err != nil {
		return nil, err
	}
	return &response.Network, nil
}

func (c *Client) GetNetwork(ctx context.Context, id int64) (*Network, error) {
	var response struct {
		Network Network `json:"network"`
	}
	if err := c.get(ctx, fmt.Sprintf("/networks/%d", id), nil, &response); err != nil {
		return nil, err
	}
	return &response.Network, nil
}

func (c *Client) GetNetworkByName(ctx context.Context, name string, zoneID int64) (*Network, error) {
	query := map[string]string{"name": name, "max": "1"}
	if zoneID != 0 {
		query["zoneId"] = strconv.FormatInt(zoneID, 10)
	}
	var response struct {
		Networks []Network `json:"networks"`
	}
	if err := c.get(ctx, "/networks", query, &response); err != nil {
		return nil, err
	}
	if len(response.Networks) == 0 {
		return nil, &APIError{StatusCode: 404, Message: fmt.Sprintf("network %q not found", name)}
	}
	return &response.Networks[0], nil
}

func (c *Client) UpdateNetwork(ctx context.Context, id int64, input NetworkInput) (*Network, error) {
	var response struct {
		Network Network `json:"network"`
	}
	if err := c.put(ctx, fmt.Sprintf("/networks/%d", id), networkBody(input, false), &response); err != nil {
		return nil, err
	}
	return &response.Network, nil
}

func (c *Client) DeleteNetwork(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/networks/%d", id), nil)
}

func (c *Client) ListNetworkTypes(ctx context.Context) ([]NetworkType, error) {
	var response struct {
		NetworkTypes []NetworkType `json:"networkTypes"`
	}
	if err := c.get(ctx, "/network-types", map[string]string{"max": "500"}, &response); err != nil {
		return nil, err
	}
	return response.NetworkTypes, nil
}

func (c *Client) GetNetworkTypeByName(ctx context.Context, nameOrCode string, openstackOnly bool) (*NetworkType, error) {
	types, err := c.ListNetworkTypes(ctx)
	if err != nil {
		return nil, err
	}
	for i := range types {
		networkType := types[i]
		if openstackOnly && !isOpenStackNetworkType(networkType) {
			continue
		}
		if networkType.Name == nameOrCode || networkType.Code == nameOrCode {
			return &networkType, nil
		}
	}
	return nil, &APIError{StatusCode: 404, Message: fmt.Sprintf("network type %q not found", nameOrCode)}
}

func isOpenStackNetworkType(networkType NetworkType) bool {
	return strings.Contains(strings.ToLower(networkType.Category), "openstack") ||
		strings.Contains(strings.ToLower(networkType.Code), "openstack")
}
