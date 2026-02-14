//go:build integration

package integration

import (
	"context"
	"testing"
)

func TestConfigFileSave(t *testing.T) {
	client := vyosClient(t)

	if err := client.SaveConfig(context.Background()); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
}
