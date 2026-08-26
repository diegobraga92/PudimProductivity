package com.pudimproductivity.api

import android.content.Context
import android.content.SharedPreferences
import androidx.core.content.edit
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject

/**
 * A real-time task event from the backend sync hub (see api/ws/events-v1.json).
 */
data class WsEvent(
    val type: String,
    val seq: Long,
    val timestamp: String,
    val payload: JSONObject?
)

/**
 * Singleton WebSocket client mirroring the web client (web/src/api/sync.ts):
 *
 *  - Connects to `<API_BASE_URL>/ws?last_seq=N`.
 *  - Reconnects with exponential backoff on failure.
 *  - Persists the last sequence number in SharedPreferences so reconnects resume
 *    without missing events; a `stale` event signals a full REST refresh.
 *  - Exposes events as a [SharedFlow]; no replay of old events (clients keep
 *    their own state and treat `stale` as "refetch").
 *
 * Start it once from Application/MainActivity — it lives for the app process.
 * Only an application-scoped [SharedPreferences] handle is retained (not a
 * Context) so no activity/application reference is leaked.
 */
object SyncClient {
    private const val PREFS_NAME = "pudim"
    private const val LAST_SEQ_KEY = "ws.lastSeq"
    private const val INITIAL_RECONNECT_MS = 1_000L
    private const val MAX_RECONNECT_MS = 30_000L

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val _events = MutableSharedFlow<WsEvent>(extraBufferCapacity = 64)
    val events: SharedFlow<WsEvent> = _events.asSharedFlow()

    private val _connected = MutableStateFlow(false)

    /**
     * Whether the WebSocket is currently connected to the backend. The app
     * watches this to flush local dirty rows and pull server changes the moment
     * the connection is (re)established — the "reconnect to the server" hook for
     * offline sync.
     */
    val connected: StateFlow<Boolean> = _connected.asStateFlow()

    private var prefs: SharedPreferences? = null
    private var webSocket: WebSocket? = null
    private var reconnectJob: Job? = null
    private var reconnectDelayMs = INITIAL_RECONNECT_MS
    private var lastSeq: Long = 0
    private var stopped = true

    /** Starts (or restarts) the connection. Idempotent. */
    @Synchronized
    fun start(appContext: Context) {
        prefs = appContext.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        stopped = false
        if (webSocket != null) return // already connected/connecting
        connect()
    }

    /** Stops the connection and cancels pending reconnects. */
    @Synchronized
    fun stop() {
        stopped = true
        _connected.value = false
        reconnectJob?.cancel()
        reconnectJob = null
        webSocket?.close(1000, "bye")
        webSocket = null
    }

    private fun connect() {
        val preferences = prefs ?: return
        lastSeq = preferences.getLong(LAST_SEQ_KEY, 0L)

        val wsUrl = ServerConfig.url.value
            .replaceFirst("http", "ws")
            .trimEnd('/') + "/ws?last_seq=$lastSeq"

        val request = Request.Builder().url(wsUrl).build()
        webSocket = ApiClient.client.newWebSocket(request, listener(preferences))
    }

    private fun listener(preferences: SharedPreferences): WebSocketListener =
        object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                reconnectDelayMs = INITIAL_RECONNECT_MS
                _connected.value = true
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val json = JSONObject(text)
                    val seq = json.optLong("seq", 0L)
                    if (seq > lastSeq) {
                        lastSeq = seq
                        preferences.edit { putLong(LAST_SEQ_KEY, lastSeq) }
                    }
                    val event = WsEvent(
                        type = json.optString("type", ""),
                        seq = seq,
                        timestamp = json.optString("timestamp", ""),
                        payload = json.optJSONObject("payload")
                    )
                    _events.tryEmit(event)
                } catch (_: Exception) {
                    // Ignore malformed frames.
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                _connected.value = false
                scheduleReconnect(preferences)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                _connected.value = false
                scheduleReconnect(preferences)
            }
        }

    private fun scheduleReconnect(preferences: SharedPreferences) {
        if (stopped) return
        if (reconnectJob != null) return
        reconnectJob = scope.launch {
            delay(reconnectDelayMs)
            reconnectDelayMs = minOf(reconnectDelayMs * 2, MAX_RECONNECT_MS)
            reconnectJob = null
            if (!stopped) connect()
        }
    }
}
