package provider

import (
	"context"
	"fmt"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

type routerScope struct {
	Group   *client.Group
	CloudID int64
	Pool    client.ResourcePool
}

// Discovery and creation must use the same group, cloud, and project resolution.
func resolveRouterScope(ctx context.Context, api *client.Client, defaults *mtnCloudProviderData, groupName, poolName string, cloudID int64) (*routerScope, error) {
	if defaults != nil {
		if groupName == "" {
			groupName = defaults.Group
		}
		if poolName == "" {
			poolName = defaults.ResourcePool
		}
	}
	if groupName == "" {
		return nil, fmt.Errorf("set group on the resource/data source or provider")
	}
	if poolName == "" {
		return nil, fmt.Errorf("set resource_pool on the resource/data source or provider")
	}
	group, err := api.GetGroupByName(ctx, groupName)
	if err != nil {
		return nil, err
	}
	if cloudID == 0 {
		if len(group.CloudIDs) != 1 {
			return nil, fmt.Errorf("set cloud_id explicitly: group %q has %d clouds", groupName, len(group.CloudIDs))
		}
		cloudID = group.CloudIDs[0]
	}
	found := false
	for _, id := range group.CloudIDs {
		if id == cloudID {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("cloud_id %d is not attached to group %q", cloudID, groupName)
	}
	pools, err := api.ListResourcePools(ctx, cloudID, group.ID)
	if err != nil {
		return nil, err
	}
	var match *client.ResourcePool
	for i := range pools {
		if pools[i].Name == poolName || pools[i].Code == poolName {
			if match != nil {
				return nil, fmt.Errorf("resource pool %q is ambiguous; use its unique code", poolName)
			}
			match = &pools[i]
		}
	}
	if match == nil {
		return nil, fmt.Errorf("resource pool %q not found in cloud %d and group %q", poolName, cloudID, groupName)
	}
	if match.ID <= 0 || match.Code == "" {
		return nil, fmt.Errorf("resource pool %q has no valid ID or code", poolName)
	}
	return &routerScope{Group: group, CloudID: cloudID, Pool: *match}, nil
}
