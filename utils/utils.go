package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/cli/config"
	"github.com/docker/docker/cli/config/configfile"
	"github.com/docker/docker/cli/config/credentials"
	"github.com/urfave/cli"
)

// GetAuthConfig returns the docker registry AuthConfig.
func GetAuthConfig(c *cli.Context) (types.AuthConfig, error) {
	if c.GlobalString("username") != "" && c.GlobalString("password") != "" && c.GlobalString("registry") != "" {
		return types.AuthConfig{
			Username:      c.GlobalString("username"),
			Password:      c.GlobalString("password"),
			ServerAddress: c.GlobalString("registry"),
		}, nil
	}

	dcfg, err := config.Load(config.Dir())
	if err != nil {
		return types.AuthConfig{}, fmt.Errorf("Loading config file failed: %v", err)
	}

	// return error early if there are no auths saved
	if !dcfg.ContainsAuth() {
		if c.GlobalString("registry") != "" {
			return types.AuthConfig{
				ServerAddress: c.GlobalString("registry"),
			}, nil
		}
		return types.AuthConfig{}, fmt.Errorf("No auth was present in %s, please pass a registry, username, and password", config.Dir())
	}

	// if they passed a specific registry, return those creds _if_ they exist
	if registry := c.GlobalString("registry"); registry != "" {
		// if credential helper exists, return the creds from the credential store
		if store := getConfiguredCredentialStore(dcfg, registry); store != "" {
			creds, err := credentials.NewNativeStore(dcfg, store).Get(registry)
			if err != nil {
				return types.AuthConfig{}, fmt.Errorf("Unable to retrieve auth from Credential Store: %v", err)
			}
			return creds, nil
		}

		if creds, ok := dcfg.AuthConfigs[registry]; ok {
			return creds, nil
		}
		return types.AuthConfig{}, fmt.Errorf("No authentication credentials exist for %s", registry)
	}

	// set the auth config as the registryURL, username and Password
	for _, creds := range dcfg.AuthConfigs {
		return creds, nil
	}

	return types.AuthConfig{}, fmt.Errorf("Could not find any authentication credentials")
}

// GetRepoAndRef parses the repo name and reference.
func GetRepoAndRef(c *cli.Context) (repo, ref string, err error) {
	if len(c.Args()) < 1 {
		return "", "", errors.New("pass the name of the repository")
	}

	arg := c.Args()[0]
	parts := []string{}
	if strings.Contains(arg, "@") {
		parts = strings.Split(c.Args()[0], "@")
	} else if strings.Contains(arg, ":") {
		parts = strings.Split(c.Args()[0], ":")
	} else {
		parts = []string{arg}
	}

	repo = parts[0]
	ref = "latest"
	if len(parts) > 1 {
		ref = parts[1]
	}

	return
}

// https://github.com/moby/moby/blob/603dd8b3b48273c0c7e1f2aef5f344acac66424e/cli/command/cli.go#L141
// getConfiguredCredentialStore returns the credential helper configured for the
// given registry, the default credsStore, or the empty string if neither are
// configured.
func getConfiguredCredentialStore(c *configfile.ConfigFile, serverAddress string) string {
	if c.CredentialHelpers != nil && serverAddress != "" {
		if helper, exists := c.CredentialHelpers[serverAddress]; exists {
			return helper
		}
	}
	return c.CredentialsStore
}
