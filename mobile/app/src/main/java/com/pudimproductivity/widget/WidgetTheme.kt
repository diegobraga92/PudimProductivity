package com.pudimproductivity.widget

import android.content.Context
import android.content.res.Configuration
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.ui.graphics.Color
import androidx.glance.material3.ColorProviders
import androidx.glance.unit.ColorProvider
import com.pudimproductivity.ui.theme.ThemeMode
import com.pudimproductivity.ui.theme.ThemePreferences

/**
 * Brand palette for the Glance widgets, mirroring ui/theme/Theme.kt token for
 * token (light + dark) so the home-screen cards share the app's exact colors:
 * warm orange primary, the habit-green secondary, cream background and the
 * same tertiary/error accents.
 *
 * Glance has its own theme system (it cannot reuse the Compose
 * [com.pudimproductivity.ui.theme.PudimProductivityTheme]), so the tokens are
 * re-declared here. The widgets pick the light/dark scheme the same way the
 * app does — from ThemePreferences (System / Light / Dark) — via [resolve].
 */
object WidgetColors {

    val light = lightColorScheme(
        primary = Color(0xFFEA580C),
        onPrimary = Color.White,
        primaryContainer = Color(0xFFFFDBC7),
        onPrimaryContainer = Color(0xFF3B0A00),
        secondary = Color(0xFF10B981),
        onSecondary = Color.White,
        secondaryContainer = Color(0xFFECFDF5),
        onSecondaryContainer = Color(0xFF047857),
        tertiary = Color(0xFFF59E0B),
        onTertiary = Color(0xFF3B2A00),
        tertiaryContainer = Color(0xFFFFFBEB),
        onTertiaryContainer = Color(0xFFB45309),
        background = Color(0xFFFEF6E9),
        onBackground = Color(0xFF1F1B16),
        surface = Color(0xFFFFFFFF),
        onSurface = Color(0xFF1F1B16),
        surfaceVariant = Color(0xFFF2E0D0),
        onSurfaceVariant = Color(0xFF52443C),
        outline = Color(0xFF85736A),
        error = Color(0xFFEF4444),
        onError = Color.White,
        errorContainer = Color(0xFFFEF2F2),
        onErrorContainer = Color(0xFFB91C1C)
    )

    val dark = darkColorScheme(
        primary = Color(0xFFFB923C),
        onPrimary = Color(0xFF2A1205),
        primaryContainer = Color(0xFF431407),
        onPrimaryContainer = Color(0xFFFDBA74),
        secondary = Color(0xFF34D399),
        onSecondary = Color(0xFF053B2A),
        secondaryContainer = Color(0xFF022C22),
        onSecondaryContainer = Color(0xFFA7F3D0),
        tertiary = Color(0xFFFBBF24),
        onTertiary = Color(0xFF3B2A00),
        tertiaryContainer = Color(0xFF451A03),
        onTertiaryContainer = Color(0xFFFDE68A),
        background = Color(0xFF0F172A),
        onBackground = Color(0xFFF1F5F9),
        surface = Color(0xFF1E293B),
        onSurface = Color(0xFFF1F5F9),
        surfaceVariant = Color(0xFF273449),
        onSurfaceVariant = Color(0xFFCBD5E1),
        outline = Color(0xFF334155),
        error = Color(0xFFF87171),
        onError = Color(0xFF450A0A),
        errorContainer = Color(0xFF450A0A),
        onErrorContainer = Color(0xFFFFD0CE)
    )

    /**
     * Resolves the widget palette the same way the app resolves its theme:
     * ThemePreferences.SYSTEM follows the system night mode, while LIGHT and
     * DARK force the scheme regardless of the system setting.
     */
    fun resolve(context: Context) =
        if (shouldUseDark(context)) ColorProviders(dark) else ColorProviders(light)

    /**
     * Web-aligned card border (`--color-border` in web/src/styles.css):
     * #E2E8F0 in light, #334155 in dark. Glance has no border modifier, so the
     * widget's row cards paint this colour as a thin ring behind the
     * surface-coloured card to fake the web's 1px border.
     */
    fun cardBorder(context: Context): ColorProvider =
        ColorProvider(if (shouldUseDark(context)) Color(0xFF334155) else Color(0xFFE2E8F0))

    private fun shouldUseDark(context: Context): Boolean = when (ThemePreferences.load(context)) {
        ThemeMode.SYSTEM ->
            (context.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK) ==
                Configuration.UI_MODE_NIGHT_YES
        ThemeMode.LIGHT -> false
        ThemeMode.DARK -> true
    }
}
