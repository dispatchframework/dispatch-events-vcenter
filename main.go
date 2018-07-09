package main

import (
	"flag"
	"log"
	"os"

	"github.com/vmware/dispatch/pkg/events/driverclient"
)

var vcenterURL = flag.String("vcenterurl", "https://vcenter.corp.local:443", "URL to vCenter instance")

func main() {
	flag.Parse()
	var url string
	if url = os.Getenv("VCENTERURL"); url == "" {
		url = *vcenterURL
	}

	driver, err := newDriver(url, true)
	if err != nil {
		log.Fatalf("Error when creating the driver: %s", err.Error())
	}
	defer driver.close()

	client, err := driverclient.NewHTTPClient()
	if err != nil {
		log.Fatalf("Error when creating the events client: %s", err.Error())
	}

	eventsChan, err := driver.consume(nil)
	if err != nil {
		log.Fatalf("Error when consuming vcenter events: %s", err.Error())
	}
	for event := range eventsChan {
		err = client.SendOne(event)
		if err != nil {
			// TODO: implement retry with exponential back-off
			log.Fatalf("Error when sending event: %s", err.Error())

		}
	}

}
