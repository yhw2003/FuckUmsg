export type Todo = {
  id: number
  title: string
  detail: string
  source_type: 'group' | 'private'
  source_id: string
  source_name: string
  sender_id: number
  sender_name: string
  raw_message: string
  message_id: string
  created_at: number
  deadline_at: number
  completed_at: number | null
  status: 'open' | 'done'
}

export type FailedMessage = {
  id: number
  user_id: number
  source_type: 'group' | 'private'
  source_id: string
  message_id: string
  raw_message: string
  fail_stage: 'classify' | 'summarize' | string
  error_text: string
  created_at: number
}

export type LoginResponse = {
  token: string
  expires_at: number
  expires_in: number
}

export type ActiveView = 'todos' | 'failed'
