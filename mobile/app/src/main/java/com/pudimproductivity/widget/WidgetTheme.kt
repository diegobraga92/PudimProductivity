package com.pudimproductivity.widget

import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.ui.graphics.Color
import androidx.glance.material3.ColorProviders

/**
 * Brand palette for the Glance widgets, mirroring ui/theme/Theme.kt.
 *
 * Glance has its own theme system (it cannot reuse the Compose
 * [com.pudimproductivity.ui.theme.PudimProductivityTheme]), so the same warm
 * orange tokens are re-declared here as a [ColorProviders]. The widgets pick
 * the light/dark scheme automatically based on the system setting.
 */
object WidgetColors {

    val providers = ColorProviders(
        light = lightColorScheme(
            primary = Color(0xFFEA580C),
            onPrimary = Color.White,
            primaryContainer = Color(0xFFFFDBC7),
            onPrimaryContainer = Color(0xFF3B0A00),
            secondary = Color(0xFFC2410C),
            background = Color(0xFFFEF6E9),
            onBackground = Color(0xFF1F1B16),
            surface = Color(0xFFFFFBFF),
            onSurface = Color(0xFF1F1B16),
            surfaceVariant = Color(0xFFF2E0D0),
            onSurfaceVariant = Color(0xFF52443C),
            outline = Color(0xFF85736A),
            error = Color(0xFFBA1A1A),
            onError = Color.White
        ),
        dark = darkColorScheme(
            primary = Color(0xFFFFB59C),
            onPrimary = Color(0xFF5D1800),
            primaryContainer = Color(0xFFC2410C),
            onPrimaryContainer = Color(0xFFFFDBC7),
            secondary = Color(0xFFE5BFA9),
            background = Color(0xFF1F150F),
            onBackground = Color(0xFFF0E0D8),
            surface = Color(0xFF1F150F),
            onSurface = Color(0xFFF0E0D8),
            surfaceVariant = Color(0xFF52443C),
            onSurfaceVariant = Color(0xFFD6C3B8),
            outline = Color(0xFF9E8E84),
            error = Color(0xFFFFB4AB),
            onError = Color(0xFF690005)
        )
    )
}
