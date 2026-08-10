package com.pudimproductivity.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext

// Pudim brand palette — warm orange, aligned with the web design tokens.
private val BrandOrange = Color(0xFFEA580C)
private val BrandOrangeLight = Color(0xFFF97316)
private val BrandOrangeDark = Color(0xFFC2410C)
private val BrandCream = Color(0xFFFEF6E9)

private val LightColorScheme = lightColorScheme(
    primary = BrandOrange,
    onPrimary = Color.White,
    primaryContainer = Color(0xFFFFDBC7),
    onPrimaryContainer = Color(0xFF3B0A00),
    secondary = BrandOrangeDark,
    onSecondary = Color.White,
    secondaryContainer = Color(0xFFFFDBC7),
    onSecondaryContainer = Color(0xFF2C160B),
    tertiary = Color(0xFF8A5A00),
    onTertiary = Color.White,
    tertiaryContainer = Color(0xFFFFDEA4),
    onTertiaryContainer = Color(0xFF2A1900),
    background = BrandCream,
    onBackground = Color(0xFF1F1B16),
    surface = Color(0xFFFFFBFF),
    onSurface = Color(0xFF1F1B16),
    surfaceVariant = Color(0xFFF2E0D0),
    onSurfaceVariant = Color(0xFF52443C),
    outline = Color(0xFF85736A),
    error = Color(0xFFBA1A1A),
    onError = Color.White,
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002),
)

private val DarkColorScheme = darkColorScheme(
    primary = Color(0xFFFFB59C),
    onPrimary = Color(0xFF5D1800),
    primaryContainer = BrandOrangeDark,
    onPrimaryContainer = Color(0xFFFFDBC7),
    secondary = Color(0xFFE5BFA9),
    onSecondary = Color(0xFF442919),
    secondaryContainer = Color(0xFF5E3F2E),
    onSecondaryContainer = Color(0xFFFFDBC7),
    tertiary = Color(0xFFEFBD69),
    onTertiary = Color(0xFF452B00),
    tertiaryContainer = Color(0xFF644000),
    onTertiaryContainer = Color(0xFFFFDEA4),
    background = Color(0xFF1F150F),
    onBackground = Color(0xFFF0E0D8),
    surface = Color(0xFF1F150F),
    onSurface = Color(0xFFF0E0D8),
    surfaceVariant = Color(0xFF52443C),
    onSurfaceVariant = Color(0xFFD6C3B8),
    outline = Color(0xFF9E8E84),
    error = Color(0xFFFFB4AB),
    onError = Color(0xFF690005),
    errorContainer = Color(0xFF93000A),
    onErrorContainer = Color(0xFFFFDAD6),
)

@Composable
fun PudimProductivityTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
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
        content = content
    )
}
