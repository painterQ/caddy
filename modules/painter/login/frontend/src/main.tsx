import Background from '@/background/background'
import LoginForm from '@/login/login'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Background></Background>
    <LoginForm></LoginForm>
  </StrictMode>,
)
