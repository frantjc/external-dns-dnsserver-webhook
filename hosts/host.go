package hosts

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"regexp"
	"slices"
	"strings"

	xslices "github.com/frantjc/x/slices"
)

type Host struct {
	IP        net.IP
	Hostnames []string
}

func (h *Host) String() string {
	return h.IP.String() + " " + strings.Join(h.Hostnames, " ")
}

func (h *Host) GoString() string {
	return "Host{" + h.String() + "}"
}

type Hosts struct {
	Hosts []Host
}

var hostnameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`)

func Decode(r io.Reader) (*Hosts, error) {
	var (
		scanner = bufio.NewScanner(r)
		hosts   = []Host{}
	)

	for scanner.Scan() {
		line := strings.TrimSpace(strings.Split(scanner.Text(), "#")[0])

		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid hosts line: %s", line)
		}

		ip := net.ParseIP(fields[0])
		if ip != nil {
			for _, hostname := range fields[1:] {
				if !hostnameRegexp.MatchString(hostname) {
					return nil, fmt.Errorf("invalid hostname: %s", hostname)
				}
			}

			hosts = append(hosts, Host{
				IP:        ip,
				Hostnames: fields[1:],
			})
		}
	}

	return &Hosts{hosts}, scanner.Err()
}

func (h *Host) Encode(w io.Writer) error {
	if h != nil && h.IP != nil && len(h.Hostnames) > 0 {
		_, err := fmt.Fprintln(w, h.IP, strings.Join(h.Hostnames, " "))
		return err
	}

	return nil
}

func (hs *Hosts) Encode(w io.Writer) error {
	for _, h := range hs.Hosts {
		if err := h.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

func (hs *Hosts) Add(g Host) (modified bool) {
	if len(g.Hostnames) == 0 || g.IP == nil {
		return false
	}

	for i, h := range hs.Hosts {
		if g.IP.Equal(h.IP) {
			if xslices.Every(g.Hostnames, func(hostname string, _ int) bool {
				return slices.Contains(h.Hostnames, hostname)
			}) {
				return false
			}

			hs.Hosts[i] = Host{
				IP:        g.IP,
				Hostnames: append(h.Hostnames, g.Hostnames...),
			}

			return true
		}
	}

	hs.Hosts = append(hs.Hosts, g)

	return true
}

func (hs *Hosts) Remove(g Host) (modified bool) {
	if len(g.Hostnames) == 0 || g.IP == nil {
		return false
	}

	for i, h := range hs.Hosts {
		if g.IP.Equal(h.IP) {
			hs.Hosts[i] = Host{
				IP: g.IP,
				Hostnames: xslices.Filter(h.Hostnames, func(hostname string, _ int) bool {
					return !slices.Contains(g.Hostnames, hostname)
				}),
			}

			return len(h.Hostnames) != len(hs.Hosts[i].Hostnames)
		}
	}

	hosts := xslices.Filter(hs.Hosts, func(h Host, _ int) bool {
		return len(h.Hostnames) > 0
	})
	modified = len(hosts) != len(hs.Hosts)
	hs.Hosts = hosts

	return
}
