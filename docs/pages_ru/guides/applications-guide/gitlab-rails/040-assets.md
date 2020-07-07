---
title: Генерируем и раздаем ассеты
sidebar: applications-guide
permalink: documentation/guides/applications-guide/gitlab-rails/040-assets.html
layout: guide
toc: false
---

{% filesused title="Файлы, упомянутые в главе" %}
- .helm/templates/service.yaml
- .helm/templates/ingress.yaml
- werf.yaml
{% endfilesused %}

В какой-то момент в процессе развития вашего базового приложения вам понадобятся ассеты (т.е. картинки, css, js).

Для того, чтобы обработать ассеты мы воспользуемся Asset Pipeline. Он соединяет и сжимает ассеты JavaScript и CSS, добавляет возможность писать ассеты на других языках и с использованием препроцессоров, таких как CoffeeScript, Sass и ERB.

Для генерации ассетов мы будем использовать команду `bundle exec rake assets:precompile`.

Интуитивно понятно, что на одной из стадии сборки нам надо будет вызвать скрипт, который генерирует файлы, т.е. что-то надо будет дописать в `werf.yaml`. Однако, не только там — ведь какое-то приложение в production должно непосредственно отдавать статические файлы. Мы не будем отдавать файлы с помощью Rails — хочется, чтобы статику раздавал nginx. А значит надо будет внести какие-то изменения и в helm чарт.

Реализовать раздачу сгенерированных ассетов можно двумя способами:

* Добавить в собираемый образ с Rails ещё и nginx, а потом этот образ запускать уже двумя разными способами: один раз для раздачи статики, второй — для работы Rails-приложения
* Сделать два отдельных образа: в одном только nginx и сгенерированные ассеты, во втором — Rails-приложение 

{% offtopic title="Как правильно сделать выбор?" %}
Чтобы сделать выбор, нужно учитывать:

- как часто вносятся изменения в код, относящийся к каждому образу. Например, если изменения почти всегда вносятся одновременно и в статику и в приложение — меньше смысла отделять их сборку, всё равно оба пересобирать.
- размер полученных образов и, как следствие, объём данных, которые придётся выкачать при каждом перевыкате

Сравните два кейса:

- над кодом работают fullstack-разработчики и они привыкли коммитить, когда всё уже готово. Следовательно, пересборка обычно затрагивает и frontend и backend. Собранные по отдельности образы — 100 МБ и 600МБ, вместе - 620 МБ. Очевидно, стоит один образ.
- над кодом работают отдельно frontend и backend или коммиты забрасываются и выкатываются часто, после мелких правок только в одной из частей кода. Собранные по отдельности образы сравнительно одинаковые — 100 МБ и 150 МБ, вместе — 240 МБ. Если катать вместе, то всегда будет пересобираться оба образа и кататься по 240 МБ, если отдельно — то максимум 150 МБ.
{% endofftopic %}

Мы сделаем два отдельных образа.

## Подготовка к внесению изменений

Перед тем, как вносить изменения — **необходимо убедиться, что в собранных ассетах нет привязки к конкретному окружению**. То есть в собранных образах не должно быть логинов, паролей, доменов и тому подобного. В момент сборки Asset Pipeline не должен подключаться к базе данных, использовать user-generated контент и тому подобное.

По непонятной причине, для генерации ассетов Rails ходит в базу данных, хотя не понятно для каких целей и для этого - нужен `SECRET_KEY_BASE​`. При текущей сборке - мы использовали костыль, передав фейковое значение. По этому поводу есть issue созданное более 2х лет назад, но в версии rails 2.7 проблема по-прежнему остаётся. Если вы знаете, зачем авторы Rails так сделали - сообщите нам, пожалуйста.

## Изменения в сборке

Для ассетов мы соберём отдельный образ с nginx и ассетами. Для этого нужно собрать образ с nginx и забросить туда ассеты, предварительно собранные с помощью [механизма артефактов](https://ru.werf.io/documentation/configuration/stapel_artifact.html).

{% offtopic title="Что за артефакты?" %}
[Артефакт](https://ru.werf.io/documentation/configuration/stapel_artifact.html) — это специальный образ, используемый в других артефактах или отдельных образах, описанных в конфигурации. Артефакт предназначен преимущественно для отделения ресурсов инструментов сборки от процесса сборки образа приложения. Примерами таких ресурсов могут быть — программное обеспечение или данные, которые необходимы для сборки, но не нужны для запуска приложения, и т.п.

Важный и сложный вопрос — это отладка образов, в которых используются артефакты. Представим себе артефакт:

```yaml
artifact: my-simple-artefact
from: ubuntu:latest
...
```

Чтобы увидеть, что собирается внутри артефакта и отладить процесс этой сборки — просто замените первый атрибут на `image` (и не забудьте отключить `mount` в том образе, куда монтируется артефакт)

```yaml
image: my-simple-artefact
from: ubuntu:latest
...
```

Теперь вы можете отлаживать `my-simple-artefact` с помощью [интроспекции стадий](https://ru.werf.io/documentation/reference/development_and_debug/stage_introspection.html).
{% endofftopic %}

Начнём с создания артефакта: установим необходимые пакеты и выполним сборку ассетов. Генерация ассетов должна происходить в артефакте на стадии `setup`.

{% snippetcut name="werf.yaml" url="#" %}
{% raw %}
```yaml
artifact: assets-built
from: ruby:2.7.1
ansible:
  beforeInstall:
  - name: install node
    shell: curl -sL https://deb.nodesource.com/setup_{{ .NODE_MAJOR }}.x | bash -
  - name: install yarn repo
    shell:
      curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - \
      && echo 'deb http://dl.yarnpkg.com/debian/ stable main' > /etc/apt/sources.list.d/yarn.list
  - name: Update repositories cache
    shell: apt-get update -qq
  - name: install dependencies
    apt:
      name:
      - nodejs
      - yarn
  - name: install bundler
    shell: gem install bundler:{{ .BUNDLER_VERSION }}
  install:
  - name: bundle install
    shell: bundle config set without 'development test' && bundle install
    args:
      chdir: /app
  - name: webpacker install
    shell: RAILS_ENV=production rails webpacker:install
    args:
      chdir: /app
  setup:
  - name: build assets
    shell: RAILS_ENV=production SECRET_KEY_BASE=fake bundle exec rake assets:precompile
    args:
      chdir: /app
```
{% endraw %}
{% endsnippetcut %}

Теперь, когда артефакт собран, соберём образ с nginx:

{% snippetcut name="werf.yaml" url="#" %}
{% raw %}
```yaml
image: assets
from: nginx:alpine
ansible:
  beforeInstall:
  - name: Add nginx config
    copy:
      content: |
{{ .Files.Get ".werf/nginx.conf" | indent 8 }}
      dest: /etc/nginx/nginx.conf
```
{% endraw %}
{% endsnippetcut %}

И пропишем в нём импорт из артефакта под названием `assets-built`.

{% snippetcut name="werf.yaml" url="#" %}
{% raw %}
```yaml
import:
- artifact: assets-built
  add: /app/public
  to: /www
  after: setup
```
{% endraw %}
{% endsnippetcut %}

## Изменения в деплое и роутинге

Внутри Deployment сделаем два контейнера: один с `nginx`, который будет раздавать статические файлы, второй — с Rails приложением. Запросы сперва будут приходить на nginx, а тот будет перенаправлять запрос приложению, если не найдётся статических файлов.

 Обязательно укажем `livenessProbe` и `readinessProbe`, которые будут проверять корректную работу контейнера в Pod-е, а также `preStop` команду для корректного завершение процесса nginx, чтобы при выкате новой версии приложения корректно завершались активные сессии.

{% snippetcut name=".helm/templates/deployment.yaml" url="#" %}
{% raw %}
```yaml
      - name: assets
{{ tuple "assets" . | include "werf_container_image" | indent 8 }}
        lifecycle:
          preStop:
            exec:
              command: ["/usr/sbin/nginx", "-s", "quit"]
        livenessProbe:
          httpGet:
            path: /healthz
            port: 80
            scheme: HTTP
        readinessProbe:
          httpGet:
            path: /healthz
            port: 80
            scheme: HTTP
        ports:
        - containerPort: 80
          name: http
          protocol: TCP
```
{% endraw %}
{% endsnippetcut %}

В описании Service так же должен быть указан правильный порт:

{% snippetcut name=".helm/templates/service.yaml" url="#" %}
{% raw %}
```yaml
  ports:
  - name: http
    port: 80
    protocol: TCP
```
{% endraw %}
{% endsnippetcut %}

В Ingress также необходимо отправить запросы на правильный порт, чтобы они попадали на nginx.

{% snippetcut name=".helm/templates/ingress.yaml" url="#" %}
{% raw %}
```yaml
      paths:
      - path: /
        backend:
          serviceName: {{ .Chart.Name }}
          servicePort: 80
```
{% endraw %}
{% endsnippetcut %}

{% offtopic title="А можно ли разделять трафик на уровне ingress?" %}

В некоторых случаях нужно разделить трафик на уровне ingress. В таком случае можно разделить запросы по path и портам:

{% snippetcut name=".helm/templates/ingress.yaml" url="#" %}
{% raw %}
```yaml
      paths:
      - path: /
        backend:
          serviceName: {{ .Chart.Name }}
          servicePort: 3000
      - path: /assets
        backend:
          serviceName: {{ .Chart.Name }}
          servicePort: 80
```
{% endraw %}
{% endsnippetcut %}

{% endofftopic %}

<div>
    <a href="050-files.html" class="nav-btn">Далее: Работа с файлами</a>
</div>

