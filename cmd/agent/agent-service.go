package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/minio/minio-service-broker/auth"
)

const (
	globalMinioPath = "/var/vcap/packages/minio-server/minio"

	globalRootDir = "/var/vcap/store/minio-agent"

	globalInstancesDir = globalRootDir + "/instances"
	globalMinioDir     = globalRootDir + "/minio"

	globalInstanceBasePort = 9001
	globalAgentPort        = ":9000"
)

var globalMaxInstances = 100

type instanceConfig struct {
	Port int
}

type minioCredential struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type minioConfig struct {
	Credential minioCredential `json:"credential"`
	Region     string          `json:"region"`
}

func getMinioDir(instanceID string) string {
	return fmt.Sprintf("%s/%s", globalMinioDir, instanceID)
}

func getMinioConfigDir(instanceID string) string {
	return fmt.Sprintf("%s/config", getMinioDir(instanceID))
}

func getMinioConfigFile(instanceID string) string {
	return fmt.Sprintf("%s/config.json", getMinioConfigDir(instanceID))
}

func getMinioDataDir(instanceID string) string {
	return fmt.Sprintf("%s/data", getMinioDir(instanceID))
}

func getMinioLogsDir(instanceID string) string {
	return fmt.Sprintf("%s/logs", getMinioDir(instanceID))
}

func getMinioLogFile(instanceID string) string {
	return fmt.Sprintf("%s/minio.log", getMinioLogsDir(instanceID))
}

func getInstanceConfigFile(instanceID string) string {
	return fmt.Sprintf("%s/%s.json", globalInstancesDir, instanceID)
}

func getInstanceConfig(instanceID string) (config instanceConfig, err error) {
	configPath := getInstanceConfigFile(instanceID)
	contents, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(contents, &config)
	return config, err
}

func getFreePort() (port int, err error) {
	entries, err := ioutil.ReadDir(globalInstancesDir)
	if err != nil {
		return port, err
	}
	portAllocated := make(map[int]bool)
	var config instanceConfig
	for _, entry := range entries {
		instanceID := strings.TrimSuffix(entry.Name(), ".json")
		config, err = getInstanceConfig(instanceID)
		if err != nil {
			return port, err
		}
		portAllocated[config.Port] = true
	}
	for port = globalInstanceBasePort; port < globalInstanceBasePort+globalMaxInstances; port++ {
		if !portAllocated[port] {
			return port, nil
		}
	}
	return -1, errors.New("maximum instances already allocated")
}

func getMinioConfig(instanceID string) (minioConfig, error) {
	var config minioConfig
	minioConfigFile := getMinioConfigFile(instanceID)
	contents, err := ioutil.ReadFile(minioConfigFile)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(contents, &config)
	return config, err
}

// MinioServiceAgent holds the map of service name to status TODO => Persist agent config to some config.json
type MinioServiceAgent struct {
	log lager.Logger
	sync.Mutex
}

//CreateInstanceHandler creates an instance of minio server
func (agent *MinioServiceAgent) CreateInstanceHandler(w http.ResponseWriter, r *http.Request) {
	agent.Lock()
	defer agent.Unlock()

	vars := mux.Vars(r)
	instanceID := vars["instance-id"]
	agent.log.Info("create instance::" + r.RequestURI + "::" + instanceID)

	_, err := os.Stat(getInstanceConfigFile(instanceID))
	if err == nil {
		agent.log.Error(fmt.Sprintf("instance %s already exists", instanceID), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Spawn minio instance
	port, err := getFreePort()
	if err != nil {
		agent.log.Error("getFreePort() error", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err = agent.createInstance(instanceID, port)
	if err != nil {
		agent.log.Error("Failed to provision instance", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (agent *MinioServiceAgent) DeleteInstanceHandler(w http.ResponseWriter, r *http.Request) {
	agent.Lock()
	defer agent.Unlock()

	vars := mux.Vars(r)
	instanceID := vars["instance-id"]

	_, err := os.Stat(getInstanceConfigFile(instanceID))
	if err != nil {
		agent.log.Error(fmt.Sprintf("Instance %s not found", instanceID), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	minioConf, err := getMinioConfig(instanceID)
	if err != nil {
		agent.log.Error(fmt.Sprintf("Instance %s", instanceID), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	accessKey := minioConf.Credential.AccessKey
	secretKey := minioConf.Credential.SecretKey

	instanceConf, err := getInstanceConfig(instanceID)
	if err != nil {
		agent.log.Error(fmt.Sprintf("Instance %s", instanceID), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	port := instanceConf.Port
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/?service", port), nil)
	if err != nil {
		agent.log.Error(fmt.Sprintf("Instance %s", instanceID), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Header.Set("x-minio-operation", "stop")
	auth.CredentialsV4{accessKey, secretKey, "us-east-1"}.Sign(req)

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		agent.log.Error(fmt.Sprintf("Delete instance %s", instanceID), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.RemoveAll(getMinioDir(instanceID))
	if err != nil {
		agent.log.Error(fmt.Sprintf("Delete instance %s", instanceID), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.RemoveAll(getInstanceConfigFile(instanceID))
	if err != nil {
		agent.log.Error(fmt.Sprintf("Delete instance %s", instanceID), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (agent *MinioServiceAgent) GetInstanceHandler(w http.ResponseWriter, r *http.Request) {
	agent.Lock()
	defer agent.Unlock()

	agent.log.Info("Entering GetInstanceHandler handler ...")
	vars := mux.Vars(r)
	instanceID := vars["instance-id"]

	instanceConf, err := getInstanceConfig(instanceID)
	if err != nil {
		agent.log.Error(fmt.Sprintf("Instance %s", instanceID), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	minioConf, err := getMinioConfig(instanceID)
	if err != nil {
		agent.log.Error(fmt.Sprintf("Instance %s", instanceID), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	info := struct {
		AccessKey string
		SecretKey string
		Region    string
		Port      int
	}{
		minioConf.Credential.AccessKey,
		minioConf.Credential.SecretKey,
		minioConf.Region,
		instanceConf.Port,
	}
	contents, err := json.Marshal(info)
	if err != nil {
		agent.log.Error(fmt.Sprintf("Instance %s", instanceID), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Write(contents)
}

func (agent *MinioServiceAgent) createInstance(instanceID string, port int) error {
	minioConfigDir := getMinioConfigDir(instanceID)
	minioDataDir := getMinioDataDir(instanceID)
	minioLogsDir := getMinioLogsDir(instanceID)
	minioLogFile := getMinioLogFile(instanceID)
	instanceConfigFile := getInstanceConfigFile(instanceID)

	err := os.MkdirAll(minioConfigDir, 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(minioDataDir, 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(minioLogsDir, 0755)
	if err != nil {
		return err
	}

	logFile, err := os.Create(minioLogFile)
	if err != nil {
		return err
	}

	cmd := exec.Command(globalMinioPath, "server", "--address", ":"+strconv.Itoa(port), "--config-dir", minioConfigDir, minioDataDir)
	cmd.Stderr = logFile
	cmd.Stdout = logFile

	err = cmd.Start() // will wait for command to return
	if err != nil {
		return err
	}
	go func() {
		cmd.Wait()
	}()

	configContents, err := json.Marshal(instanceConfig{port})
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(instanceConfigFile, configContents, 0644)
	return err
}

func (agent *MinioServiceAgent) Init() error {
	if _, err := os.Stat(globalInstancesDir); os.IsNotExist(err) {
		return nil
	}
	entries, err := ioutil.ReadDir(globalInstancesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		instanceID := strings.TrimSuffix(entry.Name(), ".json")
		conf, err := getInstanceConfig(instanceID)
		if err != nil {
			return err
		}
		err = agent.createInstance(instanceID, conf.Port)
		if err != nil {
			return err
		}
	}
	return nil
}
