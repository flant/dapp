---
title: Introduction
sidebar: documentation
permalink: documentation/reference/configuration/introduction.html
author: Alexey Igrychev <alexey.igrychev@flant.com>, Timofey Kirillov <timofey.kirillov@flant.com>
---

Application should be configured to use werf. This configuration includes:

1. Definition of project meta information such as project name, which will affect build, deploy and other commands.
2. Definition of the images to be built.

werf uses YAML configuration file `werf.yaml` placed in the root folder of your application. The config is a collection of config sections -- parts of YAML file separated by [three hyphens](http://yaml.org/spec/1.2/spec.html#id2800132):

```yaml
CONFIG_SECTION
---
CONFIG_SECTION
---
CONFIG_SECTION
```

Each config section, `CONFIG_SECTION`, has a type. There are currently 3 types of config sections:

1. Config section to describe project meta information, which will be referred to as **meta config section**.
2. Config section to describe image build instructions, which will be referred to as **image config section** (use as many sections as you want).

More types can be added in the future.

## Meta config section

```yaml
project: PROJECT_NAME
configVersion: CONFIG_VERSION
OTHER_FIELDS
---
```

Config section with the key `project: PROJECT_NAME` and `configVersion: CONFIG_VERSION` is the meta config section. This is required section. There should be only one meta config section in a single `werf.yaml` configuration.

There are other directives, `deploy` and `cleanup`, described in separate articles: [deploy to Kubernetes]({{ site.baseurl }}/documentation/reference/configuration/deploy_into_kubernetes.html) and [cleanup policies]({{ site.baseurl }}/documentation/reference/configuration/cleanup.html).

### Project name

`project` defines unique project name of your application. Project name affects build cache image names, Kubernetes Namespace, Helm Release name and other derived names (see [deploy to Kubernetes for detailed description]({{ site.baseurl }}/documentation/reference/configuration/deploy_into_kubernetes.html)). This is single required field of meta configuration.

Project name should be unique within group of projects that shares build hosts and deployed into the same Kubernetes cluster (i.e. unique across all groups within the same gitlab).

Project name must be maximum 50 chars, only lowercase alphabetic chars, digits and dashes are allowed.

**WARNING**. You should never change project name, once it has been set up, unless you know what you are doing.

Changing project name leads to issues:
1. Invalidation of build cache. New images must be built. Old images must be cleaned up from local host and Docker registry manually.
2. Creation of completely new Helm Release. So if you already had deployed your application, then changed project name and deployed it again, there will be created another instance of the same application.

werf cannot automatically resolve project name change. Described issues must be resolved manually.

### Config version

The `configVersion` defines a `werf.yaml` format. It should always be `1` for now.

## Image config section

Each image config section defines instructions to build one independent docker image. There may be multiple image config sections defined in the same `werf.yaml` config to build multiple images.

Config section with the key `image: IMAGE_NAME` is the image config section. `image` defines short name of the docker image to be built. This name must be unique in a single `werf.yaml` config.

```yaml
image: IMAGE_NAME_1
OTHER_FIELDS
---
image: IMAGE_NAME_2
OTHER_FIELDS
---
...
---
image: IMAGE_NAME_N
OTHER_FIELDS
```

## Minimal config example

```yaml
project: my-project
configVersion: 1
```

## Processing of config

The following steps could describe the processing of a YAML configuration file:
1. Reading `werf.yaml` and extra templates from `.werf` directory.
2. Executing Go templates.
3. Saving dump into `.werf.render.yaml` (this file remains after the command execution and will be removed automatically with GC procedure).
4. Splitting rendered YAML file into separate config sections (part of YAML stream separated by three hyphens, https://yaml.org/spec/1.2/spec.html#id2800132).
5. Validating each config section:
   * Validating YAML syntax (you could read YAML reference [here](http://yaml.org/refcard.html)).
   * Validating werf syntax.

### Go templates

Go templates are available within YAML configuration. The following functions are supported:

* [Built-in Go template functions](https://golang.org/pkg/text/template/#hdr-Functions) and other language features. E.g. using common variable:<a id="go-templates" href="#go-templates" class="anchorjs-link " aria-label="Anchor link for: go templates" data-anchorjs-icon=""></a>

  {% raw %}
  ```yaml
  {{ $base_image := "golang:1.11-alpine" }}

  project: my-project
  configVersion: 1
  ---

  image: gogomonia
  from: {{ $base_image }}
  ---
  image: saml-authenticator
  from: {{ $base_image }}
  ```
  {% endraw %}

* [Sprig functions](http://masterminds.github.io/sprig/). E.g. using environment variable:<a id="sprig-functions" href="#sprig-functions" class="anchorjs-link " aria-label="Anchor link for: sprig functions" data-anchorjs-icon=""></a>

  {% raw %}
  ```yaml
  project: my-project
  configVersion: 1
  ---

  {{ $_ := env "SPECIFIC_ENV_HERE" | set . "GitBranch" }}

  image: ~
  from: alpine
  git:
  - url: https://github.com/company/project1.git
    branch: {{ .GitBranch }}
    add: /
    to: /app/project1
  - url: https://github.com/company/project2.git
    branch: {{ .GitBranch }}
    add: /
    to: /app/project2
  ```
  {% endraw %}

* `include` function with `define` for reusing configs:<a id="include" href="#include" class="anchorjs-link " aria-label="Anchor link for: include" data-anchorjs-icon=""></a>

  {% raw %}
  ```yaml
  project: my-project
  configVersion: 1
  ---

  image: app1
  from: alpine
  ansible:
    beforeInstall:
    {{- include "(component) ruby" . }}
  ---
  image: app2
  from: alpine
  ansible:
    beforeInstall:
    {{- include "(component) ruby" . }}

  {{- define "(component) ruby" }}
    - command: gpg --keyserver hkp://keys.gnupg.net --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3
    - get_url:
        url: https://raw.githubusercontent.com/rvm/rvm/master/binscripts/rvm-installer
        dest: /tmp/rvm-installer
    - name: "Install rvm"
      command: bash -e /tmp/rvm-installer
    - name: "Install ruby 2.3.4"
      raw: bash -lec {{`{{ item | quote }}`}}
      with_items:
      - rvm install 2.3.4
      - rvm use --default 2.3.4
      - gem install bundler --no-ri --no-rdoc
      - rvm cleanup all
  {{- end }}
  ```
  {% endraw %}

* `tpl` function to evaluate strings (either content of environment variable or project file) as Go templates inside a template: [example with project files](#with-tpl-function).<a id="tpl" href="#tpl" class="anchorjs-link " aria-label="Anchor link for: tpl" data-anchorjs-icon=""></a>

* `.Files.Get` and `.Files.Glob` functions to work with project files:<a id="files-get" href="#files-get" class="anchorjs-link " aria-label="Anchor link for: .Files.Get and .Files.Glob" data-anchorjs-icon=""></a>

  <div class="tabs">
    <a href="javascript:void(0)" class="tabs__btn active" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'ansible')">Ansible</a>
    <a href="javascript:void(0)" class="tabs__btn" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'shell')">Shell</a>
  </div>
  
  <div id="ansible" class="tabs__content active" markdown="1">
  
  **.Files.Get**
  
  {% raw %}
  ```yaml
  project: my-project
  configVersion: 1
  ---

  image: app
  from: alpine
  ansible:
    setup:
    - name: "Setup /etc/nginx/nginx.conf"
      copy:
        content: |
  {{ .Files.Get ".werf/nginx.conf" | indent 8 }}
        dest: /etc/nginx/nginx.conf
  ```
  {% endraw %}
  
  **.Files.Glob**
  
  {% raw %}
    > The function supports [shell pattern matching](https://www.gnu.org/software/findutils/manual/html_node/find_html/Shell-Pattern-Matching.html) + `**`. Results can be merged with [`merge` sprig function](https://github.com/Masterminds/sprig/blob/master/docs/dicts.md#merge-mustmerge) (e.g `{{ $filesDict := merge (.Files.Glob "*/*.txt") (.Files.Glob "app/**/*.txt") }}`)
    
  {% endraw %}
  
  {% raw %}
  ```yaml
  project: my-project
  configVersion: 1
  ---
  
  image: app
  from: alpine
  ansible:
    install:
    - raw: mkdir /app
    setup:
  {{ range $path, $content := .Files.Glob ".werf/files/*" }}
    - name: "Setup /app/{{ base $path }}"
      copy:
        content: |
  {{ $content | indent 8 }}
        dest: /app/{{ base $path }}
  {{ end }}
  ```
  {% endraw %}
  </div>
  
  <div id="shell" class="tabs__content" markdown="1">
  
  **.Files.Get**
  
  {% raw %}
  ```yaml
  project: my-project
  configVersion: 1
  ---
  
  image: app
  from: alpine
  shell:
    setup:
    - |
      head -c -1 <<'EOF' > /etc/nginx/nginx.conf
  {{ .Files.Get ".werf/nginx.conf" | indent 4 }}
      EOF
  ```
  {% endraw %}
  
  **.Files.Glob**

  {% raw %}
  > The function supports [shell pattern matching](https://www.gnu.org/software/findutils/manual/html_node/find_html/Shell-Pattern-Matching.html) + `**`. Results can be merged with [`merge` sprig function](https://github.com/Masterminds/sprig/blob/master/docs/dicts.md#merge-mustmerge) (e.g `{{ $filesDict := merge (.Files.Glob "*/*.txt") (.Files.Glob "app/**/*.txt") }}`)
  
  {% endraw %}
  
  {% raw %}
  ```yaml
  project: my-project
  configVersion: 1
  ---

  image: app
  from: alpine
  shell:
    install: mkdir /app
    setup:
  {{ range $path, $content := .Files.Glob ".werf/files/*" }}
    - |
      head -c -1 <<EOF > /app/{{ base $path }}
  {{ $content | indent 4 }}
      EOF
  {{ end }}
  ```
  {% endraw %}
  
  </div>