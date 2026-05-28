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
    val list_id: String? = null,
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
    val recurrence_days: List<String>? = null,
    val list_id: String? = null
)

data class UpdateTaskRequest(
    val title: String? = null,
    val status: String? = null,
    val recurrence_days: List<String>? = null,
    val list_id: String? = null
)

/**
 * Task list data class.
 */
data class TaskList(
    val id: String,
    val name: String,
    val description: String? = null,
    val created_at: String,
    val updated_at: String
)

data class CreateTaskListRequest(
    val name: String
)

data class UpdateTaskListRequest(
    val name: String? = null,
    val description: String? = null
)

/**
 * Retrofit interface for task CRUD operations.
 */
interface TaskService {
    @GET("tasks")
    suspend fun listTasks(@Query("type") type: String? = null): List<Task>

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
    suspend fun completeTask(
        @Path("taskId") taskId: String,
        @Query("date") date: String? = null
    ): TaskCompletion

    @DELETE("tasks/{taskId}/complete")
    suspend fun uncompleteTask(
        @Path("taskId") taskId: String,
        @Query("date") date: String? = null
    )

    @GET("tasks/{taskId}/completions")
    suspend fun getTaskCompletions(
        @Path("taskId") taskId: String,
        @Query("from") from: String? = null,
        @Query("to") to: String? = null
    ): List<TaskCompletion>

    // Task Lists
    @GET("task-lists")
    suspend fun listTaskLists(): List<TaskList>

    @POST("task-lists")
    suspend fun createTaskList(@Body request: CreateTaskListRequest): TaskList

    @GET("task-lists/{listId}")
    suspend fun getTaskList(@Path("listId") listId: String): TaskList

    @PUT("task-lists/{listId}")
    suspend fun updateTaskList(
        @Path("listId") listId: String,
        @Body request: UpdateTaskListRequest
    ): TaskList

    @DELETE("task-lists/{listId}")
    suspend fun deleteTaskList(@Path("listId") listId: String)

    @GET("task-lists/{listId}/tasks")
    suspend fun listTasksByListID(
        @Path("listId") listId: String,
        @Query("type") type: String? = null
    ): List<Task>
}

/**
 * Extension property to access TaskService from ApiClient.
 */
val ApiClient.taskService: TaskService
    get() = retrofit.create(TaskService::class.java)
