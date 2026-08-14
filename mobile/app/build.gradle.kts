import java.io.FileInputStream
import java.util.Properties

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.plugin.compose")
    id("org.owasp.dependencycheck")
}

fun readLocalProperty(key: String): String? {
    return try {
        val props = Properties()
        FileInputStream(rootProject.file("local.properties")).use { props.load(it) }
        props.getProperty(key)
    } catch (_: Exception) {
        null
    }
}

android {
    namespace = "com.pudimproductivity"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.pudimproductivity"
        minSdk = 26
        targetSdk = 35
        versionCode = 1
        versionName = "0.0.1"

        // Backend URL.
        // Priority: -P flag > local.properties > default emulator loopback.
        //   ./gradlew -Papi.base.url=http://10.0.2.2:8080/api/v1 assembleDebug
        //  or in mobile/local.properties:
        //   api.base.url=http://192.168.3.99:8080/api/v1
        val apiBaseUrl: String = (project.findProperty("api.base.url") as String?)
            ?: readLocalProperty("api.base.url")
            ?: "http://10.0.2.2:8080/api/v1"
        buildConfigField("String", "API_BASE_URL", "\"$apiBaseUrl\"")

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    signingConfigs {
        // Release keystore. Supplied via env vars (CI secrets) or
        // mobile/local.properties (local builds):
        //   keystore.file=...            KEYSTORE_FILE
        //   keystore.password=...        KEYSTORE_PASSWORD
        //   keystore.key.alias=...       KEY_ALIAS
        //   keystore.key.password=...    KEY_PASSWORD
        // With no keystore configured the release build is unsigned (dev-only).
        create("release") {
            val storePath = System.getenv("KEYSTORE_FILE") ?: readLocalProperty("keystore.file")
            if (storePath != null) {
                storeFile = file(storePath)
                storePassword = System.getenv("KEYSTORE_PASSWORD") ?: readLocalProperty("keystore.password") ?: ""
                keyAlias = System.getenv("KEY_ALIAS") ?: readLocalProperty("keystore.key.alias") ?: ""
                keyPassword = System.getenv("KEY_PASSWORD") ?: readLocalProperty("keystore.key.password") ?: ""
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            // Wire the keystore only when one is configured; otherwise the
            // release build remains unsigned so `assembleRelease` still works
            // for local dev without a keystore.
            signingConfig = signingConfigs.getByName("release").takeIf {
                System.getenv("KEYSTORE_FILE") != null || readLocalProperty("keystore.file") != null
            }
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlin {
        jvmToolchain(21)
    }

    buildFeatures {
        buildConfig = true
    }
}

// i18n: copy the shared English/pt-BR dictionaries (single source of truth in
// `shared/i18n/`) into the app assets so the Android UI and widgets use the
// exact same strings as the web app. Runs before every build.
tasks.register("syncI18nAssets") {
    val sharedDir = rootProject.file("../shared/i18n")
    val assetsDir = file("src/main/assets/i18n")
    inputs.dir(sharedDir)
    outputs.dir(assetsDir)
    doLast {
        assetsDir.mkdirs()
        copy {
            from(sharedDir)
            into(assetsDir)
            include("en.json", "pt-BR.json")
        }
    }
}
tasks.named("preBuild") {
    dependsOn("syncI18nAssets")
}

// OWASP dependency-check tuning (task: ./gradlew dependencyCheckAnalyze).
// - OSS Index analyzer requires a Sonatype PAT and is unreliable in CI (remote
//   errors per jar) — disabled; NVD + CISA KEV remain the coverage sources.
// - NVD API key (optional) is provided via -PdependencyCheck.nvd.apiKey in CI;
//   with no key the plugin uses the (slow, rate-limited) anonymous path.
dependencyCheck {
    autoUpdate = true
    failOnError = true
    analyzers {
        ossIndex.enabled = false
    }
}

dependencies {
    // Core
    implementation("androidx.core:core-ktx:1.15.0")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.7")
    implementation("androidx.activity:activity-compose:1.9.3")

    // Fragment (androidx.fragment:fragment-ktx). Not used directly, but pinned
    // >= 1.3.0: Firebase's play-services-basement pulls in fragment 1.1.0, and
    // androidx.activity's ActivityResult APIs (registerForActivityResult — used
    // for the Android 13+ POST_NOTIFICATIONS permission prompt) require
    // fragment >= 1.3.0 (lint: InvalidFragmentVersionForActivityResult).
    implementation("androidx.fragment:fragment-ktx:1.8.9")

    // Compose
    implementation(platform("androidx.compose:compose-bom:2026.05.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-graphics")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-extended")

    // Networking: Retrofit + OkHttp
    implementation("com.squareup.retrofit2:retrofit:2.11.0")
    implementation("com.squareup.retrofit2:converter-gson:2.11.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.okhttp3:logging-interceptor:4.12.0")

    // Push notifications (Firebase Cloud Messaging) — Phase 3.
    // Requires google-services.json + the google-services Gradle plugin for a
    // real Firebase project (see README); the code below is guarded so the app
    // runs without it.
    implementation("com.google.firebase:firebase-messaging:24.1.0")

    // Coroutines
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.9.0")

    // Phase 9c: offline-first — local SQLite persistence + WorkManager.
    // (Room/KSP would require migrating off AGP 9's built-in Kotlin, which is
    // unsupported by KSP — see docs/adr/012. A hand-rolled SQLiteOpenHelper
    // layer with a Room-style DAO API gives the same offline capability with
    // zero annotation-processing build risk.)
    implementation("androidx.work:work-runtime-ktx:2.10.1")

    // Phase 10: home-screen widgets — Jetpack Glance (Compose for widgets).
    // 1.1.1 is the newest stable release (1.2/1.3 are pre-release only). Glance
    // needs no KSP, so it works with AGP 9's built-in Kotlin that blocked Room
    // (docs/adr/012). glance-material3 provides the M3 color scheme +
    // LinearProgressIndicator used by the task/habit widgets.
    implementation("androidx.glance:glance-appwidget:1.1.1")
    implementation("androidx.glance:glance-material3:1.1.1")

    // Testing
    testImplementation("junit:junit:4.13.2")
    androidTestImplementation("androidx.test.ext:junit:1.2.1")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.6.1")
    androidTestImplementation(platform("androidx.compose:compose-bom:2026.05.00"))
    androidTestImplementation("androidx.compose.ui:ui-test-junit4")

    // Debug
    debugImplementation("androidx.compose.ui:ui-tooling")
    debugImplementation("androidx.compose.ui:ui-test-manifest")
}