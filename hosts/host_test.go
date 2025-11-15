package hosts_test

import (
	"bytes"
	"net"
	"testing"

	"github.com/frantjc/external-dns-dnsserver-webhook/hosts"
)

func TestHosts(t *testing.T) {
	h, err := hosts.Decode(bytes.NewReader([]byte("0.0.0.0 frantj.cc")))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	b := new(bytes.Buffer)
	if err = h.Encode(b); err != nil {
		t.Error(err)
		t.FailNow()
	}

	expected := "0.0.0.0 frantj.cc\n"
	if b.String() != expected {
		t.Error("actual", `"`+b.String()+`"`, `does not equal expected "`+expected+`"`)
		t.FailNow()
	}

	if !h.Add(hosts.Host{
		IP:        net.IPv4(127, 0, 0, 1),
		Hostnames: []string{"localhost.frantj.cc"},
	}) {
		t.Error("host add did not modify")
		t.FailNow()
	}

	if !h.Add(hosts.Host{
		IP:        net.IPv4(0, 0, 0, 0),
		Hostnames: []string{"homelab.frantj.cc"},
	}) {
		t.Error("host add did not modify")
		t.FailNow()
	}

	b = new(bytes.Buffer)
	if err = h.Encode(b); err != nil {
		t.Error(err)
		t.FailNow()
	}

	expected = "0.0.0.0 frantj.cc homelab.frantj.cc\n127.0.0.1 localhost.frantj.cc\n"
	if b.String() != expected {
		t.Error("actual", `"`+b.String()+`"`, `does not equal expected "`+expected+`"`)
		t.FailNow()
	}
}
