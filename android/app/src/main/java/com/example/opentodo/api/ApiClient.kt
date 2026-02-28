package com.example.opentodo.api

import android.util.Log
import com.example.opentodo.BuildConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

object ApiClient {
    private const val TAG = "ApiClient"
    private val BASE_URL = BuildConfig.BACKEND_URL
    private const val CONNECT_TIMEOUT_MS = 5000
    private const val READ_TIMEOUT_MS = 5000

    suspend fun getLists(): List<ApiList> = withContext(Dispatchers.IO) {
        try {
            val url = URL("$BASE_URL/lists")
            val connection = url.openConnection() as HttpURLConnection
            connection.apply {
                requestMethod = "GET"
                connectTimeout = CONNECT_TIMEOUT_MS
                readTimeout = READ_TIMEOUT_MS
                setRequestProperty("Accept", "application/json")
            }

            val responseCode = connection.responseCode
            if (responseCode == HttpURLConnection.HTTP_OK) {
                val response = connection.inputStream.bufferedReader().use { it.readText() }
                Log.d(TAG, "Lists response: $response")
                
                val jsonObject = JSONObject(response)
                val data = jsonObject.getJSONArray("data")
                
                val lists = mutableListOf<ApiList>()
                for (i in 0 until data.length()) {
                    val listJson = data.getJSONObject(i)
                    lists.add(
                        ApiList(
                            id = listJson.getString("id"),
                            name = listJson.getString("name"),
                            type = listJson.getString("type")
                        )
                    )
                }
                lists
            } else {
                Log.e(TAG, "Failed to get lists. Response code: $responseCode")
                emptyList()
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error getting lists", e)
            emptyList()
        }
    }

    suspend fun getTasksForList(listId: String): List<ApiTask> = withContext(Dispatchers.IO) {
        try {
            val url = URL("$BASE_URL/lists/$listId/tasks")
            val connection = url.openConnection() as HttpURLConnection
            connection.apply {
                requestMethod = "GET"
                connectTimeout = CONNECT_TIMEOUT_MS
                readTimeout = READ_TIMEOUT_MS
                setRequestProperty("Accept", "application/json")
            }

            val responseCode = connection.responseCode
            if (responseCode == HttpURLConnection.HTTP_OK) {
                val response = connection.inputStream.bufferedReader().use { it.readText() }
                Log.d(TAG, "Tasks response for list $listId: $response")
                
                val jsonObject = JSONObject(response)
                val data = jsonObject.getJSONArray("data")
                
                val tasks = mutableListOf<ApiTask>()
                for (i in 0 until data.length()) {
                    val taskJson = data.getJSONObject(i)
                    tasks.add(
                        ApiTask(
                            id = taskJson.getString("id"),
                            listId = taskJson.getString("listId"),
                            title = taskJson.getString("title"),
                            completed = taskJson.getBoolean("completed")
                        )
                    )
                }
                tasks
            } else {
                Log.e(TAG, "Failed to get tasks for list $listId. Response code: $responseCode")
                emptyList()
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error getting tasks for list $listId", e)
            emptyList()
        }
    }

    suspend fun createTask(listId: String, title: String): ApiTask? = withContext(Dispatchers.IO) {
        try {
            val url = URL("$BASE_URL/tasks")
            val connection = url.openConnection() as HttpURLConnection
            connection.apply {
                requestMethod = "POST"
                connectTimeout = CONNECT_TIMEOUT_MS
                readTimeout = READ_TIMEOUT_MS
                setRequestProperty("Content-Type", "application/json")
                setRequestProperty("Accept", "application/json")
                doOutput = true
            }

            val requestBody = JSONObject().apply {
                put("listId", listId)
                put("title", title)
                put("completed", false)
                put("order", 0)
            }

            connection.outputStream.bufferedWriter().use {
                it.write(requestBody.toString())
            }

            val responseCode = connection.responseCode
            if (responseCode == HttpURLConnection.HTTP_OK) {
                val response = connection.inputStream.bufferedReader().use { it.readText() }
                Log.d(TAG, "Create task response: $response")
                
                val jsonObject = JSONObject(response)
                val data = jsonObject.getJSONObject("data")
                
                ApiTask(
                    id = data.getString("id"),
                    listId = data.getString("listId"),
                    title = data.getString("title"),
                    completed = data.getBoolean("completed")
                )
            } else {
                Log.e(TAG, "Failed to create task. Response code: $responseCode")
                null
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error creating task", e)
            null
        }
    }

    suspend fun completeTask(taskId: String): Boolean = withContext(Dispatchers.IO) {
        try {
            val url = URL("$BASE_URL/tasks/$taskId/complete")
            val connection = url.openConnection() as HttpURLConnection
            connection.apply {
                requestMethod = "PATCH"
                connectTimeout = CONNECT_TIMEOUT_MS
                readTimeout = READ_TIMEOUT_MS
                setRequestProperty("Accept", "application/json")
            }

            val responseCode = connection.responseCode
            if (responseCode == HttpURLConnection.HTTP_OK) {
                Log.d(TAG, "Task $taskId completed successfully")
                true
            } else {
                Log.e(TAG, "Failed to complete task $taskId. Response code: $responseCode")
                false
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error completing task $taskId", e)
            false
        }
    }

    suspend fun reopenTask(taskId: String): Boolean = withContext(Dispatchers.IO) {
        try {
            val url = URL("$BASE_URL/tasks/$taskId/reopen")
            val connection = url.openConnection() as HttpURLConnection
            connection.apply {
                requestMethod = "PATCH"
                connectTimeout = CONNECT_TIMEOUT_MS
                readTimeout = READ_TIMEOUT_MS
                setRequestProperty("Accept", "application/json")
            }

            val responseCode = connection.responseCode
            if (responseCode == HttpURLConnection.HTTP_OK) {
                Log.d(TAG, "Task $taskId reopened successfully")
                true
            } else {
                Log.e(TAG, "Failed to reopen task $taskId. Response code: $responseCode")
                false
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error reopening task $taskId", e)
            false
        }
    }

    suspend fun deleteTask(taskId: String): Boolean = withContext(Dispatchers.IO) {
        try {
            val url = URL("$BASE_URL/tasks/$taskId")
            val connection = url.openConnection() as HttpURLConnection
            connection.apply {
                requestMethod = "DELETE"
                connectTimeout = CONNECT_TIMEOUT_MS
                readTimeout = READ_TIMEOUT_MS
                setRequestProperty("Accept", "application/json")
            }

            val responseCode = connection.responseCode
            if (responseCode == HttpURLConnection.HTTP_OK) {
                Log.d(TAG, "Task $taskId deleted successfully")
                true
            } else {
                Log.e(TAG, "Failed to delete task $taskId. Response code: $responseCode")
                false
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error deleting task $taskId", e)
            false
        }
    }
}

data class ApiList(
    val id: String,
    val name: String,
    val type: String
)

data class ApiTask(
    val id: String,
    val listId: String,
    val title: String,
    val completed: Boolean
)
// TODO
