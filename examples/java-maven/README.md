# Java Maven example

A Java 21 fixture with a JUnit test and multi-stage container build.

```sh
cd examples/java-maven
mvn -B test
docker build -t cloudivision-java:local .
docker run --rm cloudivision-java:local
```

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f examples/java-maven/cloudivision.yaml
```

Maven Central is the only external dependency and is free to use; image push is disabled.
