package externaldns

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/frantjc/external-dns-dnsserver-webhook/hosts"
	xslices "github.com/frantjc/x/slices"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

type StandaloneCoreDNSProvider struct {
	provider.BaseProvider
	sync.Mutex
	File string

	Hosts *hosts.Hosts

	Endpoints []*endpoint.Endpoint
}

var _ provider.Provider = &StandaloneCoreDNSProvider{}

func (p *StandaloneCoreDNSProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	if p == nil || p.Endpoints == nil {
		return []*endpoint.Endpoint{}, nil
	}

	return p.Endpoints, nil
}

func (p *StandaloneCoreDNSProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	if p == nil {
		return fmt.Errorf("nil provider")
	} else if p.Endpoints == nil {
		p.Endpoints = []*endpoint.Endpoint{}
	}

	var (
		modified        bool
		addEndpoints    = []*endpoint.Endpoint{}
		removeEndpoints = []*endpoint.Endpoint{}
	)

	p.Lock()
	defer p.Unlock()

	if changes != nil {
		if changes.HasChanges() {
			addEndpoints = append(addEndpoints, changes.Create...)
			addEndpoints = append(addEndpoints, changes.UpdateNew...)
			removeEndpoints = append(removeEndpoints, changes.Delete...)

			if p.Hosts == nil {
				p.Hosts = &hosts.Hosts{}
			}

			if p.Hosts.Hosts == nil {
				p.Hosts.Hosts = []hosts.Host{}
			}

			for _, ep := range addEndpoints {
				if ep.RecordType == endpoint.RecordTypeA {
					for _, target := range ep.Targets {
						if ip := net.ParseIP(target); ip != nil {
							modified = p.Hosts.Add(hosts.Host{
								IP:        ip,
								Hostnames: []string{ep.DNSName},
							}) || modified
						} else {
							return fmt.Errorf("invalid IP: %s", target)
						}
					}
				}
			}

			for _, ep := range removeEndpoints {
				if ep.RecordType == endpoint.RecordTypeA {
					for _, target := range ep.Targets {
						if ip := net.ParseIP(target); ip != nil {
							modified = p.Hosts.Remove(hosts.Host{
								IP:        ip,
								Hostnames: []string{ep.DNSName},
							}) || modified
						} else {
							return fmt.Errorf("invalid IP: %s", target)
						}
					}
				}
			}

			if modified {
				p.Endpoints = append(p.Endpoints, changes.Create...)

				for _, up := range changes.UpdateNew {
					for i, ex := range p.Endpoints {
						if ex.DNSName == up.DNSName {
							p.Endpoints[i] = up
						}
					}
				}

				for _, del := range changes.Delete {
					for i, ex := range p.Endpoints {
						if ex.DNSName == del.DNSName {
							p.Endpoints[i] = nil
						}
					}
				}

				p.Endpoints = xslices.Filter(p.Endpoints, func(ep *endpoint.Endpoint, _ int) bool {
					return ep != nil
				})

				file, err := os.Create(fmt.Sprintf("%s.tmp", p.File))
				if err != nil {
					return nil
				}
				defer os.Remove(file.Name())
				defer file.Close()

				if err := p.Hosts.Encode(file); err != nil {
					return err
				}

				if err := os.Rename(file.Name(), p.File); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
