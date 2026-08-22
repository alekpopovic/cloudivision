# Frontend development

The web UI is an Angular standalone application written in TypeScript and styled
with Tailwind CSS. Angular services own API access, routes select pages, and
reactive forms drive create and approval flows.

Install dependencies and start the local server:

```sh
npm --prefix web ci
npm --prefix web start
```

By default the application reads runtime configuration from
`web/public/assets/config.json`. Set `apiBaseUrl` there for local development, or
use the Helm value `web.config.apiBaseUrl` in a cluster. Do not compile a production
API hostname into the TypeScript bundle.

Before committing frontend changes run:

```sh
npm --prefix web run build
npm --prefix web test -- --watch=false --browsers=ChromeHeadless
```

The current project does not define a separate lint script. UI error panels should
show the backend `code`, `message`, request ID and policy
violations without rendering secrets. Tailwind utility changes should retain focus
states, readable contrast, keyboard navigation and responsive layouts.
