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
 * The base URL comes from [ServerConfig] — a persisted, runtime-editable value
 * that defaults to the build-time [BuildConfig.API_BASE_URL] (injected via
 * `buildConfigField` in app/build.gradle.kts). Users can change the server
 * address in-app without rebuilding; call [invalidate] after changing it so
 * the next Retrofit service access is created against the new URL.
 */
object ApiClient {
    /** Shared OkHttpClient, reused by Retrofit and the WebSocket sync client. */
    val client: OkHttpClient by lazy {
        val logging = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }

        OkHttpClient.Builder()
            .addInterceptor(logging)
            // Dev-mode identity headers, mirroring the backend's shared.AuthMiddleware.
            // The backend trusts X-User-ID / X-User-Role in development; production
            // will validate JWTs. Required for protected (POST/PUT/DELETE) endpoints.
            .addInterceptor { chain ->
                val request = chain.request().newBuilder()
                    .header("X-User-ID", "dev-user")
                    .header("X-User-Role", "user")
                    .build()
                chain.proceed(request)
            }
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(10, TimeUnit.SECONDS)
            .build()
    }

    @Volatile
    private var _retrofit: Retrofit? = null

    /**
     * Retrofit instance for the current [ServerConfig.url]. Recreated lazily
     * after [invalidate] so a URL change takes effect on the next service call.
     */
    val retrofit: Retrofit
        get() {
            _retrofit?.let { return it }
            return synchronized(this) {
                _retrofit ?: buildRetrofit().also { _retrofit = it }
            }
        }

    /** Drops the cached Retrofit so the next access rebuilds against the new URL. */
    fun invalidate() {
        synchronized(this) {
            _retrofit = null
        }
    }

    private fun buildRetrofit(): Retrofit {
        val baseUrl = ServerConfig.url.value.trimEnd('/') + "/"

        return Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
    }

    val healthService: HealthService
        get() = retrofit.create(HealthService::class.java)
}
