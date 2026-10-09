import axios from 'axios'
import type {
  HostVO,
  CreateHostDTO,
  UpdateHostDTO,
  TestHostDTO,
  TunnelVO,
  CreateTunnelDTO,
  UpdateTunnelDTO,
  DashboardStats,
  LogEntry,
} from '../types'

const api = axios.create({
  baseURL: '/api',
  timeout: 15000,
})

api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const message = error.response?.data?.error || error.message || '请求失败'
    return Promise.reject(new Error(message))
  }
)

export const hostApi = {
  list: (): Promise<{ data: HostVO[] }> => api.get('/hosts'),
  get: (id: number): Promise<{ data: HostVO }> => api.get(`/hosts/${id}`),
  create: (data: CreateHostDTO): Promise<{ data: HostVO }> => api.post('/hosts', data),
  update: (id: number, data: UpdateHostDTO): Promise<{ data: HostVO }> => api.put(`/hosts/${id}`, data),
  delete: (id: number): Promise<{ message: string }> => api.delete(`/hosts/${id}`),
  testSaved: (id: number): Promise<{ success: boolean; latency: number; banner?: string; error?: string }> =>
    api.post(`/hosts/${id}/test`),
  testRaw: (data: TestHostDTO): Promise<{ success: boolean; latency: number; banner?: string; error?: string }> =>
    api.post('/hosts/test', data),
}

export const tunnelApi = {
  list: (): Promise<{ data: TunnelVO[] }> => api.get('/tunnels'),
  get: (id: number): Promise<{ data: TunnelVO }> => api.get(`/tunnels/${id}`),
  create: (data: CreateTunnelDTO): Promise<{ data: TunnelVO }> => api.post('/tunnels', data),
  update: (id: number, data: UpdateTunnelDTO): Promise<{ data: TunnelVO }> => api.put(`/tunnels/${id}`, data),
  delete: (id: number): Promise<{ message: string }> => api.delete(`/tunnels/${id}`),
  start: (id: number): Promise<{ message: string }> => api.post(`/tunnels/${id}/start`),
  stop: (id: number): Promise<{ message: string }> => api.post(`/tunnels/${id}/stop`),
  restart: (id: number): Promise<{ message: string }> => api.post(`/tunnels/${id}/restart`),
  getStats: (): Promise<{ data: DashboardStats }> => api.get('/dashboard/stats'),
}

export const logApi = {
  list: (params?: { level?: string; tag?: string; tunnel_id?: number; keyword?: string; limit?: number }): Promise<{ data: LogEntry[] }> =>
    api.get('/logs', { params }),
  clear: (): Promise<{ message: string }> => api.delete('/logs'),
}

export default api
