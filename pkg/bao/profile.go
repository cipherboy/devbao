package bao

import (
	"fmt"
	"strings"

	"github.com/openbao/openbao/api"
)

func PolicySetup(client *api.Client, policy string) error {
	switch strings.ToLower(policy) {
	case "transit":
		return PolicyTransitSealMountSetup(client)
	default:
		return fmt.Errorf("unknown policy to apply: %v", policy)
	}
}

func PolicyTransitSealMountSetup(client *api.Client) error {
	if err := client.Sys().Mount("transit", &api.MountInput{
		Type: "transit",
	}); err != nil {
		return fmt.Errorf("failed to mount transit instance: %w", err)
	}

	if _, err := client.Logical().Write("transit/keys/auto-unseal", map[string]interface{}{
		"type": "aes256-gcm96",
	}); err != nil {
		return fmt.Errorf("failed to create transit unseal key: %w", err)
	}

	return nil
}
