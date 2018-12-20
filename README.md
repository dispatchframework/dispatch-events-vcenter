# Event Driver for VMware vCenter
This repo implements vCenter event driver that is compatible with both Knative and Dispatch.

1. [Using the driver with Knative](#using-the-driver-as-an-event-source-in-knative)
1. [Using the driver with Dispatch](#using-the-driver-with-dispatch)

## Using the driver as an Event Source in Knative

### Install Knative Serving w/ Istio

If you have installed Knative Serving & Istio components you can skip this topic. Otherwise,
follow the latest instructions [here](https://github.com/knative/docs/tree/master/eventing#installation) to install dependencies of Eventing.


### Install Knative Eventing

If you have already installed eventing & eventing sources, you can skip this topic.

The following command installs the Eventing version `v0.2.1`: 
```bash
kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.2.1/release.yaml
```

This driver also specifically requires the Knative Eventing Sources component. Eventing Sources implements the support for a generic
controller that can handle a `ContainerSource` CRD which is utilized by this driver. 

Install Eventing Sources 
```bash
kubectl apply --filename https://github.com/knative/eventing-sources/releases/download/v0.2.1/release.yaml
```

### Install Kafka Broker & Channel Provisioner

If you have already installed channel provisioner you can skip this topic. 

A Channel in Eventing acts like a conduit for event transport. A `Channel` is an http endpoint to which an event source can POST events and
is managed by the channel provisioner. The default installation of Eventing provides an in-memory channel provisioner but provisioner must
not be used for production. Hence, in this step we will install a Kafka based Channel Provisioner that is more reliable. Before you install
the provisioner, you must have access to an existing kafka cluster. If not, follow these [instructions](https://github.com/knative/eventing/tree/master/config/provisioners/kafka/broker)
to deploy a kafka cluster.

Note: If you followed the above instructions, the kafka broker URL will be `kafkabroker.kafka:9092`.

Install a Kafka Channel Provisioner
 
Note: Below yaml assumes the kafka broker URL as `kafkabroker.kafka:9092`.

```bash
kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.2.1/kafka.yaml
```

For a more detailed installation or additional configuration options check [here](https://github.com/knative/eventing/tree/master/config/provisioners/kafka).


### Create a Channel 

Creates a channel that uses the kafka channel provisioner

```bash
$ cat <<EOF > vcenter-channel.yaml
apiVersion: eventing.knative.dev/v1alpha1
kind: Channel
metadata:
  name: vcenter-kafka-channel
spec:
  provisioner:
    apiVersion: eventing.knative.dev/v1alpha1
    kind: ClusterChannelProvisioner
    name: kafka
EOF
kubectl apply --filename ./vcenter-channel.yaml
```

### Install the vCenter Driver as an Event Source

Note: Modify the YAML below to add your credentials.

```bash
$ cat <<EOF > vcenter-source.yaml
apiVersion: sources.eventing.knative.dev/v1alpha1
kind: ContainerSource
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: vcenter-source
spec:
  image: dispatchframework/dispatch-events-vcenter:latest
  args:
   - '--debug'
  env:
    - name: HOST
      value: vcenter.example.com
    - name: USERNAME
      value: "inputYourCreds"
    - name: PASSWORD
      value: "inputYourCreds"
  sink:
    apiVersion: eventing.knative.dev/v1alpha1
    kind: Channel
    name: vcenter-kafka-channel
EOF

kubectl apply --filename ./vcenter-source.yaml
```
### Create a subscription

Now, you can create a subscription to your knative service by customizing the following yaml with the name of the service:

```bash
$ cat <<EOF > vcenter-subscription.yaml
apiVersion: eventing.knative.dev/v1alpha1
kind: Subscription
metadata:
  name: vcenter-subscription
spec:
  channel:
    kind: Channel
    name: vcenter-kafka-channel
    apiVersion: eventing.knative.dev/v1alpha1
  subscriber:
    ref:
      kind: Service
      name: <your_service>
      apiVersion: serving.knative.dev/v1alpha1
EOF

kubectl apply --filename ./vcenter-subscription.yaml
```

### [Optional] Service Entry for vCenter host

By default, istio blocks outbound connections from the cluster. If this is the case with your cluster, you can add
a service entry like below to allow outbound connections to you vCenter host domain. 

```bash
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: vcenter-ext
spec:
  hosts:
  - "*.vmwarevmc.com"
  ports:
  - number: 443
    name: https
    protocol: HTTPS
  location: MESH_EXTERNAL
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: vcenter-ext
spec:
  hosts:
  - "*.vmwarevmc.com"
  tls:
  - match:
    - port: 443
      sni_hosts:
      - "*.vmwarevmc.com"
    route:
    - destination:
        host: "*.vmwarevmc.com"
        port:
          number: 443
      weight: 100
```

## Using the driver with Dispatch

**NOTE: Dispatch version 0.1.20 and older have vcenter driver built-in. Only use this driver with newer versions of Dispatch**


### 1. Get vCenter URL
Create connection string for your vCenter, in the form of:

```bash
export VCENTERURL="username:password@vcenter.example.com"
```

Replace `username`, `password` and `vcenter.example.com` with respective values for your environment.


Then create a secret file:
```bash
$ cat <<EOF > vcenter_secret.json
{
    "vcenterurl": "$VCENTERURL"
}
EOF
```
Next, create a Dispatch secret which contains vcenter credentials:
```bash
$ dispatch create secret vcenter vcenter_secret.json
Created secret: vcenter
```

`

### 2. Create Event driver type in Dispatch

Create a Dispatch event driver type with name *vcenter*:
```bash
$ dispatch create eventdrivertype vcenter dispatchframework/dispatch-events-vcenter
Created event driver type: vcenter
```

### 3. Create Event driver in Dispatch
When creating vCenter event driver, remember to set the secret you have created in step 1.

```bash
$ dispatch create eventdriver vcenter --secret vcenter
Created event driver: holy-grackle-805996
```

### 4. Create Subscription in Dispatch
To make events from eventdriver be processed by Dispatch, the last step is to create Dispatch subscription which sends events to a function. For example, to create a `vm.being.created` event subscription (bind to function hello-py):
```bash
$ dispatch create subscription hello-py --event-type="vm.being.created"
created subscription: innocent-werewolf-420270
```

`event-type` should be the VCenter event type that the subscription will be listening to. Please refer to [vSphere Web Services API reference](https://code.vmware.com/apis/196/vsphere#/doc/vim.event.Event.html) (with version respective to your vCenter environment) for full list of available events.
Note that event topic in Dispatch is transformed so that Event named `VmBeingCreatedEvent` becomes `vm.being.created` in Dispatch.

## Building event driver

```bash
$ docker build -t dispatch-events-vcenter .
``
