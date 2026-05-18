import ReactDOM from 'react-dom/client'
import App from './App'
import { WindowSyncFloatingToolbar } from './modules/browser/components/WindowSyncFloatingToolbar'
import './index.css'

;(window as Window & { __ANT_APP_BOOTED__?: boolean }).__ANT_APP_BOOTED__ = true

const searchParams = new URLSearchParams(window.location.search)
const isWindowSyncToolbar = searchParams.get('toolbar') === '1'

ReactDOM.createRoot(document.getElementById('root')!).render(
  isWindowSyncToolbar ? <WindowSyncFloatingToolbar /> : <App />,
)

