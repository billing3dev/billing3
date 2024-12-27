package gateways

import (
	"billing3/database"
	"context"
	"fmt"
)

func getSettings(ctx context.Context, name string) (map[string]string, error) {
	g, err := database.Q.FindGatewayByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	return g.Settings, nil
}
