---
title: Publish process
sidebar: documentation
permalink: documentation/reference/publish_process.html
author: Timofey Kirillov <timofey.kirillov@flant.com>
---

<!--Docker images should be pushed into the Docker registry for further usage in most cases. The usage includes these demands:-->

<!--1. Using an image to run an application (for example in Kubernetes). These images will be referred to as **images for running**.-->
<!--2. Using an existing old image version from a Docker registry as a cache to build a new image version. Usually, it is default behavior. However, some additional actions may be required to organize a build environment with multiple build hosts or build hosts with no persistent local storage. These images will be referred to as **distributed images cache**.-->

<!--## What can be published-->

<!--The result of werf [build commands]({{ site.baseurl }}/documentation/cli/build/build.html) is a _stages_ in _stages storage_ related to images defined in the `werf.yaml` config. -->
<!--werf can be used to publish either:-->

<!--* Images. These can only be used as _images for running_. -->
<!--These images are not suitable for _distributed images cache_, because werf build algorithm implies creating separate images for _stages_. -->
<!--When you pull a image from a Docker registry, you do not receive _stages_ for this image.-->
<!--* Images with a stages cache images. These images can be used as _images for running_ and also as a _distributed images cache_.-->

<!--werf pushes image into a Docker registry with a so-called [**image publish procedure**](#image-publish-procedure). Also, werf pushes stages cache of all images from config with a so-called [**stages publish procedure**](#stages-publish-procedure).-->

<!--Before digging into these algorithms, it is helpful to see how to publish images using Docker.-->

<!--### Stages publish procedure-->

<!--To publish stages cache of a image from the config werf implements the **stages publish procedure**. It consists of the following steps:-->

<!-- 1. Create temporary image names aliases for all docker images in stages cache, so that:-->
<!--     - [docker repository name](https://docs.docker.com/glossary/?term=repository) is a `REPO` parameter specified by the user without changes ([details about `REPO`]({{ site.baseurl }}/documentation/reference/registry/image_naming.html#repo-parameter)).-->
<!--     - [docker tag name](https://docs.docker.com/glossary/?term=tag) constructed as a signature prefixed with a word `image-stage-` (for example `image-stage-41772c141b158349804ad27b354247df8984ead077a5dd601f3940536ebe9a11`).-->
<!-- 2. Push images by newly created aliases into Docker registry.-->
<!-- 3. Delete temporary image names aliases.-->

<!--All of these steps are also performed with a single werf command, which will be described below.-->

<!--The result of this procedure is multiple images from stages cache of image pushed into the Docker registry.-->

## Image publishing procedure

Generally, the publishing process in the Docker ecosystem consists of the following steps:

```shell
docker tag REPO:TAG
docker push REPO:TAG
docker rmi REPO:TAG
```

 1. Getting a name or an id of the created local image.
 2. Creating a temporary alias-image for the above image that consists of two parts:
     - [docker repository name](https://docs.docker.com/glossary/?term=repository) with embedded Docker registry address;
     - [docker tag name](https://docs.docker.com/glossary/?term=tag).
 3. Pushing an alias into the Docker registry.
 4. Deleting a temporary alias.

To publish an [image]({{ site.baseurl }}/documentation/reference/stages_and_images.html#images) from the config, werf implements another logic:

 1. Create **a new image** based on the built image with the specified name and save the internal service information about tagging schema to this image (using docker labels). This information is referred to as an image **meta-information**. werf uses this information in the [deploying process]({{ site.baseurl }}/documentation/reference/deploy_process/deploy_into_kubernetes.html#integration-with-built-images) and the [cleaning process]({{ site.baseurl }}/documentation/reference/cleaning_process.html).
 2. Push the newly created image into the Docker registry.
 3. Delete the temporary image created in the first step.

This procedure will be referred to as the **image publishing procedure**.

The result of this procedure is an image named using the [*rules for naming images*](#naming-images) and pushed into the Docker registry. All these steps are performed with the [werf publish command]({{ site.baseurl }}/documentation/cli/main/publish.html) or the [werf build-and-publish command]({{ site.baseurl }}/documentation/cli/main/build_and_publish.html).

## Naming images

During the image publishing procedure, werf forms the image name using:
 * _images repo_ param;
 * _images repo mode_ param;
 * image name from the werf.yaml;
 * tag param.

The final name of the docker image has the form [`DOCKER_REPOSITORY`](https://docs.docker.com/glossary/?term=repository)`:`[`TAG`](https://docs.docker.com/engine/reference/commandline/tag).

The _images repo_ and _images repo mode_ params define where and how to store images.
If the werf project contains only one nameless image, then the _images repo_ is used as a docker repository as it is, and the resulting name of a docker image gets the following form: `IMAGES_REPO:TAG`.

Otherwise, werf constructs the resulting name of a docker image for every image depending on the _images repo mode_:
- `IMAGES_REPO:IMAGE_NAME-TAG` pattern for a `monorepo` mode;
- `IMAGES_REPO/IMAGE_NAME:TAG` pattern for a `multirepo` mode.

The _images repo_ param should be specified by the `--images-repo` option or `$WERF_IMAGES_REPO`.

The _images repo mode_ param should be specified by the `--images-repo-mode` option or `$WERF_IMAGES_REPO_MODE`.  The user can use different Docker registry implementation and some of them have restrictions and various default _image repo mode_ which can be based on implementation and _images_repo_ value (more about [supported Docker registry implementations and their features]({{ site.baseurl }}/documentation/reference/working_with_docker_registries.html)).

> The image naming behavior should be the same for publishing, deploying, and cleaning processes. Otherwise, the pipeline may fail, and you may end up losing images and stages during the cleanup.

The *docker tag* is taken from `--tag-*` params:

| option                       | description                                                                     |
| ---------------------------- | ------------------------------------------------------------------------------- |
| `--tag-git-tag TAG`          | Use git-tag tagging strategy and tag by the specified git tag                   |
| `--tag-git-branch BRANCH`    | Use git-branch tagging strategy and tag by the specified git branch             |
| `--tag-git-commit COMMIT`    | Use git-commit tagging strategy and tag by the specified git commit hash        |
| `--tag-custom TAG`           | Use custom tagging strategy and tag by the specified arbitrary tag              |
| `--tag-custom TAG`           | Use custom tagging strategy and tag by the specified arbitrary tag              |
| `--tag-by-stages-signature`  | Tag each image by image _stages signature_                                      |

All the specified tag params will be validated for the conformity with the tagging rules for docker images. User may explicitly apply the slug algorithm to the tag value using `werf slugify` command, learn [more about the slug]({{ site.baseurl }}/documentation/reference/toolbox/slug.html).

Also, user specifies both the tag value and the tagging strategy by using `--tag-*` options.
The tagging strategy affects [certain policies in the cleaning process]({{ site.baseurl }}/documentation/reference/cleaning_process.html#cleanup-policies).

Every `--tag-git-*` option requires a `TAG`, `BRANCH`, or `COMMIT` argument. These options are designed to be compatible with modern CI/CD systems, where a CI job is running in the detached git worktree for the specific commit, and the current git-tag, git-branch, or git-commit is passed to the job using environment variables (for example `CI_COMMIT_TAG`, `CI_COMMIT_REF_NAME` and `CI_COMMIT_SHA` for the GitLab CI).

`--tag-by-stages-signature=true` option enables content based tagging, which is preferred method of tagging images by the werf.

### Content based tagging

Werf v1.1 supports so called content based tagging. Tags of resulting docker images depend on the content of these images.

When using `werf publish --tags-by-stages-signature` or `werf ci-env --tagging-strategy=stages-signature` werf will tag result images by so called image stages signature. Each image tagged by own stages signature which calculated by the same rules as regular signature of image stage.

Image _stages signature_ depends on content of the image and depends on the git history which lead to this content.

There may be *dummy commits* into the git repo that do not change resulting images. For example empty commits, merge commits or commits which change files that are not imported into the resulting image.

When using tagging by git-commits these *dummy commits* will cause werf to create new images names even if content of these images is the same. New images names in turn will cause restarts of application Pods in Kubernetes which is totally not a desired behaviour. All in all this is the reason preventing storing multiple application services in the single git repo.

_Stages signature_ on the countrary will not change on *dummy commits*, so these commits will not cause restarts of application Pods in kubernetes, yet it similarly to commit-id relates to the git history of edits and depends on the content of the files.

Also tagging by stages signatures is more realiable tagging method than tagging by a git-branch for example, because resulting images content does not depend on order of pipelines execution. Stages signature leads to stable immutable images names which represent the address of the certain image content.

Note that image name generation template [`werf_container_image`]({{ site.baseurl }}/documentation/reference/deploy_process/deploy_into_kubernetes.html#werf_container_image) should be used in the deploy configs to generate an image name with a correct docker tags.

Stages-signature is the default tagging strategy and the only recommended one for usage. Tagging strategies are also explained in the [plugging into CI/CD articles]({{ site.baseurl }}/documentation/reference/plugging_into_cicd/overview.html#ci-env-tagging-modes).

### Combining parameters

Any combination of tagging parameters can be used simultaneously in the [werf publish command]({{ site.baseurl }}/documentation/cli/main/publish.html) or [werf build-and-publish command]({{ site.baseurl }}/documentation/cli/main/build_and_publish.html). As a result, werf will publish a separate image for each tagging parameter of every image in a project.

## Examples

### Tagging images by a stages signature

Let's suppose `werf.yaml` defines two images: `backend` and `frontend`.

The following command:

```
werf publish --stages-storage :local --images-repo registry.hello.com/web/core/system --tag-by-stages-signature
```

may produce the following images names, respectively:
 * `registry.hello.com/web/core/system/backend:4ef339f84ca22247f01fb335bb19f46c4434014d8daa3d5d6f0e386d`;
 * `registry.hello.com/web/core/system/frontend:f44206457e0a4c8a54655543f749799d10a9fe945896dab1c16996c6`.

where `4ef339f84ca22247f01fb335bb19f46c4434014d8daa3d5d6f0e386d` is the stages signature of image `backend` and
`f44206457e0a4c8a54655543f749799d10a9fe945896dab1c16996c6` is the stages signature of image `frontend`.

These tags depend on image content and git history which lead to this content. Each of these signature may be changed
when content of the image changes so user need to update kubernetes manifests accordingly.

### Linking images to a git tag

Let's suppose `werf.yaml` defines two images: `backend` and `frontend`.

The following command:

```shell
werf publish --stages-storage :local --images-repo registry.hello.com/web/core/system --tag-git-tag v1.2.0
```

produces the following image names, respectively:
 * `registry.hello.com/web/core/system/backend:v1.2.0`;
 * `registry.hello.com/web/core/system/frontend:v1.2.0`.

### Linking images to a git branch

Let's suppose `werf.yaml` defines two images: `backend` and `frontend`.

The following command:

```shell
werf publish --stages-storage :local --images-repo registry.hello.com/web/core/system --tag-git-branch my-feature-x
```

produces the following image names, respectively:
 * `registry.hello.com/web/core/system/backend:my-feature-x`;
 * `registry.hello.com/web/core/system/frontend:my-feature-x`.

### Linking images to a git branch with special characters in the name

Once again, we have a `werf.yaml` file with two defined images: `backend` and `frontend`.

The following command:

```shell
werf publish --stages-storage :local --images-repo registry.hello.com/web/core/system --tag-git-branch $(werf slugify --format docker-tag "Features/MyFeature#169")
```

produces the following image names, respectively:
 * `registry.hello.com/web/core/system/backend:features-myfeature169-3167bc8c`;
 * `registry.hello.com/web/core/system/frontend:features-myfeature169-3167bc8c`.

Note that the [`werf slugify`]({{ site.baseurl }}/documentation/cli/toolbox/slugify.html) command generates a valid docker tag. Learn [more about the slug]({{ site.baseurl }}/documentation/reference/toolbox/slug.html).

### Content based tagging in a GitLab CI job

Let's say we have a `werf.yaml` configuration file that defines two images, `backend` and `frontend`.

Running the following command in a GitLab CI job (in some git-branch or tag — irrelevant) for a project named `web/core/system` and the Docker registry configured as `registry.hello.com/web/core/system`:

```shell
type werf && source <(werf ci-env gitlab)
werf publish --stages-storage :local
```

produces the following image names, respectively:
 * `registry.hello.com/web/core/system/backend:4ef339f84ca22247f01fb335bb19f46c4434014d8daa3d5d6f0e386d`;
 * `registry.hello.com/web/core/system/frontend:f44206457e0a4c8a54655543f749799d10a9fe945896dab1c16996c6`.

where `4ef339f84ca22247f01fb335bb19f46c4434014d8daa3d5d6f0e386d` is the stages signature of image `backend` and
`f44206457e0a4c8a54655543f749799d10a9fe945896dab1c16996c6` is the stages signature of image `frontend`.

We omitted `--tagging-strategy=stages-signature` option which is default.

These tags depend on image content and git history which lead to this content. Each of these signatures may be changed
when content of the image changes so user need to update kubernetes manifests accordingly.

### Linking images to a GitLab CI job

Let's say we have a `werf.yaml` configuration file that defines two images, `backend` and `frontend`.

Running the following command in a GitLab CI job for a project named `web/core/system` with the git branch set as `core/feature/ADD_SETTINGS` and the Docker registry configured as `registry.hello.com/web/core/system`:

```shell
type werf && source <(werf ci-env gitlab --tagging-strategy tag-or-branch --verbose)
werf publish --stages-storage :local
```

yields the following image names:
 * `registry.hello.com/web/core/system/backend:core-feature-add-settings-df80fdc3`;
 * `registry.hello.com/web/core/system/frontend:core-feature-add-settings-df80fdc3`.

Note that werf automatically applies slug to the resulting tag of the docker image: `core/feature/ADD_SETTINGS` is converted to `core-feature-add-settings-df80fdc3`. This conversion occurs in the `werf ci-env` command, which determines the name of a git branch from the GitLab CI environment, automatically slugs it and sets `WERF_TAG_GIT_BRANCH` (which is alternative way to set the `--tag-git-branch` parameter). See [more about the slug]({{ site.baseurl }}/documentation/reference/toolbox/slug.html).

### Unnamed image and a GitLab CI job

Let's suppose we have a werf.yaml with a single unnamed image. Running the following command in the GitLab CI job for the project named `web/core/queue` with the git-tag named `v2.3.1` and a Docker registry configured as `registry.hello.com/web/core/queue`:

```shell
type werf && source <(werf ci-env gitlab --tagging-strategy tag-or-branch --verbose)
werf publish --stages-storage :local
```

yields the following result: `registry.hello.com/web/core/queue:v2.3.1`.
