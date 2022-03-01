# v2l-example-scheduler

This is example code to show how to use the REST API of
the Agile Content/Edgeware ESB-3019 `vod2cbm` service.

## Description

This simple service generates a linear channel from VoD
assets and ads. Every second entry in the schedule is
an ad, and every second a program.

### The assets

In general, all assets need to be coded in the same way and all video must have
the same GoP durations.

A further restriction is that all content must be in Edgeware ESF format. That is
a CMAF-based format but with some additional metadata in form of a
`content_info.json` file and some binary `.dat` files. Each such asset must be accessible
via a file path, HTTP URL or on an S3 bucket.

The example content in this repo all have a GoP duration of 2000ms.

## Compatibility

Currently, this server works towards `ew-vod2cbm` version 0.8.


