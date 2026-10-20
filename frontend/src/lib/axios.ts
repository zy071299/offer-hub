import axios from 'axios'

export const apiClient = axios.create({
  baseURL: '',
  timeout: 10_000,
})

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`)
  }
  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  (error: unknown) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      localStorage.removeItem('token')
    }
    return Promise.reject(error)
  },
)
