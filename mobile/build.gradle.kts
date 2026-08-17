// Top-level build file for PudimProductivity Android app.
plugins {
    id("com.android.application") version "9.3.0" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.4.10" apply false
    // OWASP dependency-check: vulnerability scanning of Android/Java dependencies.
    // Task: ./gradlew dependencyCheckAnalyze
    // 12.2.2 widens the NVD reference URL column (fixes the CVE-2026-6785/6786
    // 'Value too long' crash that broke 12.1.0). Do NOT bump to 13.0.0 yet: it
    // regresses no-API-key usage (#8715 — empty NVD key treated as invalid,
    // unfixed as of 2026-08).
    id("org.owasp.dependencycheck") version "12.2.2" apply false
}
