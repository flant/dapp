---
title: Stages and Images
sidebar: documentation
permalink: documentation/reference/stages_and_images.html
author: Alexey Igrychev <alexey.igrychev@flant.com>
---

We propose to divide the assembly process into steps. Every step corresponds to the intermediate image (like layers in Docker) with specific functions and assignments.
In werf, we call every such step a [stage](#stages). So the final [image](#images) consists of a set of built stages.
All stages are kept in a [stages storage](#stages-storage). You can view it as a building cache of an application, however, that isn't a cache but merely a part of a building context.

## Stages

Stages are steps in the assembly process. They act as building blocks for constructing images.
A ***stage*** is built from a logically grouped set of config instructions. It takes into account the assembly conditions and rules.
Each _stage_ relates to a single Docker image.

The werf assembly process involves a sequential build of stages using the _stage conveyor_.  A _stage conveyor_ is an ordered sequence of conditions and rules for carrying out stages. werf uses different _stage conveyors_ to assemble various types of images depending on their configuration.

<div class="tabs">
  <a href="javascript:void(0)" class="tabs__btn active" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'dockerfile-image-tab')">Dockerfile Image</a>
  <a href="javascript:void(0)" class="tabs__btn" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'stapel-image-tab')">Stapel Image</a>
  <a href="javascript:void(0)" class="tabs__btn" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'stapel-artifact-tab')">Stapel Artifact</a>
</div>

<div id="dockerfile-image-tab" class="tabs__content active">
<a class="google-drawings" href="https://docs.google.com/drawings/d/e/2PACX-1vRrzxht-PmC-4NKq95DtLS9E7JrvtuHy0JpMKdylzlZtEZ5m7bJwEMJ6rXTLevFosWZXmi9t3rDVaPB/pub?w=2031&amp;h=144" data-featherlight="image">
<img src="https://docs.google.com/drawings/d/e/2PACX-1vRrzxht-PmC-4NKq95DtLS9E7JrvtuHy0JpMKdylzlZtEZ5m7bJwEMJ6rXTLevFosWZXmi9t3rDVaPB/pub?w=821&amp;h=59">
</a>
</div>

<div id="stapel-image-tab" class="tabs__content">
<a class="google-drawings" href="https://docs.google.com/drawings/d/e/2PACX-1vRKB-_Re-ZhkUSB45jF9GcM-3gnE2snMjTOEIQZSyXUniNHKK-eCQl8jw3tHFF-a6JLAr2sV73lGAdw/pub?w=2000&amp;h=881" data-featherlight="image">
<img src="https://docs.google.com/drawings/d/e/2PACX-1vRKB-_Re-ZhkUSB45jF9GcM-3gnE2snMjTOEIQZSyXUniNHKK-eCQl8jw3tHFF-a6JLAr2sV73lGAdw/pub?w=821&amp;h=362" >
</a>
</div>

<div id="stapel-artifact-tab" class="tabs__content">
<a class="google-drawings" href="https://docs.google.com/drawings/d/e/2PACX-1vRD-K_z7KEoliEVT4GpTekCkeaFMbSPWZpZkyTDms4XLeJAWEnnj4EeAxsdwnU3OtSW_vuKxDaaFLgD/pub?w=1800&amp;h=850" data-featherlight="image">
<img src="https://docs.google.com/drawings/d/e/2PACX-1vRD-K_z7KEoliEVT4GpTekCkeaFMbSPWZpZkyTDms4XLeJAWEnnj4EeAxsdwnU3OtSW_vuKxDaaFLgD/pub?w=640&amp;h=301">
</a>
</div>

**The user only needs to write a correct configuration: werf performs the rest of the work with stages**

For each _stage_ at every build, werf calculates the unique identifier of the stage called _stage signature_.
Each _stage_ is assembled in the ***assembly container*** that is based on the previous _stage_ and saved in the [stages storage](#stages-storage).
The _stage signature_ is used for [tagging](#stage-naming) a _stage_ (signature is the part of image tag) in the _stages storage_.
werf does not build stages that already exist in the _stages storage_ (similar to caching in Docker yet more complex).

The ***stage signature*** is calculated as the checksum of:
 - checksum of [stage dependencies]({{ site.baseurl }}/documentation/reference/stages_and_images.html#stage-dependencies);
 - previous _stage signature_;
 - git commit-id related with the previous stage (if previous stage is git-related).

Signature identifier of the stage represents content of the stage and depends on git history which lead to this content. There may be multiple built images for a single signature. Stage for different git branches can have the same signature, but werf will prevent cache of different git branches from
being reused for totally different branches, [see stage selection algorithm]({{ site.baseurl }}/documentation/reference/stages_and_images.html#stage-selection).

It means that the _stage conveyor_ can be reduced to several _stages_ or even to a single _from_ stage.

<a class="google-drawings" href="https://docs.google.com/drawings/d/e/2PACX-1vR6qxP5dbQNlHXik0jCvEcKZS2gKbdNmbFa8XIem8pixSHSGvmL1n7rpuuQv64YWl48wLXfpwbLQEG_/pub?w=572&amp;h=577" data-featherlight="image">
<img src="https://docs.google.com/drawings/d/e/2PACX-1vR6qxP5dbQNlHXik0jCvEcKZS2gKbdNmbFa8XIem8pixSHSGvmL1n7rpuuQv64YWl48wLXfpwbLQEG_/pub?w=286&amp;h=288">
</a>

## Stage dependencies

_Stage dependency_ is a piece of data that affects the stage _signature_. Stage dependency may be represented by:

 - some file from a git repo with its contents;
 - instructions to build stage defined in the `werf.yaml`;
 - the arbitrary string specified by the user in the `werf.yaml`;
 - and so on.

Most _stage dependencies_ are specified in the `werf.yaml`, others relate to a runtime.

The tables below illustrate dependencies of a Dockerfile image, a Stapel image, and a [Stapel artifact]({{ site.baseurl }}/documentation/configuration/stapel_artifact.html) _stages dependencies_.
Each row describes dependencies for a certain stage.
Left column contains a short description of dependencies, right column includes related `werf.yaml` directives and contains relevant references for more information.

<div class="tabs">
  <a href="javascript:void(0)" id="image-from-dockerfile-dependencies" class="tabs__btn dependencies-btn active">Dockerfile Image</a>
  <a href="javascript:void(0)" id="image-dependencies" class="tabs__btn dependencies-btn">Stapel Image</a>
  <a href="javascript:void(0)" id="artifact-dependencies" class="tabs__btn dependencies-btn">Stapel Artifact</a>
</div>

<div id="dependencies">
{% for stage in site.data.stages.entries %}
<div class="stage {{stage.type}}">
  <div class="stage-body">
    <div class="stage-base">
      <p>stage {{ stage.name | escape }}</p>

      {% if stage.dependencies %}
      <div class="dependencies">
        {% for dependency in stage.dependencies %}
        <div class="dependency">
          {{ dependency | escape }}
        </div>
        {% endfor %}
      </div>
      {% endif %}
    </div>

<div class="werf-config" markdown="1">

{% if stage.werf_config %}
```yaml
{{ stage.werf_config }}
```
{% endif %}

{% if stage.references %}
<div class="references">
    References:
    <ul>
    {% for reference in stage.references %}
        <li><a href="{{ reference.link }}">{{ reference.name }}</a></li>
    {% endfor %}
    </ul>
</div>
{% endif %}

</div>

    </div>
</div>
{% endfor %}
</div>

{% asset stages.css %}
<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/3.4.1/jquery.min.js"></script>
<script>
function application() {
  if ($("a[id=image-from-dockerfile-dependencies]").hasClass('active')) {
    $(".image").addClass('hidden');
    $(".artifact").addClass('hidden');
    $(".image-from-dockerfile").removeClass('hidden')
  }
  else if ($("a[id=image-dependencies]").hasClass('active')) {
    $(".image-from-dockerfile").addClass('hidden');
    $(".artifact").addClass('hidden');
    $(".image").removeClass('hidden')
  }
  else if ($("a[id=artifact-dependencies]").hasClass('active')) {
    $(".image-from-dockerfile").addClass('hidden');
    $(".image").addClass('hidden');
    $(".artifact").removeClass('hidden')
  }
  else {
    $(".image-from-dockerfile").addClass('hidden');
    $(".image").addClass('hidden');
    $(".artifact").addClass('hidden')
  }
}

$('.tabs').on('click', '.dependencies-btn', function() {
  $(this).toggleClass('active').siblings().removeClass('active');
  application()
});

application();
$.noConflict();
</script>

## Stages storage

The _stages storage_ contains the stages of the project.
Stages can be stored in the Docker Repo or locally on a host machine.

Most commands use _stages_ and require the reference to a specific _stages storage_, defined by the `--stages-storage` option or `WERF_STAGES_STORAGE` environment variable.
At the moment, only the local storage, `:local`, is supported.

### Stage naming

_Stages_ in the _stages storage_ are named using the following schema — `werf-stages-storage/PROJECT_NAME:SIGNATURE-TIMESTAMP_MILLISEC`.

Signature identifier of the stage represents content of the stage and depends on git history which lead to this content.

`TIMESTAMP_MILLISEC` is generated during [stage saving procedure](#stage-building-and-saving) after stage built.

### Stage selection

Werf stage selection algorithm is based on the git commits ancestry detection:

 1. Werf calculates a stage signature for some stage.
 2. There may be multiple stages in the stages storage by this signature, werf selects all suitable stages by the signature.
 3. If current stage is related to git (git-archive, user stage with git patch or git latest patch), then werf selects only
    those stages which are relaed to the commit that is ancestor of current git commit.
 4. Select from the remaining stages the _oldest_ by the creation timestamp.

There may be multiple built images for a single signature. Stage for different git branches can have the same signature, but werf will prevent cache of different git branches from being reused for totally different branch.

### Stage building and saving

If suitable stage has not been found by target signature during stage selection, werf starts building a new image for stage.

Note that multiple processes (on a single or multiple hosts) may start building the same stage at the same time. Werf uses optimistic locking when saving newly built image into the stages storage: when a new stage has been built werf locks stages storage and saves newly built stage image into storage stages cache only if there are no suitable already existing stages exists. Newly saved image will have a guaranteed unique identifier `TIMESTAMP_MILLISEC`. In the case when already existing stage has been found in the stages storage werf will discard newly built image and use already existing one as a cache.

In other words: the first process which finishes the build (the fastest one) will have a chance to save newly built stage into the stages storage. The slow build process will not block faster processes from saving build results and building next stages.

To select stages and save new ones into the stages storage werf uses [synchronization lock manager](#synchronization-lock-manager) to coordinate multiple werf processes.

### Image stages signature

_Stages signature_ of the image is a signature which represents content of the image and depends on the history of git commits which lead to this content.

***Stages signature*** calculated similarly to the regular stage signature as the checksum of:
 - _stage signature_ of last non empty image stage;
 - git commit-id related with the last non empty image stage (if this last stage is git-related).

The ***stage signature*** is calculated as the checksum of:
 - checksum of [stage dependencies]({{ site.baseurl }}/documentation/reference/stages_and_images.html#stage-dependencies);
 - previous _stage signature_;
 - git commit-id related with the previous stage (if previous stage is git-related).

This signature used in [content based tagging]({{ site.baseurl }}/documentation/reference/publish_process.html#content-based-tagging) and used to import files from artifacts or images (stages signature of artifact or image will affect imports stage signature of the target image).

## Images

_Image_ is a **ready-to-use** Docker image corresponding to a specific application state and [tagging strategy]({{ site.baseurl }}/documentation/reference/publish_process.html).

As mentioned [above](#stages), _stages_ are steps in the assembly process. They act as building blocks for constructing _images_.
Unlike images, _stages_ are not intended for the direct use. The main difference between images and stages is in [cleaning policies]({{ site.baseurl }}/documentation/reference/cleaning_process.html#cleanup-policies) due to the stored meta-information.
The process of cleaning up the _stages storage_ is only based on the related images in the _images repo_.

werf creates _images_ using the _stages storage_.
Currently, _images_ can only be created during the [_publishing process_]({{ site.baseurl }}/documentation/reference/publish_process.html) and saved in the _images repo_.

Images should be defined in the werf configuration file `werf.yaml`.

To publish new images into the images repo werf uses [synchronization lock manager](#synchronization-lock-manager) to coordinate multiple werf processes. Only a single werf process can perform publishing of the same image at a time.

## Synchronization lock manager

Synchornization lock manager is a service component of the werf to coordinate multiple werf processes when selecting and saving stages into stages storage and publishing images into images repo.

All commands that requires stages storage (`--stages-storage`) and images repo (`--images-repo`) params also require _syncrhonization lock manager_ address, which defined by the `--synchronization` option or `WERF_SYNCHRONIZATION=...` environment variable.
At the moment, only the local syncrhonization lock manager, `:local`, is supported.

(An implementation backed up by the Redis or Kubernetes server will be added to implement distributed builds soon.)

NOTE that multiple werf processes working with the same project should use the same _stages storage_ and _syncrhonization lock manager_.

## Further reading

Learn more about the [build process of stapel and Dockerfile builders]({{ site.baseurl }}/documentation/reference/build_process.html).
