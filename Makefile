.PHONY: serve
serve:
	@bundle exec jekyll s

.PHONY: bundle
bundle:
	@bundle

.PHONY: post
post:
	@bundle exec jekyll compose "$(POST_NAME)"
