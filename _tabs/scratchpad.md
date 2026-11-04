---
layout: page
title: Scratchpad
icon: fas fa-flask
order: 5
---

A mix of tools I maintain, experiments I've shipped, and things I've learned along the way.

{% for section in site.data.projects %}
## {{ section.category }}

{% for project in section.projects %}
- [**{{ project.name }}**]({{ project.url }}) — {{ project.description }}{% if project.docs_url %} · [Docs]({{ project.docs_url }}){% endif %}
{% endfor %}
{% endfor %}
