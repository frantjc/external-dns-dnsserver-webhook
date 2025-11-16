// A generated module for ExternalDnsDnsserverWebhook functions

package main

import (
	"context"
	"dagger/external-dns-dnsserver-webhook/internal/dagger"
	"fmt"
	"strings"
)

type ExternalDnsDnsserverWebhook struct {
	Source *dagger.Directory
}

func New(
	// +optional
	// +defaultPath="."
	src *dagger.Directory,
) (*ExternalDnsDnsserverWebhook, error) {
	return &ExternalDnsDnsserverWebhook{
		Source: src,
	}, nil
}

func (m *ExternalDnsDnsserverWebhook) Fmt(
	ctx context.Context,
	// +optional
	check bool,
) (*dagger.Changeset, error) {
	goModules := []string{".dagger/"}

	root := dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: goModules,
		}),
	}).
		Container().
		WithExec([]string{"go", "fmt", "./..."}).
		Directory(".")

	for _, module := range goModules {
		root = root.WithDirectory(
			module,
			dag.Go(dagger.GoOpts{
				Module: m.Source.Directory(module),
			}).
				Container().
				WithExec([]string{"go", "fmt", "./..."}).
				Directory("."),
		)
	}

	changeset := root.Changes(m.Source)

	if check {
		if empty, err := changeset.IsEmpty(ctx); err != nil {
			return nil, err
		} else if !empty {
			return nil, fmt.Errorf("source is not formatted")
		}
	}

	return changeset, nil
}

func (m *ExternalDnsDnsserverWebhook) Test(ctx context.Context) *dagger.Container {
	return dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: []string{".dagger/"},
		}),
	}).
		Container().
		WithExec([]string{"go", "test", "-race", "-cover", "./..."})
}

const (
	gid   = "1001"
	uid   = gid
	group = "webhook"
	user  = group
	owner = user + ":" + group
	home  = "/home/" + user
)

func (m *ExternalDnsDnsserverWebhook) Container(ctx context.Context) *dagger.Container {
	return dag.Wolfi().
		Container().
		WithExec([]string{"addgroup", "-S", "-g", gid, group}).
		WithExec([]string{"adduser", "-S", "-G", group, "-u", uid, user}).
		WithEnvVariable("PATH", home+"/.local/bin:$PATH", dagger.ContainerWithEnvVariableOpts{Expand: true}).
		WithFile(
			home+"/.local/bin/webhook", m.Binary(ctx),
			dagger.ContainerWithFileOpts{Expand: true, Owner: owner, Permissions: 0700}).
		WithExec([]string{"chown", "-R", owner, home}).
		WithUser(user).
		WithEntrypoint([]string{"sindri"})
}

func (m *ExternalDnsDnsserverWebhook) Version(ctx context.Context) string {
	version := "v0.0.0-unknown"

	ref, err := m.Source.AsGit().LatestVersion().Ref(ctx)
	if err == nil {
		version = strings.TrimPrefix(ref, "refs/tags/")
	}

	if empty, _ := m.Source.AsGit().Uncommitted().IsEmpty(ctx); !empty {
		version += "*"
	}

	return version
}

func (m *ExternalDnsDnsserverWebhook) Tag(ctx context.Context) string {
	return strings.TrimSuffix(strings.TrimPrefix(m.Version(ctx), "v"), "*")
}

func (m *ExternalDnsDnsserverWebhook) Binary(ctx context.Context) *dagger.File {
	return dag.Go(dagger.GoOpts{
		Module: m.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: []string{".github/", "e2e/"},
		}),
	}).
		Build(dagger.GoBuildOpts{
			Pkg:     "./cmd/webhook",
			Ldflags: "-s -w -X main.version=" + m.Version(ctx),
		})
}
