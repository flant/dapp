package docker_registry

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/google/go-containerregistry/pkg/logs"

	"github.com/werf/logboek"
)

var generic *genericApi

func Init(ctx context.Context, insecureRegistry, skipTlsVerifyRegistry bool) error {
	if logboek.Context(ctx).Debug().IsAccepted() {
		logs.Progress.SetOutput(logboek.Context(ctx).OutStream())
		logs.Warn.SetOutput(logboek.Context(ctx).ErrStream())

		if debugDockerRegistryAPI() {
			logs.Debug.SetOutput(logboek.Context(ctx).OutStream())
		} else {
			logs.Debug.SetOutput(ioutil.Discard)
		}
	} else {
		logs.Progress.SetOutput(ioutil.Discard)
		logs.Warn.SetOutput(ioutil.Discard)
		logs.Debug.SetOutput(ioutil.Discard)
	}

	var err error
	generic, err = newGenericApi(ctx, apiOptions{
		InsecureRegistry:      insecureRegistry,
		SkipTlsVerifyRegistry: skipTlsVerifyRegistry,
	})

	if err != nil {
		return err
	}

	return nil
}

func API() *genericApi {
	return generic
}

func debugDockerRegistryAPI() bool {
	return os.Getenv("WERF_DEBUG_DOCKER_REGISTRY_API") == "1"
}
