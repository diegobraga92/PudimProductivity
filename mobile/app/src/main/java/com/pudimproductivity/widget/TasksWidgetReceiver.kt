package com.pudimproductivity.widget

import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetReceiver

/** Receiver for the "Today's Tasks" home-screen widget (Phase 10). */
class TasksWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = TasksWidget
}
