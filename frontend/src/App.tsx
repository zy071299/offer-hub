import { BrowserRouter, Route, Routes } from 'react-router-dom'

import { AppProvider } from '@/components/provider/app-provider'
import Header from '@/components/Header'
import { HomePage } from '@/pages/home-page'
import QuestionCollections from '@/pages/QuestionCollections'
import { NotFoundPage } from '@/pages/not-found-page'
import QuestionDetail from '@/pages/QuestionDetail'
import Questions from '@/pages/Questions'

export default function App() {
  return (
    <AppProvider>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <Header />
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/collections" element={<QuestionCollections />} />
          <Route path="/questions" element={<Questions />} />
          <Route path="/questions/:question_id" element={<QuestionDetail />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </BrowserRouter>
    </AppProvider>
  )
}
