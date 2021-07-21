package docker_registry

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	dockerReference "github.com/docker/distribution/reference"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/werf/logboek"

	"github.com/werf/werf/pkg/docker_registry/container_registry_extensions"
	"github.com/werf/werf/pkg/image"
)

type api struct {
	InsecureRegistry      bool
	SkipTlsVerifyRegistry bool
}

type apiOptions struct {
	InsecureRegistry      bool
	SkipTlsVerifyRegistry bool
}

func newAPI(options apiOptions) *api {
	return &api{
		InsecureRegistry:      options.InsecureRegistry,
		SkipTlsVerifyRegistry: options.SkipTlsVerifyRegistry,
	}
}

func (api *api) Tags(_ context.Context, reference string) ([]string, error) {
	tags, err := api.list(reference)
	if err != nil {
		if IsNameUnknownError(err) {
			return []string{}, nil
		}
		return nil, err
	}

	return tags, nil
}

func (api *api) IsRepoImageExists(ctx context.Context, reference string) (bool, error) {
	if imgInfo, err := api.TryGetRepoImage(ctx, reference); err != nil {
		return false, err
	} else {
		return imgInfo != nil, nil
	}
}

func (api *api) TryGetRepoImage(ctx context.Context, reference string) (*image.Info, error) {
	if imgInfo, err := api.GetRepoImage(ctx, reference); err != nil {
		if IsBlobUnknownError(err) || IsManifestUnknownError(err) || IsNameUnknownError(err) {
			// TODO: 1. auto reject images with manifest-unknown or blob-unknown errors
			// TODO: 2. why TryGetRepoImage for rejected image records gives manifest-unknown errors?
			// TODO: 3. make sure werf never ever creates rejected image records for name-unknown errors.
			// TODO: 4. werf-cleanup should remove broken images

			if os.Getenv("WERF_DOCKER_REGISTRY_DEBUG") == "1" {
				logboek.Context(ctx).Error().LogF("WARNING: Got an error when inspecting repo image %q: %s\n", reference, err)
			}

			return nil, nil
		}

		return imgInfo, err
	} else {
		return imgInfo, nil
	}
}

func (api *api) GetRepoImage(_ context.Context, reference string) (*image.Info, error) {
	imageInfo, _, err := api.image(reference)
	if err != nil {
		return nil, err
	}

	digest, err := imageInfo.Digest()
	if err != nil {
		return nil, err
	}

	manifest, err := imageInfo.Manifest()
	if err != nil {
		return nil, err
	}

	configFile, err := imageInfo.ConfigFile()
	if err != nil {
		return nil, err
	}

	var totalSize int64
	if layers, err := imageInfo.Layers(); err != nil {
		return nil, err
	} else {
		for _, l := range layers {
			if lSize, err := l.Size(); err != nil {
				return nil, err
			} else {
				totalSize += lSize
			}
		}
	}

	referenceParts, err := api.ParseReferenceParts(reference)
	if err != nil {
		return nil, fmt.Errorf("unable to parse reference %q: %s", reference, err)
	}

	repoImage := &image.Info{
		Name:       reference,
		Repository: strings.Join([]string{referenceParts.registry, referenceParts.repository}, "/"),
		ID:         manifest.Config.Digest.String(),
		Tag:        referenceParts.tag,
		RepoDigest: digest.String(),
		ParentID:   configFile.Config.Image,
		Labels:     configFile.Config.Labels,
		Size:       totalSize,
	}

	repoImage.SetCreatedAtUnix(configFile.Created.Unix())

	return repoImage, nil
}

func (api *api) list(reference string) ([]string, error) {
	repo, err := name.NewRepository(reference, api.newRepositoryOptions()...)
	if err != nil {
		return nil, fmt.Errorf("parsing repo %q: %v", reference, err)
	}

	tags, err := remote.List(repo, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithTransport(api.getHttpTransport()))
	if err != nil {
		return nil, fmt.Errorf("reading tags for %q: %v", repo, err)
	}

	return tags, nil
}

func (api *api) deleteImageByReference(reference string) error {
	r, err := name.ParseReference(reference, api.parseReferenceOptions()...)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", reference, err)
	}

	if err := remote.Delete(r, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithTransport(api.getHttpTransport())); err != nil {
		return fmt.Errorf("deleting image %q: %v", r, err)
	}

	return nil
}

func (api *api) PushImage(ctx context.Context, reference string, opts *PushImageOptions) error {
	retriesLimit := 5

attemptLoop:
	for attempt := 1; attempt <= retriesLimit; attempt++ {
		if err := api.pushImage(ctx, reference, opts); err != nil {
			for _, substr := range []string{
				"REDACTED: UNKNOWN",
				"http2: server sent GOAWAY and closed the connection",
				"http2: Transport received Server's graceful shutdown GOAWAY",
			} {
				if strings.Contains(err.Error(), substr) {
					seconds := rand.Intn(5) + 1

					msg := fmt.Sprintf("Retrying publishing in %d seconds (%d/%d) ...\n", seconds, attempt, retriesLimit)
					logboek.Context(ctx).Warn().LogLn(msg)

					time.Sleep(time.Duration(seconds) * time.Second)
					continue attemptLoop
				}
			}

			return err
		}

		return nil
	}

	return nil
}

func (api *api) pushImage(_ context.Context, reference string, opts *PushImageOptions) error {
	ref, err := name.ParseReference(reference, api.parseReferenceOptions()...)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", reference, err)
	}

	labels := map[string]string{}
	if opts != nil {
		labels = opts.Labels
	}

	img := container_registry_extensions.NewManifestOnlyImage(labels)

	oldDefaultTransport := http.DefaultTransport
	http.DefaultTransport = api.getHttpTransport()
	err = remote.Write(ref, img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	http.DefaultTransport = oldDefaultTransport

	if err != nil {
		return fmt.Errorf("write to the remote %s have failed: %s", ref.String(), err)
	}

	return nil
}

func (api *api) image(reference string) (v1.Image, name.Reference, error) {
	ref, err := name.ParseReference(reference, api.parseReferenceOptions()...)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing reference %q: %v", reference, err)
	}

	// FIXME: Hack for the go-containerregistry library,
	// FIXME: that uses default transport without options to change transport to custom.
	// FIXME: Needed for the insecure https registry to work.
	oldDefaultTransport := http.DefaultTransport
	http.DefaultTransport = api.getHttpTransport()
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	http.DefaultTransport = oldDefaultTransport

	if err != nil {
		return nil, nil, fmt.Errorf("reading image %q: %v", ref, err)
	}

	return img, ref, nil
}

func (api *api) newRepositoryOptions() []name.Option {
	return api.parseReferenceOptions()
}

func (api *api) parseReferenceOptions() []name.Option {
	var options []name.Option
	options = append(options, name.WeakValidation)

	if api.InsecureRegistry {
		options = append(options, name.Insecure)
	}

	return options
}

func (api *api) getHttpTransport() (transport http.RoundTripper) {
	transport = http.DefaultTransport

	if api.SkipTlsVerifyRegistry {
		defaultTransport := http.DefaultTransport.(*http.Transport)

		newTransport := &http.Transport{
			Proxy:                 defaultTransport.Proxy,
			DialContext:           defaultTransport.DialContext,
			MaxIdleConns:          defaultTransport.MaxIdleConns,
			IdleConnTimeout:       defaultTransport.IdleConnTimeout,
			TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
			ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		}

		transport = newTransport
	}

	return
}

type referenceParts struct {
	registry   string
	repository string
	tag        string
	digest     string
}

func (api *api) ParseReferenceParts(reference string) (referenceParts, error) {
	// validate reference
	parsedReference, err := name.ParseReference(reference, api.parseReferenceOptions()...)
	if err != nil {
		return referenceParts{}, fmt.Errorf("unable to parse reference: %s", err)
	}

	res := dockerReference.ReferenceRegexp.FindStringSubmatch(reference)

	// res[0] full match
	// res[1] repository
	// res[2] tag
	// res[3] digest
	if len(res) != 4 {
		panic(fmt.Sprintf("unexpected regexp find submatch result %v (%d)", res, len(res)))
	}

	referenceParts := referenceParts{}
	referenceParts.registry = parsedReference.Context().RegistryStr()
	referenceParts.repository = parsedReference.Context().RepositoryStr()
	referenceParts.tag = res[2]
	if referenceParts.tag == "" {
		referenceParts.tag = name.DefaultTag
	}
	referenceParts.digest = res[3]

	return referenceParts, nil
}
