import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import { ScenarioListPage } from './pages/ScenarioListPage'
import { ScenarioDetailPage } from './pages/ScenarioDetailPage'
import { CreateScenarioPage } from './pages/CreateScenarioPage'
import { TraceViewerPage } from './pages/TraceViewerPage'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<ScenarioListPage />} />
        <Route path="/scenarios/:id" element={<ScenarioDetailPage />} />
        <Route path="/new" element={<CreateScenarioPage />} />
        <Route path="/trace" element={<TraceViewerPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  )
}

export default App
