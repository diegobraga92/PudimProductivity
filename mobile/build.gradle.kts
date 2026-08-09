// Top-level build file for PudimProductivity Android app.
plugins {
    id("com.android.application") version "9.1.1" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.2.21" apply false
    // OWASP dependency-check: vulnerability scanning of Android/Java dependencies.
    // Task: ./gradlew dependencyCheckAnalyze
    id("org.owasp.dependencycheck") version "12.1.0" apply false
}
