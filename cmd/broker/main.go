package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf/brokerapi"
)

const (
	// DefaultServiceName is the name of Minio service on the marketplace
	DefaultServiceName = "minio-service"

	// DefaultServiceDescription is the description of the default service
	DefaultServiceDescription = "Minio Service Broker"

	// DefaultPlanName is the name of our supported plan
	DefaultPlanName = "minio-plan"
	// DefaultPlanID is the ID of our supported plan
	DefaultPlanID = "minio-plan-id"
	//DefaultPlanDescription describes the default plan offered.
	DefaultPlanDescription = "Secure access to a single instance Minio server"

	// DefaultServiceID is placeholder id for the service broker
	DefaultServiceID = "minio-service-id"
)

func main() {
	// Create logger
	log := lager.NewLogger("minio-servicebroker")
	log.RegisterSink(lager.NewWriterSink(os.Stderr, lager.DEBUG))
	log.RegisterSink(lager.NewWriterSink(os.Stderr, lager.INFO))

	// Ensure username and password are present
	username := os.Getenv("SECURITY_USER_NAME")
	if username == "" {
		username = "miniobroker"
	}
	password := os.Getenv("SECURITY_USER_PASSWORD")
	if password == "" {
		password = "miniobroker123"
	}
	credentials := brokerapi.BrokerCredentials{
		Username: username,
		Password: password,
	}

	u, err := url.Parse(fmt.Sprintf("http://%s:9000", os.Getenv("MINIO_AGENT_HOST")))
	if err != nil {
		return
	}

	// Setup the broker
	broker := &MinioServiceBroker{
		log:                log,
		serviceID:          DefaultServiceID,
		serviceName:        DefaultServiceName,
		serviceDescription: DefaultServiceDescription,

		planName:        DefaultPlanName,
		planID:          DefaultPlanID,
		planDescription: DefaultPlanDescription,
		bindablePlan:    true,
		agent:           agentClient{u: *u},
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	brokerAPI := brokerapi.New(broker, log, credentials)
	http.Handle("/", brokerAPI)
	log.Info("Listening for requests")

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Error("Failed to start the server", err)
	}

}
