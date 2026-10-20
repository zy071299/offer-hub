import ReactMarkdown from 'react-markdown'

import { Button } from '@/components/ui/button'
import { useLogin } from '@/hooks/useLogin'

interface DetailContentProps {
  content: string
}

export default function DetailContent({ content }: DetailContentProps) {
  const { isLoggedIn, setShowLoginDialog } = useLogin()

  return (
    <div>
      <div className="relative">
        <div className="question-markdown">
          <ReactMarkdown>{content}</ReactMarkdown>
        </div>
        {!isLoggedIn && (
          <div aria-hidden="true" className="pointer-events-none absolute inset-x-0 bottom-0 h-10 bg-gradient-to-b from-transparent to-white" />
        )}
      </div>
      {!isLoggedIn && (
        <div className="flex justify-center pt-6">
          <Button onClick={() => setShowLoginDialog(true)}>
            登录后查看完整内容
          </Button>
        </div>
      )}
    </div>
  )
}
