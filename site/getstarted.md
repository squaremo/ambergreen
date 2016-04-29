---
layout: page
title: Getting started with Flux
---

This is a guide to getting a minimal Flux system running. It assumes
only that you have a host running Docker -- to get to that point, you
can [install Docker][docker-install], or use
[docker-machine][docker-machine] to create a VM on which to run it.

I'll take some shortcuts here, in particular by using prepared images
for some of the prerequisites. This is just so we can proceed quickly
to "kicking the tires".

## Prerequisites

First we'll set up a couple of environment variables for things we'll
use repeatedly. We need a host IP address for whatever is running
Flux, and an address for things to find `etcd` on. The host IP needs
to be accessible from inside Docker containers, so `localhost` (or
`127.0.0.1`) won't do; the IP address assigned to your host on the
local network will do the trick, or if you're using Docker on a VM,
the address assigned to that VM.

```sh
# I got this by looking at the output of ifconfig
$ export HOST_IP=192.168.3.165
# If using docker-machine, use something like
# export HOST_IP=$(docker-machine ip flux)
```

OK, now let's run the two bits of infrastructure we need: etcd and
Prometheus. We'll run them in containers, of course.

```sh
$ docker run --name=etcd -d -p $HOST_IP:2379:2379 quay.io/coreos/etcd \
       --listen-client-urls http://0.0.0.0:2379 \
       --advertise-client-urls=http://localhost:2379
# ...
$ export ETCD_ADDRESS=http://$HOST_IP:2379

# And run our pre-baked image of Prometheus
$ docker run --name=prometheus -d -e ETCD_ADDRESS -p $HOST_IP::9090 \
       weaveworks/flux-prometheus-etcd
# ...
$ export PROMETHEUS_ADDRESS=http://$(docker port prometheus 9090)
```

Now we have both etcd and prometheus, and (in the environment entries)
what we need to tell Flux so it can reach them.

## Starting fluxd

`fluxd` is a container that performs two functions:

* It listens to a docker daemon running on the host to find out when
  containers that are service instances are started and stopped.

* It routes connections and requests from client containers on the
  host to service instances.

It's necessary to pass a few options to `docker run` to give `fluxd`
the privileges it needs.  If you were deploying flux as part of a
production system, you would incorporate these options in your
deployment scripts, but we show them in full here:

```sh
$ docker run --name=fluxd -d -e ETCD_ADDRESS \
       --net=host --cap-add=NET_ADMIN \
       -v /var/run/docker.sock:/var/run/docker.sock \
       weaveworks/fluxd --host-ip $HOST_IP
```

Briefly, the purpose of these options is as follows:

* `-e ETCD_ADDRESS` tells fluxd how to connect to etcd, which is used for
coordination in flux.
* `--net=host --cap-add=NET_ADMIN` permits fluxd to do connection routing.
* `-v /var/run/docker.sock:/var/run/docker.sock` allows fluxd to connect
to the local docker daemon, to listen for container events.
* `--host-ip $HOST_IP` tell fluxd what IP address should be used on other
hosts to connect to containers on this host.

Flux is now running, and you can check this in Docker:

```sh
$ docker ps
CONTAINER ID        IMAGE                             COMMAND                  CREATED             STATUS              PORTS                                                         NAMES
27d30bc8ae4a        weaveworks/fluxd                  "/home/flux/fluxd --h"   3 seconds ago      Up 3 seconds                                                                    fluxd
# ...
```

## Trying it out

Now we'll actually use Flux! Once it's running, you control Flux with
`fluxctl`. This is available as a Docker image. We'll drive it using
an alias, to avoid typing the `docker run ...` bit again and again:

```sh
$ alias fluxctl="docker run --rm -e ETCD_ADDRESS weaveworks/fluxctl"
```

To try it out, see what `fluxctl info` gives us:

```sh
$ fluxctl info
HOSTS
192.168.3.165

SERVICES
```

Not much there -- but since we haven't done anything yet, no errors is
all we should expect. You can try `fluxctl info` at each stage that
follows, to see how it reports the state of the system.

We'll start by creating a service `hello`, which will represent some
hello-world containers we'll run presently.

```sh
$ fluxctl service hello --address 10.128.0.1:80 --protocol=http
$ fluxctl select hello default --image weaveworks/hello-world
```

The first command defines the service, gives it an address, and tells
Flux that it should treat traffic to the service as HTTP (the default
is just TCP). The second command adds a rule called `default`, that
will select containers using the image `weaveworks/hello-world` to be
instances of the service; i.e., to handle requests to the service
address.

Let's start some of those containers.

```sh
$ docker run -d -P weaveworks/hello-world
# ...
$ docker run -d -P weaveworks/hello-world
# ...
$ docker run -d -P weaveworks/hello-world
# ...
```

Now we can see if Flux has noticed them:

```sh
$ fluxctl info
HOSTS
192.168.3.165

SERVICES
hello
  Address: 10.128.0.1:80
  Protocol: http
  RULES
    default {"image":"weaveworks/hello-world"}
  INSTANCES
    df83e63e1d4488126aaa2cd71af7e73b10da13d5b4e6de54bf94d1889758e10a 192.168.3.165:32768 live
    9e3f36dc6ceb7e24c2a7f06fed950e613f5c856cd2124935cd9d07b8e3872976 192.168.3.165:32769 live
    8ddfcaaf64fdf10b47440bba61d0560371828ba49d0cde06d7f8969a77ae9e09 192.168.3.165:32770 live
```

Neat. Let's check if we can get an actual web page up.

```sh
$ docker run --rm tutum/curl sh -c 'while true; do curl -s http://10.128.0.1/; done'
# ... aaaah lots of output ...
^C
```

How about seeing it in a browser? Well, at the moment Flux is only
exposing the service on the service address, and that's only available
locally on the hosts Flux is running on (a bit like 127.0.0.1). If we
want to expose the service on an externally-accessible address, we can
run an edge balancer -- so called because it runs on the edge of your
application, making it available to clients from outside.

```sh
$ docker run -p 8080:80 -d -e ETCD_ADDRESS -e SERVICE=hello \
    weaveworks/flux-edgebal
```

Now you should be able to see the service by pointing a browser at
`http://$HOST_IP:8080/`.

![Hello World in a browser](images/hello-world.png)

## Starting the web UI

Flux had a web console for seeing the services with ther instances,
and tracking their metrics.

To run it, we need the etcd address and the Prometheus address from
before.

```sh
$ docker run --name=fluxweb -d -e ETCD_ADDRESS -e PROMETHEUS_ADDRESS -P \
    weaveworks/flux-web
```

The `-P` means Docker will choose the port on which to expose the web
UI; you can see what address to open using `docker port fluxweb
7070`, or just use `-p 7070:7070` in the above to fix it to `7070`.

Behold the web UI:

![Flux web user interface example](images/flux-ui.png)

[docker-install]: https://docs.docker.com/engine/installation/
[docker-machine]: https://docs.docker.com/machine/install-machine/
