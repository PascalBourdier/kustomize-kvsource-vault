package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
)

type plugin struct{}

var KVSource plugin

func getToken() (string, error) {
	t, exists := os.LookupEnv("VAULT_TOKEN")
	if !exists {
		tokenPath, exists := os.LookupEnv("VAULT_TOKEN_PATH")
		if exists == false {
			return "", errors.New("No vault token and vault token path")
		}

		tBytes, err := ioutil.ReadFile(tokenPath)
		if err != nil {
			fmt.Println("Could not read Vault token from $VAULT_TOKEN_PATH")
			return "", err
		}

		t = strings.TrimSpace(string(tBytes))
		if len(t) == 0 {
			fmt.Println("Vault token file is empty")
			return "", err
		}
	}

	return t, nil
}

func (p plugin) Get(root string, args []string) (map[string]string, error) {
	var vaultAddr = os.Getenv("VAULT_ADDR")

	var token, err = getToken()
	if err != nil {
		panic(err)
	}

	config := &api.Config{
		Address: vaultAddr,
	}

	client, err := api.NewClient(config)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	client.SetToken(token)

	mapOfSecrets := make(map[string]string)

	for _, k := range args {
		var t = strings.Split(k, "//")
		var path = t[0]
		var splitPath = strings.Split(path, "/")
		var secretPrefix = splitPath[len(splitPath)-1]
		var key = t[1]
		var splitKey = strings.Split(key, "@")

		secret, err := client.Logical().Read(path)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		if secret == nil {
			return nil, fmt.Errorf("the path %q was not found", path)
		}

		vaultSecretData, ok := secret.Data["data"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("malformed ecret data: %q", secret.Data["data"])
		}

		secretKey := fmt.Sprintf("%s_%s", secretPrefix, key)

		if len(splitKey) == 2 {
			secretKey = splitKey[1]
		}

		mapOfSecrets[secretKey], ok = vaultSecretData[splitKey[0]].(string)
		if !ok {
			return nil, fmt.Errorf("%q was not found at %q", splitKey[0], path)
		}
	}

	return mapOfSecrets, nil
}
