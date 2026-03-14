// Auth
export interface LoginRequest {
  username: string
  password: string
  idempotencyKey?: string
}

export interface LoginResponse {
  accessToken: string
  refreshToken: string
  tokenType: string
  expiresIn: number
  userId: string
  username: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
  fullName: string
  phoneNumber?: string
}

export interface RegisterResponse {
  userId: string
  username: string
  email: string
  fullName: string
  createdAt: string
}

// Accounts
export interface Account {
  accountId: string
  accountNumber: string
  accountType: 'SAVINGS' | 'CHECKING'
  balance: number
  currency: string
  status: 'ACTIVE' | 'FROZEN' | 'CLOSED'
  interestRate: number
  createdAt: string
}

export interface CacheInfo {
  servedFromCache: boolean
  cacheKey: string
  evictionPolicy: string
  ttlRemaining: number
}

export interface ReplicaInfo {
  readFromReplica: boolean
  replicaLag: string
}

// Transactions
export interface Transaction {
  transactionId: string
  transactionRef: string
  type: string
  amount: number
  currency: string
  status: 'PENDING' | 'COMPLETED' | 'FAILED' | 'REVERSED'
  description: string
  createdAt: string
}

export interface SagaStep {
  step: string
  stepName?: string
  stepOrder: number
  status: string
  executedAt?: string
  payload?: Record<string, unknown>
  result?: Record<string, unknown>
  order?: number
}

// API Error
export interface ApiError {
  error: string
  lockedUntil?: string
  message?: string
}
