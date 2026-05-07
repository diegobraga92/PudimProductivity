package com.pudimproductivity.api

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
 */
object ApiClient {
    private const val DEFAULT_BASE_URL = "http://10.0.2.2:8080/api/v1/"

    val healthService: HealthService by lazy {
        val logging = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }

        val client = OkHttpClient.Builder()
            .addInterceptor(logging)
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(10, TimeUnit.SECONDS)
            .build()

        val baseUrl = try {
            Class.forName("com.pudimproductivity.BuildConfig")
                .getField("API_BASE_URL")
                .get(null) as? String
                ?.trim('"')
                ?.let { "$it/" }
                ?: DEFAULT_BASE_URL
        } catch (_: Exception) {
            DEFAULT_BASE_URL
        }

        Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
            .create(HealthService::class.java)
    }
}
