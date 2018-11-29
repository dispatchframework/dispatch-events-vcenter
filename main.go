package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/vmware/dispatch/pkg/events/driverclient"
)

var vcenterURL = flag.String("vcenterurl", "", "URL to vCenter instance (i.e. cloudadmin@vmc.local:<password>@vcenter.corp.local:443)")
var debug = flag.Bool("debug", false, "Enable debug mode (print more information)")
var endpoint = flag.String(driverclient.DispatchAPIEndpointFlag, "", "events api endpoint")

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
	token := os.Getenv(driverclient.AuthToken)

	driver, err := newDriver(url, true)
	if err != nil {
		log.Fatalf("Error when creating the driver: %s", err.Error())
	}
	defer driver.close()
	// Get auth token

	// Use HTTP mode of sending events
	client, err := driverclient.NewHTTPClient(driverclient.WithEndpoint(*endpoint), driverclient.WithToken(token))
	if err != nil {
		log.Fatalf("Error when creating the events client: %s", err.Error())
	}

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
