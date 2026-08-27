package com.pudimproductivity.ui.screens

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * JVM unit tests for the library score formatting (matches the web's
 * `roundScore`/`formatScore` in web/src/pages/Library.tsx).
 */
class LibraryFormatTest {

    @Test
    fun `rounds long decimals to one decimal place`() {
        assertEquals("92.3", formatScore(92.33333333333333))
        assertEquals("93.3", formatScore(93.33333333333334))
        assertEquals("92.5", formatScore(92.45))
    }

    @Test
    fun `drops the trailing dot zero for whole scores`() {
        assertEquals("92", formatScore(92.0))
        assertEquals("92", formatScore(92.00001))
        assertEquals("7", formatScore(7.04))
        assertEquals("100", formatScore(99.999))
    }

    @Test
    fun `handles zero`() {
        assertEquals("0", formatScore(0.0))
        assertEquals("0.5", formatScore(0.5))
    }
}
