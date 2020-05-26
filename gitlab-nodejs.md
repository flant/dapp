---
author_team: "Alfa"
author_name: "Andrey Koregin"
ci: "gitlab"
language: "nodejs"
framework: "react"
is_compiled: 0
package_managers_possible:
 - npm
 - yarn
 - pnpm
package_managers_chosen: "npm"
unit_tests_possible:
 - flask-sqlalchemy
 - pytest
 - unittest
 - nose
 - nose2
unit_tests_chosen: "flask-sqlalchemy"
assets_generator_possible:
 - webpack
 - gulp
assets_generator_chosen: "webpack"
---

# Чек-лист готовности статьи
<ol>
<li>Все примеры кладём в <a href="https://github.com/flant/examples">https://github.com/flant/examples</a>

<li>Для каждой статьи может и должно быть НЕСКОЛЬКО примеров, условно говоря — по примеру на главу это нормально.

<li>Делаем примеры И на Dockerfile, И на Stapel

<li>Про хельм говорим, про особенности говорим, но в подробности не вдаёмся — считаем, что человек умеет в кубовые ямлы.

<li>Обязательно тестируйте свои примеры перед публикацией
</li>
</ol>

# Введение

Рассмотрим разные способы которые помогут Nodejs программисту собрать приложение и запустить его в kubernetes кластере.

Предполагается что читатель имеет базовые знания в разработке на Nodejs а также немного знаком с Gitlab CI и примитивами kubernetes, либо готов во всём этом разобраться самостоятельно. Мы постараемся предоставить все ссылки на необходимые ресурсы, если потребуется приобрести какие то новые знания.  

Собирать приложения будем с помощью werf. Данный инструмент работает в Linux MacOS и Windows, инструкция по [установке](https://ru.werf.io/documentation/guides/installation.html) находится на официальном [сайте](https://ru.werf.io/). В качестве примера - также приложим Docker файлы.

Для иллюстрации действий в данной статье - создан репозиторий с исходным кодом, в котором находятся несколько простых приложений. Мы постараемся подготовить примеры чтобы они запускались на вашем стенде и постараемся подсказать, как отлаживать возможные проблемы при вашей самостоятельной работе.


## Подготовка приложения

Наилучшим образом приложения будут работать в Kubernetes - если они соответствуют [12 факторам heroku](https://12factor.net/). Благодаря этому - у нас в kubernetes работают stateless приложения, которые не зависят от среды. Это важно, так как кластер может самостоятельно переносить приложения с одного узла на другой, заниматься масштабированием и т.п. — и мы не указываем, где конкретно запускать приложение, а лишь формируем правила, на основании которого кластер принимает свои собственные решения.

Договоримся что наши приложения соответствуют этим требованиям. На хабре уже было описание данного подхода, вы можете почитать про него например [тут](https://12factor.net/).


## Подготовка и настройка среды

Для того, чтобы пройти по этому гайду, необходимо, чтобы

*   У вас был работающий и настроенный Kubernetes кластер
*   Код приложения находился в Gitlab
*   Был настроен Gitlab CI, подняты и подключены к нему раннеры

Для пользователя под которым будет производиться запуск runner-а - нужно установить multiwerf - данная утилита позволяет переключаться между версиями werf и автоматически обновлять его. Инструкция по установке - доступна по [ссылке](https://ru.werf.io/documentation/guides/installation.html#installing-multiwerf).

Для автоматического выбора актуальной версии werf в канале stable, релиз 1.1 выполним следующую  команду:

```
. $(multiwerf use 1.1 stable --as-file)
```

Перед деплоем нашего приложения необходимо убедиться что наша инфраструктура готова к тому чтобы использовать werf. Используя [инструкцию](https://ru.werf.io/documentation/guides/gitlab_ci_cd_integration.html#%D0%BD%D0%B0%D1%81%D1%82%D1%80%D0%BE%D0%B9%D0%BA%D0%B0-runner) по подготовке к использованию Werf в Gitlab CI, вам нужно убедиться что все следующие пункты выполнены:

*   Развернут отдельный сервер с сетевой доступностью до мастер ноды Kubernetes.
*   На данном сервере установлен gitlab-runner.
*   Gitlab-runner подключен к нашему Gitlab с тегом werf в режиме shell executor. 
*   Ранеры включены и активны для репозитория с нашим приложением.
*   Для пользователя, которого использует gitlab-runner и под которым запускается сборка и деплой, установлен kubectl и добавлен конфигурационный файл для подключения к kubernetes.
*   Для gitlab включен и настроен gitlab registry
*   Gitlab-runner имеет доступ к API kubernetes и запускается по тегу werf  


# Hello world

В первой главе мы покажем поэтапную сборку и деплой приложения без задействования внешних ресурсов таких как база данных и сборку ассетов.

Наше приложение будет состоять из одного docker образа собранного с помощью werf.

В этом образе будет работать один основной процесс `node`, который запустит приложение.

Управлять маршрутизацией запросов к приложению будет управлять Ingress в kubernetes кластере.

Мы реализуем два стенда: production и staging. В рамках hello world приложения мы предполагаем, что разработка ведётся локально, на вашем компьютере.

_В ближайшее время werf реализует удобные инструменты для локальной разработки, следите за обновлениями._


## Локальная сборка


Для того чтобы werf смогла начать работу с нашим приложением - необходимо в корне нашего репозитория создать файл werf.yaml в которым будут описаны инструкции по сборке. Для начала соберем образ локально не загружая его в registry чтобы разобраться с синтаксисом сборки.

С помощью werf можно собирать образы с используя Dockerfile или используя синтаксис, описанный в документации werf (мы называем этот синтаксис и движок, который этот синтаксис обрабатывает, stapel). Для лучшего погружения - соберем наш образ с помощью stapel.

Прежде всего нам необходимо собрать docker image с нашим приложением внутри. 

Клонируем наши исходники любым удобным способом. В нашем случае это:


```
git clone git@gitlab-example.com:article/chat.git
```

После, в корне склоненного проекта, создаём файл `werf.yaml`. Данный файл будет отвечать за сборку вашего приложения и он обязательно должен находиться в корне проекта. 

![structure]( ./gitlab-nodejs-files/images/structure.png)

Итак, начнём с самой главной секции нашего werf.yaml файла, которая должна присутствовать в нём **всегда**. Называется она [meta config section](https://werf.io/documentation/configuration/introduction.html#meta-config-section) и содержит всего два параметра.

werf.yaml:
```yaml
project: chat
configVersion: 1
```

**_project_** - поле, задающее имя для проекта, которым мы определяем связь всех docker images собираемых в данном проекте. Данное имя по умолчанию используется в имени helm релиза и имени namespace в которое будет выкатываться наше приложение. Данное имя не рекомендуется изменять (или подходить к таким изменениям с должным уровнем ответственности) так как после изменений уже имеющиеся ресурсы, которые выкачаны в кластер, не будут переименованы.

**_configVersion_** - в данном случае определяет версию синтаксиса используемую в `werf.yaml`.

После мы сразу переходим к следующей секции конфигурации, которая и будет для нас основной секцией для сборки - [image config section](https://werf.io/documentation/configuration/introduction.html#image-config-section). И чтобы werf понял что мы к ней перешли разделяем секции с помощью тройной черты.


```yaml
project: chat
configVersion: 1
---
image: node
from: node:14-stretch
```

**_image_** - поле задающее имя нашего docker image, с которым он будет запушен в registry. Должно быть уникально в рамках одного werf-файла.

**_from_** - задает имя базового образа который мы будем использовать при сборке. Задаем мы его точно так же, как бы мы это сделали в dockerfile, т.к. приложение у нас на Nodejs, мы берём готовый docker image - _node_  с тэгом _14-stretch_. (означает что будет использована 14 версия Nodejs, а базовый образ построен на debian системе)

Теперь встает вопрос о том как нам добавить исходный код приложения внутрь нашего docker image. И для этого мы можем использовать Git! И нам даже не придётся устанавливать его внутрь docker image.

**_git_** на наш взгляд это самый правильный способ добавления ваших исходников внутрь docker image, хотя существуют и другие. Его преимущество в том что он именно клонирует, и в дальнейшем накатывает коммитами изменения в тот исходный код что мы добавили внутрь нашего docker image, а не просто копирует файлы. Вскоре мы узнаем зачем это нужно.

```yaml
project: chat
configVersion: 1
---
image: node
from: node:14-stretch
git:
- add: /
  to: /app
```

Werf подразумевает что ваша сборка будет происходить внутри директории склонированного git репозитория. Потому мы списком можем указывать директории и файлы относительно корня репозитория которые нам нужно добавить внутрь image.

`add: /` - та директория которую мы хотим добавить внутрь docker image, мы указываем, что это корень, т.е. мы хотим склонировать внутрь docker image весь наш репозиторий.

`to: /app` - то куда мы клонируем наш репозиторий внутри docker image. Важно заметить что директорию назначения werf создаст сам.

 Есть возможность даже добавлять внешние репозитории внутрь проекта не прибегая к предварительному клонированию, как это сделать можно узнать [тут](https://werf.io/documentation/configuration/stapel_image/git_directive.html).

Но на этом наша сборка, сама собой, не заканчивается и теперь пора приступать к действиям непосредственно внутри image. Для этого мы будем описывать сборку через ansible.  Прежде чем описывать задачи в ansible, необходимо добавить еще два важных поля:

```yaml
---
image: node
from: node:14-stretch
git:
- add: /
  to: /app
ansible:
  beforeInstall:
  install:
```

Это поля **_beforeInstall_** и **_install_**

В их понимании нам поможет это изображение:

![]( /werf-articles/gitlab-nodejs-files/images/stages.png "Stages")

Эта картинка упрощенно иллюстрирует процесс сборки образа с помощью werf. Тут мы видим что данные поля названы как стадии сборки на картинке.

Главное, что сразу можно увидеть, **_beforeInstall_** выполняется раньше чем мы добавляем исходники с помощью **_git_**. И на данном этапе этого нам будет достаточно, т.к. мы сможем преднастроить наш образ прежде чем приступим к работе с исходниками.

Давайте посмотрим как это выглядит:


```yaml
---
image: node
from: node:14-stretch
git:
- add: /node
  to: /app
  stageDependencies:
    install:
    - package.json
ansible:
  beforeInstall:
  - name: Install dependencies
    apt:
      name:
      - tzdata
      - locales
      update_cache: yes
  install:
  - name: npm сi
    shell: npm сi
    args:
      chdir: /app
```

Мы добавили значительное количество строк, но на самом деле те кто уже имел дело с ansbile, уже должны были узнать в них обычные ansible tasks.

В **beforeInstall** мы, c помощью [apt](https://docs.ansible.com/ansible/latest/modules/apt_module.html), добавили установку обычных deb пакетов отвечающих за таймзону и локализацию. 

А в **install** у нас расположился запуск установки зависимостей с помощью npm, запуская его просто как команду через модуль [shell](https://docs.ansible.com/ansible/latest/modules/shell_module.html).

Полный список поддерживаемых модулей ansible в werf можно найти [тут](https://werf.io/documentation/configuration/stapel_image/assembly_instructions.html#supported-modules).

Но еще мы добавили в git следующую конструкцию:

```yaml
  stageDependencies:
    install:
    - package.json
```
Данная конструкция отвечает за отслеживание изменений в файле package.json и пересборки стадии install в случае нахождения таковых.

Подробнее о стадиях, для чего они и как работают, а также об отслеживании изменений будет описано в соответствующей главе - [Подключаем зависимости](#подключаем-зависимости)

И на этом всё! Мы описали необходимый минимум нужный для сборки нашего приложения. Теперь нам достаточно её запустить! 


Мы просто берём последнюю стабильную версию werf исполняя эту команду:


```bash
type multiwerf && . $(multiwerf use 1.1 stable --as-file)
```


И видим что werf был установлен:


```
user:~/chat$ type multiwerf && . $(multiwerf use 1.1 stable --as-file)
multiwerf is /home/user/bin/multiwerf
multiwerf v1.3.0
Starting multiwerf self-update ...
Self-update: Already the latest version
GC: Actual versions: [v1.0.13 v1.1.10-alpha.6 v1.1.8+fix16 v1.1.9+fix6]
GC: Local versions:  [v1.0.13 v1.1.9+fix6]
GC: Nothing to clean
The version v1.1.8+fix16 is the actual for channel 1.1/stable
Downloading the version v1.1.8+fix16 ...
The actual version has been successfully downloaded
```


Теперь наконец запускаем сборку с помощью [werf build](https://werf.io/documentation/cli/main/build.html) !


```bash
$  werf build --stages-storage :local
```

![build](/werf-articles/gitlab-nodejs-files/images/build.gif "build")

Вот и всё, наша сборка успешно завершилась. К слову если сборка падает и вы хотите изнутри контейнера её подебажить вручную, то вы можете добавить в команду сборки флаги:

```yaml
--introspect-before-error
```

или

```yaml
--introspect-error
```

Которые при падении сборки на одном из шагов автоматически откроют вам shell в контейнер, перед исполнением проблемной инструкции или после.

В конце werf отдал информацию о готовом image:

![info_image](/werf-articles/gitlab-nodejs-files/images/info_image.png)

Теперь его можно запустить локально используя image_id просто с помощью docker.
Либо вместо этого использовать [werf run](https://werf.io/documentation/cli/main/run.html):


```bash
werf run --stages-storage :local --docker-options="-d -p 8080:8080 --restart=always" -- node /app/src/js/index.js
```

Первая часть команды очень похожа на build, а во второй мы задаем [параметры](https://docs.docker.com/engine/reference/run/) docker и через двойную черту команду с которой хотим запустить наш image.

Небольшое пояснение про `--stages-storage :local `который мы использовали и при сборке и при запуске приложения. Данный параметр указывает на то где werf хранить стадии сборки. На момент написания статьи это возможно только локально, но в ближайшее время появится возможность сохранять их в registry.

Теперь наше приложение доступно локально на порту 8080:

![redyapp](/werf-articles/gitlab-nodejs-files/images/readyapp.png "readyapp")

На этом часть с локальным использованием werf мы завершаем и переходим к той части для которой werf создавался, использовании его в CI.

## Построение CI-процесса

После того как мы закончили со сборкой, которую можно производить локально, мы приступаем к базовой настройке CI/CD на базе Gitlab.

Начнем с того что добавим нашу сборку в CI с помощью .gitlab-ci.yml, который находится внутри корня проекта. Нюансы настройки CI в Gitlab можно найти [тут](https://docs.gitlab.com/ee/ci/).

Мы предлагаем простой флоу, который мы называем [fast and furious](https://docs.google.com/document/d/1a8VgQXQ6v7Ht6EJYwV2l4ozyMhy9TaytaQuA9Pt2AbI/edit#). Такой флоу позволит вам осуществлять быструю доставку ваших изменений в production согласно методологии GitOps и будут содержать два окружения, production и stage.

На стадии сборки мы будем собирать образ с помощью werf и загружать образ в registry, а затем на стадии деплоя собрать инструкции для kubernetes, чтобы он скачивал нужные образы и запускал их.

### Сборка в Gitlab CI

Для того, чтобы настроить CI-процесс создадим .gitlab-ci.yaml в корне репозитория.

Инициализируем werf перед запуском основной команды. Это необходимо делать перед каждым использованием werf поэтому мы вынесли в секцию `before_script`
Такой сложный путь с использованием multiwerf нужен для того, чтобы вам не надо было думать про обновление верфи и установке новых версий — вы просто указываете, что используете, например, use 1.1 stable и пребываете в уверенности, что у вас актуальная версия с закрытыми issues.

```yaml
before_script:
  - type multiwerf && source <(multiwerf use 1.1 stable)
  - type werf && source <(werf ci-env gitlab --verbose)
```

`werf ci-env gitlab --verbose` - готовит наш werf для работы в Gitlab, выставляя для этого все необходимые переменные.
Пример переменных автоматически выставляемых этой командой:

```bash
### DOCKER CONFIG
 export DOCKER_CONFIG="/tmp/werf-docker-config-832705503"
 ### STAGES_STORAGE
 export WERF_STAGES_STORAGE="registry.gitlab-example.com/chat/stages"
 ### IMAGES REPO
 export WERF_IMAGES_REPO="registry.gitlab-example.com/chat"
 export WERF_IMAGES_REPO_IMPLEMENTATION="gitlab"
 ### TAGGING
 export WERF_TAG_BY_STAGES_SIGNATURE="true"
 ### DEPLOY
 # export WERF_ENV=""
 export WERF_ADD_ANNOTATION_PROJECT_GIT="project.werf.io/git=https://lab.gitlab-example.com/chat"
 export WERF_ADD_ANNOTATION_CI_COMMIT="ci.werf.io/commit=61368705db8652555bd96e68aadfd2ac423ba263"
 export WERF_ADD_ANNOTATION_GITLAB_CI_PIPELINE_URL="gitlab.ci.werf.io/pipeline-url=https://lab.gitlab-example.com/chat/pipelines/71340"
 export WERF_ADD_ANNOTATION_GITLAB_CI_JOB_URL="gitlab.ci.werf.io/job-url=https://lab.gitlab-example.com/chat/-/jobs/184837"
 ### IMAGE CLEANUP POLICIES
 export WERF_GIT_TAG_STRATEGY_LIMIT="10"
 export WERF_GIT_TAG_STRATEGY_EXPIRY_DAYS="30"
 export WERF_GIT_COMMIT_STRATEGY_LIMIT="50"
 export WERF_GIT_COMMIT_STRATEGY_EXPIRY_DAYS="30"
 export WERF_STAGES_SIGNATURE_STRATEGY_LIMIT="-1"
 export WERF_STAGES_SIGNATURE_STRATEGY_EXPIRY_DAYS="-1"
 ### OTHER
 export WERF_LOG_COLOR_MODE="on"
 export WERF_LOG_PROJECT_DIR="1"
 export WERF_ENABLE_PROCESS_EXTERMINATOR="1"
 export WERF_LOG_TERMINAL_WIDTH="95"
```


Многие из этих переменных интуитивно понятны, и содержат базовую информацию о том где находится проект, где находится его registry, информацию о коммитах. \
Подробную информацию о конфигурации ci-env можно найти [тут](https://werf.io/documentation/reference/plugging_into_cicd/overview.html). От себя лишь хочется добавить, что если вы используете совместно с Gitlab внешний registry (harbor,Docker Registry,Quay etc.), то в команду билда и пуша нужно добавлять его полный адрес (включая путь внутри registry), как это сделать можно узнать [тут](https://werf.io/documentation/cli/main/build_and_publish.html). И так же не забыть первой командой выполнить [docker login](https://docs.docker.com/engine/reference/commandline/login/).

В рамках статьи нам хватит значений выставляемых по умолчанию.

Переменная [WERF_STAGES_STORAGE](https://ru.werf.io/documentation/reference/stages_and_images.html#%D1%85%D1%80%D0%B0%D0%BD%D0%B8%D0%BB%D0%B8%D1%89%D0%B5-%D1%81%D1%82%D0%B0%D0%B4%D0%B8%D0%B9) указывает где werf сохраняет свой кэш (стадии сборки) У werf есть опция распределенной сборки, про которую вы можете прочитать в нашей статье, в текущем примере мы сделаем по-простому и сделаем сборку на одном узле в один момент времени.


```yaml
variables:
    WERF_STAGES_STORAGE: ":local"
```
Дело в том что werf хранит стадии сборки раздельно, как раз для того чтобы мы могли не пересобирать весь образ, а только отдельные его части.

Плюс стадий в том, что они имеют собственный тэг, который представляет собой хэш содержимого нашего образа. Тем самым позволяя полностью избегать не нужных пересборок наших образов. Если вы собираете ваше приложение в разных ветках, и исходный код в них различается только конфигами которые используются для генерации статики на последней стадии. То при сборке образа одинаковые стадии пересобираться не будут, будут использованы уже собранные стадии из соседней ветки. Тем самым мы резко снижаем время доставки кода.

Основная команда на текущий момент - это werf build-and-publish, которая запускает сборку и публикацию в registry на gitlab runner с тегом werf для любой ветки. Путь до registry и другие параметры беруться верфью автоматически их переменных окружения gitlab ci.

```yaml
Build:
  stage: build
  script:
    - werf build-and-publish
  tags:
    - werf
```

Если вы всё правильно сделали и корректно настроен registry и gitlab ci — вы увидите собранный образ в registry. При использовании registry от gitlab — собранный образ можно увидеть через веб-интерфейс гитлаба.

Следующие параметры тем кто работал с гитлаб уже должны быть знакомы.

**_tags_** - нужен для того чтобы выбрать наш раннер, на который мы навесили этот тэг. В данном случае наш gitlab-runner в Gitlab имеет тэг werf

```yaml
  tags:
    - werf
```


Теперь мы можем запушить наши изменения и увидеть что наша стадия успешно выполнилась.
![gbuild](/werf-articles/gitlab-nodejs-files/images/gbuild.png "gbuild")


Лог в Gitlab будет выглядеть так же как и при локальной сборке, за исключением того что в конце мы увидим как werf пушит наш docker image в registry.

```
207 │ ┌ Publishing image {{node}} by stages-signature tag c905b748cb9647a03476893941837bf79910ab09e ...
208 │ ├ Info
209 │ │   images-repo: registry.gitlab-example.com/chat/node
210 │ │        image: registry.gitlab-example.com/chat/node:c905b748cb9647a03476893941 ↵
211 │ │   837bf79910ab09ef5878037592a45d
212 │ └ Publishing image node by stages-signature tag c905b748cb9647a0347689394 ... (14.90 seconds)
213 └ ⛵ image {{node}} (73.44 seconds)
214 Running time 73.47 seconds
218 Job succeeded
```

Вы можете заметить что werf протегировал наш образ, странным образом `registry.gitlab-example.com/chat/node:c905b748cb9647a03476893941837bf79910ab09ef5878037592a45d`. По умолчанию werf использует стратегию тэгирования `stages-signarue`, что означает что тэг будет проставляться на основании содержимого нашего образа. Подробнее об этом вы можете прочитать тут - [content based tagging](https://werf.io/documentation/reference/publish_process.html#content-based-tagging). Если говорить простыми словами, то тэг явлеется ничем иным как хэш-суммой содержимого нашего образа - код, который мы добавили туда, команды которые мы выполняли и т.д.

### Деплой в Kubernetes

Werf использует встроенный Helm для применения конфигурации в Kubernetes. Для описания объектов Kubernetes werf использует конфигурационные файлы Helm: шаблоны и файлы с параметрами (например, values.yaml). Помимо этого, werf поддерживает дополнительные файлы, такие как файлы c секретами и с секретными значениями (например secret-values.yaml), а также дополнительные Go-шаблоны для интеграции собранных образов.

Werf (по аналогии с helm) берет yaml шаблоны, которые описывают объекты Kubernetes, и генерирует из них общий манифест. Манифест отдается API Kubernetes, который на его основе внесет все необходимые изменения в кластер. Werf отслеживает как Kubernetes вносит изменения и сигнализирует о результатах в реальном времени. Все это благодаря встроенной в werf библиотеке [kubedog](https://github.com/flant/kubedog).

Внутри Werf доступны команды для работы с Helm, например можно проверить как сгенерируется общий манифест в результате работы werf с шаблонами:

```bash
$ werf helm render
```

Аналогично, доступны команды [helm list](https://werf.io/documentation/cli/management/helm/list.html) и другие.

#### Общее про хельм-конфиги

На сегодняшний день [Helm](https://helm.sh/) один из самых удобных способов которым вы можете описать свой deploy в Kubernetes. Кроме возможности установки готовых чартов с приложениями прямиком из репозитория, где вы можете введя одну команду, развернуть себе готовый Redis, Postgres, Rabbitmq прямиком в Kubernetes, вы также можете использовать Helm для разработки собственных чартов с удобным синтаксисом для шаблонизации выката ваших приложений.

Потому для werf это был очевидный выбор использовать такую технологию.

Мы не будем вдаваться в подробности разработки yaml манифестов с помощью Helm для Kubernetes. Осветим лишь отдельные её части, которые касаются данного приложения и werf в целом. Если у вас есть вопросы о том как именно описываются объекты Kubernetes, советуем посетить страницы документации по Kubernetes с его [концептами](https://kubernetes.io/ru/docs/concepts/) и страницы документации по разработке [шаблонов](https://helm.sh/docs/chart_template_guide/) в Helm.

Нам понадобятся следующие файлы со структурой каталогов:


```
.helm (здесь мы будем описывать деплой)
├── templates (объекты kubernetes в виде шаблонов)
│   ├── deployment.yaml (основное приложение)
│   ├── ingress.yaml (описание для ingress)
│   └── service.yaml (сервис для приложения)
├── secret-values.yaml (файл с секретными переменными)
└── values.yaml (файл с переменными для параметризации шаблонов)
```

Подробнее читайте в [нашей статье](https://habr.com/ru/company/flant/blog/423239/) из серии про Helm.


#### Описание приложения в хельме

Для работы нашего приложения в среде Kubernetes понадобится описать сущности Deployment, Service, завернуть трафик на приложение, донастроив роутинг в кластере с помощью сущности Ingress. И не забыть создать отдельную сущность Secret, которая позволит нашему kubernetes пулить собранные образа из registry.

##### Запуск контейнера

Начнем с описания [deployment.yaml](/werf-articles/gitlab-nodejs-files/examples/deployment.yaml)

<details><summary>deployment.yaml</summary>
<p>

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
spec:
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
  replicas: 1
  selector:
    matchLabels:
      app: {{ $.Chart.Name }}
  template:
    metadata:
      labels:
        app: {{ $.Chart.Name }}
    spec:
      imagePullSecrets:
      - name: "registrysecret"
      containers:
      - name: {{ $.Chart.Name }}
{{ tuple "node" . | include "werf_container_image" | indent 8 }}
        command: ["node","/app/src/js/index.js"]
        ports:
        - containerPort: 8080
          protocol: TCP
        env:
{{ tuple "node" . | include "werf_container_env" | indent 8 }}
```
</p>
</details>

Коснусь только шаблонизированных и изменяемых параметров. Значение остальных параметров можно найти в документации [Kubernetes](https://kubernetes.io/docs/concepts/).

`{{ .Chart.Name }}` - значение для данного параметра берётся из файла werf.yaml из поля **_project_**

werf.yaml:

```yaml
project: chat
configVersion: 1
```

Далее мы указываем имя сектрета в котором мы будем хранить данные для подключение к нашему registry, где хранятся наши образа.

```yaml
      imagePullSecrets:
      - name: "registrysecret"
```

О том как его создать мы опишем в конце главы.

```yaml
{{ tuple "node" . | include "werf_container_image" | indent 8 }}
```


Данный шаблон отвечает за то чтобы вставить информацию касающуюся местонахождения нашего doсker image в registry, чтобы kubernetes знал откуда его скачать. А также политику пула этого образа. И в итоге эта строка будет заменена helm’ом на это:



```yaml
   image: registry.gitlab-example.com/chat/node:6e3af42b741da90f8bc674e5646a87ad6b81d14c531cc89ef4450585   
   imagePullPolicy: IfNotPresent
```


Замену производит сам werf из собственных переменных. Изменять эту конструкцию нужно только в двух местах:
1. Рядом в первой части “node”  -  это название вашего docker image, которые мы указывали в werf.yaml в поле **image**, когда описывали сборку.

2. Intent 8 - параметр указывает какое количество пробелов вставить перед блоком, делаем мы это чтобы не нарушить синтаксис yaml, где пробелы(отступы) играют важную разделительную роль.  \
При разработке особенно важно учитывать что yaml не воспринимает табуляцию **только пробелы**!

```yaml
        command: ["node","/app/src/js/server.js"]
```
Одна из самых главных строк, отвечает непосредственно за то какую команду запустить при запуске приложения.

```yaml
        ports:
        - containerPort: 8080
          protocol: TCP
```
Блок отвечающий за то какие порты необходимо сделать доступными снаружи контейнера, и по какому протоколу.

```yaml
{{ tuple "node" . | include "werf_container_env" | indent 8 }}
```
Этот шаблон позволяет werf работать с переменными.
Его назначение подробно описано в главе [Переменные окружения](####переменные-окружения).

Теперь, как мы и обещали перейдем к созданию сущности Secret, которая будет содержать доступы до images registry.
Вы можете использовать команду kubectl, из кластера или у себя на личном компьютере даже если он не имеет доступ к кластеру (он нам не понадобится).
Вы можете запустить следующую команду:
```bash
kubectl create secret docker-registry regcred --docker-server=<your-registry-server> --docker-username=<your-name> --docker-password=<your-pword> --docker-email=<your-email> --dry-run=true -o yaml
```
В команде вы указываете данные пользователя для подключения и затем получаете такой вывод:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: registrysecret
  creationTimestamp: null
type: kubernetes.io/dockerconfigjson

data:
  .dockerconfigjson: eyJhdXRocyI6eyJyZWdpc3RyeS5leGFtcGxlLmNvbSI6eyJ1c2VybmFtZSI6InVzZXIiLCJwYXNzd29yZCI6InF3ZXJ0eSIsImVtYWlsIjoiZXhhbXBsZUBnbWFpbC5ydSIsImF1dGgiOiJkWE5sY2pweGQyVnlkSGs9In19fQ==

```
Команда сформировала готовый секрет и отдала его вам, зашифровав данные в base64. Сработало это благодаря флагам `--dry-run=true` и `-o yaml`, первый флаг говорит о том что мы хотим сымитировать создание сущности в кластере без доступа к нему, а второй о том что мы хотим видеть наши данные в формате `yaml`

Теперь вам осталось только создать отдельный файл Secret.yaml и положить в него содержимое которое выдала вам команда, предварительно удалив строку `creationTimestamp: null`.

P.S. Настоятельно не рекомендуем хранить данные подключения в сыром виде в котором нам выдала команда, о том каким образом можно зашифровать данные с помощью werf будет показано в главе [Секретные переменные](####секретные-переменные).


##### Переменные окружения

Для корректной работы с приложениями вам может понадобиться указать переменные окружения.
Например, наличие режима отладки в зависимости от окружения.

И эти переменные можно параметризовать с помощью файла `values.yaml`.

Так например, мы пробросим значение переменной NODE_ENV в наш контейнер из `values.yaml`

```yaml
app:
  debug:
    stage: "True"
    production: "False"
```
И теперь добавляем переменную в наш Deployment.
```yaml
          - name: DEBUG
            value: {{ pluck .Values.global.env .Values.app.debug | first }}
```
Конструкция указывает на то что в зависимости от значения .Values.global.env мы будем подставлять первое совпадающее значение из .Values.app.debug

Werf устанавливает значение .Values.global.env в зависимости от названия окружения указанного в .gitlab-ci.yml в стадии деплоя.

Теперь перейдем к описанию шаблона из предыдущей главы:

```yaml
        env:
{{ tuple "node" . | include "werf_container_env" | indent 8 }}
```
Werf закрывает ряд вопросов, связанных с перевыкатом контейнеров с помощью конструкции  [werf_container_env](https://ru.werf.io/documentation/reference/deploy_process/deploy_into_kubernetes.html#werf_container_env). Она возвращает блок с переменной окружения DOCKER_IMAGE_ID контейнера пода. Значение переменной будет установлено только если .Values.global.werf.is_branch=true, т.к. в этом случае Docker-образ для соответствующего имени и тега может быть обновлен, а имя и тег останутся неизменными. Значение переменной DOCKER_IMAGE_ID содержит новый ID Docker-образа, что вынуждает Kubernetes обновить объект.

Важно учесть что данный параметр не подставляет ничего при использовании стратегии тэгирования `stages-signature`, но мы настоятельно рекомендуем добавить его внутрь манифеста для удобства интеграции будущих обновлений werf.

Аналогично можно пробросить секретные переменные (пароли и т.п.) и у Верфи есть специальный механизм для этого. Но к этому вопросу мы вернёмся позже.

##### Логгирование

Важно знать что ни в коем случае нельзя записывать логи приложения в файл внутри контейнера. Это приведет к бесконтрольному росту занятого места внутри контейнера и соотвественно места на ноде кластера Kubernetes, после того как занятое место достигнет критической точки, Kubernetes начнет удалять docker контейнеры на данной ноде и перезапускать их на других нодах, тем самым пытаясь расчистить место. Таким образом вы не только потеряете все накопленные логи, но и вызовите не нужные рестарты.

Мы предлагаем:

1. Писать все логи в stdout контейнера и чтобы оттуда их собирал сторонний сервис.

Писать наши логи в stdout из Nodejs можно таким образом:

```js
console.log("I will goto the STDOUT");
console.error("I will goto the STDERR");
```
А для сбора логов можно использовать [fluentd](https://docs.fluentd.org/v/0.12/articles/kubernetes-fluentd).
И с его помощью отправлять логи в любое удобное вам хранилище, например [Elasticsearch](https://www.elastic.co/guide/index.html).

2. Ограничить их количество в stdout с помощью настройки для Docker в /etc/docker/daemon.json


Как писать в stdout????



Что за сервисы будут это принимать???




Докер ебокер


```json
{
        "log-driver": "json-file",
        "log-opts": {
                "max-file": "5",
                "max-size": "10m"
        }
}
```
В общей сложности конструкция выше понятна, но если вы хотите разобрать её подробнее вы можете обратиться к официальной [документации](https://docs.docker.com/config/containers/logging/configure/).



##### Направление трафика на приложение

Для того чтобы запросы извне попали к нам в приложение нужно открыть порт у пода, привязать к поду сервис и настроить Ingress, который выступает у нас в качестве балансера.

Если вы мало работали с Kubernetes — эта часть может вызвать у вас много проблем. Большинство тех, кто начинает работать с Kubernetes по невнимательности допускают ошибки при конфигурировании labels и затем занимаются долгой и мучительной отладкой.


###### Проброс портов

Для того чтобы мы смогли общаться с нашим приложением извне необходимо привязать к нашему deployment объект Service.

В наш service.yaml нужно добавить:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Chart.Name }}
spec:
  selector:
    app: {{ .Chart.Name }}
  clusterIP: None
  ports:
  - name: http
    port: 8080
    protocol: TCP
```
Обязательно нужно указывать порты, на котором будет слушать наше приложение внутри контейнера. И в Service, как указано выше и в Deployment:

```yaml
        ports:
        - containerPort: 8080
          protocol: TCP
```

Сама же привязка к deployment происходит с помощью блока **selector:**


```yaml
  selector:
    app: {{ .Chart.Name }}
```


Внутри селектора у нас указан лэйбл `app: {{ .Chart.Name }}` он должен полностью совпадать с блоком `labels` в Deployment который мы описывали в главах выше:



```yaml
  template:
    metadata:
      labels:
        app: {{ $.Chart.Name }}
```


Иначе Kubernetes не поймет на какой именно под или совокупность подов Service указывать. Это важно еще и из-за того что ip адреса подов попадают в DNS Kubernetes под именем сервиса, что позволяет нам обращаться к поду с нашим приложения просто по имени сервиса.

Полная запись для пода в нашем случае будет выглядеть так:
`chat.stage.svc.cluster.local` и расшифровывается так - `имя_сервиса.имя_неймспейса.svc.cluster.local` - неизменная часть это стандартный корневой домен Kubernetes.

Интересно то что поды находящиеся внутри одного неймспейса могут обращаться друг к другу просто по имени сервиса.

Подробнее о том как работать с сервисами можно узнать в [документации](connect-applications-service).

###### Роутинг на Ingress

Теперь мы можем передать nginx ingress имя сервиса на который нужно проксировать запросы извне. Для этого в [ingress.yaml](/werf-articles/gitlab-nodejs-files/examples/ingress.yaml) опишем следующий манифест:
<details><summary>ingress.yaml</summary>
<p>

```yaml
---
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: nginx
  name: {{ .Chart.Name }}
spec:
  rules:
  - host: chat.example.com
    http:
      paths:
      - backend:
          serviceName: {{ .Chart.Name }}
          servicePort: 8080
        path: /
```
</p>
</details>

Настройка роутинга происходит непосредственно в блоке `rules:`, где мы можем описать правила по которму трафик будет попадать в наше приложение.

`- host: chat.example.com` - в данном поле мы описываем тот домен на который конечный пользователь будет обращаться чтобы попасть в наше приложение. Можно сказать что это точка входа в наше приложение.

`paths:` - отвечает за настройку путей внутри нашего домена. И принимает в себя список из конфигураций этих путей.
Далее мы прямо описываем что все запросы попадающие на корень `path: /`, мы отправляем на backend, которым выступает наш сервис:
```yaml
      - backend:
          serviceName: {{ .Chart.Name }}
          servicePort: 8080
        path: /
```
Имя сервиса и его порт должны полностью совпадать с теми что мы описывали в сущности Service.
Удобство в том что описаний таких бэкендов может быть множество. И вы можете на одном домене по разным путям направлять трафик в разные приложения. Как это делать будет описано в последующих главах.


#### Секретные переменные

Мы уже рассказывали о том как использовать обычные переменные в нашем СI забирая их напрямую из values.yaml. Суть работы с секретными переменными абсолютно та же, единственное что в репозитории они будут храниться в зашифрованном виде.

Потому для хранения в репозитории паролей, файлов сертификатов и т.п., рекомендуется использовать подсистему работы с секретами werf.

Идея заключается в том, что конфиденциальные данные должны храниться в репозитории вместе с приложением, и должны оставаться независимыми от какого-либо конкретного сервера.


Для этого в werf существует инструмент [helm secret](https://werf.io/documentation/reference/deploy_process/working_with_secrets.html). Чтобы воспользоваться шифрованием нам сначала нужно создать ключ, сделать это можно так: 

```bash
$ werf helm secret generate-secret-key
ad747845284fea7135dca84bde9cff8e
$ export WERF_SECRET_KEY=ad747845284fea7135dca84bde9cff8e
```

После того как мы сгенерировали ключ, добавим его в переменные окружения у себя локально.

Секретные данные мы можем добавить создав рядом с values.yaml файл secret-values.yaml

Теперь использовав команду:


```bash
$ werf helm secret values edit ./helm/secret-values.yaml
```


Откроется текстовый редактор по-умолчанию, где мы сможем добавить наши секретные данные как обычно:


```yaml
app:
  s3:
    access_key:
      _default: bNGXXCF1GF
    secret_key:
      _default: zpThy4kGeqMNSuF2gyw48cOKJMvZqtrTswAQ
```


После того как вы закроете редактор, werf зашифрует их и secret-values.yaml будет выглядеть так:

![svalues](/werf-articles/gitlab-nodejs-files/images/secret_values.png "svalues")

И вы сможете добавить их в переменные окружения в Deployment точно так же как делали это с обычными переменными. Главное это не забыть добавить ваш WERF_SECRET_KEY в переменные репозитория гитлаба, найти их можно тут Settings -> CI/CD -> Variables. Настройки репозитория доступны только участникам репозитория с ролью выше Administrator, потому никто кроме доверенных лиц не сможет получить наш ключ. А werf при деплое нашего приложения сможет спокойно получить ключ для расшифровки наших переменных.

#### Деплой в Gitlab CI

Теперь мы наконец приступаем к описанию стадии выката. Потому продолжаем нашу работу в gitlab-ci.yml.

Мы уже решили, что у нас будет два окружения, потому под каждое из них мы должны описать свою стадию, но в общей сложности они будут отличаться только параметрами, потому мы напишем для них небольшой шаблон:

```yaml
.base_deploy: &base_deploy
  script:
    - werf deploy --stages-storage :local 
      --set "global.ci_url=$(cut -d / -f 3 <<< $CI_ENVIRONMENT_URL)"
  dependencies:
    - Build
  tags:
    - werf
```

Скрипт стадий выката отличается от сборки всего одной командой:

```yaml
    - werf deploy --stages-storage :local
      --set "global.ci_url=$(cut -d / -f 3 <<< $CI_ENVIRONMENT_URL)"
```

И тут назревает вполне логичный вопрос.

Как werf понимает куда нужно будет деплоить и каким образом? На это есть два ответа.

Первый из них вы уже видели и заключается он в команде `werf ci-env` которая берёт нужные переменные прямиком из pipeline Gitlab - и в данном случае ту что касается названия окружения.

А второй это описание стадий выката в нашем gitlab-ci.yml:

```yaml
Deploy to Stage:
  extends: .base_deploy
  stage: deploy
  environment:
    name: stage
    url: https://stage.example.com
  only:
    - merge_requests
  when: manual

Deploy to Production:
  extends: .base_deploy
  stage: deploy
  environment:
    name: production
    url: http://example.com
  only:
    - master
```

Описание деплоя содержит в себе немного. Скрипт, указание принадлежности к стадии **deploy**, которую мы описывали в начале gitlab-ci.yml, и **dependencies** что означает что стадия не может быть запущена без успешного завершения стадии **Build**. Также мы указали с помощью **only**, ветку _master_, что означает что стадия будет доступна только из этой ветки. **environment** указали потому что werf необходимо понимать в каком окружении он работает. В дальнейшем мы покажем, как создать CI для нескольких окружений. Остальные параметры вам уже известны.

И что не мало важно **url** указанный прямо в стадии. 
1. Это добавляет в MR и pipeline дополнительную кнопку по которой мы можем сразу попасть в наше приложение. Что добавляет удобства.
2. С помощью конструкции `--set "global.ci_url=$(cut -d / -f 3 <<< $CI_ENVIRONMENT_URL)"` мы добавляем адрес в глобальные переменные проекта и затем можем например использовать его динамически в качестве главного домена в нашей сущности Ingress:
```yaml
      - host: {{ .Values.global.ci_url }}
```
По умолчанию деплой будет происходить в namespace состоящий из имени проекта задаваемого в `werf.yaml` и имени окружения задаваемого в `.gitlab-ci.yml` куда мы деплоим наше приложение.

Ну а теперь достаточно создать Merge Request и нам будет доступна кнопка Deploy to Stage.

![alt_text](images/-6.png "image_tooltip")

Посмотреть статус выполнения pipeline можно в интерфейсе gitlab **CI / CD - Pipelines**

![alt_text](images/-7.png "image_tooltip")


Список всех окружений - доступен в меню **Operations - Environments**

![alt_text](images/-8.png "image_tooltip")

Из этого меню - можно так же быстро открыть приложение в браузере.

{{И тут в итоге должна быть картинка как аппка задеплоилась и объяснение картинки}}

# Подключаем зависимости

Werf подразумевает, что лучшей практикой будет разделить сборочный процесс на этапы, каждый с четкими функциями и своим назначением. Каждый такой этап соответствует промежуточному образу, подобно слоям в Docker. В werf такой этап называется стадией, и конечный образ в итоге состоит из набора собранных стадий. Все стадии хранятся в хранилище стадий, которое можно рассматривать как кэш сборки приложения, хотя по сути это скорее часть контекста сборки.

Стадии — это этапы сборочного процесса, кирпичи, из которых в итоге собирается конечный образ. Стадия собирается из группы сборочных инструкций, указанных в конфигурации. Причем группировка этих инструкций не случайна, имеет определенную логику и учитывает условия и правила сборки. С каждой стадией связан конкретный Docker-образ. Подробнее о том, какие стадии для чего предполагаются можно посмотреть в [документации](https://ru.werf.io/documentation/reference/stages_and_images.html).

Werf предлагает использовать для стадий следующую стратегию:

*   использовать стадию beforeInstall для инсталляции системных пакетов;
*   использовать стадию install для инсталляции системных зависимостей и зависимостей приложения;
*   использовать стадию beforeSetup для настройки системных параметров и установки приложения;
*   использовать стадию setup для настройки приложения.

Подробно про стадии описано в [документации](https://ru.werf.io/documentation/configuration/stapel_image/assembly_instructions.html).

Одно из основных преимуществ использования стадий в том, что мы можем не перезапускать нашу сборку с нуля, а перезапускать её только с той стадии, которая зависит от изменений в определенных файлах.

В нашем случае в качестве примера мы можем взять файл `package.json`.

Те кто уже сталкивался с разработкой на nodejs приложений знают, что в файле `package.json` указываются зависимости которые нужны для сборки приложения. Потому самое логичное указать данный файл в зависимости сборки, чтобы в случае изменений в нём, была перезапущена сборка только со стадии **_install_**.

Для этого в одной из первых глав мы сразу и добавляли наш файл package.json в зависимости werf.

Код из [werf.yaml](/werf-articles/gitlab-nodejs-files/examples/werf_1.yaml):


```yaml
  stageDependencies:
    install:
    - package.json
```
Тем самым при изменении данного файла между сборками, Werf это увидит, и перезапустит стадию в которой мы прописали установку пакетов.

```yaml
  install:
  - name: npm сi
    shell: npm сi
    args:
      chdir: /app
```

# Генерируем и раздаем ассеты

В какой-то момент в процессе разработки вам понадобятся ассеты (т.е. картинки, css, js).

Для генерации ассетов мы будем использовать webpack.

Интуитивно понятно, что на стадии сборки нам надо будет вызвать скрипт, который генерирует файлы, т.е. что-то надо будет дописать в `werf.yaml`. Однако, не только там — ведь какое-то приложение в production должно непосредственно отдавать статические файлы. Мы не будем отдавать файлики с помощью Express Nodejs. Хочется, чтобы статику раздавал nginx. А значит надо будет внести какие-то изменения и в helm чарты.

## Сценарий сборки ассетов

Webpack собирает ассеты используя конфигурацию указанную в webpack.config.js

Запуск генерации с webpack вставляют сразу в package.json как скрипт:



```json
  "scripts": {
    "test": "echo \"Error: no test specified\" && exit 1",
    "build": "webpack --mode production",
    "watch": "webpack --mode development --watch",
    "start": "webpack-dev-server --mode development --open",
    "clear": "del-cli dist",
    "migrate": "node-pg-migrate"
  },
```

Обычно используется несколько режимов для удобной разработки, но конечный вариант который идёт в kubernetes (в любое из окружений) всегда сборка как в production. Остальная отладка производится только локально. Почему необходимо чтобы сборка всегда была одинаковой?

Потому что наш docker image между окружениями должен быть неизменяемым. Создано это для того чтобы мы всегда были уверены в нашей сборке, т.е. образ оттестированный на stage окружении должен попадать в production точно таким же. Для всего остального мира наше приложение должно быть чёрным ящиком, которое лишь может принимать параметры.

Если же вам необходимо иметь внутри вашего кода ссылки или любые други изменяемые между окружениями объекты, то сделать это можно несколькими способами:

1. Мы можем динамически в зависимости от окружение монтировать в контейнер с нашим приложением json с нужными параметрами. Для этого нам нужно создать объект configmap в .helm/templates.

**10-app-config.yaml**

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Chart.Name }}-config
data:
  config.json: |-
   {
    "domain": "{{ pluck .Values.global.env .Values.domain | first }}",
    "loginUrl": "{{ pluck .Values.global.env .Values.loginUrl | first }}"
   }
```


А затем примонтировать к нашему приложению в то место где мы могли получать его по запросу от клиента: \
Код из 01-app.yaml:


```yaml
       volumeMounts:
          - name: app-config
            mountPath: /app/dist/config.json
            subPath: config.json
      volumes:
        - name: app-config
          configMap:
            name: {{ .Chart.Name }}-config
```
После того как конфиг окажется внутри вашего приложения вы сможете обращатся к нему как обычно.

2. И второй вариант перед запуском приложения получать конфиги из внешнего ресурса,например из [consul](https://www.consul.io/). Подробно не будем расписывать данный вариант, так как он достоин отдельной главы. Дадим лишь два совета: 

*   Можно запускать его перед запуском приложения добавив в `command: ["node","/app/src/js/server.js"] `его запуск через `&&` как в синтаксисе любого shell языка, прямо в манифесте описывающем приложение:
```yaml
command: 
- /usr/bin/bash
- -c
- --
- "consul kv get app/config/urls && node /app/src/js/server.js"]
```


*   Либо добавив его запуск в [init-container](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) и подключив между инит контейнером и основным контейнером общий [volume](https://kubernetes.io/docs/concepts/storage/volumes/), это означает что просто нужно смонтировать volume в оба контейнера. Например [emptyDir](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir). Тем самым консул отработав в инит контейнере, сохранит конфиг в наш volume, а приложение из основного контейнера просто заберёт этот конфиг. 


## Какие изменения необходимо внести

Генерация ассетов происходит в image `node` на стадии `setup`, так как данная стадия рекомендуется для настройки приложения

Для уменьшения нагрузки на процесс основного приложения которое обрабатыаем логику работы Nodejs приложения мы будем отдавать статические файлы через `nginx`
Мы запустим оба контейнера одним деплойментом, а запросы разделим на уровне сущности Ingress.

### Изменения в сборке

Для начала добавим стадию сборки в наш docker image в файл [werf.yaml](/werf-articles/gitlab-nodejs-files/examples/werf_2.yaml)

```yaml
  setup:
  - name: npm run build
    shell: npm run build
    args:
      chdir: /app
```
Далее пропишем отдельный образ с nginx и перенесём в него собранную статику из нашего основного образа:
```yaml
---
image: node_assets
from: nginx:stable-alpine
docker:
  EXPOSE: '80'
import:
- image: node
  add: /app/dist/assets
  to: /usr/share/nginx/html/assets
  after: setup
```
Вы можете заметить два абсолютно новых поля конфигурации.
`import:` - позволяет импортировать директории из других ранее собраных образов.
Внутри мы можем списком указать образа с директориями которые мы хотим забрать.
В нашем случае мы берем наш образ `image:node` и берем из него директорию `add: /app/dist/assets` в которую webpack положил сгенерированную статику, и копируем её в `/usr/share/nginx/html/assets` в образе с nginx, и что примечательно пишем стадию после которой нам нужно файлы импортировать `after:setup`. В нашем случае мы не описывали никаких стадий, потому мы указали выполнить импорт после последней стадии по-умолчанию.

### Изменения в деплое

При таком подходе изменим деплой нашего приложения добавив еще один контейнер в наш деплоймент с приложением.  Укажем livenessProbe и readinessProbe, которые будут проверять корректную работу контейнера в поде. preStop команда необходима для корректного завершение процесса nginx. В таком случае при новом выкате новой версии приложения будет корректное завершение всех активных сессий.

```yaml
      - name: assets
{{ tuple "node_assets" . | include "werf_container_image" | indent 8 }}
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

В описании сервиса - так же должен быть указан правильный порт

```yaml
  ports:
  - name: http
    port: 80
    protocol: TCP
```


### Изменения в роутинге

Поскольку у нас маршрутизация запросов происходит черех nginx контейнер а не на основе ingress ресурсов - нам необходимо только указать коректный порт для сервиса

```yaml
      paths:
      - path: /
        backend:
          serviceName: {{ .Chart.Name }}
          servicePort: 80
```

Если мы хотим разделять трафик на уровне ingress - нужно разделить запросы по path и портам

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

# Работа с файлами

В разработке может встретиться возможность когда требуется сохранять загружаемые пользователями файлы. Встает резонный вопрос о том каким образом их нужно хранить, и как после этого получать.

Первый и более общий способ. Это использовать как volume в подах [NFS](https://kubernetes.io/docs/concepts/storage/volumes/#nfs), [CephFS](https://kubernetes.io/docs/concepts/storage/volumes/#cephfs) или [hostPath](https://kubernetes.io/docs/concepts/storage/volumes/#hostpath), который будет направлен на директорию на ноде, куда будет подключено одно из сетевых хранилищ.

Мы не рекомендуем этот способ, потому что при возникновении неполадок с такими типами volume’ов мы будем влиять на работоспособность всего докера, контейнера и демона docker в целом, тем самым могут пострадать приложения которые даже не имеют отношения к вашему приложению.

Мы рекомендуем пользоваться S3. Такой способ выглядит намного надежнее засчет того что мы используем отдельный сервис, который имеет свойства масштабироваться, работать в HA режиме, и будет иметь высокую доступность.

Есть cloud решения S3, такие как AWS S3, Google Cloud Storage, Microsoft Blobs Storage и т.д. которые будут самым надежным решением из всех что мы можем использовать.

Но для того чтобы просто посмотреть на то как работать с S3 или построить собственное решение, хватит Minio.


## Подключаем наше приложение к S3 Minio


Сначала с помощью npm устанавливает пакет, который так и называется Minio.


```bash
$ npm install minio --save
```


После в исходники в src/js/index.js мы добавляем следующие строки: 



```js
const Minio = require("minio");
const S3_ENDPOINT = process.env.S3_ENDPOINT || "127.0.0.1";
const S3_PORT = Number(process.env.S3_PORT) || 9000;
const TMP_S3_SSL = process.env.S3_SSL || "true";
const S3_SSL = TMP_S3_SSL.toLowerCase() == "true";
const S3_ACCESS_KEY = process.env.S3_ACCESS_KEY || "SECRET";
const S3_SECRET_KEY = process.env.S3_SECRET_KEY || "SECRET";
const S3_BUCKET = process.env.S3_BUCKET || "avatars";
const CDN_PREFIX = process.env.CDN_PREFIX || "http://127.0.0.1:9000";

// S3 client
var s3Client = new Minio.Client({
  endPoint: S3_ENDPOINT,
  port: S3_PORT,
  useSSL: S3_SSL,
  accessKey: S3_ACCESS_KEY,
  secretKey: S3_SECRET_KEY,
});
```


И этого вполне хватит для того чтобы вы могли использовать minio S3 в вашем приложении.

Полный пример использования можно посмотреть в тут.

Важно заметить, что мы не указываем жестко параметры подключения прямо в коде, а производим попытку получения их из переменных. Точно так же как и с генерацией статики тут нельзя допускать фиксированных значений. \
 \
Остается только настроить наше приложение со стороны Helm.

Добавляем значения в values.yaml


```yaml
app:
  s3:
    host:
      _default: chat-test-minio
    port:
      _default: 9000
    bucket:
      _default: 'avatars'
    ssl:
      _default: 'false'
```
И в secret-values.yaml

```yaml
app:
  s3:
    access_key:
      _default: bNGXXCF1GF
    secret_key:
      _default: zpThy4kGeqMNSuF2gyw48cOKJMvZqtrTswAQ
```
Далее мы добавляем переменные непосредственно в Deployment с нашим приложением: \



```yaml
        - name: CDN_PREFIX
          value: {{ printf "%s%s" (pluck .Values.global.env .Values.app.cdn_prefix | first | default .Values.app.cdn_prefix._default) (pluck .Values.global.env .Values.app.s3.bucket | first | default .Values.app.s3.bucket._default) | quote }}
        - name: S3_SSL
          value: {{ pluck .Values.global.env .Values.app.s3.ssl | first | default .Values.app.s3.ssl._default | quote }}
        - name: S3_ENDPOINT
          value: {{ pluck .Values.global.env .Values.app.s3.host | first | default .Values.app.s3.host._default }}
        - name: S3_PORT
          value: {{ pluck .Values.global.env .Values.app.s3.port | first | default .Values.app.s3.port._default | quote }}
        - name: S3_ACCESS_KEY
          value: {{ pluck .Values.global.env .Values.app.s3.access_key | first | default .Values.app.s3.access_key._default }}
        - name: S3_SECRET_KEY
          value: {{ pluck .Values.global.env .Values.app.s3.secret_key | first | default .Values.app.s3.secret_key._default }}
        - name: S3_BUCKET
          value: {{ pluck .Values.global.env .Values.app.s3.bucket | first | default .Values.app.s3.bucket._default }}
```
Тот способ которым мы передаем переменные в под, с помощью GO шаблонов означает:
  1. `{{ pluck .Values.global.env .Values.app.s3.access_key | first` пробуем взять из поля _app.s3.access_key_ значение из поля которое равно environment которое werf берет из текущей стадии в .gitlab-ci.yml.

  2. `default .Values.app.s3.access_key._default }} `и если такого нет, то мы берём значение из поля _default.

И всё, этого достаточно!

# Работа с электронной почтой

Для того чтобы использовать почту мы предлагаем лишь один вариант - использовать внешнее API. В нашем примере это mailgun.
Внутри исходного кода подключение к API и отправка сообщения может выглядеть так:
```js
function sendMessage(message) {
  try {
    const mg = mailgun({apiKey: process.env.MAILGUN_APIKEY, domain: process.env.MAILGUN_DOMAIN});
    const email = JSON.parse(message.content.toString());
    email.from = "Mailgun Sandbox <postmaster@sandbox"+process.env.MAILGUN_FROM+">",
    email.subject = "Welcome to Chat!"
    mg.messages().send(email, function (error, body) {
      console.log(body);
    });
  } catch (error) {
    console.error(error)
  }
}

```
Главное заметить, что мы также как и в остальных случаях выносим основную конфигурацию в переменные. А далее мы по тому же принципу добавляем их параметризированно в наш secret-values.yaml не забыв использовать [шифрование](####секретные-переменные).
```yaml
  mailgun_apikey:
    _default: 192edaae18f13aaf120a66a4fefd5c4d-7fsaaa4e-kk5d08a5
  mailgun_domain:
    _default: sandboxf1b90123966447a0514easd0ea421rba.mailgun.org
```
И теперь мы можем использовать их внутри манифеста.
```yaml
        - name: MAILGUN_APIKEY
          value: {{ pluck .Values.global.env .Values.app.mailgun_apikey | first | default .Values.app.mailgun_apikey._default }}
        - name: MAILGUN_DOMAIN
          value: {{ pluck .Values.global.env .Values.app.mailgun_domain | first | default .Values.app.mailgun_domain._default | quote }}
```

# Подключаем redis

Допустим к нашему приложению нужно подключить простейшую базу данных, например, redis или memcached. Возьмем первый вариант.

В простейшем случае нет необходимости вносить изменения в сборку — всё уже собрано для нас. Надо просто подключить нужный образ, а потом в вашем Nodejs приложении корректно обратиться к этому приложению.

## Завести Redis в Kubernetes

Есть два способа подключить: прописать helm-чарт самостоятельно или подключить внешний. Мы рассмотрим второй вариант.

Подключим redis как внешний subchart.

Для этого нужно:

1. прописать изменения в yaml файлы;
2. указать редису конфиги
3. подсказать werf, что ему нужно подтягивать subchart.

Добавим в файл `.helm/requirements.yaml` следующие изменения:

```yaml
dependencies:
- name: redis
  version: 9.3.2
  repository: https://kubernetes-charts.storage.googleapis.com/
  condition: redis.enabled
```

Для того чтобы werf при деплое загрузил необходимые нам сабчарты - нужно добавить команды в `.gitlab-ci`

```yaml
.base_deploy:
  stage: deploy
  script:
    - werf helm repo init
    - werf helm dependency update
    - werf deploy
```

Опишем параметры для redis в файле `.helm/values.yaml`

```yaml
redis:
  enabled: true
```

При использовании сабчарта по умолчанию создается master-slave кластер redis.

Если посмотреть на рендер (`werf helm render`) нашего приложения с включенным сабчартом для redis, то можем увидеть какие будут созданы сервисы:

```yaml
# Source: example-2/charts/redis/templates/redis-master-svc.yaml
apiVersion: v1
kind: Service
metadata:
  name: chat-stage-redis-master

# Source: example-2/charts/redis/templates/redis-slave-svc.yaml
apiVersion: v1
kind: Service
metadata:
  name: chat-stage-redis-slave
```

## Подключение Nodejs приложения к базе redis

В нашем приложении - мы будем  подключаться к мастер узлу редиса. Нам нужно, чтобы при выкате в любое окружение приложение подключалось к правильному редису.

В src/js/index.js мы добавляем:


```js
const REDIS_URI = process.env.SESSION_REDIS || "redis://127.0.0.1:6379";
const SESSION_TTL = process.env.SESSION_TTL || 3600;
const COOKIE_SECRET = process.env.COOKIE_SECRET || "supersecret";
// Redis connect
const expSession = require("express-session");
const redis = require("redis");
let redisClient = redis.createClient(REDIS_URI);
let redisStore = require("connect-redis")(expSession);

var session = expSession({
  store: new redisStore({ client: redisClient, ttl: SESSION_TTL }),
  secret: "keyboard cat",
  resave: false,
  saveUninitialized: false,
});
var sharedsession = require("express-socket.io-session");
app.use(session);
```

Добавляем в values.yaml

values.yaml

```yaml
app:
  redis:
     host:
       master:
         stage: chat-stage-redis-master
       slave:
         stage: chat-stage-redis-slave
```


secret-values.yaml


```yaml
app:
 redis:
    password:
      _default: 100067e35229a23c5070ad5407b7406a7d58d4e54ecfa7b58a1072bc6c34cd5d443e
```


И наконец добавляем подстановку переменных в сам манифест с нашим приложением: 



```yaml
      - name: SESSION_REDIS
        value: "redis://root:{{ pluck .Values.global.env .Values.app.redis.password | first | default .Values.app.redis.password._default }}@{{ pluck .Values.global.env .Values.app.redis.host.master | first }}:6379"
```



В данном случае Redis подключается как хранилище для сессий.

# Подключаем базу данных

Для текущего примера в приложении должны быть установлены необходимые зависимости. В данном случае мы рассмотрим подключение нашего приложения на базе пакета npm `pg`.


## Как подключить БД

Внутри кода приложения подключение нашей базы будет выглядеть так:

```js
const pgconnectionString =
  process.env.DATABASE_URL || "postgresql://127.0.0.1/postgres";

// Postgres connect
const pool = new pg.Pool({
  connectionString: pgconnectionString,
});
pool.on("error", (err, client) => {
  console.error("Unexpected error on idle client", err);
  process.exit(-1);
});
```


В данном случае мы также используем сабчарт для деплоя базы из того же репозитория. Этого должно хватить для нашего небольшого приложения. В случае большой высоконагруженной инфраструктуры деплой базы непосредственно в кубернетес не рекомендуется. 

Добавляем информацию о подключении в values.yaml


```yaml
app:
 postgresql:
    host:
      stage: chat-test-postgresql
      production: postgres
    user: chat
    db: chat
```


И в secret-values.yaml


```yaml
app:
  postgresql:
    password:
      stage: 1000acb579eaee19bec317079a014346d6aab66bbf84e4a96b395d4a5e669bc32dd1
      production: 1000acb5f9eaee19basdc3127sfa79a014346qwr12b66bbf84e4a96b395d4a5e631255ad1
```


Далее привносим подключени внутрь манифеста нашего приложения переменную подключения:



```yaml
       - name: DATABASE_URL
         value: "postgresql://{{ .Values.app.postgresql.user }}:{{ pluck .Values.global.env .Values.app.postgresql.password | first }}@{{ pluck .Values.global.env .Values.app.postgresql.host | first }}:5432/{{ .Values.app.postgresql.db }}"
```


## Выполнение миграций

Для выполнения миграций в БД мы используем пакет `node-pg-migrate`, на его примере и будем рассматривать выполнение миграций.

Запуск миграции мы помещаем в package.json, чтобы его можно было  вызывать с помощью скрипта в npm: 
```json
   "migrate": "node-pg-migrate"
```
Сама конфигурация миграций у нас находится в отдельной директории `migrations`, которую мы создали на уровне исходного кода приложения.
```
node
├── migrations
│   ├── 1588019669425_001-users.js
│   └── 1588172704904_add-avatar-status.js
├── src
├── package.json
...
```

Далее нам необходимо добавить запуск миграций непосредственно в Kubernetes.
Запуск миграций производится созданием сущности Job в kubernetes. Это единоразовый запуск пода с необходимыми нам контейнерами.

Добавим запуск миграций после каждого деплоя приложения. Потому создадим в нашей директории с манифестами еще один файл - [migrations.yaml](/werf-articles/gitlab-nodejs-files/examples/migrations.yaml)

<details><summary>migrations.yaml</summary>
<p>

```yaml
---
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ .Chart.Name }}-migrate-db
  annotations:
    "helm.sh/hook": post-install,post-upgrade
    "helm.sh/hook-weight": "2"
spec:
  backoffLimit: 0
  template:
    metadata:
      name: {{ .Chart.Name }}-migrate-db
    spec:
      initContainers:
      - name: wait-postgres
        image: postgres:12
        command:
          - "sh"
          - "-c"
          - "until pg_isready -h {{ pluck .Values.global.env .Values.app.postgresql.host | first }} -U {{ .Values.app.postgresql.user }}; do sleep 2; done;"
      containers:
      - name: node
{{ tuple "node" . | include "werf_container_image" | indent 8 }}
        command: ["npm", "migrate"]
        env:
        - name: DATABASE_URL
          value: "postgresql://{{ .Values.app.postgresql.user }}:{{ pluck .Values.global.env .Values.app.postgresql.password | first }}@{{ pluck .Values.global.env .Values.app.postgresql.host | first }}:5432/{{ .Values.app.postgresql.db }}"
{{ tuple "node" . | include "werf_container_env" | indent 10 }}
      restartPolicy: Never
```

</p>
</details>

Аннотации `"helm.sh/hook": post-install,post-upgrade` указывают условия запуска job а `"helm.sh/hook-weight": "2"` указывают на порядок выполнения (от меньшего к большему)
`backoffLimit: 0` - запрещает перезапускать наш под с Job, тем самым гарантируя что он выполнится всего один раз при деплойменте.

`initContainers:` - блок описывающий контейнеры которые будут отрабатывать единоразово перед запуском основных контейнеров. Это самый быстрый и удобный способ для подготовки наших приложений к запуску в реальном времени. В данном случае мы взяли официальный образ с postgres, для того чтобы с помощью его инструментов отследить запустилась ли наша база.
```yaml
        command:
          - "sh"
          - "-c"
          - "until pg_isready -h {{ pluck .Values.global.env .Values.app.postgresql.host | first }} -U {{ .Values.app.postgresql.user }}; do sleep 2; done;"
```
Мы используем цикл, который каждые 2 секунды будет проверять нашу базу на доступность. После того как команда `pg_isready` сможет выполнится успешно, инит контейнер завершит свою работу, а основной запустится и без проблем сразу сможет подключится к базе. 

При запуске миграций мы используем тот же самый образ что и в деплойменте. Различие только в запускаемых командах.


## Накатка фикстур при первом выкате

При первом деплое вашего приложения, вам может понадобится раскатить фикстуры. В нашем случае это дефолтный пользователь нашего чата.
Мы не будем расписывать подробного этот шаг, потому что он практически не отличается от предыдущего. 

Мы добавляем наши фикстуры в отдельную директорию `fixtures` также как это было с миграциями.
А их запуск добавляем в `package.json`
```json
    "fixtures": "node fixtures/01-default-user.js"

```
И на этом все, далее мы производим те же действия что мы описывали ранее. Готовые пример вы можете найти тут - [fixtures.yaml](/werf-articles/gitlab-nodejs-files/examples/fixtures.yaml)
# Юнит-тесты и Линтеры

В качестве примера того каким образом строить в нашем CI запуск юнит тестов, мы продемонстрируем запуск ESLint.

ESLint - это линтер для языка программирования JavaScript, написанный на Node.js.

Он чрезвычайно полезен, потому что JavaScript, будучи интерпретируемым языком, не имеет этапа компиляции и многие ошибки могут быть обнаружены только во время выполнения.

Мы точно также добавляем пакет с нашим линтером в package.json и создаем к нему конфигурационный файл .eslintrc.json

```
node
├── migrations
├── src
├── package.json
├── .eslintrc.json
...
```

Для того чтобы запустить наш линтер мы добавляем наш .gitlab-ci.yml отдельную стадию:

```yaml
Run_Tests:
  stage: test
  script:
    - werf run --stages-storage :local node -- npm run pretest
  tags:
    - werf
  needs: ["Build"]
```

и не забываем добавить её в список стадий:

```yaml
stages:
  - build
  - test
  - deploy
```

В данном случае мы после сборки нашего docker image просто запускаем его командой [werf run](https://ru.werf.io/documentation/cli/main/run.html).

При таком запуске наш kubernetes кластер не задействован.

Полная конфигурация линтера доступна в примерах, тут мы описали лишь концепцию в примерах.

# Несколько приложений в одной репе

Если в одном репозитории находятся несколько приложений например для backend и frontend необходимо использовать сборку приложения с несколькими образами.

Мы рассказывали [https://www.youtube.com/watch?v=g9cgppj0gKQ](https://www.youtube.com/watch?v=g9cgppj0gKQ) о том, почему и в каких ситуациях это — хороший путь для микросервисов.

Покажем это на примере нашего приложения на node которое отправляет в rabbitmq имена тех кто зашёл в наш чат, а приложение на Python будет работать в качестве бота который приветсвует всех зашедших.

## Сборка приложений

Сборка приложения с несколькими образами описана в [статье](https://ru.werf.io/documentation/guides/advanced_build/multi_images.html). На ее основе покажем наш пример для нашего приложения.

Структура каталогов будет организована следующим образом

```
├── .helm
│   ├── templates
│   └── values.yaml
├── node
├── python
└── werf.yaml
```
Мы должны переместить весь наш исходный код Nodejs в отдельную директорию `node` и
далее собирать её как обычно, но заменив в `werf.yaml` строки клонирования кода с корня `- add: /` на нашу директорию `- add: /node`.
Сборка приложения для python структурно практически не отличается от сборки описанной в Hello World, за исключением того что импорт из git будет из директории python.

Пример конечного [werf.yaml](/werf-articles/gitlab-nodejs-files/examples/werf_3.yaml)

Для запуска подготовленных приложений отдельными деплойментами, необходимо создать еще один файл, который будет описывать запуск бота - [bot-deployment.yaml](/werf-articles/gitlab-nodejs-files/examples/python-deployment.yaml).

<details><summary>bot-deployment.yaml</summary>
<p>

```yaml
{{ $rmq_user := pluck .Values.global.env .Values.app.rabbitmq.user | first | default .Values.app.rabbitmq.user._default }}
{{ $rmq_pass := pluck .Values.global.env .Values.app.rabbitmq.password | first | default .Values.app.rabbitmq.password._default }}
{{ $rmq_host := pluck .Values.global.env .Values.app.rabbitmq.host | first | default .Values.app.rabbitmq.host._default }}
{{ $rmq_port := pluck .Values.global.env .Values.app.rabbitmq.port | first | default .Values.app.rabbitmq.port._default }}
{{ $rmq_vhost := pluck .Values.global.env .Values.app.rabbitmq.vhost | first | default .Values.app.rabbitmq.vhost._default }}
{{ $db_user := pluck .Values.global.env .Values.app.postgresql.user | first | default .Values.app.postgresql.user._default }}
{{ $db_pass := pluck .Values.global.env .Values.app.postgresql.password | first | default .Values.app.postgresql.password._default }}
{{ $db_host := pluck .Values.global.env .Values.app.postgresql.host | first | default .Values.app.postgresql.host._default }}
{{ $db_port := pluck .Values.global.env .Values.app.postgresql.port | first | default .Values.app.postgresql.port._default }}
{{ $db_name := pluck .Values.global.env .Values.app.postgresql.db | first | default .Values.app.postgresql.db._default }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}-email-consumer
spec:
  revisionHistoryLimit: 3
  strategy:
    type: RollingUpdate
  replicas: 1
  selector:
    matchLabels:
      app: {{ $.Chart.Name }}-email-consumer
  template:
    metadata:
      labels:
        app: {{ $.Chart.Name }}-email-consumer
    spec:
      imagePullSecrets:
      - name: "registrysecret"
      containers:
      - name: {{ $.Chart.Name }}
{{ tuple "python" . | include "werf_container_image" | indent 8 }}
        workingDir: /app
        command: ["python","consumer.py"]
        ports:
        - containerPort: 5000
          protocol: TCP
        env:
        - name: AMQP_URI
          value: {{ printf "amqp://%s:%s@%s:%s/%s" $rmq_user $rmq_pass $rmq_host ($rmq_port|toString) $rmq_vhost }}
        - name: DATABASE_URL
          value: {{ printf "postgresql+psycorg2://%s:%s@%s:%s/%s" $db_user $db_pass $db_host ($db_port|toString) $db_name }}
{{ tuple "python" . | include "werf_container_env" | indent 8 }}
```

</p>
</details>

Обратите внимание что мы отделили переменные от деплоймента, мы сделали это специально, чтобы строка подключения к базам в переменных манифеста не была огромной. Потому мы выделили переменные и описали их вверху манифеста, а затем просто подставили в нужные нам строки.

Таким образом мы смогли собрать и запустить несколько приложений написанных на разных языках которые находятся в одном репозитории.

Если в вашей команды фуллстэки и/или она не очень большая и хочется видеть и выкатывать приложение целиком, может быть полезно разместить приложения на нескольких языках в одном репозитории.

К слову, мы рассказывали [https://www.youtube.com/watch?v=g9cgppj0gKQ](https://www.youtube.com/watch?v=g9cgppj0gKQ) о том, почему и в каких ситуациях это — хороший путь для микросервисов.




# Динамические окружения

Не редко необходимо разрабатывать и тестировать сразу несколько feature для вашего приложения, и нет понимания как это делать, если у вас всего два окружения. Разработчику или тестеру приходится дожидаться своей очереди на контуре stage и затем проводить необходимые манипуляции с кодом (тестирование, отладка, демонстрация функционала). Таким образом разработка сильно замедляется. 

Решением этой проблемы мы предлагаем использовать динамические окружения. Их суть в том что мы можем развернуть и погасить такие окружения в любой момент, тем самым разработчик может проверить работает ли его код развернув его в динамическое окружение, после убедившись, он может его погасить до тех пор пока его feature не будет смерджена в общий контур или пока не придет тестер, который сможет развернуть окружение уже для своих нужд.

Рассмотрим примеры того что мы должны добавить в наш .gitlab-ci.yml, чтобы наши динамические окружения заработали:


```yaml
Deploy to Review:
  extends: .base_deploy
  stage: deploy
  environment:
    name: review/${CI_COMMIT_REF_SLUG}
    url: http://${CI_COMMIT_REF_SLUG}.k8s.example.com
    on_stop: Stop Review
  only:
    - feature/*
  when: manual
```
На первый взгляд стадия не отличается ничем от тех что мы описывали ранее, но мы добавили зависимость `on_stop: Stop Review` простыми словами означающую, что мы будем останавливать наше окружение следующей стадией:

```yaml
Stop Review:
  stage: deploy
  variables:
    GIT_STRATEGY: none
  script:
    - werf dismiss --env $CI_ENVIRONMENT_SLUG --namespace ${CI_ENVIRONMENT_SLUG} --with-namespace
  when: manual
  environment:
    name: review/${CI_COMMIT_REF_SLUG}
    action: stop
  only:
    - feature/*
```
`GIT_STRATEGY: none` - говорит нашему ранеру, что check out нашего кода не требуется.

`werf dismiss` - отвечает за то чтобы удалить из кластера helm релиз с нашим приложением где `$CI_ENVIRONMENT_SLUG` это переменна описывающая название нашего окружения, но как мы видим что в названии нашего окружение имеется символ `/`, `_SLUG` переменные в Gitlab отвечают за то что заменяют все невалидные символы оригинальных переменных на `-`, тем самым позволяя нам избежать проблем с их использованием, особенно в kubernetes (т.к. символ `/` запрещен в названиях любых сущностей)

Вопрос в том зачем мы вообще использовали такой символ внутри названия окружения, дело в том что Gitlab может распределять свои окружения в директории, и тем самым мы отделили все динамические окружения и определили их в директорию `review`

Подробнее о том как описываются динамические окружения в CI можно найти [тут](https://docs.gitlab.com/ee/ci/yaml/#environmentaction)

При таком ci - мы можем выкатывать каждую ветку `feature/*` в отдельный namespace с изолированной базой данных, накатом необходимых миграций и например проводить тесты для данного окружения.

В репозитории с примерами будет реализовано отдельное приложение которое показывает реализацию данного подхода.

