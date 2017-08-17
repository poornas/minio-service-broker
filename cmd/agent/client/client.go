package client

// Client does the actual instance management.
type Client interface {
	CreateInstance(parameters map[string]interface{}) (string, error)
	GetInstanceState(instanceID string) (string, error)
	DeleteInstance(instanceID string) error
	CreateBinding(parameters map[string]interface{}) (string, error)
	DeleteBinding(instanceID string, bindingID string) error
}
