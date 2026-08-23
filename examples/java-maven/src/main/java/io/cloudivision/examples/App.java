package io.cloudivision.examples;

public final class App {
    private App() {}

    public static String health() {
        return "ok";
    }

    public static void main(String[] args) {
        System.out.println(health());
    }
}
