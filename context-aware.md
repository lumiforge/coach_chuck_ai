## Минимальный must-have набор для нормального ADK-агента

* Разделить `Session / Memory / Artifacts / Working Context`.
* Собирать контекст через flow/processors.
* Фильтровать историю перед моделью.
* Включить compaction для длинных сессий.
* Хранить большие данные в Artifacts.
* Делать Memory retrieved-on-demand.
* Держать стабильный prefix для caching.
* Ограничивать handoff между агентами.
* Давать каждому агенту только минимально нужный контекст.

[Ссылка на статью о контексте](https://developers.googleblog.com/architecting-efficient-context-aware-multi-agent-framework-for-production/)
