# v2l-example-scheduler

This is example code to show how to use the REST API of
the Agile Content/Edgeware ESB-3019 `ew-vod2cbm` service to
provide a linear channel with a schedule which is updated as time passes.

## Description

This simple scheduling service generates and updates the schedule of linear channel
from VoD assets and ads. The schedule is posted to a `ew-vod2cbm` service,
that serves the actual channel via HTTP to a StreamBuilder repackager.
Every second entry in the schedule is an ad, and every second a program.
The entries are added randomly among the assets found in the `assets` directory, and are classified as ads or programs depending on their paths. In a true CMS
system, there may of course be more metadata about each asset.

One does not need to schedule the whole asset. With the `offset` and
`length` attributes in the schedule JSON structure, one can choose any
interval of an asset, or even extend it to be looped by specifying
a `length` that goes beyond its end. This is used to schedule partial assets
at the start of the schedule.

The program keeps a sliding window and adds entries to the end of the
schedule, while removing entries from the past as they are moved out
of the accessible window. The window, as many other parameters, are
measured in number of GoPs. These values are set as constants in
`main.go`.

```mermaid
sequenceDiagram
    participant v2ls as v2l-example-scheduler
    participant vod2cbm as ew-vod2cbm
    participant storage
    v2ls->>vod2cbm: DELETE ch1
    Note right of vod2cbm: delete any previous version
    v2ls->>vod2cbm: DELETE assetpaths
    Note right of vod2cbm: only asset paths not in any schedule can be removed
    v2ls->>storage: read asset directories
    v2ls->>vod2cbm: POST assetpaths
    Note right of vod2cbm: asset paths must be added before they are scheduled
    v2ls->>v2ls: CreateChannel()
    v2ls->>vod2cbm: POST ch1
    vod2cbm->>storage: Load assets in schedule
    v2ls->>vod2cbm: GET schedule/ch1
    Note right of vod2cbm: The returned schedule includes asset lengths
    loop Timer
        v2ls->>v2ls: updateSchedule
        v2ls->>vod2cbm: PUT schedule/ch1
    end
```

The `ew-vod2cbm` server has a Swagger front-end available at
`http://localhost:8090/swagger`. For testing purposes, one can also stream HLS directly from the server at the URL `http://localhost:8090/ch1/index.html`.

### The assets

In general, all assets need to be coded in the same way and all video must have
the same GoP durations.

A further restriction is that all content must be in either Edgeware ESF format
or in DASH OnDemand format. ESF is
a CMAF-based format but with some additional metadata in form of a
`content_info.json` file and some binary `.dat` files. Each such asset must be accessible
via a file path, HTTP URL or on an S3 bucket.

The example content in this repo all have a GoP duration of 2000ms.

## Compatibility

Currently, this server works towards `ew-vod2cbm` version 0.10.
It is assumed to be at
`localhost:8090`, but another address can be set in the `main.go` file.
The `ew-vod2cbm` must have access to the `assets` directory.

The media tracks being produced are described in the `content template` file
`content_template.json`, and the input tracks must be compatible
with what is to be generated.


## How to run the program

You need to have an `ESB-3015/ew-vod2cbm` server running at `localhost:8090`. It needs a config file, which currently (v0.8) needs
a value for `defaultGopDurMS`. Such a config file is provided as
`ew-vod2cbm-config/config_2s_gops.json`.

To run the program you need `go` installed. You can also get a compiled binary that you can
run on any platform.

The project has no dependencies on other repos, so the binary can be built by simply running

    $ go build .

which will build the binary `v2l-example-scheduler`.

Alternatively, you can run it without building it as

     $ go run .

In both cases, there are currently no command-line parameters.

## Checking the EPG

A very simple EPG is available at `http://localhost:8090/epg/ch1`.

## Further documentation

The `ew-vod2cbm` service is not yet generally released, but there is online documentation
describing it and its API at https://docs.agilecontent.com/docs/acp/esb3019.