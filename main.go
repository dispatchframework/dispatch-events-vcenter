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
var dispatchEndpoint = flag.String(driverclient.DispatchEventsGatewayFlag, "localhost:8080", "events api endpoint")

func getDriverClient() driverclient.Client {
	if *dryRun {
		return nil
	}
	token := os.Getenv(driverclient.AuthToken)
	client, err := driverclient.NewHTTPClient(driverclient.WithGateway(*dispatchEndpoint), driverclient.WithToken(token))
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
