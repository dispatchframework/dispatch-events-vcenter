package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/vmware/dispatch/pkg/events/driverclient"
)

var dryRun = flag.Bool("dry-run", false, "Debug, pull messages and do not send Dispatch events")
var vcenterURL = flag.String("vcenterurl", "", "URL to vCenter instance (i.e. cloudadmin@vmc.local:<password>@vcenter.corp.local:443)")
var debug = flag.Bool("debug", false, "Enable debug mode (print more information)")
// TODO: Deprecate gateway flag in favor of sink
// (optional) gateway flag must be set if this event driver is used with Dispatch Solo. Note: this flag will be
// ignored when using this event driver with knative container source.
var dispatchEndpoint = flag.String(driverclient.DispatchEventsGatewayFlag, "localhost:8080", "events api endpoint")
// (not required) sink will be automatically set when using with knative container source. Note: this flag will supersede
// gateway.
var sink = flag.String("sink", "", "knative sink url")

func getDriverClient() driverclient.Client {
	if *dryRun {
		return nil
	}
	token := os.Getenv(driverclient.AuthToken)
	var client driverclient.Client
	var err error
	// If sink is set use the sink url and ignore gateway
	if *sink != "" {
		log.Printf("Using sink URL %s", *sink)
		client, err = driverclient.NewHTTPClient(driverclient.WithURL(*sink), driverclient.WithToken(token))
	} else {
		client, err = driverclient.NewHTTPClient(driverclient.WithGateway(*dispatchEndpoint), driverclient.WithToken(token))
	}

	if err != nil {
		log.Fatalf("Error when creating the events client: %s", err.Error())
	}
	log.Println("Event driver initialized.")
	return client
}

func main() {
	flag.Parse()
	var url string
	if url = os.Getenv("VCENTERURL"); url == "" {
		if vcenterURL != nil {
			url = *vcenterURL
		}
	}
	if url == "" {
		host := os.Getenv("HOST")
		username := os.Getenv("USERNAME")
		password := os.Getenv("PASSWORD")
		url = fmt.Sprintf("%s:%s@%s:443", username, password, host)
	}

	driver, err := newDriver(url, true)
	if err != nil {
		log.Fatalf("Error when creating the driver: %s", err.Error())
	}
	defer driver.close()

	client := getDriverClient()

	eventsChan, err := driver.consume(nil)
	if err != nil {
		log.Fatalf("Error when consuming vcenter events: %s", err.Error())
	}
	for event := range eventsChan {
		if *debug {
			log.Printf("Sending event %+v", event)
		}
		err = client.SendOne(event)
		if err != nil {
			// TODO: implement retry with exponential back-off
			log.Fatalf("Error when sending event: %s", err.Error())

		}
	}

}
