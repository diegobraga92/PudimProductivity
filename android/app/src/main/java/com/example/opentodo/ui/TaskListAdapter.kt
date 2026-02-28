package com.example.opentodo.ui

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.CheckBox
import android.widget.TextView
import androidx.recyclerview.widget.RecyclerView
import com.example.opentodo.R

class TaskListAdapter(
    private var tasks: MutableList<UiTask>,
    private val onTaskToggled: ((String, Boolean) -> Unit)? = null,
    private val onTaskDeleted: ((String) -> Unit)? = null
) : RecyclerView.Adapter<TaskListAdapter.TaskViewHolder>() {

    fun setTasks(newTasks: List<UiTask>) {
        tasks.clear()
        tasks.addAll(newTasks)
        notifyDataSetChanged()
    }

    fun addTask(task: UiTask) {
        tasks.add(task)
        notifyItemInserted(tasks.size - 1)
    }

    fun removeTask(taskId: String) {
        val position = tasks.indexOfFirst { it.id == taskId }
        if (position != -1) {
            tasks.removeAt(position)
            notifyItemRemoved(position)
        }
    }

    class TaskViewHolder(view: View) : RecyclerView.ViewHolder(view) {
        val checkBox: CheckBox = view.findViewById(R.id.taskCheckbox)
        val title: TextView = view.findViewById(R.id.taskTitle)
        val deleteButton: TextView = view.findViewById(R.id.deleteButton)
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): TaskViewHolder {
        val view = LayoutInflater.from(parent.context).inflate(R.layout.item_task, parent, false)
        return TaskViewHolder(view)
    }

    override fun getItemCount(): Int = tasks.size

    override fun onBindViewHolder(holder: TaskViewHolder, position: Int) {
        val task = tasks[position]
        holder.title.text = task.title

        holder.checkBox.setOnCheckedChangeListener(null)
        holder.checkBox.isChecked = task.completed
        holder.checkBox.setOnCheckedChangeListener { _, isChecked ->
            task.completed = isChecked
            onTaskToggled?.invoke(task.id, isChecked)
        }

        holder.deleteButton.setOnClickListener {
            onTaskDeleted?.invoke(task.id)
        }
    }
}
// TODO
