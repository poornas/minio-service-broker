package main

import (
	"code.cloudfoundry.org/lager"
	"github.com/minio/minio-service-broker/cmd/agent/client"
	"github.com/minio/minio-service-broker/utils"
	"github.com/pivotal-cf/brokerapi"
)

// BindingMgr holds info about the InstanceBinders
type BindingMgr struct {
	logger lager.Logger
	conf   utils.Config
	binds  map[string]*BindingInfo
	client *client.ApiClient
}

// BindingInfo holds binding state
type BindingInfo struct {
	instanceID string
	bindingID  string
	creds      utils.Credentials
	// other state info
}

// New creates a new binder manager
func NewBindingMgr(config utils.Config, logger lager.Logger) (b *BindingMgr) {
	c, err := client.New(config, logger)
	if err != nil {
		return nil
	}
	return &BindingMgr{
		logger: logger,
		conf:   config,
		binds:  make(map[string]*BindingInfo, 5),
		client: c,
	}
}

// Returns bindinginfo if it exists
func (mgr *BindingMgr) getBindingByID(bindingID string) *BindingInfo {
	//check if binding is in the map and return state info.
	// Assuming bindingId is unique across instances.
	if binding, found := mgr.binds[bindingID]; found {
		return binding
	}
	return nil
}

// Unbind unbinds the binding for a particular instance
func (mgr *BindingMgr) Unbind(instanceID string, bindingID string) error {
	if _, found := mgr.binds[bindingID]; found {
		delete(mgr.binds, bindingID)
		return nil
	}
	return brokerapi.ErrBindingDoesNotExist
}

// Exists returns a bool on whether the instance exists
func (mgr *BindingMgr) Exists(instanceID string, bindingID string) (bool, error) {
	for _, binding := range mgr.binds {
		if binding.instanceID == instanceID && binding.bindingID == bindingID {
			return true, nil
		}
	}
	return false, nil
}

// Bind binds a particular binding to instance.
func (mgr *BindingMgr) Bind(instanceID string, bindingID string) (interface{}, error) {

	creds, err := mgr.client.GetInstanceState(instanceID)
	if err != nil {
		return nil, err
	}

	// Save binding state in memory
	binding := &BindingInfo{instanceID: instanceID,
		bindingID: bindingID,
		creds:     creds}
	mgr.binds[bindingID] = binding
	return creds, nil
}
