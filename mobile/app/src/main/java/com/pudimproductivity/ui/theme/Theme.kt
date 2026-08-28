package com.pudimproductivity.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp

private val BrandOrange = Color(0xFFEA580C)
private val BrandCream = Color(0xFFFEF6E9)

// Web-aligned accent colors: one-off/to-do items use blue on the web.
val TodoAccentLight = Color(0xFF3B82F6)
val TodoAccentDark = Color(0xFF60A5FA)

private val LightColorScheme = lightColorScheme(
    primary = BrandOrange,
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
    background = BrandCream,
    onBackground = Color(0xFF1F1B16),
    surface = Color(0xFFFFFFFF),
    onSurface = Color(0xFF1F1B16),
    surfaceVariant = Color(0xFFF2E0D0),
    onSurfaceVariant = Color(0xFF52443C),
    outline = Color(0xFF85736A),
    error = Color(0xFFEF4444),
    onError = Color.White,
    errorContainer = Color(0xFFFEF2F2),
    onErrorContainer = Color(0xFFB91C1C),
)

private val DarkColorScheme = darkColorScheme(
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
    onErrorContainer = Color(0xFFFFD0CE),
)

// Rounded-corner scale mirrors the web design tokens (--radius-sm/md/lg), so
// cards, chips and buttons feel the same on both platforms.
private val PudimShapes = Shapes(
    extraSmall = RoundedCornerShape(6.dp),
    small = RoundedCornerShape(8.dp),
    medium = RoundedCornerShape(12.dp),
    large = RoundedCornerShape(16.dp),
    extraLarge = RoundedCornerShape(20.dp),
)

@Composable
fun PudimProductivityTheme(
    mode: ThemeMode = ThemeMode.SYSTEM,
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
    // Resolve the actual dark/light flag from the user's theme mode.
    val darkTheme = when (mode) {
        ThemeMode.SYSTEM -> isSystemInDarkTheme()
        ThemeMode.LIGHT -> false
        ThemeMode.DARK -> true
    }

    val colorScheme = when {
        // Brand palette is the default so both platforms share the same orange
        // identity. Dynamic colors (Material You) are opt-in via the flag.
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val context = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }
        darkTheme -> DarkColorScheme
        else -> LightColorScheme
    }

    MaterialTheme(
        colorScheme = colorScheme,
        shapes = PudimShapes,
        content = content
    )
}
