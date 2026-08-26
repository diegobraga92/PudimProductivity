package com.pudimproductivity

import android.content.Context
import com.pudimproductivity.BuildConfig
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.ServerConfig
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject

/**
 * Reports uncaught exceptions to the backend error beacon (POST /api/v1/errors)
 * so client crashes appear in the server logs with their trace context.
 *
 * Installed once in MainActivity via Thread.setDefaultUncaughtExceptionHandler.
 * The report is best-effort (fire-and-forget); the default crash handler still
 * runs afterwards.
 */
object ErrorReporter {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /** Installs the crash reporter. Safe to call multiple times. */
    fun install(context: Context) {
        val previous = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { thread, throwable ->
            report(context, thread, throwable)
            previous?.uncaughtException(thread, throwable)
        }
    }

    private fun report(context: Context, thread: Thread, throwable: Throwable) {
        scope.launch {
            try {
                val body = JSONObject()
                    .put("message", throwable.message ?: throwable.javaClass.simpleName)
                    .put("stack", throwable.stackTraceToString())
                    .put("context", "thread=${thread.name}; app=${BuildConfig.VERSION_NAME}")

                val request = Request.Builder()
                    .url(ServerConfig.url.value.trimEnd('/') + "/errors")
                    .header("X-Error-Source", "android")
                    .header("X-User-ID", "dev-user")
                    .header("X-User-Role", "user")
                    .post(body.toString().toRequestBody("application/json".toMediaType()))
                    .build()

                ApiClient.client.newCall(request).execute().use { } // discard response
            } catch (_: Exception) {
                // Best-effort: never crash while reporting a crash.
            }
        }
    }
}
