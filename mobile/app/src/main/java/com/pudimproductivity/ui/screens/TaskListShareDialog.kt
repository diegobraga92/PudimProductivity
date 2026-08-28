package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.ShareTaskListRequest
import com.pudimproductivity.api.TaskList
import com.pudimproductivity.api.TaskListMember
import com.pudimproductivity.api.taskService
import com.pudimproductivity.i18n.Localization
import kotlinx.coroutines.launch

/**
 * Share dialog for a task list. Owners can invite (editor/viewer) and
 * revoke access. All members see the member list with live presence dots.
 */
@Composable
fun TaskListShareDialog(
    taskList: TaskList,
    onClose: () -> Unit
) {
    val scope = rememberCoroutineScope()
    val isOwner = taskList.owner_id == "dev-user"

    var members by remember { mutableStateOf<List<TaskListMember>>(emptyList()) }
    var onlineIds by remember { mutableStateOf<Set<String>>(emptySet()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    var inviteUserId by remember { mutableStateOf("") }
    var inviteRole by remember { mutableStateOf("editor") }
    var inviteError by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }

    fun loadData() {
        scope.launch {
            isLoading = true
            error = null
            try {
                members = ApiClient.taskService.listTaskListMembers(taskList.id)
                onlineIds = ApiClient.taskService.getListPresence(taskList.id).online.toSet()
            } catch (e: Exception) {
                error = e.message ?: Localization.text("mobile.share.load.failed")
            } finally {
                isLoading = false
            }
        }
    }

    LaunchedEffect(taskList.id) { loadData() }

    AlertDialog(
        onDismissRequest = onClose,
        title = { Text(Localization.text("share.title", "name" to taskList.name)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (error != null) {
                    Text(error ?: "", color = MaterialTheme.colorScheme.error)
                }

                if (isOwner) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        OutlinedTextField(
                            value = inviteUserId,
                            onValueChange = { inviteUserId = it },
                            label = { Text(Localization.text("share.userId")) },
                            singleLine = true,
                            modifier = Modifier.weight(1f)
                        )
                        FilterChip(
                            selected = inviteRole == "editor",
                            onClick = { inviteRole = "editor" },
                            label = { Text(Localization.text("share.editor")) }
                        )
                        FilterChip(
                            selected = inviteRole == "viewer",
                            onClick = { inviteRole = "viewer" },
                            label = { Text(Localization.text("share.viewer")) }
                        )
                    }
                    if (inviteError != null) {
                        Text(inviteError ?: "", color = MaterialTheme.colorScheme.error)
                    }
                    Button(
                        enabled = inviteUserId.isNotBlank() && !busy,
                        onClick = {
                            scope.launch {
                                busy = true
                                inviteError = null
                                try {
                                    ApiClient.taskService.shareTaskList(
                                        taskList.id,
                                        ShareTaskListRequest(inviteUserId.trim(), inviteRole)
                                    )
                                    inviteUserId = ""
                                    loadData()
                                } catch (e: Exception) {
                                    inviteError = e.message ?: Localization.text("mobile.share.invite.failed")
                                } finally {
                                    busy = false
                                }
                            }
                        }
                    ) {
                        Text(Localization.text("share.invite"))
                    }
                } else {
                    Text(Localization.text("share.notOwner"))
                }


                if (isLoading) {
                    Text(Localization.text("share.loadingMembers"))
                } else {
                    LazyColumn(modifier = Modifier.heightIn(max = 240.dp)) {
                        items(members, key = { it.shared_with }) { member ->
                            val online = member.shared_with in onlineIds
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(vertical = 4.dp),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                // Presence dot
                                Surface(
                                    modifier = Modifier
                                        .padding(end = 8.dp)
                                        .size(8.dp),
                                    shape = MaterialTheme.shapes.small,
                                    color = if (online) {
                                        MaterialTheme.colorScheme.primary
                                    } else {
                                        MaterialTheme.colorScheme.outlineVariant
                                    }
                                ) {}
                                Text(
                                    text = member.shared_with,
                                    style = MaterialTheme.typography.bodyMedium,
                                    modifier = Modifier.weight(1f)
                                )
                                Text(
                                    text = member.role,
                                    style = MaterialTheme.typography.labelMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                                if (isOwner) {
                                    IconButton(
                                        onClick = {
                                            scope.launch {
                                                try {
                                                    ApiClient.taskService.unshareTaskList(taskList.id, member.shared_with)
                                                    loadData()
                                                } catch (e: Exception) {
                                                    error = e.message ?: Localization.text("mobile.share.revoke.failed")
                                                }
                                            }
                                        }
                                    ) {
                                        Text("✕")
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onClose) { Text(Localization.text("common.close")) }
        }
    )
}
