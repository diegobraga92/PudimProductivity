package com.pudimproductivity.focus

import android.content.Context
import android.content.Intent
import androidx.core.content.ContextCompat
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.PomodoroSession
import com.pudimproductivity.api.StartSessionRequest
import com.pudimproductivity.api.pomodoroService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

data class FocusTimerState(
    val active: Boolean = false,
    val running: Boolean = false,
    val paused: Boolean = false,
    val session: PomodoroSession? = null,
    val remainingSeconds: Int = 0,
    val focusMinutes: Int = 25
)

/**
 * Single source of truth for the focus timer. Calls the backend pomodoro API on
 * start/pause/resume/stop and runs a local 1-second ticker while running. A
 * foreground service (FocusTimerService) keeps the process alive in the
 * background and renders the state to a persistent notification.
 */
object FocusTimerManager {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val _state = MutableStateFlow(FocusTimerState())
    val state: StateFlow<FocusTimerState> = _state.asStateFlow()

    private var ticker: Job? = null
    private var appContext: Context? = null

    /** Call once from Application/MainActivity. */
    fun init(context: Context) {
        appContext = context.applicationContext
    }

    fun start(focusMinutes: Int) {
        scope.launch {
            val focus = if (focusMinutes > 0) focusMinutes else 25
            val session = ApiClient.pomodoroService.startSession(
                StartSessionRequest(focus_duration = focus, break_duration = 5)
            )
            _state.value = FocusTimerState(
                active = true, running = true,
                session = session, remainingSeconds = session.remaining_seconds, focusMinutes = focus
            )
            ensureServiceRunning()
            beginTicker()
        }
    }

    fun pause() {
        scope.launch {
            val session = ApiClient.pomodoroService.pause()
            _state.value = _state.value.copy(running = false, paused = true, session = session, remainingSeconds = session.remaining_seconds)
        }
    }

    fun resume() {
        scope.launch {
            val session = ApiClient.pomodoroService.resume()
            _state.value = _state.value.copy(running = true, paused = false, session = session, remainingSeconds = session.remaining_seconds)
            beginTicker()
        }
    }

    fun stop() {
        scope.launch {
            ticker?.cancel()
            ticker = null
            try {
                ApiClient.pomodoroService.stop()
            } catch (_: Exception) {
                // The backend may have no active session (e.g. already stopped).
            }
            _state.value = FocusTimerState(focusMinutes = _state.value.focusMinutes)
            ensureServiceStopped()
        }
    }

    private fun beginTicker() {
        ticker?.cancel()
        ticker = scope.launch {
            while (true) {
                delay(1_000)
                val current = _state.value
                if (current.active && current.running) {
                    val remaining = (current.remainingSeconds - 1).coerceAtLeast(0)
                    _state.value = current.copy(remainingSeconds = remaining)
                    if (remaining == 0) {
                        // Session elapsed; refresh from the backend.
                        loadCurrent()
                        break
                    }
                } else {
                    break
                }
            }
        }
    }

    fun loadCurrent() {
        scope.launch {
            try {
                val resp = ApiClient.pomodoroService.getCurrent()
                if (resp.active && resp.session != null) {
                    val s = resp.session
                    _state.value = FocusTimerState(
                        active = true,
                        running = s.status == "running",
                        paused = s.status == "paused",
                        session = s,
                        remainingSeconds = s.remaining_seconds,
                        focusMinutes = s.focus_duration
                    )
                    if (s.status == "running") beginTicker()
                    ensureServiceRunning()
                } else {
                    _state.value = FocusTimerState()
                    ensureServiceStopped()
                }
            } catch (_: Exception) {
                // Backend unavailable — leave state as-is.
            }
        }
    }

    private fun ensureServiceRunning() {
        val ctx = appContext ?: return
        ContextCompat.startForegroundService(
            ctx,
            Intent(ctx, FocusTimerService::class.java).setAction(ACTION_START)
        )
    }

    private fun ensureServiceStopped() {
        val ctx = appContext ?: return
        ctx.stopService(Intent(ctx, FocusTimerService::class.java))
    }

    const val ACTION_START = "com.pudimproductivity.focus.START"
    const val ACTION_STOP = "com.pudimproductivity.focus.STOP"
}
