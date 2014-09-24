sdocker
=======

## Overview

This is a simple program written in Golang that is effectively a proxy for
the `docker` client program.  For any existing value for `DOCKER_HOST`, it
just acts as a "passthrough" (i.e., it does nothing).  However, if the
value of `DOCKER_HOST` uses the `ssh` scheme, then it opens an ssh tunnel
runs a `docker` and then closes the tunnel.  In this way, it allows a secure
connection ot the docker server.

## Examples

Here are some example values for `DOCKER_HOST` using the `ssh` scheme:

```
DOCKER_HOST=ssh://example.com
```

Creates a tunnel to `example.com` as the current user where the server is running
docker on the default docker port (4243).

```
DOCKER_HOST=ssh://example.com:2375
```

Creates a tunnel to `example.com` as the current user where the server is running
docker on port 2375.

```
DOCKER_HOST=ssh://nobody@example.com
```

Creates a tunnel to `example.com` as user `nobody` where the server is running
docker on the default docker port (4243).

```
DOCKER_HOST=ssh://nobody@example.com:2375/35532
```

Creates a tunnel to `example.com` as user `nobody` where the server is running
docker on port 2375 but the tunnel opens locally on port 35532.


## Conclusion

Of course, all of this could be built directly into the `docker` client.  The
nice thing about this approach is that it uses ssh keys, which most people are
familiar with, it uses an established security scheme (instead of inventing
something new) and working with ssh keys is generally pretty easy (and still
allows you to audit how made the connections).  Hopefully, this scheme will
be incorporated into the real docker client at some point.
