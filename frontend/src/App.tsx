import { BrowserRouter, Route, Routes } from 'react-router-dom'

import { AppProvider } from '@/components/provider/app-provider'
import { HomePage } from '@/pages/home-page'
import { NotFoundPage } from '@/pages/not-found-page'
import { QuestionDetailPage } from '@/pages/question-detail-page'
import { QuestionsPage } from '@/pages/questions-page'

export default function App() {
  return (
    <AppProvider>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/questions" element={<QuestionsPage />} />
          <Route path="/questions/:id" element={<QuestionDetailPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </BrowserRouter>
    </AppProvider>
  )
}
