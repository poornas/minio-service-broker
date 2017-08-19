package client

import "github.com/minio/minio-service-broker/utils"

// Client does the actual instance management.
type Client interface {
	CreateInstance(parameters map[string]interface{}) (string, error)
	GetInstanceStatus(instanceID string) (*utils.Credentials, error)
	DeleteInstance(instanceID string) error
}
