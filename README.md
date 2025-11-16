# external-dns-dnsserver-webhook [![CI](https://github.com/frantjc/external-dns-dnsserver-webhook/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/frantjc/external-dns-dnsserver-webhook/actions) [![godoc](https://pkg.go.dev/badge/github.com/frantjc/external-dns-dnsserver-webhook.svg)](https://pkg.go.dev/github.com/frantjc/external-dns-dnsserver-webhook) [![goreportcard](https://goreportcard.com/badge/github.com/frantjc/external-dns-dnsserver-webhook)](https://goreportcard.com/report/github.com/frantjc/external-dns-dnsserver-webhook)

external-dns-dnsserver-webhook is an external-dns webhook provider that configures an in-process DNS server with DNS entries rather than configuring some third party DNS provider.

This is useful for making a Kubernetes cluster expose a DNS server that advertises the DNS of its own resources.
