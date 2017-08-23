package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/minio/minio-service-broker/auth"
)

type Credentials struct {
	EndpointURL string
	AccessKey   string
	SecretKey   string
	Region      string
}

// Config - TODO needs to change
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Secure    bool
}

// AccessCredentials container for access and secret keys.
type AccessCredentials struct {
	AccessKey     string `json:"accessKey,omitempty"`
	SecretKey     string `json:"secretKey,omitempty"`
	secretKeyHash []byte
}

type serverConfig struct {
	Version string `json:"version"`

	// S3 API configuration.
	Credential AccessCredentials `json:"credential"`
	Region     string            `json:"region"`
}

// GetCredentialsFromConfig fetches access key and secret key from config file
func GetCredentialsFromConfig(configFilePath string) (auth.CredentialsV4, error) {

	srvCfg := &serverConfig{}
	configFile, err := os.Open(configFilePath)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)

	jsonParser.Decode(&srvCfg)
	return auth.CredentialsV4{srvCfg.Credential.AccessKey, srvCfg.Credential.SecretKey, srvCfg.Region}, nil
}
