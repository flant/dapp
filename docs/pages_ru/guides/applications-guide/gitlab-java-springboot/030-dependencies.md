---
title: Подключение зависимостей
sidebar: applications-guide
permalink: documentation/guides/applications-guide/gitlab-java-springboot/030-dependencies.html
layout: guide
---

{% filesused title="Файлы, упомянутые в главе" %}
- werf.yaml
{% endfilesused %}

В этой главе мы настроим в нашем базовом приложении работу с зависимостями. Важно корректно вписать зависимости в [стадии сборки](https://ru.werf.io/documentation/reference/stages_and_images.html), что позволит не тратить время на пересборку зависимостей тогда, когда зависимости не изменились.

{% offtopic title="Что за стадии?" %}
Werf подразумевает, что лучшей практикой будет разделить сборочный процесс на этапы, каждый с четкими функциями и своим назначением. Каждый такой этап соответствует промежуточному образу, подобно слоям в Docker. В werf такой этап называется стадией, и конечный образ в итоге состоит из набора собранных стадий. Все стадии хранятся в хранилище стадий, которое можно рассматривать как кэш сборки приложения, хотя по сути это скорее часть контекста сборки.

Стадии — это этапы сборочного процесса, кирпичи, из которых в итоге собирается конечный образ. Стадия собирается из группы сборочных инструкций, указанных в конфигурации. Причем группировка этих инструкций не случайна, имеет определенную логику и учитывает условия и правила сборки. С каждой стадией связан конкретный Docker-образ. Подробнее о том, какие стадии для чего предполагаются можно посмотреть в [документации](https://ru.werf.io/documentation/reference/stages_and_images.html).

Werf предлагает использовать для стадий следующую стратегию:

*   использовать стадию beforeInstall для инсталляции системных пакетов;
*   использовать стадию install для инсталляции системных зависимостей и зависимостей приложения;
*   использовать стадию beforeSetup для настройки системных параметров и установки приложения;
*   использовать стадию setup для настройки приложения.

Подробно про стадии описано в [документации](https://ru.werf.io/documentation/configuration/stapel_image/assembly_instructions.html).

Одно из основных преимуществ использования стадий в том, что мы можем не перезапускать нашу сборку с нуля, а перезапускать её только с той стадии, которая зависит от изменений в определенных файлах.
{% endofftopic %}

В Java, в частности в spring, в качестве менеджера зависимостей может использоваться maven, gradle. Мы будем, как и ранее использовать maven, но для gradle. Пропишем его использование в файле `werf.yaml` и затем оптимизируем его использование.

## Подключение менеджера зависимостей

Пропишем разрешение зависимостей в нужную стадию сборки в `werf.yaml`

{% snippetcut name="werf.yaml" url="files/examples/example_1/werf.yaml" %}
```yaml
    shell: |
      mvn -B -f pom.xml package dependency:resolve
```
{% endsnippetcut %}

Однако, если оставить всё так — стадия `beforeInstall` не будет запускаться при изменении `pom.xml` и любого кода в `src/`. Подобная зависимость пользовательской стадии от изменений [указывается с помощью параметра git.stageDependencies](https://ru.werf.io/documentation/configuration/stapel_image/assembly_instructions.html#%D0%B7%D0%B0%D0%B2%D0%B8%D1%81%D0%B8%D0%BC%D0%BE%D1%81%D1%82%D1%8C-%D0%BE%D1%82-%D0%B8%D0%B7%D0%BC%D0%B5%D0%BD%D0%B5%D0%BD%D0%B8%D0%B9-%D0%B2-git-%D1%80%D0%B5%D0%BF%D0%BE%D0%B7%D0%B8%D1%82%D0%BE%D1%80%D0%B8%D0%B8):

{% snippetcut name="werf.yaml" url="template-files/examples/example_1/werf.yaml#L10" %}
```yaml
git:
- add: /
  to: /app
  stageDependencies:
    setup:
    - pom.xml
    - src
```
{% endsnippetcut %}

При изменении файла `pom.xml` или любого из файлов в `src/` стадия `setup` будет запущена заново.

## Оптимизация сборки

Сборка занимает много времени, поэтому оптимизировать её — важная задача. Применим два приёма:

* Уменьшим объём скачиваемых файлов благодаря улучшенному использованию кэша maven.
* Усовершенствуем использование пользовательских стадий

Даже в пустом проекте сборщику нужно скачать приличное количество файлов. Cкачивать эти файлы раз за разом выглядит нецелесообразным, поэтому разумно **переиспользовать кэш в `.m2/repository` между сборками**. С помощью директивы `mount` будем хранить кэш на раннере:

{% snippetcut name="werf.yaml" url="gitlab-java-springboot-files/01-demo-optimization/werf.yaml:14-1" %}
```yaml
mount:
- from: build_dir
  to: /root/.m2/repository
```
{% endsnippetcut %}

**Усовершенствуем использование пользовательских стадий**: отделим resolve зависимостей от сборки jar — таким образом те коммиты, в которых правится исходный код, но не меняются зависимости, будут собираться быстрее.

{% snippetcut name="werf.yaml" url="gitlab-java-springboot-files/01-demo-optimization/werf.yaml:17-31" %}
```yaml
ansible:
  beforeSetup:
  - name: dependency resolve
    shell: |
      mvn -B -f pom.xml dependency:resolve
    args:
      chdir: /app
      executable: /bin/bash
  setup:
  - name: Build jar
    shell: |
      mvn -B -f pom.xml package
    args:
      chdir: /app
      executable: /bin/bash
```
{% endsnippetcut %}


<div>
    <a href="040-assets.html" class="nav-btn">Далее: Генерируем и раздаем ассеты</a>
</div>
