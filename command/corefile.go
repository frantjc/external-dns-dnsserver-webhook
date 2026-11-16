package command

import (
	_ "embed"
	"fmt"
	"strings"
)

// defaultDirectives are used in place of the --corefile directives when none
// are given.
//
//go:embed Corefile
var defaultDirectives string

type corefileOpts struct {
	readyPort, healthPort, metricsPort int
	hostsFile                          string
	// directives are injected into the managed server block as-is. CoreDNS
	// executes plugins in the order defined by its plugin.cfg, not the order
	// that they appear in the Corefile, so where they are injected doesn't
	// matter. directives must not repeat a plugin that's already managed.
	directives string
}

// corefile returns the contents of the Corefile for the webhook's DNS server.
// It manages only the plugins that the webhook depends on; everything else,
// e.g. how to resolve names that aren't in the hosts file, comes from
// opts.directives.
func corefile(opts corefileOpts) string {
	var directives strings.Builder
	for line := range strings.SplitSeq(strings.TrimSpace(opts.directives), "\n") {
		if line = strings.TrimRight(line, " \t\r"); line != "" {
			directives.WriteString("  ")
			directives.WriteString(line)
			directives.WriteString("\n")
		}
	}

	return fmt.Sprintf(
		`. {
  ready :%d
  health :%d {
    lameduck 5s
  }
  prometheus :%d
  hosts %s {
    fallthrough
  }
%s}
`,
		opts.readyPort,
		opts.healthPort,
		opts.metricsPort,
		opts.hostsFile,
		directives.String(),
	)
}
