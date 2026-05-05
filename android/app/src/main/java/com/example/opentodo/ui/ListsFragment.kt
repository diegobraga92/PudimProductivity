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
import com.example.opentodo.api.ApiList
import com.example.opentodo.api.ApiTask
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class ListsFragment : Fragment(R.layout.fragment_lists_page) {
    private val TAG = "ListsFragment"
    private val scope = CoroutineScope(Dispatchers.Main)
    private var loadJob: Job? = null
    
    private lateinit var selectedListLabel: TextView
    private lateinit var newTaskInput: EditText
    private lateinit var addTaskButton: Button
    private lateinit var tasksRecycler: RecyclerView
    private lateinit var collectionRecycler: RecyclerView
    
    private var selectedList: ApiList? = null
    private var lists: List<ApiList> = emptyList()
    private var tasks: List<ApiTask> = emptyList()
    
    private lateinit var taskAdapter: TaskListAdapter
    private lateinit var collectionAdapter: CollectionListAdapter

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        selectedListLabel = view.findViewById(R.id.selectedListLabel)
        newTaskInput = view.findViewById(R.id.newCollectionTaskInput)
        addTaskButton = view.findViewById(R.id.addCollectionTaskButton)
        tasksRecycler = view.findViewById(R.id.collectionTaskRecycler)
        collectionRecycler = view.findViewById(R.id.collectionRecycler)

        // Hide the collection creation UI since we're using backend lists
        view.findViewById<EditText>(R.id.newCollectionInput).visibility = View.GONE
        view.findViewById<Button>(R.id.addCollectionButton).visibility = View.GONE

        // Setup task adapter
        taskAdapter = TaskListAdapter(mutableListOf(),
            onTaskToggled = { taskId, completed ->
                toggleTaskCompletion(taskId, completed)
            },
            onTaskDeleted = { taskId ->
                deleteTask(taskId)
            }
        )
        tasksRecycler.layoutManager = LinearLayoutManager(requireContext())
        tasksRecycler.adapter = taskAdapter

        // Setup collection adapter
        collectionAdapter = CollectionListAdapter(mutableListOf()) { uiCollection ->
            // Find the corresponding ApiList
            selectedList = lists.find { it.id == uiCollection.id }
            selectedList?.let {
                loadTasksForList(it.id)
                selectedListLabel.text = it.name
            }
        }
        collectionRecycler.layoutManager = LinearLayoutManager(requireContext())
        collectionRecycler.adapter = collectionAdapter

        // Load lists from backend
        loadLists()

        addTaskButton.setOnClickListener {
            val selected = selectedList ?: return@setOnClickListener
            val title = newTaskInput.text.toString().trim()
            if (title.isEmpty()) return@setOnClickListener

            createTask(selected.id, title)
            newTaskInput.text?.clear()
        }
    }

    private fun loadLists() {
        loadJob?.cancel()
        loadJob = scope.launch {
            try {
                val apiLists = withContext(Dispatchers.IO) {
                    ApiClient.getLists()
                }
                
                lists = apiLists
                collectionAdapter.setCollections(apiLists.map { 
                    UiCollection(id = it.id, name = it.name, tasks = mutableListOf())
                })
                
                // Select first list if available
                if (apiLists.isNotEmpty()) {
                    selectedList = apiLists.first()
                    selectedListLabel.text = selectedList?.name ?: "Select a list"
                    loadTasksForList(selectedList!!.id)
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error loading lists", e)
                Toast.makeText(requireContext(), "Failed to load lists", Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun loadTasksForList(listId: String) {
        scope.launch {
            try {
                val apiTasks = withContext(Dispatchers.IO) {
                    ApiClient.getTasksForList(listId)
                }
                
                tasks = apiTasks
                taskAdapter.setTasks(apiTasks.map {
                    UiTask(id = it.id, title = it.title, completed = it.completed)
                })
            } catch (e: Exception) {
                Log.e(TAG, "Error loading tasks for list $listId", e)
                Toast.makeText(requireContext(), "Failed to load tasks", Toast.LENGTH_SHORT).show()
            }
        }
    }

    private fun createTask(listId: String, title: String) {
        scope.launch {
            try {
                val newTask = withContext(Dispatchers.IO) {
                    ApiClient.createTask(listId, title)
                }
                
                if (newTask != null) {
                    // Add to UI
                    val uiTask = UiTask(id = newTask.id, title = newTask.title, completed = newTask.completed)
                    taskAdapter.addTask(uiTask)
                    Toast.makeText(requireContext(), "Task created", Toast.LENGTH_SHORT).show()
                } else {
                    Toast.makeText(requireContext(), "Failed to create task", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error creating task", e)
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
                    taskAdapter.removeTask(taskId)
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
