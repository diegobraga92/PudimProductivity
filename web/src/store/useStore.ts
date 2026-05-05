import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User, List, Task } from '../types';
import { api, ServerUnavailableError } from '../lib/api';

type AppState = {
  // Auth state
  user: User | null;
  isAuthenticated: boolean;
  
  // Data state
  lists: List[];
  tasks: Record<string, Task[]>; // listId -> tasks
  
  // UI state
  isLoading: boolean;
  error: string | null;
  isServerDown: boolean;
  
  // Actions
  login: (email: string, password: string) => Promise<void>;
  fetchLists: () => Promise<void>;
  createList: (name: string, type?: List['type']) => Promise<void>;
  createTask: (task: Omit<Task, 'id' | 'createdAt' | 'updatedAt'>) => Promise<void>;
  completeTask: (id: string) => Promise<void>;
  reopenTask: (id: string) => Promise<void>;
  deleteTask: (id: string) => Promise<void>;
  clearError: () => void;
};

export const useStore = create<AppState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      isAuthenticated: false,
      lists: [],
      tasks: {},
      isLoading: false,
      error: null,
      isServerDown: false,

      // Actions
      login: async (email: string, password: string) => {
        console.log('STORE: login called with', email);
        set({ isLoading: true, error: null });
        try {
          const response = await api.login(email, password);
          console.log('STORE: login successful', response.data.user);
          set({
            user: response.data.user,
            isAuthenticated: true,
            isServerDown: false,
            isLoading: false
          });
          await get().fetchLists();
        } catch (error: unknown) {
          console.error('STORE: login error', error);
          set({ 
            error: error instanceof Error ? error.message : 'Login failed',
            isServerDown: error instanceof ServerUnavailableError,
            isLoading: false 
          });
          throw error;
        }
      },

      fetchLists: async () => {
        console.log('STORE: fetchLists called');
        set({ isLoading: true, error: null });
        try {
          const response = await api.getLists();
          console.log('STORE: fetchLists successful', response.data.length, 'lists');
          const fetchedLists = response.data;

          // Load tasks for all lists so simplified dashboard has complete data.
          const taskResponses = await Promise.all(
            fetchedLists.map(async (list) => {
              const tasksResponse = await api.getTasks(list.id);
              return [list.id, tasksResponse.data] as const;
            })
          );

          const tasksByList = Object.fromEntries(taskResponses);

          set(() => ({
            lists: fetchedLists,
            tasks: tasksByList,
            isServerDown: false,
            isLoading: false
          }));
        } catch (error: unknown) {
          console.error('STORE: fetchLists error', error);
          set({ 
            error: error instanceof Error ? error.message : 'Failed to fetch lists', 
            isServerDown: error instanceof ServerUnavailableError,
            isLoading: false 
          });
        }
      },

      createList: async (name, type = 'todo') => {
        console.log('STORE: createList called', name, type);
        set({ isLoading: true, error: null });
        try {
          const userId = get().user?.id;
          if (!userId) {
            throw new Error('User not authenticated');
          }

          const response = await api.createList({
            userId,
            name,
            type,
            config: {
              showCompleted: false,
            },
          });

          console.log('STORE: createList successful', response.data);

          set((state) => ({
            lists: [...state.lists, response.data],
            tasks: {
              ...state.tasks,
              [response.data.id]: [],
            },
            isServerDown: false,
            isLoading: false,
          }));
        } catch (error: unknown) {
          console.error('STORE: createList error', error);
          set({
            error: error instanceof Error ? error.message : 'Failed to create list',
            isServerDown: error instanceof ServerUnavailableError,
            isLoading: false,
          });
        }
      },

      createTask: async (taskData) => {
        console.log('STORE: createTask called', taskData);
        set({ isLoading: true, error: null });
        try {
          const response = await api.createTask(taskData);
          console.log('STORE: createTask successful', response.data);
          const listId = taskData.listId;
          
          set(state => ({
            tasks: {
              ...state.tasks,
              [listId]: [...(state.tasks[listId] || []), response.data]
            },
            isServerDown: false,
            isLoading: false
          }));
        } catch (error: unknown) {
          console.error('STORE: createTask error', error);
          set({ 
            error: error instanceof Error ? error.message : 'Failed to create task', 
            isServerDown: error instanceof ServerUnavailableError,
            isLoading: false 
          });
        }
      },

      completeTask: async (id) => {
        console.log('STORE: completeTask called', id);
        set({ isLoading: true, error: null });
        try {
          const response = await api.completeTask(id);
          console.log('STORE: completeTask successful', response.data);
          const updatedTask = response.data;
          
          set(state => ({
            tasks: {
              ...state.tasks,
              [updatedTask.listId]: (state.tasks[updatedTask.listId] || [])
                .map(task => task.id === id ? updatedTask : task)
            },
            isServerDown: false,
            isLoading: false
          }));
        } catch (error: unknown) {
          console.error('STORE: completeTask error', error);
          set({ 
            error: error instanceof Error ? error.message : 'Failed to complete task', 
            isServerDown: error instanceof ServerUnavailableError,
            isLoading: false 
          });
        }
      },

      reopenTask: async (id) => {
        console.log('STORE: reopenTask called', id);
        set({ isLoading: true, error: null });
        try {
          const response = await api.reopenTask(id);
          console.log('STORE: reopenTask successful', response.data);
          const updatedTask = response.data;
          
          set(state => ({
            tasks: {
              ...state.tasks,
              [updatedTask.listId]: (state.tasks[updatedTask.listId] || [])
                .map(task => task.id === id ? updatedTask : task)
            },
            isServerDown: false,
            isLoading: false
          }));
        } catch (error: unknown) {
          console.error('STORE: reopenTask error', error);
          set({ 
            error: error instanceof Error ? error.message : 'Failed to reopen task', 
            isServerDown: error instanceof ServerUnavailableError,
            isLoading: false 
          });
        }
      },

      deleteTask: async (id) => {
        console.log('STORE: deleteTask called', id);
        set({ isLoading: true, error: null });
        try {
          // First, find the task to get its listId
          const state = get();
          let taskListId: string | null = null;
          for (const [listId, taskList] of Object.entries(state.tasks)) {
            const task = taskList.find(t => t.id === id);
            if (task) {
              taskListId = listId;
              break;
            }
          }
          
          if (!taskListId) {
            throw new Error('Task not found');
          }

          await api.deleteTask(id);
          console.log('STORE: deleteTask successful', id);
          
          set(state => ({
            tasks: {
              ...state.tasks,
              [taskListId!]: (state.tasks[taskListId!] || []).filter(task => task.id !== id)
            },
            isServerDown: false,
            isLoading: false
          }));
        } catch (error: unknown) {
          console.error('STORE: deleteTask error', error);
          set({ 
            error: error instanceof Error ? error.message : 'Failed to delete task', 
            isServerDown: error instanceof ServerUnavailableError,
            isLoading: false 
          });
        }
      },

      clearError: () => {
        console.log('STORE: clearError called');
        set({ error: null });
      },
    }),
    {
      name: 'todo-app-storage',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        lists: state.lists,
        tasks: state.tasks
      })
    }
  )
);
// TODO
