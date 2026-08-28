package com.pudimproductivity.api

import retrofit2.http.GET
import retrofit2.http.Query

data class SyncTask(
    val id: String,
    val title: String,
    val status: String,
    val recurrence_days: List<String>? = null,
    val list_id: String? = null,
    val start_time: String? = null,
    val end_time: String? = null,
    val color: String? = null,
    val scheduled_date: String? = null,
    val alarm_minutes: Int? = null,
    val created_at: String,
    val updated_at: String
)

data class SyncCompletion(
    val id: String,
    val task_id: String,
    val completed_date: String,
    val created_at: String
)

data class SyncTaskList(
    val id: String,
    val name: String,
    val description: String = "",
    val owner_id: String = "",
    val created_at: String,
    val updated_at: String
)

data class SyncShare(
    val list_id: String,
    val shared_with: String,
    val role: String,
    val created_at: String
)

data class SyncBundle(
    val timestamp: String,
    val tasks: List<SyncTask> = emptyList(),
    val deleted_task_ids: List<String> = emptyList(),
    val completions: List<SyncCompletion> = emptyList(),
    val deleted_completion_ids: List<String> = emptyList(),
    val task_lists: List<SyncTaskList> = emptyList(),
    val deleted_task_list_ids: List<String> = emptyList(),
    val shares: List<SyncShare> = emptyList(),
    val deleted_share_keys: List<String> = emptyList()
)

interface SyncService {
    @GET("sync")
    suspend fun getChanges(@Query("since") since: String? = null): SyncBundle
}

/** Extension property to access SyncService from ApiClient. */
val ApiClient.syncService: SyncService
    get() = retrofit.create(SyncService::class.java)
