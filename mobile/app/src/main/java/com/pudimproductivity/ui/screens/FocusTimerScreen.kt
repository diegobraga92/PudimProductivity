package com.pudimproductivity.ui.screens

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pudimproductivity.focus.FocusTimerManager
import com.pudimproductivity.focus.FocusTimerState

private fun formatTime(totalSeconds: Int): String {
    val m = totalSeconds / 60
    val s = totalSeconds % 60
    return "%02d:%02d".format(m, s)
}

/** Circular countdown ring. */
@Composable
private fun CountdownCircle(state: FocusTimerState) {
    val total = state.focusMinutes * 60
    val remaining = if (total > 0) state.remainingSeconds.coerceIn(0, total) else 0
    val progress = if (total > 0) remaining.toFloat() / total else 0f

    val ringColor = MaterialTheme.colorScheme.surfaceVariant
    val progressColor =
        if (state.running) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.tertiary

    Canvas(modifier = Modifier.size(260.dp)) {
        val stroke = 14.dp.toPx()
        val inset = stroke / 2
        val size = Size(this.size.width - stroke, this.size.height - stroke)
        val topLeft = Offset(inset, inset)

        // Background ring
        drawArc(
            color = ringColor,
            startAngle = 0f,
            sweepAngle = 360f,
            useCenter = false,
            topLeft = topLeft,
            size = size,
            style = Stroke(width = stroke, cap = StrokeCap.Round)
        )
        // Progress ring (remaining time)
        drawArc(
            color = progressColor,
            startAngle = -90f,
            sweepAngle = 360f * progress,
            useCenter = false,
            topLeft = topLeft,
            size = size,
            style = Stroke(width = stroke, cap = StrokeCap.Round)
        )
    }

    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                text = formatTime(state.remainingSeconds),
                fontSize = 48.sp,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface
            )
            Text(
                text = when {
                    state.running -> "Focusing…"
                    state.paused -> "Paused"
                    !state.active -> "Ready"
                    else -> ""
                },
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

/**
 * Focus timer screen (Phase 4): circular countdown + start/pause/resume/stop.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FocusTimerScreen(onBack: () -> Unit) {
    val state by FocusTimerManager.state.collectAsState()
    var selectedMinutes by remember { mutableStateOf(25) }
    val durationOptions = listOf(15, 25, 45)

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Focus Timer") },
                navigationIcon = {
                    TextButton(onClick = onBack) { Text("← Back") }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            CountdownCircle(state)

            Spacer(modifier = Modifier.height(24.dp))

            if (!state.active) {
                Text("Duration", style = MaterialTheme.typography.labelMedium)
                Spacer(modifier = Modifier.height(8.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    durationOptions.forEach { minutes ->
                        FilterChip(
                            selected = selectedMinutes == minutes,
                            onClick = { selectedMinutes = minutes },
                            label = { Text("$minutes min") }
                        )
                    }
                }
                Spacer(modifier = Modifier.height(24.dp))
                Button(onClick = { FocusTimerManager.start(selectedMinutes) }) {
                    Text("Start Focus")
                }
            } else {
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    if (state.running) {
                        Button(onClick = { FocusTimerManager.pause() }) { Text("Pause") }
                    } else if (state.paused) {
                        Button(onClick = { FocusTimerManager.resume() }) { Text("Resume") }
                    }
                    OutlinedButton(onClick = { FocusTimerManager.stop() }) { Text("Stop") }
                }
                Spacer(modifier = Modifier.height(16.dp))
                Text(
                    text = "Cycle ${state.session?.current_cycle ?: 1} • Focus ${state.focusMinutes} min",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}
