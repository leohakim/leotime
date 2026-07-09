package store

import (
	"context"
	"strings"
)

func hasBillableRate(client *Client, project *Project) bool {
	if project != nil && project.DefaultHourlyRateMinor != nil && *project.DefaultHourlyRateMinor > 0 {
		return true
	}
	if client != nil && client.DefaultHourlyRateMinor > 0 {
		return true
	}
	return false
}

func defaultBillableFromRates(client *Client, project *Project) bool {
	return hasBillableRate(client, project)
}

func (s *Store) loadClientProjectForBillable(ctx context.Context, userID, clientID, projectID string) (*Client, *Project, error) {
	var client *Client
	var project *Project

	if strings.TrimSpace(clientID) != "" {
		loaded, err := s.ClientByID(ctx, userID, clientID)
		if err != nil {
			return nil, nil, err
		}
		client = loaded
	}

	if strings.TrimSpace(projectID) != "" {
		loaded, err := s.ProjectByID(ctx, userID, projectID)
		if err != nil {
			return nil, nil, err
		}
		project = loaded
		if client == nil && project.ClientID != "" {
			loadedClient, err := s.ClientByID(ctx, userID, project.ClientID)
			if err != nil {
				return nil, nil, err
			}
			client = loadedClient
		}
	}

	return client, project, nil
}

func (s *Store) resolveBillableFlag(ctx context.Context, userID, clientID, projectID string, requested bool, creating bool) (bool, error) {
	if !creating {
		return requested, nil
	}
	if !requested {
		return false, nil
	}

	client, project, err := s.loadClientProjectForBillable(ctx, userID, clientID, projectID)
	if err != nil {
		return false, err
	}
	return defaultBillableFromRates(client, project), nil
}
