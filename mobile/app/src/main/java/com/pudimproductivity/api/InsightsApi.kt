package com.pudimproductivity.api

import retrofit2.http.GET
import retrofit2.http.Query

data class HabitStat(
    val task_id: String,
    val title: String,
    val count: Int
)

data class WeeklyStats(
    val week_start: String,
    val total_completions: Int,
    val completions_per_day: Double,
    val top_habits: List<HabitStat>? = null,
    val focus_minutes: Int,
    val focus_sessions: Int,
    val recipes_created: Int
)

data class InsightReport(
    val week_start: String,
    val stats: WeeklyStats,
    val report_text: String,
    val llm_summary: String? = null,
    val generated_at: String
)

interface InsightsService {
    @GET("insights/weekly")
    suspend fun getWeeklyInsights(@Query("date") date: String? = null): InsightReport
}

/** Extension property to access InsightsService from ApiClient. */
val ApiClient.insightsService: InsightsService
    get() = retrofit.create(InsightsService::class.java)
