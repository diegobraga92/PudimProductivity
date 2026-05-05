package com.example.opentodo.ui

import java.util.UUID

data class UiTask(
    val id: String = UUID.randomUUID().toString(),
    var title: String,
    var completed: Boolean = false
)

data class UiCollection(
    val id: String = UUID.randomUUID().toString(),
    var name: String,
    val tasks: MutableList<UiTask> = mutableListOf()
)

object InMemoryUiStore {
    val dailyTasks: MutableList<UiTask> = mutableListOf(
        UiTask(title = "Drink water"),
        UiTask(title = "Stretch for 10 minutes")
    )

    val todoTasks: MutableList<UiTask> = mutableListOf(
        UiTask(title = "Finish Android screen"),
        UiTask(title = "Review backend endpoints")
    )

    val collections: MutableList<UiCollection> = mutableListOf(
        UiCollection(
            name = "Shopping",
            tasks = mutableListOf(
                UiTask(title = "Buy milk"),
                UiTask(title = "Buy coffee")
            )
        ),
        UiCollection(
            name = "Project Ideas",
            tasks = mutableListOf(
                UiTask(title = "Sync improvements"),
                UiTask(title = "Widget prototype")
            )
        )
    )
}
// TODO
