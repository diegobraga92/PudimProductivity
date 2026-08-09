package com.pudimproductivity.api

import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST

/**
 * Pomodoro session DTO matching api/openapi/pomodoro-v1.yaml.
 */
data class PomodoroSession(
    val id: String,
    val status: String,
    val focus_duration: Int,
    val break_duration: Int,
    val current_cycle: Int,
    val elapsed_seconds: Int,
    val remaining_seconds: Int,
    val started_at: String,
    val paused_at: String? = null,
    val completed_at: String? = null,
    val noise_config: NoiseConfig? = null
)

data class NoiseConfig(
    val enabled: Boolean,
    val track_id: String? = null
)

data class CurrentSessionResponse(
    val active: Boolean,
    val session: PomodoroSession? = null
)

data class StartSessionRequest(
    val focus_duration: Int? = null,
    val break_duration: Int? = null,
    val noise_config: NoiseConfig? = null
)

/**
 * Retrofit interface for the pomodoro (focus timer) endpoints.
 */
interface PomodoroService {
    @POST("pomodoro/start")
    suspend fun startSession(@Body request: StartSessionRequest): PomodoroSession

    @GET("pomodoro/current")
    suspend fun getCurrent(): CurrentSessionResponse

    @POST("pomodoro/pause")
    suspend fun pause(): PomodoroSession

    @POST("pomodoro/resume")
    suspend fun resume(): PomodoroSession

    @POST("pomodoro/stop")
    suspend fun stop(): PomodoroSession
}

/** Extension property to access PomodoroService from ApiClient. */
val ApiClient.pomodoroService: PomodoroService
    get() = retrofit.create(PomodoroService::class.java)
