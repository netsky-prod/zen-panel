export interface User {
  id: number
  name: string
  uuid: string
  enabled: boolean
  data_limit: number
  data_used: number
  expires_at: string | null
  created_at: string
  updated_at: string
  inbounds?: Inbound[]
}

export interface Node {
  id: number
  name: string
  address: string
  api_port: number
  api_token: string
  enabled: boolean
  created_at: string
  updated_at: string
  status?: 'online' | 'offline'
  inbounds?: Inbound[]
}

export interface Inbound {
  id: number
  node_id: number
  name: string
  protocol: 'reality' | 'ws-tls' | 'hysteria2'
  listen_port: number
  sni: string
  fallback_addr: string
  fallback_port: number
  private_key: string
  public_key: string
  short_id: string
  up_mbps: number
  down_mbps: number
  ws_path: string
  fingerprint: string
  enabled: boolean
  created_at: string
  node?: Node
}

export interface TrafficStats {
  id: number
  user_id: number
  inbound_id: number
  upload: number
  download: number
  recorded_at: string
}

export interface Stats {
  total_users: number
  active_users: number
  total_traffic: number
  total_upload: number
  total_download: number
}

export interface DashboardData {
  stats: Stats
  nodes: NodeStatus[]
  traffic_chart: TrafficChartData[]
}

export interface NodeStatus {
  id: number
  name: string
  address: string
  online: boolean
  users_count: number
  inbounds_count: number
}

export interface TrafficChartData {
  date: string
  upload: number
  download: number
}

export interface UserConfig {
  singbox: object
  share_url: string
  share_urls: ShareUrl[]
}

export interface ShareUrl {
  inbound_name: string
  node_name: string
  url: string
}

export interface AuthUser {
  id: number
  username: string
}

export interface LoginCredentials {
  username: string
  password: string
}

export interface RealityKeys {
  private_key: string
  public_key: string
  short_id: string
}

export interface CreateUserInput {
  name: string
  enabled: boolean
  data_limit: number
  expires_at: string | null
  inbound_ids: number[]
}

export interface UpdateUserInput extends Partial<CreateUserInput> {
  id: number
}

export interface CreateNodeInput {
  name: string
  address: string
  api_port: number
  api_token: string
  enabled: boolean
}

export interface UpdateNodeInput extends Partial<CreateNodeInput> {
  id: number
}

export interface CreateInboundInput {
  node_id: number
  name: string
  protocol: 'reality' | 'ws-tls' | 'hysteria2'
  listen_port: number
  sni?: string
  fallback_addr?: string
  fallback_port?: number
  private_key?: string
  public_key?: string
  short_id?: string
  up_mbps?: number
  down_mbps?: number
  ws_path?: string
  fingerprint?: string
  enabled: boolean
}

export interface UpdateInboundInput extends Partial<CreateInboundInput> {
  id: number
}
