package image

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"

	"github.com/golang/example/stringutil"
)

type Info struct {
	Name              string            `json:"name"`
	Repository        string            `json:"repository"`
	Tag               string            `json:"tag"`
	RepoDigest        string            `json:"repoDigest"`
	ID                string            `json:"ID"`
	ParentID          string            `json:"parentID"`
	Labels            map[string]string `json:"labels"`
	Size              int64             `json:"size"`
	CreatedAtUnixNano int64             `json:"createdAtUnixNano"`
}

func (info *Info) SetCreatedAtUnix(seconds int64) {
	info.CreatedAtUnixNano = seconds * 1000_000_000
}

func (info *Info) SetCreatedAtUnixNano(seconds int64) {
	info.CreatedAtUnixNano = seconds
}

func (info *Info) GetCreatedAt() time.Time {
	return time.Unix(info.CreatedAtUnixNano/1000_000_000, info.CreatedAtUnixNano%1000_000_000)
}

func NewInfoFromInspect(ref string, inspect *types.ImageInspect) *Info {
	var repository, tag string
	parts := strings.SplitN(stringutil.Reverse(ref), ":", 2)
	if len(parts) == 2 {
		repository = stringutil.Reverse(parts[0])
		tag = stringutil.Reverse(parts[1])
	}

	return &Info{
		Name:              ref,
		Repository:        repository,
		Tag:               tag,
		Labels:            inspect.Config.Labels,
		CreatedAtUnixNano: mustParseTimestampString(inspect.Created).UnixNano(),
		ID:                inspect.ID,
		ParentID:          inspect.Parent,
		Size:              inspect.Size,
	}
}

func mustParseTimestampString(timestampString string) time.Time {
	t, err := time.Parse(time.RFC3339, timestampString)
	if err != nil {
		panic(fmt.Sprintf("got bad timestamp %q: %s", timestampString, err))
	}
	return t
}
