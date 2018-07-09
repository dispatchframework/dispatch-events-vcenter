///////////////////////////////////////////////////////////////////////
// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
///////////////////////////////////////////////////////////////////////

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/vmware/dispatch/pkg/events"
	"github.com/vmware/dispatch/pkg/utils"
)

// NO TESTS

const eventTypeVersion = "0.1"

type vCenterEvent struct {
	Metadata interface{} `json:"metadata"`
	Time     time.Time   `json:"time"`
	Category string      `json:"category"`
	Message  string      `json:"message"`
}

// newDriver creates a new vCenter event driver
func newDriver(vcenterURL string, insecure bool) (*vCenterDriver, error) {
	vClient, err := newVCenterClient(context.Background(), vcenterURL, insecure)
	if err != nil {
		return nil, err
	}
	manager := event.NewManager(vClient.Client)
	return &vCenterDriver{
		vcenterURL: vcenterURL,
		insecure:   insecure,
		manager:    manager,
		client:     vClient,
	}, nil
}

type vCenterDriver struct {
	vcenterURL string
	insecure   bool
	manager    *event.Manager
	client     *govmomi.Client
	done       func()
}

func (d *vCenterDriver) consume(topics []string) (<-chan *events.CloudEvent, error) {
	ctx, cancel := context.WithCancel(context.Background())
	d.done = cancel
	eventsChan := make(chan *events.CloudEvent)
	go func() {
		err := d.manager.Events(
			ctx, // context
			// TODO: add support for filter customization
			[]types.ManagedObjectReference{d.client.ServiceContent.RootFolder}, // object(s) to monitor
			10,   // maximum number of events per page passed to handler
			true, // poll for events indefinitely
			true, // ignore limit of monitored objects (10)
			d.handler(eventsChan, false), // handler executed for each event page
		)
		if err != nil {
			log.Printf("Error when reading events from vCenter: %+v", err)
		}
		close(eventsChan)
	}()

	return eventsChan, nil
}

func (d *vCenterDriver) topics() []string {
	// TODO: generate it based on API WSDL
	return nil
}

func (d *vCenterDriver) close() error {
	d.done()
	return nil
}

func (d *vCenterDriver) handler(events chan *events.CloudEvent, multiple bool) func(types.ManagedObjectReference, []types.BaseEvent) error {

	return func(obj types.ManagedObjectReference, page []types.BaseEvent) error {

		event.Sort(page) // sort by event time

		for _, e := range page {
			processedEvent, err := d.processEvent(e)
			if err != nil {
				return errors.Wrap(err, "error processing event")
			}

			events <- processedEvent
		}

		return nil
	}
}

func (d *vCenterDriver) processEvent(e types.BaseEvent) (*events.CloudEvent, error) {
	eventType := reflect.TypeOf(e).Elem().Name()

	cat, err := d.manager.EventCategory(context.Background(), e)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving event category")
	}

	ve := &vCenterEvent{
		Time:     e.GetEvent().CreatedTime,
		Category: cat,
		Message:  strings.TrimSpace(e.GetEvent().FullFormattedMessage),
	}

	// if this is a TaskEvent gather a little more information
	if t, ok := e.(*types.TaskEvent); ok {
		// some tasks won't have this information, so just use the event message
		if t.Info.Entity != nil {
			ve.Message = fmt.Sprintf("%s (target=%s %s)", ve.Message, t.Info.Entity.Type, t.Info.EntityName)
		}
	}
	ve.Metadata = processEventMetadata(e)

	topic := convertToTopic(eventType)

	return d.dispatchEvent(topic, ve)
}

func (d *vCenterDriver) dispatchEvent(topic string, ve *vCenterEvent) (*events.CloudEvent, error) {

	encoded, err := json.Marshal(*ve)
	if err != nil {
		return nil, err
	}

	event := events.CloudEvent{
		EventType:          topic,
		EventTypeVersion:   eventTypeVersion,
		CloudEventsVersion: events.CloudEventsVersion,
		Source:             "vcenter1", // TODO: make this configurable
		EventID:            uuid.NewV4().String(),
		EventTime:          time.Time{},
		ContentType:        "application/json",
		Data:               encoded,
	}

	return &event, nil
}

func newVCenterClient(ctx context.Context, vcenterURL string, insecure bool) (*govmomi.Client, error) {

	url, err := soap.ParseURL(vcenterURL)
	if err != nil {
		return nil, err
	}

	return govmomi.NewClient(ctx, url, insecure)
}

func convertToTopic(eventType string) string {

	eventType = strings.Replace(eventType, "Event", "", -1)
	return utils.CamelCaseToLowerSeparated(eventType, ".")
}
