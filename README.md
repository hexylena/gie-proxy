# GIE Proxy

The Galaxy Interactive Environments proxy is a websocket aware HTTP proxy with
cookie based authentication.

## Features

- [x] Proxy HTTP + WS
- [x] Supports running under a proxy prefix
- [x] API to allow dynamically adding new proxy routes
- [x] save/restore proxy routes across restarts
- [ ] kill routes after N minutes
- [ ] execute docker-compose kill/docker kills on route finish

## Building

First, make sure you have the Go build environment correctly installed. See
http://golang.org/ for more information.

Then run "make". This will in turn call the go utility to build the load
balancer, resulting in a binary named `gxproxy`


## Configuration

TODO!

## License

MIT Licensed. Forked and rewriten from
[upstream](https://github.com/akrennmair/drunken-hipster). See the file LICENSE
for license information.

## Author

Eric Rasche <esr@tamu.edu>

Based on code from https://github.com/akrennmair/drunken-hipster, however a
substantial rewrite took place, leaving only a small portion of the original
code base intact (the copy functions in util.go).
