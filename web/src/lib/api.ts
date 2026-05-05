import type { User, List, Task, ApiResponse } from '../types';

type RequestMethod = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';

export class ServerUnavailableError extends Error {
  constructor(message = 'Server is down') {
    super(message);
    this.name = 'ServerUnavailableError';
  }
}

export class ApiClient {
  private readonly baseUrl = import.meta.env.VITE_API_URL ?? 'http://localhost:3000';
  private readonly requestTimeoutMs = 5000;
  private token: string | null = null;

  constructor() {
    console.log(`API: Using backend at ${this.baseUrl}`);
  }

  private async request<T>(
    path: string,
    method: RequestMethod = 'GET',
    body?: unknown
  ): Promise<ApiResponse<T>> {
    let response: Response;
    const controller = new AbortController();
    const timeoutId = window.setTimeout(() => controller.abort(), this.requestTimeoutMs);

    try {
      response = await fetch(`${this.baseUrl}${path}`, {
        method,
        headers: {
          'Content-Type': 'application/json',
          ...(this.token ? { Authorization: `Bearer ${this.token}` } : {}),
        },
        signal: controller.signal,
        body: body ? JSON.stringify(body) : undefined,
      });
    } catch {
      throw new ServerUnavailableError();
    } finally {
      clearTimeout(timeoutId);
    }

    const text = await response.text();
    const parsed = text ? (JSON.parse(text) as unknown) : null;

    if (!response.ok) {
      const errorMessage =
        typeof parsed === 'object' &&
        parsed !== null &&
        'message' in parsed &&
        typeof (parsed as { message: unknown }).message === 'string'
          ? (parsed as { message: string }).message
          : `Request failed: ${response.status} ${response.statusText}`;

      throw new Error(errorMessage);
    }

    if (
      typeof parsed === 'object' &&
      parsed !== null &&
      'data' in parsed &&
      'success' in parsed
    ) {
      return parsed as ApiResponse<T>;
    }

    return {
      data: parsed as T,
      success: true,
    };
  }

  // Auth endpoints
  async login(email: string, password: string): Promise<ApiResponse<{ token: string; user: User }>> {
    const result = await this.request<{ token: string; user: User }>('/auth/login', 'POST', {
      email,
      password,
    });
    this.token = result.data.token;
    return result;
  }

  // List endpoints
  async getLists(): Promise<ApiResponse<List[]>> {
    return this.request<List[]>('/lists');
  }

  async createList(list: Omit<List, 'id' | 'createdAt' | 'updatedAt'>): Promise<ApiResponse<List>> {
    return this.request<List>('/lists', 'POST', list);
  }

  // Task endpoints
  async getTasks(listId: string): Promise<ApiResponse<Task[]>> {
    return this.request<Task[]>(`/lists/${listId}/tasks`);
  }

  async createTask(task: Omit<Task, 'id' | 'createdAt' | 'updatedAt'>): Promise<ApiResponse<Task>> {
    return this.request<Task>('/tasks', 'POST', task);
  }

  async completeTask(id: string): Promise<ApiResponse<Task>> {
    return this.request<Task>(`/tasks/${id}/complete`, 'PATCH');
  }

  async reopenTask(id: string): Promise<ApiResponse<Task>> {
    return this.request<Task>(`/tasks/${id}/reopen`, 'PATCH');
  }

  async deleteTask(id: string): Promise<ApiResponse<void>> {
    return this.request<void>(`/tasks/${id}`, 'DELETE');
  }
}

// Singleton instance
export const api = new ApiClient();
// TODO
