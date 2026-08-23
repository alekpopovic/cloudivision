FROM maven:3.9-eclipse-temurin-21-alpine AS build
WORKDIR /src
COPY pom.xml ./
COPY src ./src
RUN mvn -B package

FROM eclipse-temurin:21-jre-alpine
RUN addgroup -S app && adduser -S -G app app
COPY --from=build --chown=app:app /src/target/java-api-1.0.0.jar /app.jar
USER app
ENTRYPOINT ["java", "-cp", "/app.jar", "io.cloudivision.examples.App"]
