# Mixing live

Mixing live is an experimental phase of a new mode where
a live source can be mixed during an interval with ordinary
VoD assets.

## Setup

The setup here is rather simple.

* The basis is still a random schedule of VoD assets and ads
* A live asset URL is specified in the main.go configuration
* The live asset needs to create segments compatible with
  the output of this service.
* A simple HTTP API allows for changing the schedule as soon as
  possible from VoD to live and back again.
  This is typically done on the upcoming segment boundary.
In this demo, a live source is mixed with a schedule of VoD and ad
asset.

The scheduler connects to the REST API of
the Agile Content/Edgeware ESB-3019 `ew-vod2cbm` service to
provide a linear channel with a schedule which is updated as time passes.

## Description

A simple test setup is to run an extra ew-vod2cbm instance that generates a
"live" source. For example, from the `sintel` asset by loopin it infintely.
Assuming that such a service is running on `localhost:8091` and producing the
channel `live`, one can add the source `http://localhost:8091/live/` in the
configuration

To switch to sending live and back to VoD, the scheduler must get a notification.

This can be done as

```
    $ curl -X http://localhost:8888/live
    $ curl -X http://localhost:8888/vod
```

In the output of the scheduler, one will see that a new schedule is sent and accepted by the `ew-vod2cbm` server.

## Current issues/shortcomings

Issue 1: A schedule entry that is less than two segments long is not allowed and will not be
accepted by `ew-vod2cbm`. This restriction must be implemented in the scheduler since it accepts a change at any moment, which may cause such a short entry.

Issue 2: At present, `ew-vod2cbm` can only switch between VoD and live at segment boundaries.
In practice, that means that one should configure the GoP duration to be the same as
segment duration, e.g. 2000ms.

Issue 3: The service has only been tested with live input from another `ew-vod2cbm` node.
If an ordinary ESB3003 live-ingest node is used, there may be issued with segments
not being ready in time. This should be handled.

Of this, issues 1 and 3 should be fixed before using this in a reliable way.

## Compatibility

Currently, this server works towards `ew-vod2cbm` with live extensions.
It is assumed to be at `localhost:8090`, but another address can be set in the `main.go` file.
The `ew-vod2cbm` must have access to the `assets` directory.

The media tracks being produced are described in the `content template` file
`content_template.json`, and the input tracks must be compatible
with what is to be generated.




## Checking the EPG

A very simple EPG is available at `http://localhost:8090/epg/ch1`.

## Further documentation

The `ew-vod2cbm` service is not yet generally released, but there is online documentation
describing it and its API at https://docs.agilecontent.com/docs/acp/esb3019.