package docker_registry

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/werf/werf/pkg/docker"
	"github.com/werf/werf/pkg/image"
)

type genericApi struct {
	commonApi *api
	mirrors   []string
}

func newGenericApi(ctx context.Context, options apiOptions) (*genericApi, error) {
	d := &genericApi{}
	d.commonApi = newAPI(options)

	// init registry mirrors if docker cli initialized in context
	if docker.IsContext(ctx) {
		info, err := docker.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to get docker system info: %s", err)
		}

		if info.RegistryConfig != nil {
			d.mirrors = info.RegistryConfig.Mirrors
		}
	}

	return d, nil
}

func (api *genericApi) GetRepoImageConfigFile(ctx context.Context, reference string) (*v1.ConfigFile, error) {
	mirrorReferenceList, err := api.mirrorReferenceList(reference)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare mirror reference list: %s", err)
	}

	for _, mirrorReference := range mirrorReferenceList {
		config, err := api.getRepoImageConfigFile(ctx, mirrorReference)
		if err != nil {
			if IsBlobUnknownError(err) || IsManifestUnknownError(err) || IsNameUnknownError(err) {
				continue
			}

			return nil, err
		}

		return config, nil
	}

	return api.getRepoImageConfigFile(ctx, reference)
}

func (api *genericApi) getRepoImageConfigFile(_ context.Context, reference string) (*v1.ConfigFile, error) {
	imageInfo, _, err := api.commonApi.image(reference)
	if err != nil {
		return nil, err
	}

	return imageInfo.ConfigFile()
}

func (api *genericApi) GetRepoImage(ctx context.Context, reference string) (*image.Info, error) {
	mirrorReferenceList, err := api.mirrorReferenceList(reference)
	if err != nil {
		return nil, fmt.Errorf("unable to prepare mirror reference list: %s", err)
	}

	for _, mirrorReference := range mirrorReferenceList {
		info, err := api.commonApi.TryGetRepoImage(ctx, mirrorReference)
		if err != nil {
			return nil, fmt.Errorf("unable to try getting mirror repo image %q: %s", mirrorReference, err)
		}

		if info != nil {
			return info, nil
		}
	}

	return api.commonApi.GetRepoImage(ctx, reference)
}

func (api *genericApi) mirrorReferenceList(reference string) ([]string, error) {
	var referenceList []string

	referenceParts, err := api.commonApi.ParseReferenceParts(reference)
	if err != nil {
		return nil, err
	}

	// nothing if container registry is not Docker Hub
	if referenceParts.registry != name.DefaultRegistry {
		return nil, nil
	}

	for _, mirrorRegistry := range api.mirrors {
		mirrorRegistryUrl, err := url.Parse(mirrorRegistry)
		if err != nil {
			return nil, fmt.Errorf("unable to parse mirror registry url %q: %s", mirrorRegistry, err)
		}

		mirrorReference := mirrorRegistryUrl.Host
		mirrorReference += "/" + referenceParts.repository
		mirrorReference += ":" + referenceParts.tag

		if referenceParts.digest != "" {
			mirrorReference += "@" + referenceParts.digest
		}

		referenceList = append(referenceList, mirrorReference)
	}

	return referenceList, nil
}
