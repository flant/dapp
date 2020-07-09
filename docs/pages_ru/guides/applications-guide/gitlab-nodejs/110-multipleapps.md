---
title: Несколько приложений в одном репозитории
sidebar: applications-guide
permalink: documentation/guides/applications-guide/gitlab-nodejs/110-multipleapps.html
layout: guide
toc: false
---

{% filesused title="Файлы, упомянутые в главе" %}
- .helm/templates/deployment-frontend.yaml
- .helm/templates/deployment-backend.yaml
- .helm/templates/service-backend.yaml
- .helm/templates/ingress.yaml
- werf.yaml
{% endfilesused %}

В этой главе мы добавим к нашему базовому приложению ещё одно, находящееся в том же репозитории. Это корректная ситуация:

* для сложных случаев с двумя приложениями на двух разных языках
* для ситуации, когда есть более одного запускаемого процесса (например, сервис, отвечающий на http-запросы и worker)
* для ситуации, когда в одном репозитории находится frontend и backend

Рекомендуем также посмотреть [доклад Дмитрия Столярова](https://www.youtube.com/watch?v=g9cgppj0gKQ) о том, почему и в каких ситуациях это — хороший путь для микросервисов. Также вы можете посмотреть [аналогичную статью](https://ru.werf.io/documentation/guides/advanced_build/multi_images.html) о приложении с несколькими образами.

Добавим к нашему приложению, Python-приложение с ботом, который приветствует всех кто заходит в наш чат. Наш подход будет очень похож на то, что делалось в главе [Генерируем и раздаем ассеты](040-assets.html), с одним существенным отличием: изменения в коде Python сильно отделены от изменений в NodeJS приложении. Как следствие — мы разнесём их в разные папки, а также в различные Pod-ы. 

Мы рассмотрим вопрос организации структуры файлов и папок, соберём два образа: для NodeJS приложения и для Python приложения и сконфигурируем запуск этих образов в kubernetes.

## Структура файлов и папок

Структура каталогов будет организована следующим образом:

```
├── .helm/
│   ├── templates/
│   └── values.yaml
├── node/
├── python/
└── werf.yaml
```

Для одного репозитория рекомендуется использовать один файл `werf.yaml` и одну папку `.helm` с конфигурацией инфраструктуры. Такой подход делает работу над кодом прозрачнее и помогает избегать рассинхронизации в двух частях одного проекта.

{% offtopic title="А если получится слишком много информации в одном месте и станет сложно ориентироваться?" %}
Helm обрабатывает все файлы, которые находятся в папке `.helm/templates`, а значит их может быть столько, сколько удобно вам. Для упрощения кода можно использовать [общие блоки](https://helm.sh/docs/chart_template_guide/named_templates/).

Кроме того `werf.yaml` также поддерживает [Описание конфигурации в нескольких файлах](https://ru.werf.io/documentation/configuration/introduction.html#%D0%BE%D0%BF%D0%B8%D1%81%D0%B0%D0%BD%D0%B8%D0%B5-%D0%BA%D0%BE%D0%BD%D1%84%D0%B8%D0%B3%D1%83%D1%80%D0%B0%D1%86%D0%B8%D0%B8-%D0%B2-%D0%BD%D0%B5%D1%81%D0%BA%D0%BE%D0%BB%D1%8C%D0%BA%D0%B8%D1%85-%D1%84%D0%B0%D0%B9%D0%BB%D0%B0%D1%85) и вынесение части кода в общие блоки.
{% endofftopic %}

## Сборка приложений

На стадии сборки приложения нам необходимо правильно организовать структуру файла `werf.yaml`, описав в нём сборку двух приложений на разном стеке.

Мы соберём два образа: `node` c NodeJS-приложением и `bot` c Python-приложением.

{% offtopic title="Как конкретно?" %}

Сборка образа `node` аналогична ранее описанному [базовому приложению](020-basic.html) с [зависимостями](030-dependencies.html), за исключением того, откуда берётся исходный код:

{% snippetcut name="werf.yaml" url="#" %}
{% raw %}
```yaml
git:
- add: /node
  to: /app
```
{% endraw %}
{% endsnippetcut %}

Мы добавляем в собираемый образ только исходные коды, относящиеся к Python приложению. Таким образом, пересборка этой части проекта не будет срабатывать, когда изменился только NodeJs-код.

Сборка для python приложения описана в файле `werf.yaml` как отдельный образ. 

{% snippetcut name="werf.yaml" url="#" %}
{% raw %}
```yaml
image: bot
from: node:{{ .NODE_MAJOR }}
git:
- add: /python
  to: /app
```
{% endraw %}
{% endsnippetcut %}


## Конфигурация инфраструктуры в Kubernetes

Подготовленные приложения мы будем запускать отдельными объектами Deployment: таким образом в случае изменений только в одной из частей приложения будет перевыкатываться только эта часть. Создадим два отдельных файла для описания объектов: `app.yaml` и `bot.yaml`. В условиях, когда в одном сервисе меньше 15-20 объектов — удобно следовать принципу максимальной атомарности в шаблонах.

При деплое нескольких Deployment крайне важно правильно прописать `selector`-ы в Service и Deployment:

{% snippetcut name="service-app.yaml" url="#" %}
{% raw %}
```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Chart.Name }}
spec:
  selector:
    app: {{ .Chart.Name }}
```
{% endraw %}
{% endsnippetcut %}

{% snippetcut name="app.yaml" url="#" %}
{% raw %}
```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
spec:
  selector:
    matchLabels:
      app: {{ .Chart.Name }}
  template:
    metadata:
      labels:
        app: {{ .Chart.Name }}
```
{% endraw %}
{% endsnippetcut %}

Маршрутизация запросов будет осуществляться через Ingress:

{% snippetcut name="ingress.yaml" url="#" %}
{% raw %}
```yaml
  rules:
  - host: {{ .Values.global.ci_url }}
    http:
      paths:
      - path: /
        backend:
          serviceName: {{ .Chart.Name }}
          servicePort: 3000
```
{% endraw %}
{% endsnippetcut %}

<div>
    <a href="120-dynamicenvs.html" class="nav-btn">Далее: Динамические окружения</a>
</div>
