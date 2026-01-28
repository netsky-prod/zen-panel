import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import type {
  User,
  Node,
  Inbound,
  DashboardData,
  UserConfig,
  RealityKeys,
  CreateUserInput,
  UpdateUserInput,
  CreateNodeInput,
  UpdateNodeInput,
  CreateInboundInput,
  UpdateInboundInput,
  LoginCredentials,
  AuthUser,
} from '../types'

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api'

const client = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor for JWT token
client.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor for error handling
client.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ message?: string; error?: string }>) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    const message = error.response?.data?.message || error.response?.data?.error || error.message
    return Promise.reject(new Error(message))
  }
)

// Auth API
export const authApi = {
  login: async (credentials: LoginCredentials): Promise<{ token: string; user: AuthUser }> => {
    const { data } = await client.post('/auth/login', credentials)
    // API returns { success: true, data: { token, admin: { id, username } } }
    return {
      token: data.data.token,
      user: {
        id: data.data.admin.id,
        username: data.data.admin.username,
      },
    }
  },
  logout: async (): Promise<void> => {
    await client.post('/auth/logout')
  },
  me: async (): Promise<AuthUser> => {
    const { data } = await client.get('/auth/me')
    // API returns { success: true, data: { id, username } }
    return data.data
  },
  changePassword: async (oldPassword: string, newPassword: string): Promise<void> => {
    await client.post('/auth/change-password', { old_password: oldPassword, new_password: newPassword })
  },
}

// Users API
export const usersApi = {
  list: async (): Promise<User[]> => {
    const { data } = await client.get('/users')
    return data.data
  },
  get: async (id: number): Promise<User> => {
    const { data } = await client.get(`/users/${id}`)
    return data.data
  },
  create: async (input: CreateUserInput): Promise<User> => {
    const { data } = await client.post('/users', input)
    return data.data
  },
  update: async ({ id, ...input }: UpdateUserInput): Promise<User> => {
    const { data } = await client.put(`/users/${id}`, input)
    return data.data
  },
  delete: async (id: number): Promise<void> => {
    await client.delete(`/users/${id}`)
  },
  getConfig: async (id: number): Promise<UserConfig> => {
    const { data } = await client.get(`/users/${id}/config`)
    return data.data
  },
  resetUUID: async (id: number): Promise<User> => {
    const { data } = await client.post(`/users/${id}/reset-uuid`)
    return data.data
  },
  resetTraffic: async (id: number): Promise<User> => {
    const { data } = await client.post(`/users/${id}/reset-traffic`)
    return data.data
  },
  enable: async (id: number): Promise<User> => {
    const { data } = await client.put(`/users/${id}`, { enabled: true })
    return data.data
  },
  disable: async (id: number): Promise<User> => {
    const { data } = await client.put(`/users/${id}`, { enabled: false })
    return data.data
  },
}

// Nodes API
export const nodesApi = {
  list: async (): Promise<Node[]> => {
    const { data } = await client.get('/nodes')
    return data.data
  },
  get: async (id: number): Promise<Node> => {
    const { data } = await client.get(`/nodes/${id}`)
    return data.data
  },
  create: async (input: CreateNodeInput): Promise<Node> => {
    const { data } = await client.post('/nodes', input)
    return data.data
  },
  update: async ({ id, ...input }: UpdateNodeInput): Promise<Node> => {
    const { data } = await client.put(`/nodes/${id}`, input)
    return data.data
  },
  delete: async (id: number): Promise<void> => {
    await client.delete(`/nodes/${id}`)
  },
  getStatus: async (id: number): Promise<{ online: boolean }> => {
    const { data } = await client.get(`/nodes/${id}/status`)
    return data.data
  },
  sync: async (id: number): Promise<void> => {
    await client.post(`/nodes/${id}/sync`)
  },
}

// Inbounds API
export const inboundsApi = {
  listByNode: async (nodeId: number): Promise<Inbound[]> => {
    const { data } = await client.get(`/nodes/${nodeId}/inbounds`)
    return data.data
  },
  create: async (input: CreateInboundInput): Promise<Inbound> => {
    const { data } = await client.post(`/nodes/${input.node_id}/inbounds`, input)
    return data.data
  },
  update: async ({ id, ...input }: UpdateInboundInput): Promise<Inbound> => {
    const { data } = await client.put(`/inbounds/${id}`, input)
    return data.data
  },
  delete: async (id: number): Promise<void> => {
    await client.delete(`/inbounds/${id}`)
  },
  generateKeys: async (id: number): Promise<RealityKeys> => {
    const { data } = await client.post(`/inbounds/${id}/generate-keys`)
    return data.data
  },
}

// Dashboard API
export const dashboardApi = {
  get: async (): Promise<DashboardData> => {
    const { data } = await client.get('/dashboard')
    return data.data
  },
}

// Stats API
export const statsApi = {
  getOverall: async () => {
    const { data } = await client.get('/stats')
    return data.data
  },
  getUser: async (userId: number) => {
    const { data } = await client.get(`/stats/users/${userId}`)
    return data.data
  },
  getNode: async (nodeId: number) => {
    const { data } = await client.get(`/stats/nodes/${nodeId}`)
    return data.data
  },
}

export default client
