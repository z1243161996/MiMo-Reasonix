import { useState, useEffect, useRef } from 'react'
import { Submit, Cancel, GetHistory, GetStatus } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

interface Message {
  role: string
  content: string
}

interface Status {
  running: boolean
  plan: boolean
  label: string
}

function App() {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [status, setStatus] = useState<Status | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  // Load initial history and status
  useEffect(() => {
    const init = async () => {
      try {
        const history = await GetHistory()
        setMessages(history)
        const s = await GetStatus()
        setStatus(s)
      } catch (err) {
        console.error('Failed to load initial data:', err)
      }
    }
    init()
  }, [])

  // Listen for agent events
  useEffect(() => {
    const unsubscribe = EventsOn('agent:event', (data: string) => {
      try {
        const event = JSON.parse(data)
        if (event.kind === 'text' || event.kind === 'message') {
          setMessages(prev => [...prev, {
            role: 'assistant',
            content: event.text
          }])
        } else if (event.kind === 'notice') {
          setMessages(prev => [...prev, {
            role: 'system',
            content: event.text
          }])
        }
      } catch (err) {
        console.error('Failed to parse event:', err)
      }
    })

    return () => {
      unsubscribe()
    }
  }, [])

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSubmit = async () => {
    if (!input.trim()) return

    const userMessage = input.trim()
    setInput('')

    // Add user message to UI
    setMessages(prev => [...prev, {
      role: 'user',
      content: userMessage
    }])

    try {
      await Submit(userMessage)
    } catch (err) {
      console.error('Submit failed:', err)
      setMessages(prev => [...prev, {
        role: 'system',
        content: `Error: ${err}`
      }])
    }
  }

  const handleCancel = async () => {
    try {
      await Cancel()
    } catch (err) {
      console.error('Cancel failed:', err)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <div className="app">
      <div className="sidebar">
        <h3>MiMo-Reasonix</h3>
        <p>Model: {status?.label || 'Loading...'}</p>
        <p>Status: {status?.running ? 'Running' : 'Idle'}</p>
        {status?.plan && <p>Mode: Plan</p>}
      </div>

      <div className="chat">
        <div className="messages">
          {messages.map((msg, i) => (
            <div key={i} className={`message ${msg.role}`}>
              <strong>{msg.role}: </strong>
              {msg.content}
            </div>
          ))}
          <div ref={messagesEndRef} />
        </div>

        <div className="input-area">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a message..."
            disabled={status?.running}
          />
          <button onClick={handleSubmit} disabled={status?.running || !input.trim()}>
            Send
          </button>
          <button onClick={handleCancel} disabled={!status?.running}>
            Cancel
          </button>
        </div>
      </div>
    </div>
  )
}

export default App
