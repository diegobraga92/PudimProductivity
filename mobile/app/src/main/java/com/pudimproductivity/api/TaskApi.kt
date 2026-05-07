package com.pudimproductivity.api

import retrofit2.http.*

/**
 * Task data class matching the OpenAPI spec.
 */
data class Task(
    val id: String,
    val title: String,
    val description: String?,
    val status: String,
    val priority: String,
    val due_date: String?,
    val created_at: String,
    val updated_at: String
)

data class CreateTaskRequest(
    val title: String,
    val description: String? = null,
    val priority: String? = "medium",
    val due_date: String? = null
)

data class UpdateTaskRequest(
    val title: String? = null,
    val description: String? = null,
    val status: String? = null,
    val priority: String? = null,
    val due_date: String? = null
)

/**
 * Retrofit interface for task CRUD operations.
 */
interface TaskService {
    @GET("tasks")
    suspend fun listTasks(
        @Query("status") status: String? = null,
        @Query("priority") priority: String? = null
    ): List<Task>

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
}

/**
 * Extension property to access TaskService from ApiClient.
 */
val ApiClient.taskService: TaskService
    get() = retrofit.create(TaskService::class.java)
