# Documentation site development

The `docs/` directory is both the documentation source and a Jekyll site. GitHub
Pages builds it on pull requests and deploys it after changes reach `main`.

## Run locally

Install Ruby 3.3 (matching the GitHub Pages builder) and Bundler, then run:

```sh
cd docs
bundle install
bundle exec jekyll build --baseurl ""
python3 -m http.server 4000 --directory _site
```

Open `http://127.0.0.1:4000`. The production site uses the `/cloudivision`
base URL; all theme assets and navigation links must use Jekyll's `relative_url`
filter so both environments work.

## Author a page

Markdown files do not need explicit front matter. The optional-front-matter and
titles-from-headings plugins render the first H1 as the page title. Add the page
to `_data/navigation.yml` when it belongs in the primary navigation.

Keep CI and CD terminology precise: CI creates artifacts; CD updates Git and
GitOps controllers deploy. Put reusable presentation rules in `assets/css`,
behavior in `assets/js`, and shared structure in `_layouts` or `_includes`.
