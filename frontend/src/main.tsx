import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles.css'
import { reportClientError } from './lib/user-errors'

window.addEventListener('error', (event) => reportClientError('unexpected interface error', event.error ?? event.message))
window.addEventListener('unhandledrejection', (event) => reportClientError('unhandled interface task', event.reason))

ReactDOM.createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>)
