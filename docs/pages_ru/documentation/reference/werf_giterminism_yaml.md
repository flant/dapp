---
title: werf-giterminism.yaml
permalink: documentation/reference/werf_giterminism_yaml.html
description: Werf giterminism config file example
toc: false
---

Конфигурация `werf-giterminism.yaml` позволяет использовать определённый набор конфигурационных файлов из директории проекта и точечно включить функционал, который потенциально может зависеть от внешних факторов. Чтобы правила вступили в силу, этот файл должен быть добавлен в git-репозиторий проекта.

> Мы рекомендуем минимизировать использования конфигурации `werf-giterminism.yaml` для того, чтобы конфигурация проекта была надёжной и легко воспроизводимой

Все пути и глобы в конфигурации должны быть описаны относительно директории проекта.

{% include documentation/reference/werf_giterminism_yaml/table.html %}
