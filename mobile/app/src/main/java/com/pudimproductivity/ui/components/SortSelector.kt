package com.pudimproductivity.ui.components

import androidx.compose.foundation.layout.Box
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pudimproductivity.i18n.Localization
import com.pudimproductivity.utils.SortOption

private fun sortLabelKey(option: SortOption): String = when (option) {
    SortOption.ALPHA_ASC -> "sort.alphaAsc"
    SortOption.ALPHA_DESC -> "sort.alphaDesc"
    SortOption.CREATED_ASC -> "sort.createdAsc"
    SortOption.CREATED_DESC -> "sort.createdDesc"
    SortOption.TIME_ASC -> "sort.timeAsc"
    SortOption.TIME_DESC -> "sort.timeDesc"
}

/**
 * Ordering dropdown for task/habit lists.
 */
@Composable
fun SortSelector(
    value: SortOption,
    options: List<SortOption>,
    onSelect: (SortOption) -> Unit,
    modifier: Modifier = Modifier
) {
    var expanded by remember { mutableStateOf(false) }

    Box(modifier = modifier) {
        TextButton(onClick = { expanded = true }) {
            Text("↕ " + Localization.text(sortLabelKey(value)), style = MaterialTheme.typography.labelMedium)
        }
        DropdownMenu(
            expanded = expanded,
            onDismissRequest = { expanded = false }
        ) {
            options.forEach { option ->
                DropdownMenuItem(
                    text = { Text(Localization.text(sortLabelKey(option))) },
                    onClick = {
                        expanded = false
                        onSelect(option)
                    }
                )
            }
        }
    }
}
