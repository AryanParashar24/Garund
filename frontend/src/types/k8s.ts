export interface Pod {
  name: string
  namespace: string
  status: string
  node: string
}

export interface Namespace {
  name: string
  status: string
}

export interface Deployment {
  name: string
  namespace: string
  replicas: number
}