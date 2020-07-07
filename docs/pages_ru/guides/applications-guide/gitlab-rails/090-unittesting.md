---
title: Юнит-тесты и Линтеры
sidebar: applications-guide
permalink: documentation/guides/applications-guide/gitlab-rails/090-unittesting.html
layout: guide
toc: false
---

{% filesused title="Файлы, упомянутые в главе" %}
- .gitlab-ci.yml
{% endfilesused %}

В этой главе мы настроим в нашем базовом приложении выполнение тестов/линтеров. Запуск тестов и линтеров - это отдельная стадия в pipelinе Gitlab CI для выполнения которых могут быть нужны определенные условия. Рассмотрим на примере линтера rubocop.

Если мы хотим воспользоваться пакетом rubocop-rails нам нужно добавить эту зависимость в наше приложение (в файл `Gemfile`) и прописать выполнение задания отдельной стадией на нашем gitlab runner командной [werf run](https://ru.werf.io/documentation/cli/main/run.html).

{% snippetcut name=".gitlab-ci.yaml" url="#" %}
{% raw %}
```yaml
Rubocop check:
  script:
    - werf run rails -- rubocop --require rubocop-rails
  stage: test
  tags:
    - werf
  needs: ["Build"]
```
{% endraw %}
{% endsnippetcut %}

Созданную стадию нужно добавить в список стадий

{% snippetcut name=".gitlab-ci.yaml" url="#" %}
{% raw %}
```yaml
stages:
  - build
  - test
  - deploy
```
{% endraw %}
{% endsnippetcut %}

Обратите внимание, что процесс будет выполняться на runner-е, внутри собранного контейнера, но без доступа к базе данных и каким-либо ресурсам Kubernetes-кластера.

{% offtopic title="А если нужно больше?" %}
Если нужен доступ к ресурсам кластера или база данных — это уже не линтер: нужно собирать отдельный образ и прописывать сложный сценарий деплоя объектов Kubernetes. Эти кейсы выходят за рамки нашего гайда для начинающих.
{% endofftopic %}

<div>
    <a href="110-multipleapps.html" class="nav-btn">Далее: Несколько приложений в одном репозитории</a>
</div>
