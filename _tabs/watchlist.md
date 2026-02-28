---
layout: page
title: Watchlist
icon: fas fa-photo-film
order: 5
---

A curated list of talks and videos I find insightful.


{% assign watchlist_posts = site.posts | where_exp: "post", "post.categories contains 'Watchlist'" %}

{% if watchlist_posts.size > 0 %}
{% for post in watchlist_posts %}
- [`{{ post.title }}`]({{ post.url | relative_url }})
  <br>
  <small>{{ post.date | date: "%Y-%m-%d" }}{% if post.description %} - {{ post.description }}{% endif %}</small>
{% endfor %}
{% else %}
No entries yet.
{% endif %}
