# duuh - docker unattended upgrades helper

[![Build Status](https://travis-ci.org/philpep/duuh.svg?branch=master)](https://travis-ci.org/philpep/duuh)
[![Go Report Card](https://goreportcard.com/badge/github.com/philpep/duuh)](https://goreportcard.com/report/github.com/philpep/duuh)

This project aims to build docker images with unattended upgrades, i.e. with fixables CVEs fixed.

## The problem with docker build cache

Docker build cache is great, it help to build docker images faster and avoid users to re-download layers upon rebuilds. You *want* docker build cache features.
But it also have a terrible drawback of ignoring unattended security upgrades coming from underlying OS package manager.
Most of the images available in registries have pending security updates :/

## Solution

`Duuh` can detect such pending updates by running os package manager commands.

    $ duuh --help
    Usage of duuh: duuh <docker image>
      -build
            Build image with unattended upgrades
      -check
            check current container and output json unattended upgrades (internal use)
      -pull
            force pull image from registry before processing
      -push
            push image to registry after processing


`duuh` will exit with status 2 if pending updates are available.

With the `-build` flag, `duuh` will build and tag a new image with pending
updates installed. A list of these pending updates are available in the
`duuh.upgrades` docker label.

Currently, `duuh` support Alpine (apk), Debian (apt), and Centos (yum) based distros.

## Example output

    $ duuh python:alpine
    2019/12/23 00:51:53 checking unattended upgrades in python:alpine
    2019/12/22 23:51:55 detected os type: alpine
    2019/12/22 23:51:55 detected upgrade: busybox-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: busybox-1.30.1-r2]
    2019/12/22 23:51:55 detected upgrade: ssl_client-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: ssl_client-1.30.1-r2]

	$ duuh -build python:alpine
    2019/12/23 00:52:40 checking unattended upgrades in python:alpine
    2019/12/22 23:52:44 detected os type: alpine
    2019/12/22 23:52:44 detected upgrade: busybox-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: busybox-1.30.1-r2]
    2019/12/22 23:52:44 detected upgrade: ssl_client-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: ssl_client-1.30.1-r2]
    FROM python:alpine
    LABEL duuh.upgrades="busybox-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: busybox-1.30.1-r2]\ 
    ssl_client-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: ssl_client-1.30.1-r2]"
    RUN apk --no-cache upgrade
    Sending build context to Docker daemon  2.048kB
    Step 1/3 : FROM python:alpine
     ---> dca462abc566
    Step 2/3 : LABEL duuh.upgrades="busybox-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: busybox-1.30.1-r2]ssl_client-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: ssl_client-1.30.1-r2]"
     ---> Running in afedc777bd90
    Removing intermediate container afedc777bd90
     ---> 5f92a9fce90b
    Step 3/3 : RUN apk --no-cache upgrade
     ---> Running in 225bf2ae2720
    fetch http://dl-cdn.alpinelinux.org/alpine/v3.10/main/x86_64/APKINDEX.tar.gz
    fetch http://dl-cdn.alpinelinux.org/alpine/v3.10/community/x86_64/APKINDEX.tar.gz
    (1/2) Upgrading busybox (1.30.1-r2 -> 1.30.1-r3)
    Executing busybox-1.30.1-r3.post-upgrade
    (2/2) Upgrading ssl_client (1.30.1-r2 -> 1.30.1-r3)
    Executing busybox-1.30.1-r3.trigger
    OK: 18 MiB in 35 packages
    Removing intermediate container 225bf2ae2720
     ---> b6638f090d74
    Successfully built b6638f090d74
    Successfully tagged python:alpine

	$ duuh python:alpine
    2019/12/23 00:53:08 checking unattended upgrades in python:alpine
    2019/12/22 23:53:09 detected os type: alpine
    2019/12/23 00:53:09 image has no unattended upgrades

    $ docker inspect -f '{{.Config.Labels}}' python:alpine
    map[duuh.upgrades:busybox-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: busybox-1.30.1-r2]ssl_client-1.30.1-r3 x86_64 {busybox} (GPL-2.0) [upgradable from: ssl_client-1.30.1-r2]]

## Install and run

### From the command line

Just download and build the code:

    $ go get github.com/philpep/duuh/...
    $ $(go env GOPATH)/bin/duuh --help
