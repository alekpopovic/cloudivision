package io.cloudivision.examples;

import static org.junit.jupiter.api.Assertions.assertEquals;
import org.junit.jupiter.api.Test;

class AppTest {
    @Test
    void reportsHealthy() {
        assertEquals("ok", App.health());
    }
}
