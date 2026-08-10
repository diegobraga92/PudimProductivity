package com.pudimproductivity.api

import retrofit2.http.GET
import retrofit2.http.Query

data class ScheduleSlot(
    val task_id: String,
    val title: String,
    val start_time: String,
    val end_time: String,
    val kind: String
)

data class Suggestion(
    val date: String,
    val slots: List<ScheduleSlot> = emptyList(),
    val free_hours: Int = 0,
    val avg_per_day: Double = 0.0,
    val pending_count: Int = 0
)

interface ScheduleService {
    @GET("schedule")
    suspend fun getDailySchedule(@Query("date") date: String? = null): Suggestion
}

/** Extension property to access ScheduleService from ApiClient. */
val ApiClient.scheduleService: ScheduleService
    get() = retrofit.create(ScheduleService::class.java)
