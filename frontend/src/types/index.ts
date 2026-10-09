export interface HostVO {
  id: number
  created_at: string
  updated_at: string
  name: string
  host: string
  port: number
  username: string
  auth_type: 'password' | 'private_key'
  has_password: boolean
  has_private_key: boolean
  has_passphrase: boolean
  remark: string
}

export interface CreateHostDTO {
  name: string
  host: string
  port: number
  username: string
  auth_type: 'password' | 'private_key'
  password?: string
  private_key?: string
  passphrase?: string
  remark?: string
}

export interface UpdateHostDTO {
  name: string
  host: string
  port: number
  username: string
  auth_type: 'password' | 'private_key'
  password?: string
  private_key?: string
  passphrase?: string
  remark?: string
}

export interface TestHostDTO {
  host: string
  port: number
  username: string
  auth_type: 'password' | 'private_key'
  password?: string
  private_key?: string
  passphrase?: string
}

export interface TunnelRuntime {
  status: 'stopped' | 'starting' | 'running' | 'reconnecting' | 'error'
  last_error: string
  active_conns: number
  bytes_in: number
  bytes_out: number
  uptime: number
  connected_at?: string
  health?: 'healthy' | 'unhealthy' | 'unknown'
  health_message?: string
}

export interface TunnelVO {
  id: number
  created_at: string
  updated_at: string
  name: string
  host_id: number
  host?: HostVO
  type: 'forward' | 'reverse'
  listen_host: string
  listen_port: number
  target_host: string
  target_port: number
  auto_start: boolean
  remark: string
  runtime: TunnelRuntime
}

export interface CreateTunnelDTO {
  name: string
  host_id: number
  type: 'forward' | 'reverse'
  listen_host: string
  listen_port: number
  target_host: string
  target_port: number
  auto_start: boolean
  remark?: string
}

export interface UpdateTunnelDTO {
  name: string
  host_id: number
  type: 'forward' | 'reverse'
  listen_host: string
  listen_port: number
  target_host: string
  target_port: number
  auto_start: boolean
  remark?: string
}

export interface DashboardStats {
  total_hosts: number
  total_tunnels: number
  running_tunnels: number
  total_conns: number
  total_bytes_in: number
  total_bytes_out: number
}

export interface LogEntry {
  id: number
  timestamp: string
  level: 'INFO' | 'WARN' | 'ERROR' | 'SUCCESS'
  tag: string
  tunnel_id?: number
  tunnel_name?: string
  message: string
}

export interface AuthStatusVO {
  initialized: boolean
  from_env: boolean
}

export interface LoginResponse {
  token: string
  expires_in: number
  message?: string
}
