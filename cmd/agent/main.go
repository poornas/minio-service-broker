package main

import (
	"net/http"
)

import (
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
)

var log = lager.NewLogger("minio-serviceagent")

func main() {
	//creds := auth.CredentialsV4{"miniobroker", "miniobroker123", "us-east-1"}

	// Create logger
	log.RegisterSink(lager.NewWriterSink(os.Stderr, lager.DEBUG))
	log.RegisterSink(lager.NewWriterSink(os.Stderr, lager.INFO))

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "9001"
	}
	r := mux.NewRouter()
	r.HandleFunc("/instance/create", CreateInstanceHandler)
	r.HandleFunc("/instance/delete", DeleteInstanceHandler)
	r.HandleFunc("/instance/status", InstanceStatusHandler)
	//r.HandleFunc("/binding/create", CreateBindingHandler)
	//r.HandleFunc("/binding/delete", DeleteBindingHandler)
	//r.HandleFunc("/binding/status", BindingStatusHandler)

	http.ListenAndServe(":9001", r)
}
