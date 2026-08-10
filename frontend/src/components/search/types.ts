export interface SearchResult {
  kind: string
  name: string
  namespace: string
  status?: string
  uid?: string
  score?: number
}