# Angular application example

A real standalone Angular 20 application with a component test, production build, and unprivileged nginx image.

```sh
cd examples/angular-app
npm ci
npm test -- --browsers=ChromeHeadless
npm run build
docker build -t cloudivision-angular:local .
```

Chrome or Chromium is required for the local Karma test. To exercise the repository's already-installed Angular toolchain as a dogfood check, run `npm --prefix web test -- --watch=false` from the repository root.

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f examples/angular-app/cloudivision.yaml
```

Dependencies come from the public npm registry; image push is disabled and no paid service is required.
