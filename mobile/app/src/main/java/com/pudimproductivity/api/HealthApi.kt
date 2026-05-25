package com.pudimproductivity.api

import com.pudimproductivity.BuildConfig
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import retrofit2.http.GET
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import java.util.concurrent.TimeUnit

/**
 * Response from GET /api/v1/health.
 */
data class HealthResponse(
    val status: String,
    val version: String,
    val db: String
)

/**
 * Retrofit interface for the health endpoint.
 */
interface HealthService {
    @GET("health")
    suspend fun getHealth(): HealthResponse
}

/**
 * Singleton Retrofit client configured for the backend API.
 *
 * The base URL is taken directly from [BuildConfig.API_BASE_URL] which is injected
 * at build time via the `buildConfigField` in `app/build.gradle.kts`.
 * Override it in a local `gradle.properties` or CI environment variable to point
 * at a real device / LAN server instead of the AVD emulator loopback.
 */
object ApiClient {
    val retrofit: Retrofit by lazy {
        val logging = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }

        val client = OkHttpClient.Builder()
            .addInterceptor(logging)
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(10, TimeUnit.SECONDS)
            .build()

        val baseUrl = BuildConfig.API_BASE_URL.trimEnd('/') + "/"

        Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
    }

    val healthService: HealthService by lazy {
        retrofit.create(HealthService::class.java)
    }
}
