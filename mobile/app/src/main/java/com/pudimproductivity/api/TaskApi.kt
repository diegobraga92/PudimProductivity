package com.pudimproductivity.api

import retrofit2.http.*

/**
 * Task data class matching the OpenAPI spec.
 */
data class Task(
    val id: String,
    val title: String,
    val status: String,
    val recurrence_days: List<String>? = null,
    val created_at: String,
    val updated_at: String
)

data class TaskCompletion(
    val id: String,
    val task_id: String,
    val completed_date: String,
    val created_at: String
)

data class CreateTaskRequest(
    val title: String,
    val recurrence_days: List<String>? = null
)

data class UpdateTaskRequest(
    val title: String? = null,
    val status: String? = null,
    val recurrence_days: List<String>? = null
)

/**
 * Retrofit interface for task CRUD operations.
 */
interface TaskService {
    @GET("tasks")
    suspend fun listTasks(): List<Task>

    @POST("tasks")
    suspend fun createTask(@Body request: CreateTaskRequest): Task

    @GET("tasks/{taskId}")
    suspend fun getTask(@Path("taskId") taskId: String): Task

    @PUT("tasks/{taskId}")
    suspend fun updateTask(
        @Path("taskId") taskId: String,
        @Body request: UpdateTaskRequest
    ): Task

    @DELETE("tasks/{taskId}")
    suspend fun deleteTask(@Path("taskId") taskId: String)

    @POST("tasks/{taskId}/complete")
    suspend fun completeTask(@Path("taskId") taskId: String): TaskCompletion

    @DELETE("tasks/{taskId}/complete")
    suspend fun uncompleteTask(@Path("taskId") taskId: String)

    @GET("tasks/{taskId}/completions")
    suspend fun getTaskCompletions(
        @Path("taskId") taskId: String,
        @Query("from") from: String? = null,
        @Query("to") to: String? = null
    ): List<TaskCompletion>
}

/**
 * Extension property to access TaskService from ApiClient.
 */
val ApiClient.taskService: TaskService
    get() = retrofit.create(TaskService::class.java)
