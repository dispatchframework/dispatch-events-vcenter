## Dispatch Event Driver for VMware vCenter
This repo implements vCenter event driver for Dispatch.

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
