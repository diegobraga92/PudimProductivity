import { useMemo, useState } from 'react';
import { Check, ChevronLeft, ChevronRight, Plus, X } from 'lucide-react';
import { addDays, format, isSameDay } from 'date-fns';
import { useStore } from '../store/useStore';

export function DashboardPage() {
  const { lists, tasks, createList, createTask, completeTask, reopenTask, deleteTask } = useStore();

  const [selectedListId, setSelectedListId] = useState<string>('');
  const [newListName, setNewListName] = useState('');
  const [newTaskTitle, setNewTaskTitle] = useState('');
  const [newDailyTaskTitle, setNewDailyTaskTitle] = useState('');
  const [newTodoTaskTitle, setNewTodoTaskTitle] = useState('');
  const [selectedDailyDate, setSelectedDailyDate] = useState(new Date());

  const todayDate = new Date();
  const selectedDailyDateLabel = format(selectedDailyDate, 'EEEE, MMMM d');

  const dailyList = useMemo(() => lists.find((list) => list.type === 'daily'), [lists]);
  const todoList = useMemo(() => lists.find((list) => list.type === 'todo'), [lists]);

  const dailyTasks = useMemo(() => (dailyList ? tasks[dailyList.id] || [] : []), [dailyList, tasks]);
  const todoTasks = useMemo(() => (todoList ? tasks[todoList.id] || [] : []), [todoList, tasks]);

  const todaysDailyTasks = useMemo(() => {
    if (!dailyList) return [];

    const dayOfWeek = selectedDailyDate.getDay();
    const daysOfWeek = dailyList.config.daysOfWeek || [0, 1, 2, 3, 4, 5, 6];

    if (!daysOfWeek.includes(dayOfWeek)) return [];
    return dailyTasks;
  }, [dailyList, dailyTasks, selectedDailyDate]);

  const canGoToNextDailyDate = !isSameDay(selectedDailyDate, todayDate);

  const selectableLists = useMemo(() => lists.filter((list) => list.type === 'collection'), [lists]);

  const effectiveSelectedListId = useMemo(() => {
    if (selectableLists.length === 0) return '';
    const selectedStillExists = selectableLists.some((list) => list.id === selectedListId);
    return selectedStillExists ? selectedListId : selectableLists[0].id;
  }, [selectableLists, selectedListId]);

  const selectedList = useMemo(
    () => selectableLists.find((list) => list.id === effectiveSelectedListId),
    [selectableLists, effectiveSelectedListId]
  );

  const selectedListTasks = useMemo(
    () => (selectedList ? tasks[selectedList.id] || [] : []),
    [selectedList, tasks]
  );

  const handleCreateList = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newListName.trim()) return;

    await createList(newListName.trim(), 'collection');
    setNewListName('');
  };

  const handleCreateDailyTask = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDailyTaskTitle.trim() || !dailyList) return;

    await createTask({
      listId: dailyList.id,
      title: newDailyTaskTitle,
      completed: false,
      order: dailyTasks.length + 1,
    });

    setNewDailyTaskTitle('');
  };

  const handleCreateTodoTask = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTodoTaskTitle.trim() || !todoList) return;

    await createTask({
      listId: todoList.id,
      title: newTodoTaskTitle,
      completed: false,
      order: todoTasks.length + 1,
    });

    setNewTodoTaskTitle('');
  };

  const handleCreateTask = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTaskTitle.trim() || !selectedList) return;

    await createTask({
      listId: selectedList.id,
      title: newTaskTitle,
      completed: false,
      order: selectedListTasks.length + 1,
    });

    setNewTaskTitle('');
  };

  const renderTask = (
    task: { id: string; title: string; completed: boolean; completedAt?: string },
    isDaily = false
  ) => {
    const isDone = isDaily
      ? task.completed && !!task.completedAt && isSameDay(new Date(task.completedAt), selectedDailyDate)
      : task.completed;

    return (
      <div
        key={task.id}
        className="px-4 py-3 flex items-center justify-between gap-3 border-b border-gray-100 last:border-b-0"
      >
        <div className="flex items-center gap-3 flex-1">
          <button
            onClick={() => (isDone ? reopenTask(task.id) : completeTask(task.id))}
            className={`w-5 h-5 rounded-sm border flex items-center justify-center ${
              isDone ? 'bg-green-500 border-green-500' : 'border-gray-300 hover:border-green-500'
            }`}
          >
            {isDone && <Check className="h-3 w-3 text-white" />}
          </button>
          <span className={isDone ? 'text-gray-500 line-through' : 'text-gray-800'}>{task.title}</span>
        </div>
        <button
          onClick={() => deleteTask(task.id)}
          className="p-1 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-md transition-colors"
          aria-label="Delete task"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    );
  };

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <section className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden">
          <div className="px-4 py-3 border-b border-gray-200">
            <div className="flex items-center justify-between gap-3">
              <button
                type="button"
                onClick={() => setSelectedDailyDate((prev) => addDays(prev, -1))}
                className="p-1 rounded-md hover:bg-gray-100 text-gray-600"
                aria-label="View previous day"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>

              <h2 className="text-lg font-semibold text-gray-800">Daily</h2>

              <button
                type="button"
                onClick={() => {
                  if (!canGoToNextDailyDate) return;
                  setSelectedDailyDate((prev) => addDays(prev, 1));
                }}
                disabled={!canGoToNextDailyDate}
                className="p-1 rounded-md hover:bg-gray-100 text-gray-600 disabled:opacity-40 disabled:cursor-not-allowed"
                aria-label="View next day"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
            <p className="text-sm text-gray-500">{selectedDailyDateLabel} · Tasks for your recurring routine</p>
          </div>

          <form onSubmit={handleCreateDailyTask} className="px-4 py-3 border-b border-gray-200 flex gap-2">
            <input
              type="text"
              value={newDailyTaskTitle}
              onChange={(e) => setNewDailyTaskTitle(e.target.value)}
              placeholder="Add daily task"
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="submit"
              disabled={!newDailyTaskTitle.trim()}
              className="px-3 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Plus className="h-4 w-4" />
            </button>
          </form>

          <div>
            {!dailyList ? (
              <p className="px-4 py-6 text-sm text-gray-500">Daily list is not available.</p>
            ) : todaysDailyTasks.length === 0 ? (
              <p className="px-4 py-6 text-sm text-gray-500">No daily tasks for today.</p>
            ) : (
              todaysDailyTasks
                .slice()
                .sort((a, b) => a.order - b.order)
                .map((task) => renderTask(task, true))
            )}
          </div>
        </section>

        <section className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden">
          <div className="px-4 py-3 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-800">To Do</h2>
            <p className="text-sm text-gray-500">Main fixed to-do list</p>
          </div>

          <form onSubmit={handleCreateTodoTask} className="px-4 py-3 border-b border-gray-200 flex gap-2">
            <input
              type="text"
              value={newTodoTaskTitle}
              onChange={(e) => setNewTodoTaskTitle(e.target.value)}
              placeholder="Add to do task"
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
            <button
              type="submit"
              disabled={!newTodoTaskTitle.trim()}
              className="px-3 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Plus className="h-4 w-4" />
            </button>
          </form>

          <div>
            {!todoList ? (
              <p className="px-4 py-6 text-sm text-gray-500">To Do list is not available.</p>
            ) : todoTasks.length === 0 ? (
              <p className="px-4 py-6 text-sm text-gray-500">No to do tasks yet.</p>
            ) : (
              todoTasks
                .slice()
                .sort((a, b) => a.order - b.order)
                .map((task) => renderTask(task))
            )}
          </div>
        </section>
      </div>

      <div className="bg-white border border-gray-200 rounded-lg shadow-sm overflow-hidden grid grid-cols-1 lg:grid-cols-12 gap-0">
        <aside className="lg:col-span-4 xl:col-span-3 h-fit min-w-0 border-b lg:border-b-0 lg:border-r border-gray-200">
          <div className="px-4 py-3 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-800">List of Lists</h2>
            <p className="text-sm text-gray-500">Create or select a categorized list</p>
          </div>

          <form onSubmit={handleCreateList} className="px-4 py-3 border-b border-gray-200 flex items-center gap-2 min-w-0">
            <input
              type="text"
              value={newListName}
              onChange={(e) => setNewListName(e.target.value)}
              placeholder="New list name"
              className="flex-1 min-w-0 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="submit"
              disabled={!newListName.trim()}
              className="flex-none px-3 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              aria-label="Create list"
            >
              <Plus className="h-4 w-4" />
            </button>
          </form>

          <div className="py-2">
            {selectableLists.length === 0 ? (
              <p className="px-4 py-3 text-sm text-gray-500">No lists yet. Create your first list.</p>
            ) : (
              selectableLists.map((list) => (
                <button
                  key={list.id}
                  onClick={() => setSelectedListId(list.id)}
                  className={`w-full text-left px-4 py-3 border-l-4 transition-colors ${
                    list.id === effectiveSelectedListId
                      ? 'border-blue-500 bg-blue-50 text-blue-800'
                      : 'border-transparent hover:bg-gray-50 text-gray-700'
                  }`}
                >
                  <div className="font-medium">{list.name}</div>
                  <div className="text-xs text-gray-500 uppercase mt-1">{list.type}</div>
                </button>
              ))
            )}
          </div>
        </aside>

        <section className="lg:col-span-8 xl:col-span-9 min-h-[360px] min-w-0">
          {!selectedList ? (
            <div className="p-6 text-gray-600">Select or create a list to start adding tasks.</div>
          ) : (
            <>
              <div className="px-4 py-3 border-b border-gray-200">
                <h2 className="text-lg font-semibold text-gray-800">{selectedList.name}</h2>
                <p className="text-sm text-gray-500">To Do items for this list</p>
              </div>

              <form onSubmit={handleCreateTask} className="px-4 py-3 border-b border-gray-200 flex gap-2">
                <input
                  type="text"
                  value={newTaskTitle}
                  onChange={(e) => setNewTaskTitle(e.target.value)}
                  placeholder={`Add task to ${selectedList.name}`}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
                />
                <button
                  type="submit"
                  disabled={!newTaskTitle.trim()}
                  className="px-3 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  aria-label="Create task"
                >
                  <Plus className="h-4 w-4" />
                </button>
              </form>

              <div>
                {selectedListTasks.length === 0 ? (
                  <p className="px-4 py-6 text-sm text-gray-500">No tasks yet for this list.</p>
                ) : (
                  selectedListTasks
                    .slice()
                    .sort((a, b) => a.order - b.order)
                    .map((task) => renderTask(task))
                )}
              </div>
            </>
          )}
        </section>
      </div>
    </div>
  );
}
// TODO
