package com.example.opentodo.ui

import android.os.Bundle
import android.util.Log
import android.view.View
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.example.opentodo.R
import com.example.opentodo.api.ApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class DailyFragment : Fragment(R.layout.fragment_task_page) {
    private val TAG = "DailyFragment"
    private val scope = CoroutineScope(Dispatchers.Main)
    private var loadJob: Job? = null
    
    // Daily list ID from backend
    private val DAILY_LIST_ID = "10000000-0000-0000-0000-000000000001"
    
    private lateinit var taskInput: EditText
    private lateinit var addButton: Button
    private lateinit var recycler: RecyclerView
    private lateinit var adapter: TaskListAdapter

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        view.findViewById<TextView>(R.id.pageTitle).text = "Daily"
        view.findViewById<TextView>(R.id.pageSubtitle).text = "Recurring routine tasks"

        taskInput = view.findViewById(R.id.newTaskInput)
        addButton = view.findViewById(R.id.addTaskButton)
        recycler = view.findViewById(R.id.taskRecycler)

        adapter = TaskListAdapter(mutableListOf(),
            onTaskToggled = { taskId, completed ->
                toggleTaskCompletion(taskId, completed)
            },
            onTaskDeleted = { taskId ->
                deleteTask(taskId)
            }
        )
        recycler.layoutManager = LinearLayoutManager(requireContext())
        recycler.adapter = adapter

        // Load tasks from backend
        loadTasks()

        addButton.setOnClickListener {
            val title = taskInput.text.toString().trim()
            if (title.isEmpty()) return@setOnClickListener

            createTask(title)
            taskInput.text?.clear()
        }
    }

    private fun loadTasks() {
        loadJob?.cancel()
        loadJob = scope.launch {
            try {
                val apiTasks = withContext(Dispatchers.IO) {
                    ApiClient.getTasksForList(DAILY_LIST_ID)
                }
                
                adapter.setTasks(apiTasks.map {
                    UiTask(id = it.id, title = it.title, completed = it.completed)
                })
            } catch (e: Exception) {
                Log.e(TAG, "Error loading daily tasks", e)
                Toast.makeText(requireContext(), "Failed to load daily tasks", Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun createTask(title: String) {
        scope.launch {
            try {
                val newTask = withContext(Dispatchers.IO) {
                    ApiClient.createTask(DAILY_LIST_ID, title)
                }
                
                if (newTask != null) {
                    // Add to UI
                    val uiTask = UiTask(id = newTask.id, title = newTask.title, completed = newTask.completed)
                    adapter.addTask(uiTask)
                    Toast.makeText(requireContext(), "Task created", Toast.LENGTH_SHORT).show()
                } else {
                    Toast.makeText(requireContext(), "Failed to create task", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error creating daily task", e)
                Toast.makeText(requireContext(), "Error creating task", Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun toggleTaskCompletion(taskId: String, completed: Boolean) {
        scope.launch {
            try {
                val success = withContext(Dispatchers.IO) {
                    if (completed) {
                        ApiClient.completeTask(taskId)
                    } else {
                        ApiClient.reopenTask(taskId)
                    }
                }
                
                if (!success) {
                    Toast.makeText(requireContext(), "Failed to update task", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error toggling task completion", e)
                Toast.makeText(requireContext(), "Error updating task", Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun deleteTask(taskId: String) {
        scope.launch {
            try {
                val success = withContext(Dispatchers.IO) {
                    ApiClient.deleteTask(taskId)
                }
                
                if (success) {
                    // Remove from UI
                    adapter.removeTask(taskId)
                    Toast.makeText(requireContext(), "Task deleted", Toast.LENGTH_SHORT).show()
                } else {
                    Toast.makeText(requireContext(), "Failed to delete task", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error deleting task", e)
                Toast.makeText(requireContext(), "Error deleting task", Toast.LENGTH_SHORT).show()
            }
        }
    }

    override fun onDestroyView() {
        super.onDestroyView()
        loadJob?.cancel()
    }
}
// TODO
